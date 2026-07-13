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

	"fmt"

	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta2"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/telemetry"
	infrav1util "github.com/microsoft/cluster-api-provider-azurestackhci/pkg/util"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// AzureStackHCIMachineReconciler reconciles a AzureStackHCIMachine object
type AzureStackHCIMachineReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

func (r *AzureStackHCIMachineReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		WithLogConstructor(r.ConstructLogger).
		For(&infrav1.AzureStackHCIMachine{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("AzureStackHCIMachine"))),
		).
		Watches(
			&infrav1.AzureStackHCICluster{},
			handler.EnqueueRequestsFromMapFunc(r.AzureStackHCIClusterToAzureStackHCIMachines),
		).
		Watches(
			&infrav1.AzureStackHCIVirtualMachine{},
			handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &infrav1.AzureStackHCIMachine{}),
		).
		Complete(r)
}

func (r *AzureStackHCIMachineReconciler) ConstructLogger(req *reconcile.Request) logr.Logger {
	log := r.Log.WithName("")
	if req == nil {
		return log
	}
	log = log.WithValues("azureStackHCIMachine", req.NamespacedName)
	cxt := context.Background()
	azureStackHCIMachine := &infrav1.AzureStackHCIMachine{}
	err := r.Get(cxt, req.NamespacedName, azureStackHCIMachine)
	if err != nil {
		log.Error(err, "failed to get azureStackHCIMachine")
		return log
	}
	return log.WithValues("operationId", azureStackHCIMachine.GetAnnotations()[infrav1.AzureOperationIDAnnotationKey],
		"correlationId", azureStackHCIMachine.GetAnnotations()[infrav1.AzureCorrelationIDAnnotationKey])
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhcimachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhcimachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch

func (r *AzureStackHCIMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := r.Log.WithValues("azureStackHCIMachine", req.NamespacedName, "reconcileID", infrav1util.GetReconcileID(ctx))
	logger.Info("Attempt to reconcile resource")

	// Fetch the AzureStackHCIMachine VM.
	azureStackHCIMachine := &infrav1.AzureStackHCIMachine{}
	err := r.Get(ctx, req.NamespacedName, azureStackHCIMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	logger = logger.WithValues("operationId", azureStackHCIMachine.GetAnnotations()[infrav1.AzureOperationIDAnnotationKey], "correlationId", azureStackHCIMachine.GetAnnotations()[infrav1.AzureCorrelationIDAnnotationKey])

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, azureStackHCIMachine.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if machine == nil {
		logger.Info("Machine Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	logger = logger.WithValues("machine", machine.Name)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("Machine is missing cluster label or cluster does not exist")
		return reconcile.Result{}, nil
	}

	logger = logger.WithValues("cluster", cluster.Name)

	azureStackHCICluster := &infrav1.AzureStackHCICluster{}

	azureStackHCIClusterName := client.ObjectKey{
		Namespace: azureStackHCIMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, azureStackHCIClusterName, azureStackHCICluster); err != nil {
		logger.Info("AzureStackHCICluster is not available yet")
		return reconcile.Result{}, nil
	}

	logger = logger.WithValues("azureStackHCICluster", azureStackHCICluster.Name)

	// Create the cluster scope
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:               r.Client,
		Logger:               &logger,
		Cluster:              cluster,
		AzureStackHCICluster: azureStackHCICluster,
	})
	if err != nil {
		r.Recorder.Eventf(azureStackHCIMachine, corev1.EventTypeWarning, "CreateClusterScopeFailed", errors.Wrapf(err, "failed to create cluster scope").Error())
		return reconcile.Result{}, err
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Logger:               &logger,
		Client:               r.Client,
		Cluster:              cluster,
		Machine:              machine,
		AzureStackHCICluster: azureStackHCICluster,
		AzureStackHCIMachine: azureStackHCIMachine,
		Context:              ctx,
	})
	if err != nil {
		r.Recorder.Eventf(azureStackHCIMachine, corev1.EventTypeWarning, "FailureCreateMachineScope", errors.Wrapf(err, "failed to create machine scope").Error())
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any AzureStackHCIMachine changes.
	defer func() {
		if err := machineScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted machines
	if !azureStackHCIMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(machineScope, clusterScope)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(machineScope, clusterScope)
}

func (r *AzureStackHCIMachineReconciler) reconcileNormal(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling AzureStackHCIMachine")
	// NOTE: Do not gate reconciliation on stale Machine conditions (e.g. VMRunningCondition=False).
	// The VM controller owns VM state and can recover asynchronously after transient failures
	// (e.g. MOC I/O errors). The Machine controller must always re-fetch the VM CR to get
	// current status, otherwise a stale error condition permanently blocks reconciliation
	// even after the VM has been successfully created.

	// If the AzureMachine doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(machineScope.AzureStackHCIMachine, infrav1.MachineFinalizer)
	// Register the finalizer immediately to avoid orphaning Azure resources on delete
	if err := machineScope.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}

	// Check if the infrastructure cluster is ready by checking our own AzureStackHCICluster status
	if clusterScope.AzureStackHCICluster.Status.Initialization == nil ||
		clusterScope.AzureStackHCICluster.Status.Initialization.Provisioned == nil ||
		!*clusterScope.AzureStackHCICluster.Status.Initialization.Provisioned {
		machineScope.Info("Cluster infrastructure is not ready yet")
		return reconcile.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	// NOTE: CAPI's Machine controller populates DataSecretName via the bootstrap phase
	// independently of InfraMachine's Provisioned status, so no circular dependency exists.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		return reconcile.Result{}, nil
	}

	vm, err := r.reconcileVirtualMachineNormal(machineScope, clusterScope)

	if err != nil {
		return reconcile.Result{}, err
	}

	// TODO(ncdc): move this validation logic into a validating webhook
	if errs := r.validateUpdate(&machineScope.AzureStackHCIMachine.Spec, vm); len(errs) > 0 {
		agg := kerrors.NewAggregate(errs)
		r.Recorder.Eventf(machineScope.AzureStackHCIMachine, corev1.EventTypeWarning, "InvalidUpdate", "Invalid update: %s", agg.Error())
		return reconcile.Result{}, nil
	}

	// Make sure Spec.ProviderID is always set.
	machineScope.SetProviderID(fmt.Sprintf("moc://%s", vm.Name))

	// TODO(vincepri): Remove this annotation when clusterctl is no longer relevant.
	machineScope.SetAnnotation("cluster-api-provider-azurestackhci", "true")

	// Merge VM conditions into the Machine using conditions.Set (update and insert by type)
	// to avoid duplicating conditions on every reconcile, which would cause
	// infinite growth and a hot reconcile loop.
	for _, vmCond := range vm.Status.Conditions {
		conditions.Set(machineScope.AzureStackHCIMachine, vmCond)
	}

	if vm.Status.VMState == nil {
		machineScope.Info("Waiting for VM controller to set vm state")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
	}

	// changed to avoid using dereference in function param for deep copying
	machineScope.SetVMState(vm.Status.VMState)

	switch *machineScope.GetVMState() {
	case infrav1.VMStateSucceeded:
		machineScope.Info("Machine VM is running", "name", vm.Name)
		machineScope.SetReady()
		conditions.Set(machineScope.AzureStackHCIMachine, metav1.Condition{
			Type:   infrav1.VMRunningCondition,
			Status: metav1.ConditionTrue,
			Reason: "VMRunning",
		})
	case infrav1.VMStateUpdating:
		machineScope.Info("Machine VM is updating", "name", vm.Name)
		conditions.Set(machineScope.AzureStackHCIMachine, metav1.Condition{
			Type:   infrav1.VMRunningCondition,
			Status: metav1.ConditionFalse,
			Reason: "VMUpdating",
		})
	default:
		conditions.Set(machineScope.AzureStackHCIMachine, fallbackVMRunningCondition(
			*machineScope.GetVMState(),
			conditions.Get(machineScope.AzureStackHCIMachine, infrav1.VMRunningCondition),
		))
	}

	// Mirror VMRunningCondition into the standard CAPI "Ready" condition (clusterv1.ReadyCondition)
	// so that CAPI's generic Machine controller (setInfrastructureReadyCondition, which looks
	// specifically for a condition of Type "Ready" on the InfraMachine via
	// contract.InfrastructureMachine().ReadyConditionType()) has something real to mirror onto
	// Machine.Status.Conditions[InfrastructureReadyCondition], instead of always falling back to a
	// generic message derived only from status.initialization.provisioned. See AB#38717753
	// investigation ("alternative fix" track) for full analysis.
	if vmRunning := conditions.Get(machineScope.AzureStackHCIMachine, infrav1.VMRunningCondition); vmRunning != nil {
		conditions.Set(machineScope.AzureStackHCIMachine, deriveMachineReadyCondition(*vmRunning))
	}

	return reconcile.Result{}, nil
}

// deriveMachineReadyCondition mirrors the given VMRunningCondition verbatim into the standard CAPI
// "Ready" condition type. Status/Reason/Message are copied as-is: on success this reproduces
// today's correct True/empty-message behavior (see AB#38717753 investigation §9); on failure this
// is what lets CAPH-specific detail (OutOfCapacity/MocUnreachable/etc.) survive into
// Machine.Status.Conditions[InfrastructureReadyCondition] via CAPI's mirror, instead of being
// replaced by a generic boilerplate message.
//
// OPEN DESIGN QUESTION -- NOT YET RESOLVED, flagging for review rather than deciding unilaterally:
// VMUpdating (set while the VM is being routinely updated, e.g. resized) is mirrored here as
// Ready=False like any other non-success state. Today's CAPI fallback does NOT do this --
// status.initialization.provisioned stays true through a routine update, so Machine readiness
// does not flip during normal operations. Mirroring VMUpdating faithfully (as done here) means
// Ready will flip False during expected, routine VM updates too, which may introduce new flapping
// for autoscaler / MachineHealthCheck / other automation watching Machine readiness. This
// prototype intentionally mirrors verbatim to match the "mirror VMRunning into Ready" idea as
// proposed -- revisit this specific case before considering this fix ready to ship.
func deriveMachineReadyCondition(vmRunning metav1.Condition) metav1.Condition {
	return metav1.Condition{
		Type:    clusterv1.ReadyCondition,
		Status:  vmRunning.Status,
		Reason:  vmRunning.Reason,
		Message: vmRunning.Message,
	}
}

// fallbackVMRunningCondition decides the VMRunningCondition to set on the AzureStackHCIMachine
// when the VM is in a state other than Succeeded/Updating. The VM's own conditions are merged
// onto the Machine (via conditions.Set) immediately before this runs, so existingCondition
// reflects whatever the VM controller itself already reported for this reconcile pass.
//
// If the VM already reported a specific reason (e.g. OutOfCapacityReason, MocUnreachableReason)
// for VMRunningCondition=False, that is preserved as-is -- it is the detail that downstream
// consumers (detector.go regex matching, AKSControlPlaneReadyCondition readers, ICM triage) need.
// Only when the VM did not already report anything specific do we synthesize the generic
// "unexpected state" fallback, so we never silently clobber a more informative message.
//
// (Ported from PR #314 / zilingzhou/moc-unreachable-reason -- included here too so this
// alternative-fix prototype isn't undermined by the same overwrite bug it's meant to fix around.)
func fallbackVMRunningCondition(vmState infrav1.VMState, existingCondition *metav1.Condition) metav1.Condition {
	if existingCondition != nil && existingCondition.Status == metav1.ConditionFalse && existingCondition.Reason != "" {
		return *existingCondition
	}
	return metav1.Condition{
		Type:    infrav1.VMRunningCondition,
		Status:  metav1.ConditionFalse,
		Reason:  infrav1.VMProvisionFailedReason,
		Message: fmt.Sprintf("AzureStackHCI VM state %q is unexpected", vmState),
	}
}

func (r *AzureStackHCIMachineReconciler) reconcileVirtualMachineNormal(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (*infrav1.AzureStackHCIVirtualMachine, error) {
	vm := &infrav1.AzureStackHCIVirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterScope.Namespace(),
			Name:      machineScope.Name(),
		},
	}

	mutateFn := func() (err error) {
		// Mark the AzureStackHCIMachine as the owner of the AzureStackHCIVirtualMachine
		vm.SetOwnerReferences(util.EnsureOwnerRef(
			vm.OwnerReferences,
			metav1.OwnerReference{
				APIVersion: machineScope.Machine.APIVersion,
				Kind:       machineScope.Machine.Kind,
				Name:       machineScope.Machine.Name,
				UID:        machineScope.Machine.UID,
			}))

		vm.Spec.ResourceGroup = clusterScope.AzureStackHCICluster.Spec.ResourceGroup
		vm.Spec.VnetName = clusterScope.AzureStackHCICluster.Spec.NetworkSpec.Vnet.Name
		vm.Spec.ClusterName = clusterScope.AzureStackHCICluster.Name

		backendPoolNames := []string{}
		switch role := machineScope.Role(); role {
		case infrav1.Node:
			vm.Spec.SubnetName = azurestackhci.GenerateNodeSubnetName(clusterScope.Name())
		case infrav1.ControlPlane:
			vm.Spec.SubnetName = azurestackhci.GenerateControlPlaneSubnetName(clusterScope.Name())
			if clusterScope.AzureStackHCILoadBalancer() != nil {
				backendPoolNames = append(backendPoolNames, azurestackhci.GenerateControlPlaneBackendPoolName(clusterScope.Name()))
			}
		default:
			return errors.Errorf("unknown value %s for label `set` on machine %s, unable to create virtual machine resource", role, machineScope.Name())
		}
		//add worker and control plane nodes to the lb backend
		if clusterScope.AzureStackHCILoadBalancer() != nil {
			backendPoolNames = append(backendPoolNames, azurestackhci.GenerateBackendPoolName(clusterScope.Name()))
		}
		vm.Spec.BackendPoolNames = backendPoolNames

		var bootstrapData string
		bootstrapData, err = machineScope.GetBootstrapData()
		if err != nil {
			return errors.Wrap(err, "failed to retrieve bootstrap data")
		}

		image, err := r.getVMImage(machineScope)
		if err != nil {
			return errors.Wrap(err, "failed to get VM image")
		}
		vm.Spec.Image = image.DeepCopy()

		vm.Spec.VMSize = machineScope.AzureStackHCIMachine.Spec.VMSize
		vm.Spec.GpuCount = machineScope.AzureStackHCIMachine.Spec.GpuCount
		if machineScope.AzureStackHCIMachine.Spec.AvailabilityZone != nil {
			vm.Spec.AvailabilityZone = machineScope.AzureStackHCIMachine.Spec.AvailabilityZone.DeepCopy()
		}
		if machineScope.AzureStackHCIMachine.Spec.OSDisk != nil {
			vm.Spec.OSDisk = machineScope.AzureStackHCIMachine.Spec.OSDisk.DeepCopy()
		}
		vm.Spec.Location = machineScope.AzureStackHCIMachine.Spec.Location
		vm.Spec.SSHPublicKey = machineScope.AzureStackHCIMachine.Spec.SSHPublicKey
		vm.Spec.BootstrapData = &bootstrapData
		vm.Spec.AdditionalSSHKeys = machineScope.AzureStackHCIMachine.Spec.AdditionalSSHKeys
		vm.Spec.StorageContainer = machineScope.AzureStackHCIMachine.Spec.StorageContainer
		vm.Spec.AvailabilitySetName = machineScope.AzureStackHCIMachine.Spec.AvailabilitySetName
		vm.Spec.PlacementGroupName = machineScope.AzureStackHCIMachine.Spec.PlacementGroupName

		machineScope.AzureStackHCIMachine.Spec.NetworkInterfaces.DeepCopyInto(&vm.Spec.NetworkInterfaces)

		infrav1util.CopyCorrelationID(machineScope.AzureStackHCIMachine, vm)

		return nil
	}

	operationResult, err := controllerutil.CreateOrUpdate(clusterScope.Context, r.Client, vm, mutateFn)
	if telemetry.IsCRDUpdate(operationResult) {
		operation, resourceType := telemetry.ConvertOperationResult(operationResult)
		telemetry.RecordHybridAKSCRDChange(
			machineScope.GetLogger(),
			clusterScope.GetCustomResourceTypeWithName(),
			fmt.Sprintf("%s/%s/%s", vm.TypeMeta.Kind, vm.ObjectMeta.Namespace, vm.ObjectMeta.Name),
			operation,
			resourceType,
			nil,
			err)
	}
	if err != nil {
		// If CreateOrUpdate throws AlreadyExists, we know that we have encountered an edge case where
		// Get with the cached client returned NotFound and then Create returned AlreadyExists.
		//
		// Because of this, it should be safe to ignore an AlreadyExists from this function. There
		// is the gap that this opens up:
		// 1. CreateOrUpdate is called and creates an object.
		// 2. CreateOrUpdate is called a second time and returns AlreadyExists due to the cache reasoning.
		//
		// Since we are ignoring AlreadyExists, if the second call to CreateOrUpdate has updates to the object,
		// they will not be applied silently.
		//
		// We believe this is ok as this cache issue is only going to be seen very close to when the object was
		// initially created. We are also looking to improve this behavior by introducing live clients or polling.
		if apierrors.IsAlreadyExists(err) {
			machineScope.Info("CreateOrUpdate in reconcileVirtualMachineNormal returned AlreadyExists", "vmName", vm.Name)
		} else {
			return nil, errors.Wrapf(err, "failed to CreateOrUpdate AzureStackHCIVirtualMachine %s", vm.Name)
		}
	}

	azureStackHCIVirtualMachine := &infrav1.AzureStackHCIVirtualMachine{}
	key := client.ObjectKey{
		Namespace: clusterScope.Namespace(),
		Name:      machineScope.Name(),
	}

	err = r.Client.Get(clusterScope.Context, key, azureStackHCIVirtualMachine)
	if err != nil {
		return nil, err
	}

	return azureStackHCIVirtualMachine, nil
}

