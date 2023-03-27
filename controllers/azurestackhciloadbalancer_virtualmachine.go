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
	"sort"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *AzureStackHCILoadBalancerReconciler) reconcileVirtualMachines(lbs *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	loadBalancerVMs, err := r.getVirtualMachinesForLoadBalancer(lbs, clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to get loadbalancer virtual machine list")
	}

	for _, vm := range loadBalancerVMs {
		if conditions.IsFalse(vm, infrav1.VMRunningCondition) {
			cond := conditions.Get(vm, infrav1.VMRunningCondition)
			if cond.Severity == clusterv1.ConditionSeverityError {
				conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition, cond.Reason, cond.Severity, cond.Message)
				return reconcile.Result{}, nil
			}
		}
	}

	r.updateReplicaStatus(lbs, clusterScope, loadBalancerVMs)

	// before we handle any scaling or upgrade operations, we make sure that all existing replicas we have created are ready
	if lbs.GetReadyReplicas() < lbs.GetReplicas() {
		// continue waiting for upgrade/scaleup to complete, this is a best effort until health checks and remediation are available
		if r.replicasAreUpgrading(lbs) || r.replicasAreScalingUp(lbs) {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
		}

		// some replicas are no longer ready. Unless a scale down operation was requested we will wait for them to become ready again
		if !r.isScaleDownRequired(lbs) {
			conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition, infrav1.LoadBalancerWaitingForReplicasReadyReason, clusterv1.ConditionSeverityInfo, "")
			return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
		}
	}

	// check if we need to scale up
	if r.isScaleUpRequired(lbs) {
		if !r.replicasAreUpgrading(lbs) {
			conditions.MarkFalse(lbs.AzureStackHCILoadBalancer, infrav1.LoadBalancerReplicasReadyCondition, infrav1.LoadBalancerReplicasScalingUpReason, clusterv1.ConditionSeverityInfo, "")
		}

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
		labels[infrav1.OSVersionLabelName] = loadBalancerScope.OSVersion()
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

// deleteVirtualMachine deletes a virtual machine
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

	// find the oldest machine which is not running the latest os version
	sort.Sort(infrav1.VirtualMachinesByCreationTimestamp(vmList))
	for _, vm := range vmList {
		if vm.Labels[infrav1.OSVersionLabelName] != lbs.OSVersion() {
			return vm, nil
		}
	}

	// all machines are running the latest os version, so we just select the oldest machine
	return vmList[0], nil
}

// getVMImage returns the image to use for a virtual machine
func (r *AzureStackHCILoadBalancerReconciler) getVMImage(loadBalancerScope *scope.LoadBalancerScope) (*infrav1.Image, error) {
	// Use custom image if provided
	if loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image.Name != nil && *loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image.Name != "" {
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
	for _, vm := range vmList {
		if vm.Labels[infrav1.OSVersionLabelName] != lbs.OSVersion() {
			return true
		}
	}
	return false
}
