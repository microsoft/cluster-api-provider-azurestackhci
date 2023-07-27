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

package secrets

import (
	"context"

	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/telemetry"
	"github.com/microsoft/moc-sdk-for-go/services/security/keyvault"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

// Spec specification for secret
type Spec struct {
	Name      string
	VaultName string
	FileName  string
	Value     string
}

// Get provides information about a secret.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	telemetry.WriteMocDeploymentIdLog(ctx, s.Scope.GetCloudAgentFqdn(), s.Scope.GetAuthorizer())
	secretSpec, ok := spec.(*Spec)
	if !ok {
		return keyvault.Secret{}, errors.New("Invalid secret specification")
	}
	secret, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), secretSpec.Name, secretSpec.VaultName)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		return nil, errors.Wrapf(err, "secret %s not found", secretSpec.Name)
	} else if err != nil {
		return nil, err
	}
	if secret == nil || len(*secret) == 0 {
		return nil, errors.New("Not Found")
	}
	return (*secret)[0], nil
}

// Reconcile gets/creates/updates a secret.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocDeploymentIdLog(ctx, s.Scope.GetCloudAgentFqdn(), s.Scope.GetAuthorizer())
	secretSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid secret specification")
	}

	if _, err := s.Get(ctx, secretSpec); err == nil {
		// secret already exists, cannot update since its immutable
		return nil
	}

	keyvaultSecret := keyvault.Secret{
		Name:  &secretSpec.Name,
		Value: &secretSpec.Value,
		SecretProperties: &keyvault.SecretProperties{
			VaultName: &secretSpec.VaultName,
			FileName:  &secretSpec.FileName,
		},
	}

	keyvaultSecretCopy := keyvaultSecret
	keyvaultSecretCopy.Value = nil

	klog.V(2).Infof("creating secret %s ", secretSpec.Name)
	_, err := s.Client.CreateOrUpdate(ctx, s.Scope.GetResourceGroup(), secretSpec.Name, &keyvaultSecret)
	telemetry.WriteMocOperationLog(telemetry.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), telemetry.Secret,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), secretSpec.VaultName, secretSpec.Name), keyvaultSecretCopy, err)
	if err != nil {
		return err
	}

	klog.V(2).Infof("successfully created secret %s ", secretSpec.Name)
	return err
}

// Delete deletes a secret.
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocDeploymentIdLog(ctx, s.Scope.GetCloudAgentFqdn(), s.Scope.GetAuthorizer())
	secretSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid secret specification")
	}
	klog.V(2).Infof("deleting secret %s", secretSpec.Name)
	err := s.Client.Delete(ctx, s.Scope.GetResourceGroup(), secretSpec.Name, secretSpec.VaultName)
	telemetry.WriteMocOperationLog(telemetry.Delete, s.Scope.GetCustomResourceTypeWithName(), telemetry.Secret,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), secretSpec.VaultName, secretSpec.Name), nil, err)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		// already deleted
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete secret %s in resource group %s", secretSpec.Name, s.Scope.GetResourceGroup())
	}

	klog.V(2).Infof("successfully deleted secret %s", secretSpec.Name)
	return err
}
