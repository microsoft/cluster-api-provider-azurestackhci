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

package controllers

import (
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/keyvaults"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/virtualnetworks"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

const (
	KubeConfigSecretName    = "kubeconf" // lgtm - Semmle Suppression [SM03415] Not a secret
	KubeConfigDataFieldName = "value"
)

// azureStackHCIClusterReconciler are list of services required by cluster controller
type azureStackHCIClusterReconciler struct {
	scope       *scope.ClusterScope
	vnetSvc     azurestackhci.Service
	keyvaultSvc azurestackhci.Service
}

// newAzureStackHCIClusterReconciler populates all the services based on input scope
func newAzureStackHCIClusterReconciler(scope *scope.ClusterScope) *azureStackHCIClusterReconciler {
	return &azureStackHCIClusterReconciler{
		scope:       scope,
		vnetSvc:     virtualnetworks.NewService(scope),
		keyvaultSvc: keyvaults.NewService(scope),
	}
}

// Reconcile reconciles all the services in pre determined order
func (r *azureStackHCIClusterReconciler) Reconcile() error {
	klog.V(2).Infof("reconciling cluster %s", r.scope.Name())

	r.createOrUpdateVnetName()

	vnetSpec := &virtualnetworks.Spec{
		Name: r.scope.Vnet().Name,
		CIDR: azurestackhci.DefaultVnetCIDR,
	}
	if r.scope.Vnet().Group != "" {
		vnetSpec.Group = r.scope.Vnet().Group
	} else {
		vnetSpec.Group = r.scope.GetResourceGroup()
	}

	if err := r.vnetSvc.Reconcile(r.scope.Context, vnetSpec); err != nil {
		return errors.Wrapf(err, "failed to reconcile virtual network for cluster %s", r.scope.Name())
	}

	vaultSpec := &keyvaults.Spec{
		Name: r.scope.Name(),
	}
	if err := r.keyvaultSvc.Reconcile(r.scope.Context, vaultSpec); err != nil {
		return errors.Wrapf(err, "failed to reconcile keyvault for cluster %s", r.scope.Name())
	}

	return nil
}

// Delete reconciles all the services in pre determined order
func (r *azureStackHCIClusterReconciler) Delete() error {
	vaultSpec := &keyvaults.Spec{
		Name: r.scope.Name(),
	}
	if err := r.keyvaultSvc.Delete(r.scope.Context, vaultSpec); err != nil {
		if !azurestackhci.ResourceNotFound(err) {
			return errors.Wrapf(err, "failed to delete keyvault %s for cluster %s", r.scope.Name(), r.scope.Name())
		}
	}

	vnetSpec := &virtualnetworks.Spec{
		Name: r.scope.Vnet().Name,
		CIDR: azurestackhci.DefaultVnetCIDR,
	}
	if r.scope.Vnet().Group != "" {
		vnetSpec.Group = r.scope.Vnet().Group
	} else {
		vnetSpec.Group = r.scope.GetResourceGroup()
	}

	if err := r.vnetSvc.Delete(r.scope.Context, vnetSpec); err != nil {
		if !azurestackhci.ResourceNotFound(err) {
			return errors.Wrapf(err, "failed to delete virtual network %s for cluster %s", r.scope.Vnet().Name, r.scope.Name())
		}
	}

	return nil
}

// createOrUpdateVnetName creates or updates the virtual network (vnet) name
func (r *azureStackHCIClusterReconciler) createOrUpdateVnetName() {
	if r.scope.Vnet().Name == "" {
		r.scope.Vnet().Name = azurestackhci.GenerateVnetName(r.scope.Name())
	}
}
