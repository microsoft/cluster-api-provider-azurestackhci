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
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	"github.com/microsoft/moc/pkg/diagnostics"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// LoadBalancerScopeParams defines the input parameters used to create a new LoadBalancerScope.
type LoadBalancerScopeParams struct {
	Client                    client.Client
	Logger                    *logr.Logger
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
		log := zap.New(zap.UseDevMode(true))
		params.Logger = &log
	}

	helper, err := patch.NewHelper(params.AzureStackHCILoadBalancer, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	scopeContext := diagnostics.NewContextWithCorrelationId(context.Background(), params.AzureStackHCILoadBalancer.GetAnnotations()[infrav1.AzureCorrelationIDAnnotationKey])
	return &LoadBalancerScope{
		client:                    params.Client,
		AzureStackHCILoadBalancer: params.AzureStackHCILoadBalancer,
		Cluster:                   params.Cluster,
		AzureStackHCICluster:      params.AzureStackHCICluster,
		Logger:                    *params.Logger,
		patchHelper:               helper,
		Context:                   scopeContext,
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

const (
	// The maximum number of Replicas that can be created above the desired amount.
	MaxSurge = 1
)

// Name returns the Name of the AzureStackHCILoadBalancer
func (l *LoadBalancerScope) Name() string {
	return l.AzureStackHCILoadBalancer.Name
}

// Address returns the address of the AzureStackHCILoadBalancer, if it exists.
func (l *LoadBalancerScope) Address() string {
	return l.AzureStackHCILoadBalancer.Status.Address
}

// OSVersion returns the AzureStackHCILoadBalancer image OS version
func (l *LoadBalancerScope) OSVersion() string {
	if l.AzureStackHCILoadBalancer.Spec.Image.Version != nil {
		return *l.AzureStackHCILoadBalancer.Spec.Image.Version
	}
	return ""
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

// GetReady returns the AzureStackHCILoadBalancer Ready Status
func (l *LoadBalancerScope) GetReady() bool {
	return l.AzureStackHCILoadBalancer.Status.Ready
}

// AddReplica increments the AzureStackHCILoadBalancer Replica status
func (l *LoadBalancerScope) AddReplica() {
	l.AzureStackHCILoadBalancer.Status.Replicas++
}

// RemoveReplica decrements the AzureStackHCILoadBalancer Replica status
func (l *LoadBalancerScope) RemoveReplica() {
	if l.AzureStackHCILoadBalancer.Status.Replicas > 0 {
		l.AzureStackHCILoadBalancer.Status.Replicas--
	}
}

// SetReplicas sets the AzureStackHCILoadBalancer Replica status
func (l *LoadBalancerScope) SetReplicas(replicas int32) {
	l.AzureStackHCILoadBalancer.Status.Replicas = replicas
}

// GetReplicas returns the AzureStackHCILoadBalancer Replica status
func (l *LoadBalancerScope) GetReplicas() int32 {
	return l.AzureStackHCILoadBalancer.Status.Replicas
}

// SetReadyReplicas sets the AzureStackHCILoadBalancer ReadyReplica status
func (l *LoadBalancerScope) SetReadyReplicas(replicas int32) {
	l.AzureStackHCILoadBalancer.Status.ReadyReplicas = replicas
}

// GetReadyReplicas returns the AzureStackHCILoadBalancer ReadyReplica status
func (l *LoadBalancerScope) GetReadyReplicas() int32 {
	return l.AzureStackHCILoadBalancer.Status.ReadyReplicas
}

// SetFailedReplicas sets the AzureStackHCILoadBalancer FailedReplica status
func (l *LoadBalancerScope) SetFailedReplicas(replicas int32) {
	l.AzureStackHCILoadBalancer.Status.FailedReplicas = replicas
}

// GetFailedReplicas returns the AzureStackHCILoadBalancer FailedReplica status
func (l *LoadBalancerScope) GetFailedReplicas() int32 {
	return l.AzureStackHCILoadBalancer.Status.FailedReplicas
}

// GetDesiredReplicas returns the AzureStackHCILoadBalancer spec.Replicas
func (l *LoadBalancerScope) GetDesiredReplicas() int32 {
	if l.AzureStackHCILoadBalancer.Spec.Replicas == nil {
		return 0
	}
	return *l.AzureStackHCILoadBalancer.Spec.Replicas
}

// GetMaxReplicas returns the maximum number of Replicas that can be created
func (l *LoadBalancerScope) GetMaxReplicas() int32 {
	return (l.GetDesiredReplicas() + MaxSurge)
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

// SetPhase sets the Phase field of the AzureStackHCILoadBalancer Status.
func (l *LoadBalancerScope) SetPhase(p infrav1.AzureStackHCILoadBalancerPhase) {
	l.AzureStackHCILoadBalancer.Status.SetTypedPhase(p)
}

// SetSelector sets the Selector field of the AzureStackHCILoadBalancer Status.
func (l *LoadBalancerScope) SetSelector(selector string) {
	l.AzureStackHCILoadBalancer.Status.Selector = selector
}

// GetLogger returns the logger.
func (l *LoadBalancerScope) GetLogger() logr.Logger {
	return l.Logger
}
