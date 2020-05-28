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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// AzureStackHCILoadBalancerFinalizer allows ReconcileLoadBalancer to clean up the load balancer resources before removing it from the apiserver.
	AzureStackHCILoadBalancerFinalizer = "azurestackhciloadbalancer.infrastructure.cluster.x-k8s.io"
)

type AzureStackHCILoadBalancerSpec struct {
	SSHPublicKey string `json:"sshPublicKey"`
	Image        Image  `json:"image"`
	VMSize       string `json:"vmSize"`
}

type AzureStackHCILoadBalancerStatus struct {
	// +optional
	Ready bool `json:"ready,omitempty"`

	// VMState is the provisioning state of the AzureStackHCI virtual machine.
	// +optional
	VMState *VMState `json:"vmState,omitempty"`

	// Address is the IP address of the load balancer.
	// +optional
	Address string `json:"address,omitempty"`

	// Port is the port of the azureStackHCIloadbalancers frontend.
	Port int32 `json:"port,omitempty"`

	// ErrorReason will be set in the event that there is a terminal problem
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
	ErrorReason *errors.MachineStatusError `json:"errorReason,omitempty"`

	// ErrorMessage will be set in the event that there is a terminal problem
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
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=azurestackhciloadbalancers,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status

// AzureStackHCILoadBalancer is the Schema for the azurestackhciloadbalancers API
type AzureStackHCILoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureStackHCILoadBalancerSpec   `json:"spec,omitempty"`
	Status AzureStackHCILoadBalancerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AzureStackHCILoadBalancerList contains a list of AzureStackHCILoadBalancers
type AzureStackHCILoadBalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureStackHCILoadBalancer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzureStackHCILoadBalancer{}, &AzureStackHCILoadBalancerList{})
}
