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
	"encoding/base64"

	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/klogr"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	AzureStackHCIClients
	Client               client.Client
	Logger               *logr.Logger
	Cluster              *clusterv1.Cluster
	Machine              *clusterv1.Machine
	AzureStackHCICluster *infrav1.AzureStackHCICluster
	AzureStackHCIMachine *infrav1.AzureStackHCIMachine
}

// NewMachineScope creates a new MachineScope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("machine is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("cluster is required when creating a MachineScope")
	}
	if params.AzureStackHCICluster == nil {
		return nil, errors.New("azurestackhci cluster is required when creating a MachineScope")
	}
	if params.AzureStackHCIMachine == nil {
		return nil, errors.New("azurestackhci machine is required when creating a MachineScope")
	}

	if params.Logger == nil {
		log := klogr.New()
		params.Logger = &log
	}

	helper, err := patch.NewHelper(params.AzureStackHCIMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &MachineScope{
		client:               params.Client,
		Cluster:              params.Cluster,
		Machine:              params.Machine,
		AzureStackHCICluster: params.AzureStackHCICluster,
		AzureStackHCIMachine: params.AzureStackHCIMachine,
		Logger:               *params.Logger,
		patchHelper:          helper,
	}, nil
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Cluster              *clusterv1.Cluster
	Machine              *clusterv1.Machine
	AzureStackHCICluster *infrav1.AzureStackHCICluster
	AzureStackHCIMachine *infrav1.AzureStackHCIMachine
}

// Location returns the AzureStackHCIMachine location.
func (m *MachineScope) Location() string {
	return m.AzureStackHCICluster.Spec.Location
}

// AvailabilityZone returns the AzureStackHCIMachine Availability Zone.
func (m *MachineScope) AvailabilityZone() string {
	return *m.AzureStackHCIMachine.Spec.AvailabilityZone.ID
}

// Name returns the AzureStackHCIMachine name.
func (m *MachineScope) Name() string {
	return m.AzureStackHCIMachine.Name
}

// Namespace returns the namespace name.
func (m *MachineScope) Namespace() string {
	return m.AzureStackHCIMachine.Namespace
}

// IsControlPlane returns true if the machine is a control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// Role returns the machine role from the labels.
func (m *MachineScope) Role() string {
	if util.IsControlPlaneMachine(m.Machine) {
		return infrav1.ControlPlane
	}
	return infrav1.Node
}

// GetVMID returns the AzureStackHCIMachine instance id by parsing Spec.ProviderID.
func (m *MachineScope) GetVMID() *string {
	parsed, err := noderefutil.NewProviderID(m.GetProviderID())
	if err != nil {
		return nil
	}
	return pointer.StringPtr(parsed.ID())
}

// GetLogger returns the logger.
func (m *MachineScope) GetLogger() logr.Logger {
	return m.Logger
}

// GetProviderID returns the AzureStackHCIMachine providerID from the spec.
func (m *MachineScope) GetProviderID() string {
	if m.AzureStackHCIMachine.Spec.ProviderID != nil {
		return *m.AzureStackHCIMachine.Spec.ProviderID
	}
	return ""
}

// SetProviderID sets the AzureStackHCIMachine providerID in spec.
func (m *MachineScope) SetProviderID(v string) {
	m.AzureStackHCIMachine.Spec.ProviderID = pointer.StringPtr(v)
}

// GetVMState returns the AzureStackHCIMachine VM state.
func (m *MachineScope) GetVMState() *infrav1.VMState {
	return m.AzureStackHCIMachine.Status.VMState
}

// SetVMState sets the AzureStackHCIMachine VM state.
func (m *MachineScope) SetVMState(v *infrav1.VMState) {
	m.AzureStackHCIMachine.Status.VMState = new(infrav1.VMState)
	*m.AzureStackHCIMachine.Status.VMState = *v
}

// SetReady sets the AzureStackHCIMachine Ready Status
func (m *MachineScope) SetReady() {
	m.AzureStackHCIMachine.Status.Ready = true
}

// SetFailureMessage sets the AzureStackHCIMachine status failure message.
func (m *MachineScope) SetFailureMessage(v error) {
	m.AzureStackHCIMachine.Status.FailureMessage = pointer.StringPtr(v.Error())
}

// SetFailureReason sets the AzureStackHCIMachine status failure reason.
func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.AzureStackHCIMachine.Status.FailureReason = &v
}

// SetAnnotation sets a key value annotation on the AzureStackHCIMachine.
func (m *MachineScope) SetAnnotation(key, value string) {
	if m.AzureStackHCIMachine.Annotations == nil {
		m.AzureStackHCIMachine.Annotations = map[string]string{}
	}
	m.AzureStackHCIMachine.Annotations[key] = value
}

// PatchObject persists the machine spec and status.
func (m *MachineScope) PatchObject() error {
	conditions.SetSummary(m.AzureStackHCIMachine,
		conditions.WithConditions(
			infrav1.VMRunningCondition,
		),
		conditions.WithStepCounterIfOnly(
			infrav1.VMRunningCondition,
		),
	)

	return m.patchHelper.Patch(
		context.TODO(),
		m.AzureStackHCIMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.VMRunningCondition,
		}})
}

// Close the MachineScope by updating the machine spec, machine status.
func (m *MachineScope) Close() error {
	return m.PatchObject()
}

// GetBootstrapData returns the bootstrap data from the secret in the Machine's bootstrap.dataSecretName.
func (m *MachineScope) GetBootstrapData() (string, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.Namespace(), Name: *m.Machine.Spec.Bootstrap.DataSecretName}
	if err := m.client.Get(context.TODO(), key, secret); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for AzureStackHCIMachine %s/%s", m.Namespace(), m.Name())
	}

	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}
	return base64.StdEncoding.EncodeToString(value), nil
}
