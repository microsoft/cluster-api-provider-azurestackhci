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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
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

	AvailabilityZone AvailabilityZone `json:"availabilityZone,omitempty"`

	Image Image `json:"image,omitempty"`

	OSDisk OSDisk `json:"osDisk,omitempty"`

	Location string `json:"location"`

	SSHPublicKey string `json:"sshPublicKey"`

	// AllocatePublicIP allows the ability to create dynamic public ips for machines where this value is true.
	// +optional
	AllocatePublicIP bool `json:"allocatePublicIP,omitempty"`

	AdditionalSSHKeys []string `json:"additionalSSHKeys,omitempty"`

	// +optional
	NetworkInterfaces NetworkInterfaces `json:"networkInterfaces,omitempty"`
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
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
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

func (c *AzureStackHCIMachine) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

func (c *AzureStackHCIMachine) SetConditions(conditions clusterv1.Conditions) {
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
	SchemeBuilder.Register(&AzureStackHCIMachine{}, &AzureStackHCIMachineList{})
}
