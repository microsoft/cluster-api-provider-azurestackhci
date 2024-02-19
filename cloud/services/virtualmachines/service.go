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

package virtualmachines

import (
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/moc-sdk-for-go/services/compute/baremetalmachine"
	"github.com/microsoft/moc-sdk-for-go/services/compute/virtualmachine"
	"github.com/microsoft/moc/pkg/auth"
)

var _ azurestackhci.Service = (*Service)(nil)

// Service provides operations on virtual machines.
type Service struct {
	Client          virtualmachine.VirtualMachineClient
	BareMetalClient baremetalmachine.BareMetalMachineClient
	Scope           scope.ScopeInterface
}

// getVirtualMachinesClient creates a new virtual machines client.
func getVirtualMachinesClient(cloudAgentFqdn string, authorizer auth.Authorizer) virtualmachine.VirtualMachineClient {
	vmClient, _ := virtualmachine.NewVirtualMachineClient(cloudAgentFqdn, authorizer)
	return *vmClient
}

func getBareMetalMachinesClient(cloudAgentFqdn string, authorizer auth.Authorizer) baremetalmachine.BareMetalMachineClient {
	bareMetalClient, _ := baremetalmachine.NewBareMetalMachineClient(cloudAgentFqdn, authorizer)
	return *bareMetalClient
}

// NewService creates a new virtual machines service.
func NewService(scope scope.ScopeInterface) *Service {
	return &Service{
		Client:          getVirtualMachinesClient(scope.GetCloudAgentFqdn(), scope.GetAuthorizer()),
		BareMetalClient: getBareMetalMachinesClient(scope.GetCloudAgentFqdn(), scope.GetAuthorizer()),
		Scope:           scope,
	}
}
