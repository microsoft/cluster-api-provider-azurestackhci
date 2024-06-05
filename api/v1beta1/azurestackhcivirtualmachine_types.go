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

package v1beta1

import (
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// VirtualMachineFinalizer allows ReconcileVirtualAzureStackHCIMachine to clean up AzureStackHCI resources associated with VirtualAzureStackHCIMachine before
	// removing it from the apiserver.
	VirtualMachineFinalizer = "azurestackhcivirtualmachine.infrastructure.cluster.x-k8s.io"
)

// AzureStackHCIVirtualMachineSpec defines the desired state of AzureStackHCIVirtualMachine
type AzureStackHCIVirtualMachineSpec struct {
	VMSize           string           `json:"vmSize"`
	AvailabilityZone AvailabilityZone `json:"availabilityZone,omitempty"`
	Image            Image            `json:"image"`
	OSDisk           OSDisk           `json:"osDisk,omitempty"`
	BootstrapData    *string          `json:"bootstrapData,omitempty"`
	Identity         VMIdentity       `json:"identity,omitempty"`
	Location         string           `json:"location"` // does location belong here?
	SSHPublicKey     string           `json:"sshPublicKey"`

	// +optional
	StorageContainer string `json:"storageContainer"`

	// come from the cluster scope for machine and lb controller creation path
	ResourceGroup    string   `json:"resourceGroup"`
	VnetName         string   `json:"vnetName"`
	ClusterName      string   `json:"clusterName"`
	SubnetName       string   `json:"subnetName"`
	BackendPoolNames []string `json:"backendPoolNames,omitempty"`

	AdditionalSSHKeys []string `json:"additionalSSHKeys,omitempty"`

	// +optional
	NetworkInterfaces NetworkInterfaces `json:"networkInterfaces,omitempty"`

	// +optional
	AvailabilitySetName string `json:"availabilitySetName,omitempty"`
}

// AzureStackHCIVirtualMachineStatus defines the observed state of AzureStackHCIVirtualMachine
type AzureStackHCIVirtualMachineStatus struct {
	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the AzureStackHCI instance associated addresses.
	Addresses []v1core.NodeAddress `json:"addresses,omitempty"`

	// VMState is the provisioning state of the AzureStackHCI virtual machine.
	// +optional
	VMState *VMState `json:"vmState,omitempty"`

	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the AzureStackHCIVirtualMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=azurestackhcivirtualmachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// AzureStackHCIVirtualMachine is the Schema for the azurestackhcivirtualmachines API
type AzureStackHCIVirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureStackHCIVirtualMachineSpec   `json:"spec,omitempty"`
	Status AzureStackHCIVirtualMachineStatus `json:"status,omitempty"`
}

// GetConditions returns the list of conditions for the AzureStackHCIVirtualMachine.
func (m *AzureStackHCIVirtualMachine) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

// SetConditions sets the conditions for the AzureStackHCIVirtualMachine.
func (m *AzureStackHCIVirtualMachine) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}

// VirtualMachinesByCreationTimestamp sorts a list of AzureStackHCIVirtualMachine by creation timestamp, using their names as a tie breaker.
type VirtualMachinesByCreationTimestamp []*AzureStackHCIVirtualMachine

func (o VirtualMachinesByCreationTimestamp) Len() int      { return len(o) }
func (o VirtualMachinesByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o VirtualMachinesByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}

// +kubebuilder:object:root=true

// AzureStackHCIVirtualMachineList contains a list of AzureStackHCIVirtualMachine
type AzureStackHCIVirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureStackHCIVirtualMachine `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &AzureStackHCIVirtualMachine{}, &AzureStackHCIVirtualMachineList{})
}
