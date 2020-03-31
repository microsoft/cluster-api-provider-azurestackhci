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

package loadbalancers

import (
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/moc-sdk-for-go/services/network/loadbalancer"
	"github.com/microsoft/moc/pkg/auth"
)

var _ azurestackhci.Service = (*Service)(nil)

// Service provides operations on load balancers.
type Service struct {
	Client loadbalancer.LoadBalancerClient
	Scope  scope.ScopeInterface
}

// getLoadBalancersClient creates a new load balancers client.
func getLoadBalancersClient(cloudAgentFqdn string, authorizer auth.Authorizer) loadbalancer.LoadBalancerClient {
	lbClient, _ := loadbalancer.NewLoadBalancerClient(cloudAgentFqdn, authorizer)
	return *lbClient
}

// NewService creates a new load balancers service.
func NewService(scope scope.ScopeInterface) *Service {
	return &Service{
		Client: getLoadBalancersClient(scope.GetCloudAgentFqdn(), scope.GetAuthorizer()),
		Scope:  scope,
	}
}
