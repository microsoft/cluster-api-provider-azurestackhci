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
package health

import (
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/moc-sdk-for-go/services/admin/health"
	"github.com/microsoft/moc/pkg/auth"
)

var _ azurestackhci.Service = (*Service)(nil)

// Service provides operations on resource groups
type Service struct {
	Client health.HealthClient
	Scope  scope.ScopeInterface
}

// getHealthClient creates a new health client.
func getHealthClient(cloudAgentFqdn string, authorizer auth.Authorizer) health.HealthClient {
	healthClient, _ := health.NewHealthClient(cloudAgentFqdn, authorizer)
	return *healthClient
}

// NewService creates a new health service.
func NewService(scope scope.ScopeInterface) *Service {
	return &Service{
		Client: getHealthClient(scope.GetCloudAgentFqdn(), scope.GetAuthorizer()),
		Scope:  scope,
	}
}
