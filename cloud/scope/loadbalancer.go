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

package scope

import (
	"context"

	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha3"
	"github.com/pkg/errors"
	"k8s.io/klog/klogr"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LoadBalancerScopeParams defines the input parameters used to create a new LoadBalancerScope.
type LoadBalancerScopeParams struct {
	Client               client.Client
	Logger               logr.Logger
	LoadBalancer         *infrav1.LoadBalancer
	Cluster              *clusterv1.Cluster
	AzureStackHCICluster *infrav1.AzureStackHCICluster
}

// NewLoadBalancerScope creates a new LoadBalancerScope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewLoadBalancerScope(params LoadBalancerScopeParams) (*LoadBalancerScope, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a LoadBalancerScope")
	}

	if params.LoadBalancer == nil {
		return nil, errors.New("load balancer is required when creating a LoadBalancerScope")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.LoadBalancer, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &LoadBalancerScope{
		client:               params.Client,
		LoadBalancer:         params.LoadBalancer,
		Cluster:              params.Cluster,
		AzureStackHCICluster: params.AzureStackHCICluster,
		Logger:               params.Logger,
		patchHelper:          helper,
		Context:              context.Background(),
	}, nil
}

// LoadBalancerScope defines a scope defined around a machine.
type LoadBalancerScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper
	Context     context.Context

	LoadBalancer         *infrav1.LoadBalancer
	Cluster              *clusterv1.Cluster
	AzureStackHCICluster *infrav1.AzureStackHCICluster
}

// Name returns the Name of the load balancer.
func (l *LoadBalancerScope) Name() string {
	return l.LoadBalancer.Name
}

// Address returns the address of the load balancer, if it exists.
func (l *LoadBalancerScope) Address() string {
	return l.LoadBalancer.Status.Address
}

// SetAnnotation sets a key value annotation on the LoadBalancer.
func (l *LoadBalancerScope) SetAnnotation(key, value string) {
	if l.LoadBalancer.Annotations == nil {
		l.LoadBalancer.Annotations = map[string]string{}
	}
	l.LoadBalancer.Annotations[key] = value
}

// PatchObject persists the loadbalancer spec and status.
func (l *LoadBalancerScope) PatchObject() error {
	return l.patchHelper.Patch(context.TODO(), l.LoadBalancer)
}

// Close the LoadBalancerScope by updating the loadBalancer spec and status.
func (l *LoadBalancerScope) Close() error {
	return l.patchHelper.Patch(context.TODO(), l.LoadBalancer)
}

// SetReady sets the LoadBalancer Ready Status
func (l *LoadBalancerScope) SetReady() {
	l.LoadBalancer.Status.Ready = true
}

// GetVMState returns the LoadBalancer VM state.
func (l *LoadBalancerScope) GetVMState() *infrav1.VMState {
	return l.LoadBalancer.Status.VMState
}

// SetVMState sets the LoadBalancer VM state.
func (l *LoadBalancerScope) SetVMState(v *infrav1.VMState) {
	l.LoadBalancer.Status.VMState = new(infrav1.VMState)
	*l.LoadBalancer.Status.VMState = *v
}

// SetErrorMessage sets the LoadBalancer status error message.
func (l *LoadBalancerScope) SetErrorMessage(v error) {
	l.LoadBalancer.Status.ErrorMessage = pointer.StringPtr(v.Error())
}

// SetErrorReason sets the LoadBalancer status error reason.
func (l *LoadBalancerScope) SetErrorReason(v capierrors.MachineStatusError) {
	l.LoadBalancer.Status.ErrorReason = &v
}

// SetLoadBalancerAddress sets the Address field of the Load Balancer Status.
func (l *LoadBalancerScope) SetAddress(address string) {
	l.LoadBalancer.Status.Address = address
}