func (r *AzureStackHCIMachineReconciler) reconcileDelete(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Handling deleted AzureStackHCIMachine", "MachineName", machineScope.AzureStackHCIMachine.Name)

	result, err := r.reconcileVirtualMachineDelete(machineScope, clusterScope)
	if err != nil || result.RequeueAfter > 0 {
		return result, err
	}

	controllerutil.RemoveFinalizer(machineScope.AzureStackHCIMachine, infrav1.MachineFinalizer)

	return reconcile.Result{}, nil
}

func (r *AzureStackHCIMachineReconciler) reconcileVirtualMachineDelete(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	// Use Get to find the VM
	vm := &infrav1.AzureStackHCIVirtualMachine{}
	vmName := apitypes.NamespacedName{
		Namespace: clusterScope.Namespace(),
		Name:      machineScope.Name(),
	}

	if err := r.Client.Get(clusterScope.Context, vmName, vm); err != nil {
		// If the error is other than NotFound, return with error
		if !apierrors.IsNotFound(err) {
			machineScope.Error(err, "failed to get AzureStackHCIVirtualMachine", "vm", vmName)
			return reconcile.Result{}, errors.Wrapf(err, "failed to get AzureStackHCIVirtualMachine %s", vmName)
		}
		// If the VM resource is not found, no need to reconcile again
		return reconcile.Result{}, nil
	}

	// If the VM resource exists and has a deletion timestamp, it means a deletion has been requested.
	// In this case, requeue the request after a delay to check again later if the deletion has been completed.
	if !vm.DeletionTimestamp.IsZero() {
		machineScope.Info("Waiting for AzureStackHCIVirtualMachine deletion to complete", "vm", vm.Name)
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}

	// If the VM resource exists and does not have a deletion timestamp, proceed with the deletion process.
	infrav1util.CopyCorrelationID(machineScope.AzureStackHCIMachine, vm)
	if err := r.Client.Update(clusterScope.Context, vm); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to update AzureStackHCIVirtualMachine %s", vmName)
	}

	// Delete the VM resource
	err := r.Client.Delete(clusterScope.Context, vm)
	telemetry.RecordHybridAKSCRDChange(
		clusterScope.GetLogger(),
		clusterScope.GetCustomResourceTypeWithName(),
		fmt.Sprintf("%s/%s/%s", vm.TypeMeta.Kind, vm.ObjectMeta.Namespace, vm.ObjectMeta.Name),
		telemetry.Delete,
		telemetry.CRD,
		nil,
		err)
	if err != nil && !apierrors.IsNotFound(err) {
		machineScope.Error(err, "failed to delete AzureStackHCIVirtualMachine", "vm", vmName)
		return reconcile.Result{}, errors.Wrapf(err, "failed to delete AzureStackHCIVirtualMachine %s", vmName)
	}

	// Requeue the reconciliation after a delay to check if the deletion has been completed
	return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
}

