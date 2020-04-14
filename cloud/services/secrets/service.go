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

package secrets

import (
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/moc/pkg/auth"
	"github.com/microsoft/wssdcloud-sdk-for-go/services/security/keyvault/secret"
)

var _ azurestackhci.Service = (*Service)(nil)

// Service provides operations on secrets.
type Service struct {
	Client secret.SecretClient
	Scope  scope.ScopeInterface
}

// getSecretClient creates a new secret client.
func getSecretClient(cloudAgentFqdn string, authorizer auth.Authorizer) secret.SecretClient {
	secretClient, _ := secret.NewSecretClient(cloudAgentFqdn, authorizer)
	return *secretClient
}

// NewService creates a new secret service.
func NewService(scope scope.ScopeInterface) *Service {
	return &Service{
		Client: getSecretClient(scope.GetCloudAgentFqdn(), scope.GetAuthorizer()),
		Scope:  scope,
	}
}
