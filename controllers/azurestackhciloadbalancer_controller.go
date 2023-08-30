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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/loadbalancers"
	infrav1util "github.com/microsoft/cluster-api-provider-azurestackhci/pkg/util"
	"github.com/microsoft/moc-sdk-for-go/services/network"
	mocerrors "github.com/microsoft/moc/pkg/errors"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// AzureStackHCILoadBalancerReconciler reconciles a AzureStackHCILoadBalancer object
type AzureStackHCILoadBalancerReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

func (r *AzureStackHCILoadBalancerReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	// later we will also want to watch the cluster which owns the LB
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.AzureStackHCILoadBalancer{}).
		Watches(
			&source.Kind{Type: &infrav1.AzureStackHCIVirtualMachine{}},
			&handler.EnqueueRequestForOwner{OwnerType: &infrav1.AzureStackHCILoadBalancer{}, IsController: false},
		).
		Complete(r)
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhciloadbalancers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhciloadbalancers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

func (r *AzureStackHCILoadBalancerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := r.Log.WithValues("azureStackHCILoadBalancer", req.NamespacedName)
	logger = infrav1util.AttachReconcileIDToLogger(ctx, logger)
	logger.Info("Attempt to reconcile resource")

	// Fetch the AzureStackHCILoadBalancer resource.
	azureStackHCILoadBalancer := &infrav1.AzureStackHCILoadBalancer{}
	err := r.Get(ctx, req.NamespacedName, azureStackHCILoadBalancer)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the CAPI Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, azureStackHCILoadBalancer.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		logger.Info("AzureStackHCICluster Controller has not set OwnerRef on AzureStackHCILoadBalancer")
		return reconcile.Result{}, fmt.Errorf("Expected Cluster OwnerRef is missing from AzureStackHCILoadBalancer %s", req.Name)
	}

	logger = logger.WithValues("cluster", cluster.Name)

	azureStackHCICluster := &infrav1.AzureStackHCICluster{}
	azureStackHCIClusterName := client.ObjectKey{
		Namespace: azureStackHCILoadBalancer.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, azureStackHCIClusterName, azureStackHCICluster); err != nil {
		logger.Info("AzureStackHCICluster is not available yet")
		return reconcile.Result{}, nil
	}

	logger = logger.WithValues("azureStackHCICluster", azureStackHCICluster.Name)

	// create a cluster scope for the request.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:               r.Client,
		Logger:               &logger,
		Cluster:              cluster,
		AzureStackHCICluster: azureStackHCICluster,
	})
	if err != nil {
		r.Recorder.Eventf(azureStackHCICluster, corev1.EventTypeWarning, "CreateClusterScopeFailed", errors.Wrapf(err, "failed to create cluster scope").Error())
		return reconcile.Result{}, err
	}

	// create a lb scope for this request.
	loadBalancerScope, err := scope.NewLoadBalancerScope(scope.LoadBalancerScopeParams{
		Logger:                    &logger,
		Client:                    r.Client,
		AzureStackHCILoadBalancer: azureStackHCILoadBalancer,
		AzureStackHCICluster:      azureStackHCICluster,
		Cluster:                   cluster,
	})
	if err != nil {
		r.Recorder.Eventf(azureStackHCILoadBalancer, corev1.EventTypeWarning, "CreateLoadBalancerScopeFailed", errors.Wrapf(err, "failed to create loadbalancer scope").Error())
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any AzureStackHCILoadBalancer changes.
	defer func() {
		r.reconcileStatus(loadBalancerScope)

		if err := loadBalancerScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted LoadBalancers.
	if !azureStackHCILoadBalancer.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(loadBalancerScope, clusterScope)
	}

	// Handle non-deleted LoadBalancers.
	return r.reconcileNormal(loadBalancerScope, clusterScope)
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileNormal(lbs *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	lbs.Info("Reconciling AzureStackHCILoadBalancer")

	// If the AzureStackHCILoadBalancer doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(lbs.AzureStackHCILoadBalancer, infrav1.AzureStackHCILoadBalancerFinalizer)
	// Register the finalizer immediately to avoid orphaning resources on delete
	if err := lbs.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}

	if lbs.AzureStackHCILoadBalancer.Spec.Replicas == nil {
		err := errors.Errorf("No replica count was specified for loadbalancer %s", lbs.Name())
		r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeWarning, "InvalidConfig", err.Error())
		return reconcile.Result{}, err
	}

	result, err := r.reconcileVirtualMachines(lbs, clusterScope)
	if err != nil {
		r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeWarning, "FailureReconcileLBMachines", errors.Wrapf(err, "Failed to reconcile LoadBalancer machines").Error())
		conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, infrav1.LoadBalancerMachineReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcile.Result{}, err
	}

	// reconcile the loadbalancer service and the lb frontend ip address
	err = r.reconcileLoadBalancerService(lbs, clusterScope)
	if err != nil {
		switch mocerrors.GetErrorCode(err) {
		case mocerrors.OutOfMemory.Error():
			conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, infrav1.OutOfMemoryReason, clusterv1.ConditionSeverityError, err.Error())
		case mocerrors.OutOfCapacity.Error():
			conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, infrav1.OutOfCapacityReason, clusterv1.ConditionSeverityError, err.Error())
		default:
			conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, infrav1.LoadBalancerServiceReconciliationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		}

		r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeWarning, "FailureReconcileLB", errors.Wrapf(err, "Failed to reconcile LoadBalancer service").Error())
		return reconcile.Result{}, err
	}

	if lbs.Address() == "" {
		err := r.reconcileLoadBalancerServiceStatus(lbs, clusterScope)
		if err != nil {
			r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeWarning, "FailureReconcileLBStatus", errors.Wrapf(err, "Failed to reconcile LoadBalancer service status").Error())
			conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, infrav1.LoadBalancerServiceStatusFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
			return reconcile.Result{}, err
		}
		if lbs.Address() == "" {
			lbs.Info("LoadBalancer service address is not available yet")
			conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, infrav1.LoadBalancerAddressUnavailableReason, clusterv1.ConditionSeverityInfo, "")
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 20}, nil
		}
	}

	// When a SDN integration is present, LB replica count will be 0 as the loadbalancing is handled by SDN.
	// So fail only if the configured replica count is not 0.
	if lbs.GetReplicas() != 0 && lbs.GetReadyReplicas() < 1 {
		if lbs.GetReady() {
			// we achieved ready state at any earlier point, but have now lost all ready replicas
			r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeWarning, "FailureLBNoReadyReplicas", "No replicas are ready for LoadBalancer %s", lbs.Name())
		}

		lbs.Info("Waiting for at least one replica to be ready")
		conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, infrav1.LoadBalancerNoReplicasReadyReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 20}, nil
	}

	if conditions.IsFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition) {
		conditions.MarkTrue(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition)
		r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeNormal, "LoadBalancerReady", "AzureStackHCILoadBalancer %s infrastructure is ready", lbs.Name())
	}

	lbs.SetReady()
	return result, nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileLoadBalancerServiceStatus(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	loadBalancerScope.Info("Attempting to get status for loadbalancer service", "name", loadBalancerScope.AzureStackHCILoadBalancer.Name)
	lbSpec := &loadbalancers.Spec{
		Name: loadBalancerScope.AzureStackHCILoadBalancer.Name,
	}
	lbInterface, err := loadbalancers.NewService(clusterScope).Get(clusterScope.Context, lbSpec)
	if err != nil {
		return err
	}

	lb, ok := lbInterface.(network.LoadBalancer)
	if !ok {
		return errors.New("error getting status for loadbalancer service")
	}

	if loadBalancerScope.Address() == "" {
		loadBalancerScope.SetAddress(*((*lb.FrontendIPConfigurations)[0].IPAddress))
	}

	loadBalancerScope.SetReadyReplicas(int32(lb.ReplicationCount))
	return nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileLoadBalancerService(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	backendPoolName := azurestackhci.GenerateControlPlaneBackendPoolName(clusterScope.Name())
	loadBalancerScope.SetPort(clusterScope.APIServerPort())
	role := azurestackhci.LBRoleAksHciApiServer
	tags := map[string]*string{azurestackhci.LBRoleTagName: &role}
	lbSpec := &loadbalancers.Spec{
		Name:            loadBalancerScope.AzureStackHCILoadBalancer.Name,
		BackendPoolName: backendPoolName,
		FrontendPort:    loadBalancerScope.GetPort(),
		BackendPort:     clusterScope.APIServerPort(),
		VnetName:        clusterScope.AzureStackHCICluster.Spec.NetworkSpec.Vnet.Name,
		Tags:            tags,
	}

	if err := loadbalancers.NewService(clusterScope).Reconcile(clusterScope.Context, lbSpec); err != nil {
		return errors.Wrapf(err, "failed to reconcile loadbalancer %s", loadBalancerScope.AzureStackHCILoadBalancer.Name)
	}

	return nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileDelete(lbs *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	lbs.Info("Handling deleted AzureStackHCILoadBalancer", "LoadBalancer", lbs.AzureStackHCILoadBalancer.Name)

	if err := r.reconcileDeleteLoadBalancerService(lbs, clusterScope); err != nil {
		r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeWarning, "FailureDeleteLoadBalancer", errors.Wrapf(err, "Error deleting AzureStackHCILoadBalancer %s", lbs.Name()).Error())
		conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, clusterv1.DeletionFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcile.Result{}, err
	}

	if err := r.reconcileDeleteVirtualMachines(lbs, clusterScope); err != nil {
		r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeWarning, "FailureDeleteLoadBalancerMachines", errors.Wrapf(err, "Error deleting machines for AzureStackHCILoadBalancer %s", lbs.Name()).Error())
		conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, clusterv1.DeletionFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcile.Result{}, err
	}

	controllerutil.RemoveFinalizer(lbs.AzureStackHCILoadBalancer, infrav1.AzureStackHCILoadBalancerFinalizer)
	conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, infrav1.LoadBalancerDeletingReason, clusterv1.ConditionSeverityInfo, "")

	r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeNormal, "SuccessfulDeleteLoadBalancer", "Successfully deleted AzureStackHCILoadBalancer %s", lbs.Name())
	return reconcile.Result{}, nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileDeleteLoadBalancerService(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	lbSpec := &loadbalancers.Spec{
		Name: loadBalancerScope.AzureStackHCILoadBalancer.Name,
	}
	if err := loadbalancers.NewService(clusterScope).Delete(clusterScope.Context, lbSpec); err != nil {
		if !azurestackhci.ResourceNotFound(err) {
			return errors.Wrapf(err, "failed to delete loadbalancer %s", loadBalancerScope.AzureStackHCILoadBalancer.Name)
		}
	}

	return nil
}

