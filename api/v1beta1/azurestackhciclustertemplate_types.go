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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AzureStackHCIClusterTemplateSpec defines the desired state of AzureStackHCIClusterTemplate
type AzureStackHCIClusterTemplateSpec struct {
	Template AzureStackHCIClusterTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true

// AzureStackHCIClusterTemplate is the Schema for the azurestackhciclustertemplates API
type AzureStackHCIClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AzureStackHCIClusterTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// AzureStackHCIClusterTemplateList contains a list of AzureStackHCIClusterTemplate
type AzureStackHCIClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureStackHCIClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzureStackHCIClusterTemplate{}, &AzureStackHCIClusterTemplateList{})
}

// AzureStackHCIClusterTemplateResource describes the data needed to create an AzureStackHCICluster from a template
type AzureStackHCIClusterTemplateResource struct {
	Spec AzureStackHCIClusterSpec `json:"spec"`
}
