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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/moc-sdk-for-go/services/security/authentication"
	"github.com/microsoft/moc/pkg/auth"
	"github.com/microsoft/moc/pkg/config"
	mocerrors "github.com/microsoft/moc/pkg/errors"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	Scheme = scheme.Scheme
)

const (
	AzHCIAccessCreds          = "caphlogintoken"
	AzHCICreds                = "cloudconfig"
	AzHCIAccessTokenFieldName = "value"
)

var mut sync.Mutex

func GetAuthorizerFromKubernetesCluster(ctx context.Context, cloudFqdn string) (auth.Authorizer, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	config.Timeout = 10 * time.Second

	logger := zap.New(zap.UseDevMode(true))
	c, err := client.New(config, client.Options{Scheme: Scheme})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a client")
	}

	return ReconcileAzureStackHCIAccess(logger, ctx, c, cloudFqdn)
}

func ReconcileAzureStackHCIAccess(logger logr.Logger, ctx context.Context, cli client.Client, cloudFqdn string) (auth.Authorizer, error) {

	wssdconfigpath := os.Getenv("WSSD_CONFIG_PATH")
	if wssdconfigpath == "" {
		return nil, errors.New("ReconcileAzureStackHCIAccess: Environment variable WSSD_CONFIG_PATH is not set")
	}

	if strings.ToLower(os.Getenv("WSSD_DEBUG_MODE")) != "on" {
		_, err := os.Stat(wssdconfigpath)
		if err != nil {
			return login(logger, ctx, cli, cloudFqdn)
		}
		go UpdateLoginConfig(logger, ctx, cli)
	}
	authorizer, err := auth.NewAuthorizerFromEnvironment(cloudFqdn)
	if err != nil {
		// Return for any errors other than cert expiry
		if !mocerrors.IsExpired(err) {
			return nil, errors.Wrap(err, "error: new authorizer failed")
		}
		// Login if certificate expired
		return login(logger, ctx, cli, cloudFqdn)
	}
	return authorizer, nil
}

func UpdateLoginConfig(logger logr.Logger, ctx context.Context, cli client.Client) {
	secret, err := GetSecret(ctx, cli, AzHCIAccessCreds)
	if err != nil {
		logger.Error(err, "error: failed to create wssd session, missing login credentials secret")
		return
	}

	data, ok := secret.Data[AzHCIAccessTokenFieldName]
	if !ok {
		logger.Error(nil, "could not parse kubernetes secret")
		return
	}

	loginconfig := auth.LoginConfig{}
	err = config.LoadYAMLConfig(string(data), &loginconfig)
	if err != nil {
		logger.Error(err, "failed to create wssd session: parse yaml login config failed")
		return
	}

	// update login config to moc-sdk for recovery
	authentication.UpdateLoginConfig(loginconfig)

}

func login(logger logr.Logger, ctx context.Context, cli client.Client, cloudFqdn string) (auth.Authorizer, error) {
	wssdconfigpath := os.Getenv("WSSD_CONFIG_PATH")
	if wssdconfigpath == "" {
		return nil, errors.New("ReconcileAzureStackHCIAccess: Environment variable WSSD_CONFIG_PATH is not set")
	}

	mut.Lock()
	defer mut.Unlock()
	if _, err := os.Stat(wssdconfigpath); err == nil {
		if authorizer, err := auth.NewAuthorizerFromEnvironment(cloudFqdn); err == nil {
			return authorizer, nil
		}
	}
	logger.Info("AzureStackHCI: Login attempt")
	secret, err := GetSecret(ctx, cli, AzHCIAccessCreds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create wssd session, missing login credentials secret")
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

	authenticationClient, err := authentication.NewAuthenticationClientAuthMode(cloudFqdn, loginconfig)
	if err != nil {
		return nil, err
	}

	_, err = authenticationClient.LoginWithConfig(ctx, "", loginconfig, true)
	if err != nil && !azurestackhci.ResourceAlreadyExists(err) {
		return nil, errors.Wrap(err, "failed to create wssd session: login failed")
	}
	if _, err := os.Stat(wssdconfigpath); err != nil {
		return nil, errors.Wrapf(err, "Missing wssdconfig %s after login", wssdconfigpath)
	}
	logger.Info("AzureStackHCI: Login successful")
	return auth.NewAuthorizerFromEnvironment(cloudFqdn)
}

func GetSecret(ctx context.Context, cli client.Client, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: "default",
		Name:      name,
	}

	if err := cli.Get(ctx, secretKey, secret); err != nil {
		return nil, errors.Wrapf(err, "kubernetes secret query failed")
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
		return nil, errors.Wrapf(err, "kubernetes secret create failed")
	}

	return secret, nil
}
