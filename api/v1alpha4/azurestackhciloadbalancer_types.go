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

package v1alpha4

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
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
	// Number of desired loadbalancer machines. Defaults to 1.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`
}

type AzureStackHCILoadBalancerStatus struct {
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Total number of non-terminated replicas for this loadbalancer
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Total number of ready (service connected) replicas for this loadbalancer
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Total number of failed replicas for this loadbalancer.
	// +optional
	FailedReplicas int32 `json:"failedReplicas,omitempty"`

	// Address is the IP address of the load balancer.
	// +optional
	Address string `json:"address,omitempty"`

	// Port is the port of the azureStackHCIloadbalancers frontend.
	Port int32 `json:"port,omitempty"`

	// Phase represents the current phase of loadbalancer actuation.
	// E.g. Pending, Running, Terminating, Failed etc.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions defines current service state of the AzureStackHCILoadBalancer.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

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

	// Selector is the label selector in string format to avoid introspection
	// by clients, and is used to provide the CRD-based integration for the
	// scale subresource and additional integrations for things like kubectl
	// describe.. The string will be in the same format as the query-param syntax.
	// More info about label selectors: http://kubernetes.io/docs/user-guide/labels#label-selectors
	// +optional
	Selector string `json:"selector,omitempty"`
}

// SetTypedPhase sets the Phase field to the string representation of AzureStackHCILoadBalancerPhase
func (c *AzureStackHCILoadBalancerStatus) SetTypedPhase(p AzureStackHCILoadBalancerPhase) {
	c.Phase = string(p)
}

// GetTypedPhase attempts to parse the Phase field and return
// the typed AzureStackHCILoadBalancerPhase representation as described in `types.go`.
func (c *AzureStackHCILoadBalancerStatus) GetTypedPhase() AzureStackHCILoadBalancerPhase {
	switch phase := AzureStackHCILoadBalancerPhase(c.Phase); phase {
	case
		AzureStackHCILoadBalancerPhasePending,
		AzureStackHCILoadBalancerPhaseProvisioning,
		AzureStackHCILoadBalancerPhaseProvisioned,
		AzureStackHCILoadBalancerPhaseScaling,
		AzureStackHCILoadBalancerPhaseUpgrading,
		AzureStackHCILoadBalancerPhaseDeleting,
		AzureStackHCILoadBalancerPhaseFailed:
		return phase
	default:
		return AzureStackHCILoadBalancerPhaseUnknown
	}
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=azurestackhciloadbalancers,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="The current phase/status of the loadbalancer"
// +kubebuilder:printcolumn:name="IP",type="string",JSONPath=".status.address",description="The frontend VIP address assigned to the loadbalancer"
// +kubebuilder:printcolumn:name="Port",type="integer",JSONPath=".status.port",description="The frontend port assigned to the loadbalancer"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas",description="Total number of desired machine replicas for this loadbalancer"
// +kubebuilder:printcolumn:name="Created",type="integer",JSONPath=".status.replicas",description="Total number of machine replicas created to service this loadbalancer"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas",description="Total number of machine replicas that are actively connected to the loadbalancer service"
// +kubebuilder:printcolumn:name="Unavailable",type="integer",JSONPath=".status.failedReplicas",description="Total number of machine replicas that are in a failed or unavailable state"

// AzureStackHCILoadBalancer is the Schema for the azurestackhciloadbalancers API
type AzureStackHCILoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureStackHCILoadBalancerSpec   `json:"spec,omitempty"`
	Status AzureStackHCILoadBalancerStatus `json:"status,omitempty"`
}

// GetConditions returns the list of conditions for AzureStackHCILoadBalancer.
func (c *AzureStackHCILoadBalancer) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

// SetConditions sets the conditions for AzureStackHCILoadBalancer.
func (c *AzureStackHCILoadBalancer) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
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
