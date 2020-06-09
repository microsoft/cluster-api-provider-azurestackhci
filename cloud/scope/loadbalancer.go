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
	Client                    client.Client
	Logger                    logr.Logger
	AzureStackHCILoadBalancer *infrav1.AzureStackHCILoadBalancer
	Cluster                   *clusterv1.Cluster
	AzureStackHCICluster      *infrav1.AzureStackHCICluster
}

// NewLoadBalancerScope creates a new LoadBalancerScope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewLoadBalancerScope(params LoadBalancerScopeParams) (*LoadBalancerScope, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a LoadBalancerScope")
	}

	if params.AzureStackHCILoadBalancer == nil {
		return nil, errors.New("azurestackhci loadbalancer is required when creating a LoadBalancerScope")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.AzureStackHCILoadBalancer, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &LoadBalancerScope{
		client:                    params.Client,
		AzureStackHCILoadBalancer: params.AzureStackHCILoadBalancer,
		Cluster:                   params.Cluster,
		AzureStackHCICluster:      params.AzureStackHCICluster,
		Logger:                    params.Logger,
		patchHelper:               helper,
		Context:                   context.Background(),
	}, nil
}

// LoadBalancerScope defines a scope defined around a LoadBalancer.
type LoadBalancerScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper
	Context     context.Context

	AzureStackHCILoadBalancer *infrav1.AzureStackHCILoadBalancer
	Cluster                   *clusterv1.Cluster
	AzureStackHCICluster      *infrav1.AzureStackHCICluster
}

// Name returns the Name of the AzureStackHCILoadBalancer
func (l *LoadBalancerScope) Name() string {
	return l.AzureStackHCILoadBalancer.Name
}

// Address returns the address of the AzureStackHCILoadBalancer, if it exists.
func (l *LoadBalancerScope) Address() string {
	return l.AzureStackHCILoadBalancer.Status.Address
}

// SetAnnotation sets a key value annotation on the AzureStackHCILoadBalancer
func (l *LoadBalancerScope) SetAnnotation(key, value string) {
	if l.AzureStackHCILoadBalancer.Annotations == nil {
		l.AzureStackHCILoadBalancer.Annotations = map[string]string{}
	}
	l.AzureStackHCILoadBalancer.Annotations[key] = value
}

// PatchObject persists the AzureStackHCILoadBalancer spec and status.
func (l *LoadBalancerScope) PatchObject() error {
	return l.patchHelper.Patch(context.TODO(), l.AzureStackHCILoadBalancer)
}

// Close the LoadBalancerScope by updating the AzureStackHCILoadBalancer spec and status.
func (l *LoadBalancerScope) Close() error {
	return l.patchHelper.Patch(context.TODO(), l.AzureStackHCILoadBalancer)
}

// SetReady sets the AzureStackHCILoadBalancer Ready Status
func (l *LoadBalancerScope) SetReady() {
	l.AzureStackHCILoadBalancer.Status.Ready = true
}

// GetVMState returns the AzureStackHCILoadBalancer VM state.
func (l *LoadBalancerScope) GetVMState() *infrav1.VMState {
	return l.AzureStackHCILoadBalancer.Status.VMState
}

// SetVMState sets the AzureStackHCILoadBalancer VM state.
func (l *LoadBalancerScope) SetVMState(v *infrav1.VMState) {
	l.AzureStackHCILoadBalancer.Status.VMState = new(infrav1.VMState)
	*l.AzureStackHCILoadBalancer.Status.VMState = *v
}

// SetErrorMessage sets the AzureStackHCILoadBalancer status error message.
func (l *LoadBalancerScope) SetErrorMessage(v error) {
	l.AzureStackHCILoadBalancer.Status.ErrorMessage = pointer.StringPtr(v.Error())
}

// SetErrorReason sets the AzureStackHCILoadBalancer status error reason.
func (l *LoadBalancerScope) SetErrorReason(v capierrors.MachineStatusError) {
	l.AzureStackHCILoadBalancer.Status.ErrorReason = &v
}

// SetAddress sets the Address field of the AzureStackHCILoadBalancer Status.
func (l *LoadBalancerScope) SetAddress(address string) {
	l.AzureStackHCILoadBalancer.Status.Address = address
}

// SetPort sets the Port field of the AzureStackHCILoadBalancer Status.
func (l *LoadBalancerScope) SetPort(port int32) {
	l.AzureStackHCILoadBalancer.Status.Port = port
}

// GetPort returns the Port field of the AzureStackHCILoadBalancer Status.
func (l *LoadBalancerScope) GetPort() int32 {
	return l.AzureStackHCILoadBalancer.Status.Port
}
