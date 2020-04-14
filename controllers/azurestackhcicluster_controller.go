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
	"context"
	"time"

	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha3"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// AzureStackHCIClusterReconciler reconciles a AzureStackHCICluster object
type AzureStackHCIClusterReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

func (r *AzureStackHCIClusterReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.AzureStackHCICluster{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhciclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhciclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *AzureStackHCIClusterReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.TODO()
	log := r.Log.WithValues("namespace", req.Namespace, "azureStackHCICluster", req.Name)

	// Fetch the AzureStackHCICluster instance
	azureStackHCICluster := &infrav1.AzureStackHCICluster{}
	err := r.Get(ctx, req.NamespacedName, azureStackHCICluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, azureStackHCICluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	// Create the scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:               r.Client,
		Logger:               log,
		Cluster:              cluster,
		AzureStackHCICluster: azureStackHCICluster,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any AzureStackHCIMachine changes.
	defer func() {
		if err := clusterScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted clusters
	if !azureStackHCICluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(clusterScope)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(clusterScope)
}

func (r *AzureStackHCIClusterReconciler) reconcileNormal(clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconciling AzureStackHCICluster")

	azureStackHCICluster := clusterScope.AzureStackHCICluster

	// If the AzureCluster doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(azureStackHCICluster, infrav1.ClusterFinalizer)
	// Register the finalizer immediately to avoid orphaning Azure resources on delete
	if err := clusterScope.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}

	err := newAzureStackHCIClusterReconciler(clusterScope).Reconcile()
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to reconcile cluster services")
	}

	if ok, err := r.reconcileLoadBalancer(clusterScope); !ok {
		if err != nil {
			return reconcile.Result{}, err
		}
		clusterScope.Info("Load balancer was not reconciled")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
	}

	// No errors, so mark us ready so the Cluster API Cluster Controller can pull it
	azureStackHCICluster.Status.Ready = true

	// We mark the Cluster as Ready so CAPI can progress on ... but we still need to wait for
	// the kubeconfig to be written to secrets.
	err = newAzureStackHCIClusterReconciler(clusterScope).ReconcileKubeConfig()
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to reconcile cluster services")
	}

	return reconcile.Result{}, nil
}

func (r *AzureStackHCIClusterReconciler) reconcileDelete(clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconciling AzureStackHCICluster delete")

	azureStackHCICluster := clusterScope.AzureStackHCICluster

	if err := newAzureStackHCIClusterReconciler(clusterScope).Delete(); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "error deleting AzureStackHCICluster %s/%s", azureStackHCICluster.Namespace, azureStackHCICluster.Name)
	}

	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(clusterScope.AzureStackHCICluster, infrav1.ClusterFinalizer)

	return reconcile.Result{}, nil
}

func (r *AzureStackHCIClusterReconciler) reconcileLoadBalancer(clusterScope *scope.ClusterScope) (bool, error) {
	if clusterScope.LoadBalancerRef() == nil {
		clusterScope.Info("Skipping load balancer reconciliation since AzureStackHCICluster.Spec.LoadBalancerRef is nil")
		return true, nil
	}

	// if there are some existing control plane endpoints, skip reconciler reconcile
	if clusterScope.AzureStackHCICluster.Spec.ControlPlaneEndpoint.Host != "" {
		clusterScope.Info("Skipping load balancer reconciliation since a control plane endpoint is already present")
		return true, nil
	}

	// find the load balancer
	loadBalancer := &infrav1.LoadBalancer{}
	loadBalancerNamespacedName := types.NamespacedName{
		Namespace: clusterScope.AzureStackHCICluster.Spec.LoadBalancerRef.Namespace,
		Name:      clusterScope.AzureStackHCICluster.Spec.LoadBalancerRef.Name,
	}
	if err := r.Get(clusterScope.Context, loadBalancerNamespacedName, loadBalancer); err != nil {
		if apierrors.IsNotFound(err) {
			clusterScope.Info("resource specified by LoadBalancerRef not found", "Namespaced name", loadBalancerNamespacedName)
			return false, nil
		}
		return false, err
	}

	// ensure there is an owner reference
	clusterOwnerRef := metav1.OwnerReference{
		APIVersion:         clusterScope.APIVersion(),
		Kind:               clusterScope.Kind(),
		Name:               clusterScope.Name(),
		UID:                clusterScope.UID(),
		BlockOwnerDeletion: pointer.BoolPtr(true),
	}
	loadBalancerOwnerRefs := loadBalancer.GetOwnerReferences()
	if !util.HasOwnerRef(loadBalancerOwnerRefs, clusterOwnerRef) {
		loadBalancerPatchHelper, err := patch.NewHelper(loadBalancer, r.Client)
		if err != nil {
			return false, err
		}
		loadBalancer.SetOwnerReferences(util.EnsureOwnerRef(loadBalancerOwnerRefs, clusterOwnerRef))
		if err := loadBalancerPatchHelper.Patch(clusterScope.Context, loadBalancer); err != nil {
			return false, err
		}
		clusterScope.Info("LoadBalancer owner reference set")
	}

	// wait for the load balancer ip to be available and update the control plane endpoints list
	if loadBalancer.Status.Address == "" {
		return false, nil
	}

	// Set APIEndpoints so the Cluster API Cluster Controller can pull them
	clusterScope.AzureStackHCICluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
		Host: loadBalancer.Status.Address,
		Port: clusterScope.APIServerPort(),
	}

	return true, nil
}
