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
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	infrav1util "github.com/microsoft/cluster-api-provider-azurestackhci/pkg/util"
	mocerrors "github.com/microsoft/moc/pkg/errors"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
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
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io;bootstrap.cluster.x-k8s.io;controlplane.cluster.x-k8s.io,resources=*,verbs=get;list;watch

func (r *AzureStackHCIClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := r.Log.WithValues("azureStackHCICluster", req.NamespacedName, "reconcileID", infrav1util.GetReconcileID(ctx))
	log.Info("Attempt to reconcile resource")

	// Fetch the AzureStackHCICluster instance
	azureStackHCICluster := &infrav1.AzureStackHCICluster{}

	err := r.Get(ctx, req.NamespacedName, azureStackHCICluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	log = log.WithValues("operationId", azureStackHCICluster.GetAnnotations()[infrav1.AzureOperationIDAnnotationKey], "correlationId", azureStackHCICluster.GetAnnotations()[infrav1.AzureCorrelationIDAnnotationKey])

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
		Logger:               &log,
		Cluster:              cluster,
		AzureStackHCICluster: azureStackHCICluster,
	})
	if err != nil {
		r.Recorder.Eventf(azureStackHCICluster, corev1.EventTypeWarning, "CreateClusterScopeFailed", errors.Wrapf(err, "failed to create cluster scope").Error())
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any AzureStackHCIMachine changes.
	defer func() {
		r.reconcilePhase(clusterScope)

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
		switch mocerrors.GetErrorCode(err) {
		case mocerrors.OutOfMemory.Error():
			conditions.MarkFalse(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, infrav1.OutOfMemoryReason, clusterv1.ConditionSeverityError, err.Error())
		case mocerrors.OutOfCapacity.Error():
			conditions.MarkFalse(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, infrav1.OutOfCapacityReason, clusterv1.ConditionSeverityError, err.Error())
		default:
			conditions.MarkFalse(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, infrav1.ClusterReconciliationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		}

		wrappedErr := errors.Wrap(err, "failed to reconcile cluster services")
		r.Recorder.Eventf(azureStackHCICluster, corev1.EventTypeWarning, "ClusterReconcileFailed", wrappedErr.Error())

		return reconcile.Result{}, wrappedErr
	}

	if ready, err := r.reconcileAzureStackHCILoadBalancer(clusterScope); !ready {
		if err != nil {
			return reconcile.Result{}, err
		}
		clusterScope.Info("AzureStackHCILoadBalancer is not ready yet")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 20}, nil
	}

	// No errors, so mark us ready so the Cluster API Cluster Controller can pull it
	azureStackHCICluster.Status.Ready = true
	conditions.MarkTrue(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition)

	return reconcile.Result{}, nil
}

