/*
Copyright 2019 The Kubernetes Authors.

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
)

// AzureStackHCIMachineTemplateSpec defines the desired state of AzureStackHCIMachineTemplate
type AzureStackHCIMachineTemplateSpec struct {
	Template AzureStackHCIMachineTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=azurestackhcimachinetemplates,scope=Namespaced,categories=cluster-api

// AzureStackHCIMachineTemplate is the Schema for the azurestackhcimachinetemplates API
type AzureStackHCIMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AzureStackHCIMachineTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// AzureStackHCIMachineTemplateList contains a list of AzureStackHCIMachineTemplate
type AzureStackHCIMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureStackHCIMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzureStackHCIMachineTemplate{}, &AzureStackHCIMachineTemplateList{})
}

// AzureStackHCIMachineTemplateResource describes the data needed to create an AzureStackHCIMachine from a template
type AzureStackHCIMachineTemplateResource struct {
	// Spec is the specification of the desired behavior of the machine.
	Spec AzureStackHCIMachineSpec `json:"spec"`
}
