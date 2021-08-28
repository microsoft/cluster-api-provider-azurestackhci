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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// SetupWebhookWithManager will setup and register the webhook with the controller mnager
func (m *AzureStackHCIMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(m).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha3-azurestackhcimachine,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=azurestackhcimachine,versions=v1alpha3,name=validation.azurestackhcimachine.infrastructure.cluster.x-k8s.io

var _ webhook.Validator = &AzureStackHCIMachine{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (m *AzureStackHCIMachine) ValidateCreate() error {

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (m *AzureStackHCIMachine) ValidateUpdate(old runtime.Object) error {

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (m *AzureStackHCIMachine) ValidateDelete() error {

	return nil
}
