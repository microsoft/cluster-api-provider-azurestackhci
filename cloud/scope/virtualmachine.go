/*
Copyright 2020 The Kubernetes Authors.
Portions Copyright © Microsoft Corporation.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scope

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha3"
	azhciauth "github.com/microsoft/cluster-api-provider-azurestackhci/pkg/auth"
	"github.com/microsoft/moc/pkg/auth"
	"github.com/pkg/errors"
	"k8s.io/klog/klogr"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams defines the input parameters used to create a new VirtualMachineScope.
type VirtualMachineScopeParams struct {
	AzureStackHCIClients
	Client                      client.Client
	Logger                      logr.Logger
	AzureStackHCIVirtualMachine *infrav1.AzureStackHCIVirtualMachine
}

// NewMachineScope creates a new VirtualMachineScope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewVirtualMachineScope(params VirtualMachineScopeParams) (*VirtualMachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a VirtualMachineScope")
	}

	if params.AzureStackHCIVirtualMachine == nil {
		return nil, errors.New("azurestackhci virtual machine is required when creating a VirtualMachineScope")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	agentFqdn := os.Getenv("AZURESTACKHCI_CLOUDAGENT_FQDN")
	if agentFqdn == "" {
		return nil, errors.New("error creating azurestackhci services. Environment variable AZURESTACKHCI_CLOUDAGENT_FQDN is not set")
	}
	params.AzureStackHCIClients.CloudAgentFqdn = agentFqdn

	authorizer, err := azhciauth.ReconcileAzureStackHCIAccess(context.Background(), params.Client, agentFqdn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create azurestackhci session")
	}
	params.AzureStackHCIClients.Authorizer = authorizer

	helper, err := patch.NewHelper(params.AzureStackHCIVirtualMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &VirtualMachineScope{
		client:                      params.Client,
		AzureStackHCIVirtualMachine: params.AzureStackHCIVirtualMachine,
		AzureStackHCIClients:        params.AzureStackHCIClients,
		Logger:                      params.Logger,
		patchHelper:                 helper,
		Context:                     context.Background(),
	}, nil
}

// VirtualMachineScope defines a scope defined around a machine.
type VirtualMachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper
	Context     context.Context

	AzureStackHCIClients
	AzureStackHCIVirtualMachine *infrav1.AzureStackHCIVirtualMachine
}

// GetResourceGroup allows VirtualMachineScope to fulfill ScopeInterface and thus to be used by the cloud services.
func (m *VirtualMachineScope) GetResourceGroup() string {
	return m.AzureStackHCIVirtualMachine.Spec.ResourceGroup
}

// GetCloudAgentFqdn returns the cloud agent fqdn string.
func (m *VirtualMachineScope) GetCloudAgentFqdn() string {
	return m.CloudAgentFqdn
}

// GetAuthorizer is a getter for the environment generated authorizer.
func (m *VirtualMachineScope) GetAuthorizer() auth.Authorizer {
	return m.Authorizer
}

// VnetName returns the vnet name given in the vm spec.
func (m *VirtualMachineScope) VnetName() string {
	return m.AzureStackHCIVirtualMachine.Spec.VnetName
}

// SubnetName returns the subnet name given in the vm spec.
func (m *VirtualMachineScope) SubnetName() string {
	return m.AzureStackHCIVirtualMachine.Spec.SubnetName
}

// ClusterName returns the cluster name in the vm spec.
func (m *VirtualMachineScope) ClusterName() string {
	return m.AzureStackHCIVirtualMachine.Spec.ClusterName
}

// Location returns the AzureStackHCIVirtualMachine location.
func (m *VirtualMachineScope) Location() string {
	return m.AzureStackHCIVirtualMachine.Spec.Location
}

// AvailabilityZone returns the AzureStackHCIVirtualMachine Availability Zone.
func (m *VirtualMachineScope) AvailabilityZone() string {
	return *m.AzureStackHCIVirtualMachine.Spec.AvailabilityZone.ID
}

// Name returns the AzureStackHCIVirtualMachine name.
func (m *VirtualMachineScope) Name() string {
	return m.AzureStackHCIVirtualMachine.Name
}

// Namespace returns the namespace name.
func (m *VirtualMachineScope) Namespace() string {
	return m.AzureStackHCIVirtualMachine.Namespace
}

// GetVMState returns the AzureStackHCIVirtualMachine VM state.
func (m *VirtualMachineScope) GetVMState() *infrav1.VMState {
	return m.AzureStackHCIVirtualMachine.Status.VMState
}

// SetVMState sets the AzureStackHCIVirtualMachine VM state.
func (m *VirtualMachineScope) SetVMState(v infrav1.VMState) {
	m.AzureStackHCIVirtualMachine.Status.VMState = new(infrav1.VMState)
	*m.AzureStackHCIVirtualMachine.Status.VMState = v
}

// SetReady sets the AzureStackHCIVirtualMachine Ready Status
func (m *VirtualMachineScope) SetReady() {
	m.AzureStackHCIVirtualMachine.Status.Ready = true
}

// SetFailureMessage sets the AzureStackHCIVirtualMachine status failure message.
func (m *VirtualMachineScope) SetFailureMessage(v error) {
	m.AzureStackHCIVirtualMachine.Status.FailureMessage = pointer.StringPtr(v.Error())
}

// SetFailureReason sets the AzureStackHCIVirtualMachine status failure reason.
func (m *VirtualMachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.AzureStackHCIVirtualMachine.Status.FailureReason = &v
}

// SetAnnotation sets a key value annotation on the AzureStackHCIVirtualMachine.
func (m *VirtualMachineScope) SetAnnotation(key, value string) {
	if m.AzureStackHCIVirtualMachine.Annotations == nil {
		m.AzureStackHCIVirtualMachine.Annotations = map[string]string{}
	}
	m.AzureStackHCIVirtualMachine.Annotations[key] = value
}

// SetResourceName sets the AzureStackHCIVirtualMachine resource name.
func (m *VirtualMachineScope) SetResourceName(resourceName string) {
	m.AzureStackHCIVirtualMachine.Status.ResourceName = resourceName
}

// PatchObject persists the virtual machine spec and status.
func (m *VirtualMachineScope) PatchObject() error {
	conditions.SetSummary(m.AzureStackHCIVirtualMachine,
		conditions.WithConditions(
			infrav1.VMRunningCondition,
		),
		conditions.WithStepCounterIfOnly(
			infrav1.VMRunningCondition,
		),
	)

	return m.patchHelper.Patch(m.Context,
		m.AzureStackHCIVirtualMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.VMRunningCondition,
		}})

}

// Close the VirtualMachineScope by updating the machine spec, machine status.
func (m *VirtualMachineScope) Close() error {
	return m.PatchObject()
}

// AzureStackHCILoadBalancerVM returns true if the AzureStackHCIVirtualMachine is owned by a LoadBalancer resource and false otherwise (Tenant).
func (m *VirtualMachineScope) AzureStackHCILoadBalancerVM() bool {
	for _, ref := range m.AzureStackHCIVirtualMachine.ObjectMeta.GetOwnerReferences() {
		m.Info("owner references", "type", ref.Kind)
		if ref.Kind == "AzureStackHCILoadBalancer" && ref.APIVersion == m.AzureStackHCIVirtualMachine.APIVersion {
			return true
		}
	}
	return false
}

// BackendPoolNames returns the backend pool name for the virtual machine
func (m *VirtualMachineScope) BackendPoolNames() []string {
	return m.AzureStackHCIVirtualMachine.Spec.BackendPoolNames
}
