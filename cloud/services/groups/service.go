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

package groups

import (
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/moc/pkg/auth"
	"github.com/microsoft/wssdcloud-sdk-for-go/services/cloud/group"
)

var _ azurestackhci.Service = (*Service)(nil)

// Service provides operations on groups
type Service struct {
	Client group.GroupClient
	Scope  *scope.ClusterScope
}

// getGroupClient creates a new group client.
func getGroupClient(cloudAgentFqdn string, authorizer auth.Authorizer) group.GroupClient {
	groupClient, _ := group.NewGroupClient(cloudAgentFqdn, authorizer)
	return *groupClient
}

// NewService creates a new group service.
func NewService(scope *scope.ClusterScope) *Service {
	return &Service{
		Client: getGroupClient(scope.CloudAgentFqdn, scope.Authorizer),
		Scope:  scope,
	}
}