// validateUpdate checks that no immutable fields have been updated and
// returns a slice of errors representing attempts to change immutable state.
func (r *AzureStackHCIMachineReconciler) validateUpdate(spec *infrav1.AzureStackHCIMachineSpec, i *infrav1.AzureStackHCIVirtualMachine) (errs []error) {
	// TODO: Add comparison logic for immutable fields
	return errs
}

// AzureStackHCIClusterToAzureStackHCIMachines is a handler.ToRequestsFunc to be used to enqueue requests for reconciliation
// of AzureStackHCIMachines.
func (r *AzureStackHCIMachineReconciler) AzureStackHCIClusterToAzureStackHCIMachines(ctx context.Context, o client.Object) []ctrl.Request {
	result := []ctrl.Request{}

	c, ok := o.(*infrav1.AzureStackHCICluster)
	if !ok {
		r.Log.Error(errors.Errorf("expected a AzureStackHCICluster but got a %T", o), "failed to get AzureStackHCIMachine for AzureStackHCICluster")
		return nil
	}
	log := r.Log.WithValues("AzureStackHCICluster", c.Name, "Namespace", c.Namespace)

	cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
	switch {
	case apierrors.IsNotFound(err) || cluster == nil:
		return result
	case err != nil:
		log.Error(err, "failed to get owning cluster")
		return result
	}

	labels := map[string]string{clusterv1.ClusterNameLabel: cluster.Name}
	machineList := &clusterv1.MachineList{}
	if err := r.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
		log.Error(err, "failed to list Machines")
		return nil
	}
	for _, m := range machineList.Items {
		if m.Spec.InfrastructureRef.Name == "" {
			continue
		}
		name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}

	return result
}

// Pick image from the machine configuration, or use a default one.
func (r *AzureStackHCIMachineReconciler) getVMImage(scope *scope.MachineScope) (*infrav1.Image, error) {
	// Use custom image if provided
	if scope.AzureStackHCIMachine.Spec.Image != nil &&
		scope.AzureStackHCIMachine.Spec.Image.Name != nil &&
		*scope.AzureStackHCIMachine.Spec.Image.Name != "" {
		scope.Info("Using custom image name for machine", "machine", scope.AzureStackHCIMachine.GetName(), "imageName", scope.AzureStackHCIMachine.Spec.Image.Name)
		return scope.AzureStackHCIMachine.Spec.Image, nil
	}

	osType := infrav1.OSTypeLinux
	if scope.AzureStackHCIMachine.Spec.Image != nil {
		osType = scope.AzureStackHCIMachine.Spec.Image.OSType
	}
	return azurestackhci.GetDefaultImage(osType, scope.Machine.Spec.Version)
}