func (r *AzureStackHCIClusterReconciler) reconcileDelete(clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconciling AzureStackHCICluster delete")

	azureStackHCICluster := clusterScope.AzureStackHCICluster
	conditions.MarkFalse(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, clusterv1.DeletedReason, clusterv1.ConditionSeverityInfo, "")

	// Steps to delete a cluster
	// 1. Wait for machines in the cluster to be deleted
	// 2. Delete the AzureStackHCILoadBalancer
	// 3. Wait for AzureStackHCILoadBalancer Deletion
	// 4. Delete the Cluster
	azhciMachines, err := infrav1util.GetAzureStackHCIMachinesInCluster(clusterScope.Context, clusterScope.Client, clusterScope.AzureStackHCICluster.Namespace, clusterScope.AzureStackHCICluster.Name)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "unable to list AzureStackHCIMachines part of AzureStackHCIClusters %s/%s", clusterScope.AzureStackHCICluster.Namespace, clusterScope.AzureStackHCICluster.Name)
		r.Recorder.Eventf(azureStackHCICluster, corev1.EventTypeWarning, "FailureListMachinesInCluster", wrappedErr.Error())
		conditions.MarkFalse(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, clusterv1.DeletionFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcile.Result{}, wrappedErr
	}

	if len(azhciMachines) > 0 {

		err := r.deleteOrphanedMachines(clusterScope, azhciMachines)
		if err != nil {
			wrappedErr := errors.Wrapf(err, "failed to delete orphaned AzureStackHCIMachines part of AzureStackHCIClusters %s/%s", clusterScope.AzureStackHCICluster.Namespace, clusterScope.AzureStackHCICluster.Name)
			r.Recorder.Eventf(azureStackHCICluster, corev1.EventTypeWarning, "FailureListMachinesInCluster", wrappedErr.Error())
			conditions.MarkFalse(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, clusterv1.DeletionFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
			return reconcile.Result{}, wrappedErr
		}

		clusterScope.Info("Waiting for AzureStackHCIMachines to be deleted", "count", len(azhciMachines))
		conditions.MarkFalse(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, infrav1.AzureStackHCIMachinesDeletingReason, clusterv1.ConditionSeverityWarning, "")
		return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
	}

	if err := r.reconcileDeleteAzureStackHCILoadBalancer(clusterScope); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Failed to delete AzureStackHCICluster AzureStackHCILoadBalancer")
	}

	// Initialize the AzureStackHCILoadBalancer struct and namespaced name for lookup
	azureStackHCILoadBalancer := &infrav1.AzureStackHCILoadBalancer{}
	azureStackHCILoadBalancerName := apitypes.NamespacedName{
		Namespace: clusterScope.Namespace(),
		Name:      azurestackhci.GenerateAzureStackHCILoadBalancerName(clusterScope.Name()),
	}

	// Try to get the AzureStackHCILoadBalancer; if it still exists, requeue
	if err := r.Client.Get(clusterScope.Context, azureStackHCILoadBalancerName, azureStackHCILoadBalancer); err == nil {
		clusterScope.Info("Waiting for AzureStackHCILoadBalancer to be deleted", "name", azureStackHCILoadBalancerName.Name)
		conditions.MarkFalse(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, infrav1.LoadBalancerDeletingReason, clusterv1.ConditionSeverityWarning, "")
		return reconcile.Result{RequeueAfter: 20 * time.Second}, nil
	}

	if err := newAzureStackHCIClusterReconciler(clusterScope).Delete(); err != nil {
		wrappedErr := errors.Wrapf(err, "error deleting AzureStackHCICluster %s/%s", azureStackHCICluster.Namespace, azureStackHCICluster.Name)
		r.Recorder.Eventf(azureStackHCICluster, corev1.EventTypeWarning, "FailureClusterDelete", wrappedErr.Error())
		conditions.MarkFalse(azureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, clusterv1.DeletionFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcile.Result{}, wrappedErr
	}

	r.Recorder.Eventf(azureStackHCICluster, corev1.EventTypeNormal, "SuccessfulDeleteCluster", "Successfully deleted AzureStackHCICluster %s/%s", azureStackHCICluster.Namespace, azureStackHCICluster.Name)

	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(clusterScope.AzureStackHCICluster, infrav1.ClusterFinalizer)

	return reconcile.Result{}, nil
}

func (r *AzureStackHCIClusterReconciler) deleteOrphanedMachines(clusterScope *scope.ClusterScope, azhciMachines []*infrav1.AzureStackHCIMachine) error {

	for _, azhciMachine := range azhciMachines {
		machine, err := util.GetOwnerMachine(clusterScope.Context, clusterScope.Client, azhciMachine.ObjectMeta)
		if err != nil {
			return err
		}
		if machine == nil {
			// update correlation id before deletion
			infrav1util.CopyCorrelationId(clusterScope.AzureStackHCICluster, azhciMachine)
			if err := r.Client.Update(clusterScope.Context, azhciMachine); err != nil {
				if !apierrors.IsNotFound(err) {
					return errors.Wrapf(err, "Failed to update AzureStackHCIMachine %s", azhciMachine)
				}
			}
			clusterScope.Info("Deleting Orphaned Machine", "Name", azhciMachine.Name, "AzureStackHCICluster", clusterScope.AzureStackHCICluster.Name)
			if err := r.Client.Delete(clusterScope.Context, azhciMachine); err != nil {
				if !apierrors.IsNotFound(err) {
					return errors.Wrapf(err, "Failed to delete AzureStackHCIMachine %s", azhciMachine)
				}
			}
		}
	}

	return nil
}

