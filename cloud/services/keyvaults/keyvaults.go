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

package keyvaults

import (
	"context"

	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/telemetry"
	"github.com/microsoft/moc-sdk-for-go/services/security"
	"github.com/pkg/errors"
)

// Spec specification for keyvault
type Spec struct {
	Name string
}

// Get provides information about a keyvault.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	vaultSpec, ok := spec.(*Spec)
	if !ok {
		return security.KeyVault{}, errors.New("Invalid keyvault specification")
	}
	vault, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), vaultSpec.Name)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		return nil, errors.Wrapf(err, "keyvault %s not found", vaultSpec.Name)
	} else if err != nil {
		return nil, err
	}
	return (*vault)[0], nil
}

// Reconcile gets/creates/updates a keyvault.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	vaultSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid keyvault specification")
	}

	if _, err := s.Get(ctx, vaultSpec); err == nil {
		// vault already exists, cannot update since its immutable
		return nil
	}

	logger := s.Scope.GetLogger()
	logger.Info("creating keyvault", "name", vaultSpec.Name)
	_, err := s.Client.CreateOrUpdate(ctx, s.Scope.GetResourceGroup(), vaultSpec.Name,
		&security.KeyVault{
			Name:               &vaultSpec.Name,
			KeyVaultProperties: &security.KeyVaultProperties{},
		})
	telemetry.WriteMocOperationLog(logger, telemetry.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), telemetry.KeyVault,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), vaultSpec.Name), nil, err)
	if err != nil {
		return err
	}

	logger.Info("successfully created keyvault", "name", vaultSpec.Name)
	return err
}

// Delete deletes a keyvault.
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	vaultSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid keyvault specification")
	}
	logger := s.Scope.GetLogger()
	logger.Info("deleting keyvault", "name", vaultSpec.Name)
	err := s.Client.Delete(ctx, s.Scope.GetResourceGroup(), vaultSpec.Name)
	telemetry.WriteMocOperationLog(logger, telemetry.Delete, s.Scope.GetCustomResourceTypeWithName(), telemetry.KeyVault,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), vaultSpec.Name), nil, err)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		// already deleted
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete keyvault %s in resource group %s", vaultSpec.Name, s.Scope.GetResourceGroup())
	}

	logger.Info("successfully deleted keyvault", "name", vaultSpec.Name)
	return err
}
