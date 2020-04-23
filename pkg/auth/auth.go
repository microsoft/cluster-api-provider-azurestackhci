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

package auth

import (
	"context"
	"time"

	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/moc-sdk-for-go/services/security"
	"github.com/microsoft/moc-sdk-for-go/services/security/authentication"
	"github.com/microsoft/moc/pkg/auth"
	"github.com/microsoft/moc/pkg/config"
	"github.com/microsoft/moc/pkg/marshal"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	Scheme = scheme.Scheme
)

const (
	AzHCIAccessCreds          = "wssdlogintoken"
	AzHCICreds                = "cloudconfig"
	AzHCIAccessTokenFieldName = "value"
)

func GetAuthorizerFromKubernetesCluster(ctx context.Context, cloudFqdn string) (auth.Authorizer, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	config.Timeout = 10 * time.Second

	c, err := client.New(config, client.Options{Scheme: Scheme})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a client")
	}

	return ReconcileAzureStackHCIAccess(ctx, c, cloudFqdn)
}

func ReconcileAzureStackHCIAccess(ctx context.Context, cli client.Client, cloudFqdn string) (auth.Authorizer, error) {

	secretAccess, err := GetSecret(ctx, cli, AzHCICreds)
	if err == nil {
		// Already have the AccessFile.
		data, ok := secretAccess.Data[AzHCIAccessTokenFieldName]
		if !ok {
			return nil, errors.New("error: could not parse kubernetes secret")
		}
		azhciObject := auth.WssdConfig{}
		err := marshal.FromJSON(string(data), &azhciObject)
		if err != nil {
			return nil, errors.Wrap(err, "error: could not parse kubernetes secret JSON")
		}
		serverPem, tlsCert, err := auth.AccessFileToTls(azhciObject)
		if err != nil {
			return nil, errors.Wrap(err, "error: could not parse accessfile")
		}
		authorizer, err := auth.NewAuthorizerFromInput(tlsCert, serverPem, cloudFqdn)
		if err != nil {
			return nil, errors.Wrap(err, "error: new authorizer failed")
		}
		return authorizer, nil
	}

	secret, err := GetSecret(ctx, cli, AzHCIAccessCreds)
	if err != nil {
		authorizer, err := auth.NewAuthorizerFromEnvironment(cloudFqdn)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create wssd session")
		}
		return authorizer, nil
	}

	data, ok := secret.Data[AzHCIAccessTokenFieldName]
	if !ok {
		return nil, errors.New("error: could not parse kubernetes secret")
	}

	loginconfig := auth.LoginConfig{}
	err = config.LoadYAMLConfig(string(data), &loginconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create wssd session: parse yaml login config failed")
	}

	authForAuth, err := auth.NewAuthorizerForAuth(loginconfig.Token, loginconfig.Certificate, cloudFqdn)
	if err != nil {
		return nil, err
	}

	authenticationClient, err := authentication.NewAuthenticationClient(cloudFqdn, authForAuth)
	if err != nil {
		return nil, err
	}

	clientCert, accessFile, err := auth.GenerateClientKey(loginconfig)
	if err != nil {
		return nil, err
	}
	id := security.Identity{
		Name:        &loginconfig.Name,
		Certificate: &clientCert,
	}

	_, err = authenticationClient.Login(ctx, "", &id)
	if err != nil && !azurestackhci.ResourceAlreadyExists(err) {
		return nil, errors.Wrap(err, "failed to create wssd session: login failed")
	}

	if !azurestackhci.ResourceAlreadyExists(err) {
		str, err := marshal.ToJSON(accessFile)
		if err != nil {
			return nil, err
		}
		CreateSecret(ctx, cli, AzHCICreds, []byte(str))
	}

	serverPem, tlsCert, err := auth.AccessFileToTls(accessFile)
	if err != nil {
		return nil, err
	}

	authorizer, err := auth.NewAuthorizerFromInput(tlsCert, serverPem, cloudFqdn)
	if err != nil {
		return nil, err
	}

	return authorizer, nil
}

func GetSecret(ctx context.Context, cli client.Client, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: "default",
		Name:      name,
	}

	if err := cli.Get(ctx, secretKey, secret); err != nil {
		return nil, errors.Wrapf(err, "kubernetes secret query for wssd access token failed")
	}

	return secret, nil
}

func CreateSecret(ctx context.Context, cli client.Client, name string, data []byte) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
		},
		Data: map[string][]byte{
			AzHCIAccessTokenFieldName: data,
		},
	}

	if err := cli.Create(ctx, secret); err != nil {
		return nil, errors.Wrapf(err, "kubernetes secret query for wssd access token failed")
	}

	return secret, nil
}
