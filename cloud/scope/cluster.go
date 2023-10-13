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
	"fmt"
	"os"

	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	azhciauth "github.com/microsoft/cluster-api-provider-azurestackhci/pkg/auth"
	"github.com/microsoft/moc/pkg/auth"
	"github.com/microsoft/moc/pkg/diagnostics"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	AzureStackHCIClients
	Client               client.Client
	Logger               *logr.Logger
	Cluster              *clusterv1.Cluster
	AzureStackHCICluster *infrav1.AzureStackHCICluster
	Context              context.Context
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil Cluster")
	}
	if params.AzureStackHCICluster == nil {
		return nil, errors.New("failed to generate new scope from nil AzureStackHCICluster")
	}

	if params.Logger == nil {
		log := klogr.New()
		params.Logger = &log
	}

	agentFqdn := os.Getenv("AZURESTACKHCI_CLOUDAGENT_FQDN")
	if agentFqdn == "" {
		return nil, errors.New("error creating azurestackhci services. Environment variable AZURESTACKHCI_CLOUDAGENT_FQDN is not set")
	}
	params.AzureStackHCIClients.CloudAgentFqdn = agentFqdn

	helper, err := patch.NewHelper(params.AzureStackHCICluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	scopeContext := diagnostics.NewContextWithCorrelationId(context.Background(), params.AzureStackHCICluster.GetAnnotations()[infrav1.AzureCorrelationIDAnnotationKey])
	scope := &ClusterScope{
		Logger:               *params.Logger,
		Client:               params.Client,
		AzureStackHCIClients: params.AzureStackHCIClients,
		Cluster:              params.Cluster,
		AzureStackHCICluster: params.AzureStackHCICluster,
		patchHelper:          helper,
		Context:              scopeContext,
	}

	authorizer, err := azhciauth.ReconcileAzureStackHCIAccess(scope.Context, scope.Client, agentFqdn)
	if err != nil {
		return nil, errors.Wrap(err, "error creating azurestackhci services. can not authenticate to azurestackhci")
	}

	scope.Authorizer = authorizer
	return scope, nil
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	logr.Logger
	Client      client.Client
	patchHelper *patch.Helper

	AzureStackHCIClients
	Cluster              *clusterv1.Cluster
	AzureStackHCICluster *infrav1.AzureStackHCICluster
	Context              context.Context
}

// GetResourceGroup allows ClusterScope to fulfill ScopeInterface and thus to be used by the cloud services.
func (s *ClusterScope) GetResourceGroup() string {
	return s.AzureStackHCICluster.Spec.ResourceGroup
}

// GetCloudAgentFqdn returns the cloud agent fqdn string.
func (s *ClusterScope) GetCloudAgentFqdn() string {
	return s.CloudAgentFqdn
}

// GetAuthorizer is a getter for the environment generated authorizer.
func (s *ClusterScope) GetAuthorizer() auth.Authorizer {
	return s.Authorizer
}

// GetLogger returns the logger.
func (s *ClusterScope) GetLogger() logr.Logger {
	return s.Logger
}

// GetCustomResourceTypeWithName return cluster resource string.
func (s *ClusterScope) GetCustomResourceTypeWithName() string {
	return fmt.Sprintf("Cluster/%s/%s", s.Namespace(), s.Name())
}

// Vnet returns the cluster Vnet.
func (s *ClusterScope) Vnet() *infrav1.VnetSpec {
	return &s.AzureStackHCICluster.Spec.NetworkSpec.Vnet
}

// Subnets returns the cluster subnets.
func (s *ClusterScope) Subnets() infrav1.Subnets {
	return s.AzureStackHCICluster.Spec.NetworkSpec.Subnets
}

// Name returns the cluster name.
func (s *ClusterScope) Name() string {
	return s.Cluster.Name
}

// Namespace returns the cluster namespace.
func (s *ClusterScope) Namespace() string {
	return s.Cluster.Namespace
}

func (s *ClusterScope) APIVersion() string {
	return s.Cluster.APIVersion
}

func (s *ClusterScope) Kind() string {
	return s.Cluster.Kind
}

func (s *ClusterScope) UID() types.UID {
	return s.Cluster.UID
}

// Location returns the cluster location.
func (s *ClusterScope) Location() string {
	return s.AzureStackHCICluster.Spec.Location
}

// ListOptionsLabelSelector returns a ListOptions with a label selector for clusterName.
func (s *ClusterScope) ListOptionsLabelSelector() client.ListOption {
	return client.MatchingLabels(map[string]string{
		clusterv1.ClusterNameLabel: s.Cluster.Name,
	})
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject() error {
	conditions.SetSummary(s.AzureStackHCICluster,
		conditions.WithConditions(
			infrav1.NetworkInfrastructureReadyCondition,
		),
		conditions.WithStepCounterIfOnly(
			infrav1.NetworkInfrastructureReadyCondition,
		),
	)

	return s.patchHelper.Patch(s.Context,
		s.AzureStackHCICluster,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.NetworkInfrastructureReadyCondition,
		}})
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close() error {
	return s.PatchObject()
}

// APIServerPort returns the APIServerPort to use when creating the load balancer.
func (s *ClusterScope) APIServerPort() int32 {
	if s.Cluster.Spec.ClusterNetwork != nil && s.Cluster.Spec.ClusterNetwork.APIServerPort != nil {
		return *s.Cluster.Spec.ClusterNetwork.APIServerPort
	}
	return 6443
}

func (s *ClusterScope) AzureStackHCILoadBalancer() *infrav1.AzureStackHCILoadBalancerSpec {
	return s.AzureStackHCICluster.Spec.AzureStackHCILoadBalancer
}

// GetNamespaceOrDefault returns the default namespace if given empty
func GetNamespaceOrDefault(namespace string) string {
	if namespace == "" {
		return corev1.NamespaceDefault
	}
	return namespace
}

func (s *ClusterScope) GetSecret(name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: GetNamespaceOrDefault(s.Cluster.Namespace),
		Name:      name,
	}

	if err := s.Client.Get(s.Context, secretKey, secret); err != nil {
		return nil, errors.Wrapf(err, "kubernetes secret query for azurestackhci access token failed")
	}

	return secret, nil
}
