/*
Copyright 2018 The Kubernetes Authors.

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
	"os"

	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha3"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/moc/pkg/auth"
	"github.com/microsoft/moc/pkg/config"
	"github.com/microsoft/moc/pkg/marshal"
	"github.com/microsoft/wssdcloud-sdk-for-go/services/security"
	"github.com/microsoft/wssdcloud-sdk-for-go/services/security/authentication"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AzureStackHCILoginCreds           = "azurestackhcilogintoken"
	AzureStackHCICreds                = "cloudconfig"
	AzureStackHCIAccessTokenFieldName = "value"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	AzureStackHCIClients
	Client               client.Client
	Logger               logr.Logger
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
		params.Logger = klogr.New()
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
	scope := &ClusterScope{
		Logger:               params.Logger,
		Client:               params.Client,
		AzureStackHCIClients: params.AzureStackHCIClients,
		Cluster:              params.Cluster,
		AzureStackHCICluster: params.AzureStackHCICluster,
		patchHelper:          helper,
		Context:              context.Background(),
	}

	// This is temp. Will be moved to the CloudController in the future
	err = scope.ReconcileAzureStackHCIAccess()
	if err != nil {
		return nil, errors.Wrap(err, "error creating azurestackhci services. can not authenticate to azurestackhci")
	}

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

// Network returns the cluster network object.
func (s *ClusterScope) Network() *infrav1.Network {
	return &s.AzureStackHCICluster.Status.Network
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
		clusterv1.ClusterLabelName: s.Cluster.Name,
	})
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject() error {
	return s.patchHelper.Patch(context.TODO(), s.AzureStackHCICluster)
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.AzureStackHCICluster)
}

// APIServerPort returns the APIServerPort to use when creating the load balancer.
func (s *ClusterScope) APIServerPort() int32 {
	if s.Cluster.Spec.ClusterNetwork != nil && s.Cluster.Spec.ClusterNetwork.APIServerPort != nil {
		return *s.Cluster.Spec.ClusterNetwork.APIServerPort
	}
	return 6443
}

func (s *ClusterScope) LoadBalancerRef() *corev1.ObjectReference {
	return s.AzureStackHCICluster.Spec.LoadBalancerRef
}

// GetNamespaceOrDefault returns the default namespace if given empty
func GetNamespaceOrDefault(namespace string) string {
	if namespace == "" {
		return corev1.NamespaceDefault
	}
	return namespace
}

// This is temp. Will be moved to the CloudController in the future
func (s *ClusterScope) ReconcileAzureStackHCIAccess() error {
	s.Logger.Info("reconciling azurestackhci access")
	secretAccess, err := s.GetSecret(AzureStackHCICreds)
	if err == nil {
		// Already have the AccessFile.
		data, ok := secretAccess.Data[AzureStackHCIAccessTokenFieldName]
		if !ok {
			return errors.New("error: could not parse kubernetes secret")
		}
		azurestackhciObject := auth.WssdConfig{}
		err := marshal.FromJSON(string(data), &azurestackhciObject)
		if err != nil {
			return errors.Wrap(err, "error: could not parse kubernetes secret JSON")
		}
		serverPem, tlsCert, err := auth.AccessFileToTls(azurestackhciObject)
		if err != nil {
			return errors.Wrap(err, "error: could not parse accessfile")
		}
		authorizer, err := auth.NewAuthorizerFromInput(tlsCert, serverPem, s.AzureStackHCIClients.CloudAgentFqdn)
		if err != nil {
			return errors.Wrap(err, "error: new authorizer failed")
		}
		s.AzureStackHCIClients.Authorizer = authorizer
		return nil
	}

	secret, err := s.GetSecret(AzureStackHCILoginCreds)
	if err != nil {
		authorizer, err := auth.NewAuthorizerFromEnvironment(s.AzureStackHCIClients.CloudAgentFqdn)
		if err != nil {
			return errors.Wrap(err, "failed to create azurestackhci session")
		}
		s.AzureStackHCIClients.Authorizer = authorizer
		return nil
	}

	s.Logger.Info("recieved azurestackhcilogintoken from the cluster")

	data, ok := secret.Data[AzureStackHCIAccessTokenFieldName]
	if !ok {
		return errors.New("error: could not parse kubernetes secret")
	}

	loginconfig := auth.LoginConfig{}
	err = config.LoadYAMLConfig(string(data), &loginconfig)
	if err != nil {
		return errors.Wrap(err, "failed to create azurestackhci session: parse yaml login config failed")
	}

	authForAuth, err := auth.NewAuthorizerForAuth(loginconfig.Token, loginconfig.Certificate, s.AzureStackHCIClients.CloudAgentFqdn)
	if err != nil {
		return err
	}

	authenticationClient, err := authentication.NewAuthenticationClient(s.AzureStackHCIClients.CloudAgentFqdn, authForAuth)
	if err != nil {
		return err
	}

	clientCert, accessFile, err := auth.GenerateClientKey(loginconfig)
	if err != nil {
		return err
	}
	id := security.Identity{
		Name:        &loginconfig.Name,
		Certificate: &clientCert,
	}

	_, err = authenticationClient.Login(s.Context, "", &id)
	if err != nil && !azurestackhci.ResourceAlreadyExists(err) {
		return errors.Wrap(err, "failed to create azurestackhci session: login failed")
	}

	if !azurestackhci.ResourceAlreadyExists(err) {
		str, err := marshal.ToJSON(accessFile)
		if err != nil {
			return err
		}
		s.CreateSecret(AzureStackHCICreds, []byte(str))
	}

	serverPem, tlsCert, err := auth.AccessFileToTls(accessFile)
	if err != nil {
		return err
	}

	authorizer, err := auth.NewAuthorizerFromInput(tlsCert, serverPem, s.AzureStackHCIClients.CloudAgentFqdn)
	if err != nil {
		return err
	}

	s.AzureStackHCIClients.Authorizer = authorizer

	return nil
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

func (s *ClusterScope) CreateSecret(name string, data []byte) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: GetNamespaceOrDefault(s.Cluster.Namespace),
			Name:      name,
		},
		Data: map[string][]byte{
			AzureStackHCIAccessTokenFieldName: data,
		},
	}

	if err := s.Client.Create(s.Context, secret); err != nil {
		return nil, errors.Wrapf(err, "kubernetes secret query for azurestackhci access token failed")
	}

	return secret, nil
}
