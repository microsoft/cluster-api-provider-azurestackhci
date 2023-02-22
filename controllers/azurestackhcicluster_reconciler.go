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
	"fmt"

	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/groups"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/keyvaults"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/secrets"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/virtualnetworks"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

const (
	KubeConfigSecretName    = "kubeconf" // lgtm - Semmle Suppression [SM03415] Not a secret
	KubeConfigDataFieldName = "value"
	MocLocation             = "MocLocation"
)

// azureStackHCIClusterReconciler are list of services required by cluster controller
type azureStackHCIClusterReconciler struct {
	scope       *scope.ClusterScope
	vnetSvc     azurestackhci.Service
	keyvaultSvc azurestackhci.Service
	secretSvc   azurestackhci.GetterService
	groupSvc    azurestackhci.Service
}

// newAzureStackHCIClusterReconciler populates all the services based on input scope
func newAzureStackHCIClusterReconciler(scope *scope.ClusterScope) *azureStackHCIClusterReconciler {
	return &azureStackHCIClusterReconciler{
		scope:       scope,
		vnetSvc:     virtualnetworks.NewService(scope),
		keyvaultSvc: keyvaults.NewService(scope),
		secretSvc:   secrets.NewService(scope),
		groupSvc:    groups.NewService(scope),
	}
}

// Reconcile reconciles all the services in pre determined order
func (r *azureStackHCIClusterReconciler) Reconcile() error {
	//TODO: remove DEBUG from logs
	klog.V(2).Infof("DEBUG reconciling cluster %s", r.scope.Name())

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

	klog.V(2).Infof("DEBUG reconciling group cluster %s", r.scope.GetResourceGroup())
	groupSpec := &groups.Spec{
		Name:     r.scope.GetResourceGroup(),
		Location: MocLocation,
	}

	if err := r.groupSvc.Reconcile(r.scope.Context, groupSpec); err != nil {
		return errors.Wrapf(err, "DEBUG failed to reconcile group for cluster %s", r.scope.Name())
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
	klog.V(2).Infof("DEBUG deleting cluster %s", r.scope.Name())
	vaultSpec := &keyvaults.Spec{
		Name: r.scope.Name(),
	}
	if err := r.keyvaultSvc.Delete(r.scope.Context, vaultSpec); err != nil {
		if !azurestackhci.ResourceNotFound(err) {
			return errors.Wrapf(err, "failed to delete keyvault %s for cluster %s", r.scope.Name(), r.scope.Name())
		}
	}
	klog.V(2).Infof("DEBUG deleting group %s", r.scope.GetResourceGroup())
	groupSpec := &groups.Spec{
		Name:     r.scope.GetResourceGroup(),
		Location: MocLocation,
	}

	if err := r.groupSvc.Delete(r.scope.Context, groupSpec); err != nil {
		return errors.Wrapf(err, "failed to delete group %s for cluster %s", r.scope.GetResourceGroup(), r.scope.Name())
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

// ReconcileKubeConfig reconciles the kubeconfig from the cluster secrets
func (r *azureStackHCIClusterReconciler) ReconcileKubeConfig() error {
	r.scope.Logger.Info("reconciling kubeconfig %s", r.scope.Name())

	cluster := r.scope.Cluster
	name := fmt.Sprintf("%s-kubeconfig", cluster.Name)
	secret, err := r.scope.GetSecret(name)
	if err != nil {
		return errors.Wrapf(err, "kubernetes secret query failed %s", r.scope.Name())
	}
	r.scope.Logger.Info("recieved kubeconfig from the cluster")

	data, ok := secret.Data[KubeConfigDataFieldName]
	if !ok {
		return nil
	}

	secretSpec := &secrets.Spec{
		Name:      KubeConfigSecretName,
		VaultName: r.scope.Name(),
		Value:     string(data),
	}

	if err := r.secretSvc.Reconcile(r.scope.Context, secretSpec); err != nil {
		return errors.Wrapf(err, "failed to reconcile secret for cluster %s", r.scope.Name())
	}
	return nil
}

// createOrUpdateVnetName creates or updates the virtual network (vnet) name
func (r *azureStackHCIClusterReconciler) createOrUpdateVnetName() {
	if r.scope.Vnet().Name == "" {
		r.scope.Vnet().Name = azurestackhci.GenerateVnetName(r.scope.Name())
	}
}