func (r *AzureStackHCIClusterReconciler) reconcileAzureStackHCILoadBalancer(clusterScope *scope.ClusterScope) (bool, error) {
	if clusterScope.AzureStackHCILoadBalancer() == nil {
		clusterScope.Info("Skipping load balancer reconciliation since AzureStackHCICluster.Spec.AzureStackHCILoadBalancer is nil")
		return true, nil
	}

	azureStackHCILoadBalancer := &infrav1.AzureStackHCILoadBalancer{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterScope.Namespace(),
			Name:      azurestackhci.GenerateAzureStackHCILoadBalancerName(clusterScope.Name()),
		},
	}

	mutateFn := func() (err error) {
		// Mark the Cluster as the owner of the AzureStackHCILoadBalancer
		azureStackHCILoadBalancer.SetOwnerReferences(util.EnsureOwnerRef(
			azureStackHCILoadBalancer.OwnerReferences,
			metav1.OwnerReference{
				APIVersion: clusterScope.APIVersion(),
				Kind:       clusterScope.Kind(),
				Name:       clusterScope.Name(),
				UID:        clusterScope.UID(),
			}))

		clusterScope.AzureStackHCILoadBalancer().Image.DeepCopyInto(&azureStackHCILoadBalancer.Spec.Image)
		azureStackHCILoadBalancer.Spec.SSHPublicKey = clusterScope.AzureStackHCILoadBalancer().SSHPublicKey
		azureStackHCILoadBalancer.Spec.VMSize = clusterScope.AzureStackHCILoadBalancer().VMSize
		azureStackHCILoadBalancer.Spec.Replicas = clusterScope.AzureStackHCILoadBalancer().Replicas
		infrav1util.CopyCorrelationId(clusterScope.AzureStackHCICluster, azureStackHCILoadBalancer)
		return nil
	}

	if _, err := controllerutil.CreateOrUpdate(clusterScope.Context, r.Client, azureStackHCILoadBalancer, mutateFn); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			conditions.MarkFalse(clusterScope.AzureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, infrav1.LoadBalancerProvisioningReason, clusterv1.ConditionSeverityWarning, err.Error())
			return false, err
		}
	}

	// Wait for the load balancer to be fully provisioned
	if conditions.IsFalse(azureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition) {
		cond := conditions.Get(azureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition)
		conditions.MarkFalse(clusterScope.AzureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, cond.Reason, cond.Severity, cond.Message)
		return false, nil
	}

	if azureStackHCILoadBalancer.Status.Address != "" {
		// Set APIEndpoints so the Cluster API Cluster Controller can pull them
		clusterScope.AzureStackHCICluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
			Host: azureStackHCILoadBalancer.Status.Address,
			Port: clusterScope.APIServerPort(),
		}
	}

	return true, nil
}

func (r *AzureStackHCIClusterReconciler) reconcileDeleteAzureStackHCILoadBalancer(clusterScope *scope.ClusterScope) error {
	if clusterScope.AzureStackHCILoadBalancer() == nil {
		clusterScope.Info("Skipping load balancer deletion since AzureStackHCICluster AzureStackHCILoadBalancer is nil")
		return nil
	}
	// Initialize the LoadBalancer struct and namespaced name for lookup
	azureStackHCILoadBalancer := &infrav1.AzureStackHCILoadBalancer{}
	azureStackHCILoadBalancerName := apitypes.NamespacedName{
		Namespace: clusterScope.Namespace(),
		Name:      azurestackhci.GenerateAzureStackHCILoadBalancerName(clusterScope.Name()),
	}

	// Try to get the AzureStackHCILoadBalancer
	if err := r.Client.Get(clusterScope.Context, azureStackHCILoadBalancerName, azureStackHCILoadBalancer); err != nil {
		// If the AzureStackHCILoadBalancer is not found, it must have already been deleted
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "Failed to get AzureStackHCILoadBalancer %s", azureStackHCILoadBalancerName)
		}
	} else if azureStackHCILoadBalancer.GetDeletionTimestamp().IsZero() {
		// If the AzureStackHCILoadBalancer is not already marked for deletion, delete it
		// Update correlation id before deletion
		infrav1util.CopyCorrelationId(clusterScope.AzureStackHCICluster, azureStackHCILoadBalancer)
		if err := r.Client.Update(clusterScope.Context, azureStackHCILoadBalancer); err != nil {
			if !apierrors.IsNotFound(err) {
				conditions.MarkFalse(clusterScope.AzureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, clusterv1.DeletionFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
				return errors.Wrapf(err, "Failed to update AzureStackHCILoadBalancer %s", azureStackHCILoadBalancerName)
			}
		}
		if err := r.Client.Delete(clusterScope.Context, azureStackHCILoadBalancer); err != nil {
			if !apierrors.IsNotFound(err) {
				conditions.MarkFalse(clusterScope.AzureStackHCICluster, infrav1.NetworkInfrastructureReadyCondition, clusterv1.DeletionFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
				return errors.Wrapf(err, "Failed to delete AzureStackHCILoadBalancer %s", azureStackHCILoadBalancerName)
			}
		}
	}

	return nil
}

func (r *AzureStackHCIClusterReconciler) reconcilePhase(clusterScope *scope.ClusterScope) {
	azureStackHCICluster := clusterScope.AzureStackHCICluster

	if azureStackHCICluster.Status.Phase == "" {
		azureStackHCICluster.Status.SetTypedPhase(infrav1.AzureStackHCIClusterPhasePending)
	}

	if !azureStackHCICluster.Status.Ready {
		azureStackHCICluster.Status.SetTypedPhase(infrav1.AzureStackHCIClusterPhaseProvisioning)
	}

	if azureStackHCICluster.Status.Ready { // && azureStackHCICluster.Spec.ControlPlaneEndpoint.IsValid() {
		azureStackHCICluster.Status.SetTypedPhase(infrav1.AzureStackHCIClusterPhaseProvisioned)
	}

	if !azureStackHCICluster.DeletionTimestamp.IsZero() {
		azureStackHCICluster.Status.SetTypedPhase(infrav1.AzureStackHCIClusterPhaseDeleting)
	}
}