// reconcileStatus is called after every reconcilitation loop in a defer statement
func (r *AzureStackHCILoadBalancerReconciler) reconcileStatus(lbs *scope.LoadBalancerScope) {
	lb := lbs.AzureStackHCILoadBalancer

	// this is necessary for CRDs including scale subresources
	if lb.Status.Selector == "" {
		lbs.SetSelector(fmt.Sprintf("%s=%s", infrav1.LoadBalancerLabel, lbs.Name()))
	}

	if !lb.DeletionTimestamp.IsZero() {
		lb.Status.SetTypedPhase(infrav1.AzureStackHCILoadBalancerPhaseDeleting)
		return
	}

	if lb.Status.ErrorReason != nil {
		lb.Status.SetTypedPhase(infrav1.AzureStackHCILoadBalancerPhaseFailed)
		return
	}

	if lb.Status.Phase == "" {
		lb.Status.SetTypedPhase(infrav1.AzureStackHCILoadBalancerPhasePending)
		return
	}

	if conditions.IsFalse(lb, infrav1.LoadBalancerReplicasReadyCondition) {
		if *conditions.GetSeverity(lb, infrav1.LoadBalancerReplicasReadyCondition) == clusterv1.ConditionSeverityError {
			cond := conditions.Get(lb, infrav1.LoadBalancerReplicasReadyCondition)
			conditions.MarkFalse(lb, infrav1.LoadBalancerInfrastructureReadyCondition, cond.Reason, cond.Severity, cond.Message)
		}

		if conditions.GetReason(lb, infrav1.LoadBalancerReplicasReadyCondition) == infrav1.LoadBalancerReplicasUpgradingReason {
			lbs.SetPhase(infrav1.AzureStackHCILoadBalancerPhaseUpgrading)
			return
		}

		lbs.SetPhase(infrav1.AzureStackHCILoadBalancerPhaseProvisioning)
		return
	}

	if conditions.IsFalse(lb, infrav1.LoadBalancerInfrastructureReadyCondition) {
		lbs.SetPhase(infrav1.AzureStackHCILoadBalancerPhaseProvisioning)
		return
	}

	lbs.SetPhase(infrav1.AzureStackHCILoadBalancerPhaseProvisioned)
}
