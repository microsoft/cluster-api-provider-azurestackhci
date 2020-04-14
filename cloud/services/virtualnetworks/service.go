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

package virtualnetworks

import (
	//"github.com/Azure/go-autorest/autorest"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/moc/pkg/auth"
	"github.com/microsoft/wssdcloud-sdk-for-go/services/network/virtualnetwork"
)

var _ azurestackhci.Service = (*Service)(nil)

// Service provides operations on virtual networks.
type Service struct {
	Client virtualnetwork.VirtualNetworkClient
	Scope  scope.ScopeInterface
}

// getVirtualNetworksClient creates a new virtual networks client.
func getVirtualNetworksClient(cloudAgentFqdn string, authorizer auth.Authorizer) virtualnetwork.VirtualNetworkClient {
	vnetsClient, _ := virtualnetwork.NewVirtualNetworkClient(cloudAgentFqdn, authorizer)
	return *vnetsClient
}

// NewService creates a new virtual networks service.
func NewService(scope scope.ScopeInterface) *Service {
	return &Service{
		Client: getVirtualNetworksClient(scope.GetCloudAgentFqdn(), scope.GetAuthorizer()),
		Scope:  scope,
	}
}
