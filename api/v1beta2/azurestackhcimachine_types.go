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

package v1beta2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// MachineFinalizer allows ReconcileAzureStackHCIMachine to clean up Azure resources associated with AzureStackHCIMachine before
	// removing it from the apiserver.
	MachineFinalizer = "azurestackhcimachine.infrastructure.cluster.x-k8s.io"
)

// AzureStackHCIMachineSpec defines the desired state of AzureStackHCIMachine
type AzureStackHCIMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	VMSize string `json:"vmSize"`

	// +optional
	AvailabilityZone *AvailabilityZone `json:"availabilityZone,omitempty"`

	// +optional
	Image *Image `json:"image,omitempty"`

	// +optional
	OSDisk *OSDisk `json:"osDisk,omitempty"`

	Location string `json:"location"`

	SSHPublicKey string `json:"sshPublicKey"`

	// +optional
	StorageContainer string `json:"storageContainer"`

	GpuCount int32 `json:"gpuCount,omitempty"`

	// AllocatePublicIP allows the ability to create dynamic public ips for machines where this value is true.
	// +optional
	AllocatePublicIP bool `json:"allocatePublicIP,omitempty"`

	AdditionalSSHKeys []string `json:"additionalSSHKeys,omitempty"`

	// +optional
	NetworkInterfaces NetworkInterfaces `json:"networkInterfaces,omitempty"`

	// +optional
	AvailabilitySetName string `json:"availabilitySetName,omitempty"`

	// +optional
	PlacementGroupName string `json:"placementGroupName,omitempty"`
}

// AzureStackHCIMachineStatus defines the observed state of AzureStackHCIMachine
type AzureStackHCIMachineStatus struct {
	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the Azure instance associated addresses.
	Addresses []v1.NodeAddress `json:"addresses,omitempty"`

	// VMState is the provisioning state of the Azure virtual machine.
	// +optional
	VMState *VMState `json:"vmState,omitempty"`

	// Conditions defines current service state of the AzureStackHCIMachine.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Initialization provides observations of the AzureStackHCIMachine initialization process.
	// +optional
	Initialization *AzureStackHCIMachineInitializationStatus `json:"initialization,omitempty,omitzero"`
}

// AzureStackHCIMachineInitializationStatus provides observations of the AzureStackHCIMachine initialization process.
// +kubebuilder:validation:MinProperties=1
type AzureStackHCIMachineInitializationStatus struct {
	// Provisioned is true when the infrastructure provider reports that the Machine's infrastructure is fully provisioned.
	// NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Machine provisioning.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=azurestackhcimachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:subresource:status

// AzureStackHCIMachine is the Schema for the azurestackhcimachines API
type AzureStackHCIMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureStackHCIMachineSpec   `json:"spec,omitempty"`
	Status AzureStackHCIMachineStatus `json:"status,omitempty"`
}

func (c *AzureStackHCIMachine) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

func (c *AzureStackHCIMachine) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

// +kubebuilder:object:root=true

// AzureStackHCIMachineList contains a list of AzureStackHCIMachine
type AzureStackHCIMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureStackHCIMachine `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &AzureStackHCIMachine{}, &AzureStackHCIMachineList{})
}
