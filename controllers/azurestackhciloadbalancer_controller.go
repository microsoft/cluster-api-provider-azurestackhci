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
	"sort"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha3"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/loadbalancers"
	"github.com/microsoft/moc-sdk-for-go/services/network"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
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

func (r *AzureStackHCILoadBalancerReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.Background()
	logger := r.Log.WithValues("azureStackHCILoadBalancer", req.Name)

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
		Logger:               logger,
		Cluster:              cluster,
		AzureStackHCICluster: azureStackHCICluster,
	})
	if err != nil {
		r.Recorder.Eventf(azureStackHCICluster, corev1.EventTypeWarning, "CreateClusterScopeFailed", errors.Wrapf(err, "failed to create cluster scope").Error())
		return reconcile.Result{}, err
	}

	// create a lb scope for this request.
	loadBalancerScope, err := scope.NewLoadBalancerScope(scope.LoadBalancerScopeParams{
		Logger:                    logger,
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
		r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeWarning, "FailureReconcileLB", errors.Wrapf(err, "Failed to reconcile LoadBalancer service").Error())
		conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerInfrastructureReadyCondition, infrav1.LoadBalancerServiceReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
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

	if lbs.GetReadyReplicas() < 1 {
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

func (r *AzureStackHCILoadBalancerReconciler) reconcileVirtualMachines(lbs *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	loadBalancerVMs, err := r.getVirtualMachinesForLoadBalancer(lbs, clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to get loadbalancer virtual machine list")
	}

	r.updateReplicaStatus(lbs, clusterScope, loadBalancerVMs)

	// before we handle any scaling or upgrade operations, we make sure that all existing replicas we have created are ready
	if lbs.GetReadyReplicas() < lbs.GetReplicas() {
		// continue waiting for upgrade/scaleup to complete, this is a best effort until health checks and remediation are available
		if r.replicasAreUpgrading(lbs) || r.replicasAreScalingUp(lbs) {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
		}

		// some replicas are no longer ready. Unless a scale down operation was requested we will wait for them to become ready again. Scale down
		// will be allowed in order to enable manual remediation for a failed deployment
		if !r.isScaleDownRequired(lbs) {
			conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition, infrav1.LoadBalancerWaitingForReplicasReadyReason, clusterv1.ConditionSeverityInfo, "")
			return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
		}
	}

	// check if we need to scale up
	if r.isScaleUpRequired(lbs) {
		conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition, infrav1.LoadBalancerReplicasScalingUpReason, clusterv1.ConditionSeverityInfo, "")

		err = r.scaleUpVirtualMachines(lbs, clusterScope)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "failed to scale up loadbalancer")
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
	}

	// check if we need to scale down
	if r.isScaleDownRequired(lbs) {
		if !r.replicasAreUpgrading(lbs) {
			conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition, infrav1.LoadBalancerReplicasScalingDownReason, clusterv1.ConditionSeverityInfo, "")
		}

		err = r.scaleDownVirtualMachines(lbs, clusterScope, loadBalancerVMs)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "failed to scale down loadbalancer")
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 20}, nil
	}

	// check if we need to upgrade
	if r.isUpgradeRequired(lbs, loadBalancerVMs) {
		conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition, infrav1.LoadBalancerReplicasUpgradingReason, clusterv1.ConditionSeverityInfo, "")
		r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeNormal, "UpgradingLoadBalancer", "Upgrading AzureStackHCILoadBalancer %s", lbs.Name())

		for lbs.GetReplicas() < lbs.GetMaxReplicas() {
			err = r.scaleUpVirtualMachines(lbs, clusterScope)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "failed to scale up loadbalancer VM for upgrade")
			}
		}
		return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
	}

	// desired state was achieved
	if conditions.IsFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition) {
		conditions.MarkTrue(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition)
		r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeNormal, "LoadBalancerReplicasReady", "All replicas for AzureStackHCILoadBalancer %s are ready", lbs.Name())
	}

	return reconcile.Result{}, nil
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
	lbSpec := &loadbalancers.Spec{
		Name:            loadBalancerScope.AzureStackHCILoadBalancer.Name,
		BackendPoolName: backendPoolName,
		FrontendPort:    loadBalancerScope.GetPort(),
		BackendPort:     clusterScope.APIServerPort(),
		VnetName:        clusterScope.AzureStackHCICluster.Spec.NetworkSpec.Vnet.Name,
	}

	if err := loadbalancers.NewService(clusterScope).Reconcile(clusterScope.Context, lbSpec); err != nil {
		return errors.Wrapf(err, "failed to reconcile loadbalancer %s", loadBalancerScope.AzureStackHCILoadBalancer.Name)
	}

	return nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileDelete(lbs *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	lbs.Info("Handling deleted AzureStackHCILoadBalancer", "LoadBalancer", lbs.AzureStackHCILoadBalancer.Name)

	if err := r.reconcileDeleteLoadBalancer(lbs, clusterScope); err != nil {
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

func (r *AzureStackHCILoadBalancerReconciler) reconcileDeleteVirtualMachines(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	vmList, err := r.getVirtualMachinesForLoadBalancer(loadBalancerScope, clusterScope)
	if err != nil {
		return errors.Wrapf(err, "failed to get loadbalancer virtual machine list")
	}

	for _, vm := range vmList {
		if vm.GetDeletionTimestamp().IsZero() {
			if err := r.Client.Delete(clusterScope.Context, vm); err != nil {
				if !apierrors.IsNotFound(err) {
					return errors.Wrapf(err, "failed to delete AzureStackHCIVirtualMachine %s", vm.Name)
				}
			}
		}
	}

	return nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileDeleteLoadBalancer(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
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

func (r *AzureStackHCILoadBalancerReconciler) createOrUpdateVirtualMachine(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (*infrav1.AzureStackHCIVirtualMachine, error) {
	vm := &infrav1.AzureStackHCIVirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterScope.Namespace(),
			Name:      azurestackhci.GenerateAzureStackHCILoadBalancerMachineName(loadBalancerScope.Name()),
		},
	}

	mutateFn := func() (err error) {
		// Mark the AzureStackHCILoadBalancer as the owner of the AzureStackHCIVirtualMachine
		vm.SetOwnerReferences(util.EnsureOwnerRef(
			vm.OwnerReferences,
			metav1.OwnerReference{
				APIVersion: loadBalancerScope.AzureStackHCILoadBalancer.APIVersion,
				Kind:       loadBalancerScope.AzureStackHCILoadBalancer.Kind,
				Name:       loadBalancerScope.AzureStackHCILoadBalancer.Name,
				UID:        loadBalancerScope.AzureStackHCILoadBalancer.UID,
			}))

		labels := vm.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels[infrav1.OSVersionLabelName] = loadBalancerScope.Cluster.Labels[infrav1.OSVersionLabelName]
		labels[infrav1.LoadBalancerLabel] = loadBalancerScope.Name()
		vm.SetLabels(labels)

		vm.Spec.ResourceGroup = clusterScope.AzureStackHCICluster.Spec.ResourceGroup
		vm.Spec.VnetName = clusterScope.AzureStackHCICluster.Spec.NetworkSpec.Vnet.Name
		vm.Spec.ClusterName = clusterScope.AzureStackHCICluster.Name
		vm.Spec.SubnetName = azurestackhci.GenerateNodeSubnetName(clusterScope.Name())
		bootstrapdata := ""
		vm.Spec.BootstrapData = &bootstrapdata
		vm.Spec.VMSize = loadBalancerScope.AzureStackHCILoadBalancer.Spec.VMSize
		vm.Spec.Location = clusterScope.Location()
		vm.Spec.SSHPublicKey = loadBalancerScope.AzureStackHCILoadBalancer.Spec.SSHPublicKey

		image, err := r.getVMImage(loadBalancerScope)
		if err != nil {
			return errors.Wrap(err, "failed to get AzureStackHCILoadBalancer image")
		}
		image.DeepCopyInto(&vm.Spec.Image)

		return nil
	}

	if _, err := controllerutil.CreateOrUpdate(clusterScope.Context, r.Client, vm, mutateFn); err != nil {
		return nil, err
	}

	return vm, nil
}

func (r *AzureStackHCILoadBalancerReconciler) deleteVirtualMachine(clusterScope *scope.ClusterScope, vm *infrav1.AzureStackHCIVirtualMachine) error {
	if vm.GetDeletionTimestamp().IsZero() {
		if err := r.Client.Delete(clusterScope.Context, vm); err != nil {
			if !apierrors.IsNotFound(err) {
				return errors.Wrapf(err, "failed to delete AzureStackHCIVirtualMachine %s", vm.Name)
			}
		}
	}
	return nil
}

func (r *AzureStackHCILoadBalancerReconciler) getVMImage(loadBalancerScope *scope.LoadBalancerScope) (*infrav1.Image, error) {
	// Use custom image if provided
	if loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image.Name != nil {
		loadBalancerScope.Info("Using custom image name for loadbalancer", "loadbalancer", loadBalancerScope.AzureStackHCILoadBalancer.GetName(), "imageName", loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image.Name)
		return &loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image, nil
	}

	return azurestackhci.GetDefaultImage(loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image.OSType, to.String(loadBalancerScope.AzureStackHCICluster.Spec.Version))
}

// getVirtualMachinesForLoadBalancer returns a list of non-deleted AzureStackHCIVirtualMachines associated with the load balancer
func (r *AzureStackHCILoadBalancerReconciler) getVirtualMachinesForLoadBalancer(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) ([]*infrav1.AzureStackHCIVirtualMachine, error) {
	labels := map[string]string{infrav1.LoadBalancerLabel: loadBalancerScope.Name()}
	vmList := &infrav1.AzureStackHCIVirtualMachineList{}

	if err := r.Client.List(
		clusterScope.Context,
		vmList,
		client.InNamespace(clusterScope.Namespace()),
		client.MatchingLabels(labels)); err != nil {
		return nil, err
	}

	filtered := make([]*infrav1.AzureStackHCIVirtualMachine, 0, len(vmList.Items))
	for idx := range vmList.Items {
		vm := &vmList.Items[idx]
		if vm.GetDeletionTimestamp().IsZero() {
			filtered = append(filtered, vm)
		}
	}

	return filtered, nil
}

// updateReplicaStatus updates the loadbalancer status using the specified replica information
func (r *AzureStackHCILoadBalancerReconciler) updateReplicaStatus(lbs *scope.LoadBalancerScope, clusterScope *scope.ClusterScope, vmList []*infrav1.AzureStackHCIVirtualMachine) {
	replicas, failedReplicas := r.getMachineReplicaCounts(vmList)

	// best effort attempt to update ready replica count
	r.reconcileLoadBalancerServiceStatus(lbs, clusterScope)

	lbs.SetReplicas(replicas)
	lbs.SetFailedReplicas(failedReplicas)

	lbs.Info("Updated replication status", "replicas", lbs.GetReplicas(), "readyReplicas", lbs.GetReadyReplicas(), "failedReplicas", lbs.GetFailedReplicas())
}

// getMachineReplicaCounts calculates the replica counts for the AzureStackHCIVirtualMachines associated with the load balancer
func (r *AzureStackHCILoadBalancerReconciler) getMachineReplicaCounts(vmList []*infrav1.AzureStackHCIVirtualMachine) (replicas, failedReplicas int32) {
	replicas = int32(len(vmList))
	for _, vm := range vmList {
		if vm.Status.VMState == nil {
			continue
		}
		switch *vm.Status.VMState {
		case infrav1.VMStateFailed:
			failedReplicas++
		}
	}
	return
}

// replicasAreUpgrading checks if the replicas are in the process of upgrading
func (r *AzureStackHCILoadBalancerReconciler) replicasAreUpgrading(lbs *scope.LoadBalancerScope) bool {
	return conditions.GetReason(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition) == infrav1.LoadBalancerReplicasUpgradingReason
}

// replicasAreScalingUp checks if the replicas are in the process of scaling up
func (r *AzureStackHCILoadBalancerReconciler) replicasAreScalingUp(lbs *scope.LoadBalancerScope) bool {
	return conditions.GetReason(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition) == infrav1.LoadBalancerReplicasScalingUpReason
}

// replicasAreScalingDown checks if the replicas are in the process of scaling down
func (r *AzureStackHCILoadBalancerReconciler) replicasAreScalingDown(lbs *scope.LoadBalancerScope) bool {
	return conditions.GetReason(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition) == infrav1.LoadBalancerReplicasScalingDownReason
}

// replicasAreScaling checks if the replicas are in the process of a scale operation
func (r *AzureStackHCILoadBalancerReconciler) replicasAreScaling(lbs *scope.LoadBalancerScope) bool {
	return r.replicasAreScalingUp(lbs) || r.replicasAreScalingDown(lbs)
}

// isScaleUpRequired determines if the loadbalancer replicas need to be scaled up to meet desired state
func (r *AzureStackHCILoadBalancerReconciler) isScaleUpRequired(loadBalancerScope *scope.LoadBalancerScope) bool {
	return loadBalancerScope.GetReplicas() < loadBalancerScope.GetDesiredReplicas()
}

// isScaleDownRequired determines if the loadbalancer replicas need to be scaled down to meet desired state
func (r *AzureStackHCILoadBalancerReconciler) isScaleDownRequired(loadBalancerScope *scope.LoadBalancerScope) bool {
	return loadBalancerScope.GetReplicas() > loadBalancerScope.GetDesiredReplicas()
}

// isUpgradeRequired determines if there are any replicas which need to be upgraded
func (r *AzureStackHCILoadBalancerReconciler) isUpgradeRequired(lbs *scope.LoadBalancerScope, vmList []*infrav1.AzureStackHCIVirtualMachine) bool {
	latestOSVersion := lbs.Cluster.Labels[infrav1.OSVersionLabelName]
	for _, vm := range vmList {
		currentOSVersion := vm.Labels[infrav1.OSVersionLabelName]
		if currentOSVersion != latestOSVersion {
			return true
		}
	}
	return false
}

// scaleUpVirtualMachines scales up by creating a virtual machine replica
func (r *AzureStackHCILoadBalancerReconciler) scaleUpVirtualMachines(lbs *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeNormal, "ScalingUpLoadBalancer", "Scaling up AzureStackHCILoadBalancer %s", lbs.Name())

	_, err := r.createOrUpdateVirtualMachine(lbs, clusterScope)
	if err != nil {
		return errors.Wrapf(err, "failed to create or update loadbalancer VM")
	}

	lbs.AddReplica()
	return nil
}

// scaleDownVirtualMachines scales down by deleting a virtual machine replica
func (r *AzureStackHCILoadBalancerReconciler) scaleDownVirtualMachines(lbs *scope.LoadBalancerScope, clusterScope *scope.ClusterScope, vmList []*infrav1.AzureStackHCIVirtualMachine) error {
	r.Recorder.Eventf(lbs.AzureStackHCILoadBalancer, corev1.EventTypeNormal, "ScalingDownLoadBalancer", "Scaling down AzureStackHCILoadBalancer %s", lbs.Name())

	vm, err := r.selectVirtualMachineForScaleDown(lbs, vmList)
	if err != nil {
		return err
	}

	err = r.deleteVirtualMachine(clusterScope, vm)
	if err != nil {
		return errors.Wrapf(err, "failed to scale down loadbalancer VM %s", vm.Name)
	}

	lbs.RemoveReplica()
	return nil
}

// selectVirtualMachineForScaleDown determines the next machine to be deleted when scaling down
func (r *AzureStackHCILoadBalancerReconciler) selectVirtualMachineForScaleDown(lbs *scope.LoadBalancerScope, vmList []*infrav1.AzureStackHCIVirtualMachine) (*infrav1.AzureStackHCIVirtualMachine, error) {
	if len(vmList) < 1 {
		return nil, fmt.Errorf("no machines were provided for scale down selection")
	}

	latestOSVersion := lbs.Cluster.Labels[infrav1.OSVersionLabelName]

	// find the oldest machine which is not running the latest os version
	sort.Sort(infrav1.VirtualMachinesByCreationTimestamp(vmList))
	for _, vm := range vmList {
		currentOSVersion := vm.Labels[infrav1.OSVersionLabelName]
		if currentOSVersion != latestOSVersion {
			return vm, nil
		}
	}

	// all machines are running the latest os version, so we just select the oldest machine
	return vmList[0], nil
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
