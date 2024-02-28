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

package availabilitysets

import (
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/moc-sdk-for-go/services/cloud/node"
	"github.com/microsoft/moc-sdk-for-go/services/compute/availabilityset"
	"github.com/microsoft/moc/pkg/auth"
)

var _ azurestackhci.Service = (*Service)(nil)

// Service provides operations on resource groups
type Service struct {
	Client     availabilityset.AvailabilitySetClient
	NodeClient node.NodeClient
	Scope      scope.ScopeInterface
}

// getAvailabilitySetsClient creates a new availability set client.
func getAvailabilitySetsClient(cloudAgentFqdn string, authorizer auth.Authorizer) availabilityset.AvailabilitySetClient {
	availabilitysetClient, _ := availabilityset.NewAvailabilitySetClient(cloudAgentFqdn, authorizer)
	return *availabilitysetClient
}

func getNodeClient(cloudAgentFqdn string, authorizer auth.Authorizer) node.NodeClient {
	nodeClient, _ := node.NewNodeClient(cloudAgentFqdn, authorizer)
	return *nodeClient
}

// NewService creates a new availability set service.
func NewService(scope scope.ScopeInterface) *Service {
	return &Service{
		// TODO: Replace with getAvailabilitySetsClient
		Client:     getAvailabilitySetsClient(scope.GetCloudAgentFqdn(), scope.GetAuthorizer()),
		NodeClient: getNodeClient(scope.GetCloudAgentFqdn(), scope.GetAuthorizer()),
		Scope:      scope,
	}
}
