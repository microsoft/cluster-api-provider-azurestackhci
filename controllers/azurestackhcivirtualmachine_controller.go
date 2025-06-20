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

	"github.com/go-logr/logr"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	infrav1util "github.com/microsoft/cluster-api-provider-azurestackhci/pkg/util"
	mocerrors "github.com/microsoft/moc/pkg/errors"
	moccodes "github.com/microsoft/moc/pkg/errors/codes"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
)

// AzureStackHCIVirtualMachineReconciler reconciles a AzureStackHCIVirtualMachine object
type AzureStackHCIVirtualMachineReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// SetupWithManager registers the controller with the k8s manager
func (r *AzureStackHCIVirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		WithLogConstructor(r.ConstructLogger).
		For(&infrav1.AzureStackHCIVirtualMachine{}).
		Complete(r)
}

func (r *AzureStackHCIVirtualMachineReconciler) ConstructLogger(req *reconcile.Request) logr.Logger {
	log := r.Log.WithName("")
	if req == nil {
		return log
	}
	log = log.WithValues("azureStackHCIVirtualMachine", req.NamespacedName)
	cxt := context.Background()
	azureStackHCIVirtualMachine := &infrav1.AzureStackHCIVirtualMachine{}
	err := r.Get(cxt, req.NamespacedName, azureStackHCIVirtualMachine)
	if err != nil {
		log.Error(err, "failed to get azureStackHCIVirtualMachine")
		return log
	}
	return log.WithValues("operationId", azureStackHCIVirtualMachine.GetAnnotations()[infrav1.AzureOperationIDAnnotationKey],
		"correlationId", azureStackHCIVirtualMachine.GetAnnotations()[infrav1.AzureCorrelationIDAnnotationKey])
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhcivirtualmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhcivirtualmachines/status,verbs=get;update;patch

// Reconcile reacts to some event on the kubernetes object that the controller has registered to handle
func (r *AzureStackHCIVirtualMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := r.Log.WithValues("azureStackHCIVirtualMachine", req.NamespacedName, "reconcileID", infrav1util.GetReconcileID(ctx))
	logger.Info("Attempt to reconcile resource")

	azureStackHCIVirtualMachine := &infrav1.AzureStackHCIVirtualMachine{}
	err := r.Get(ctx, req.NamespacedName, azureStackHCIVirtualMachine)
	if err != nil {
		logger.Info("resource not found", "name", req.NamespacedName)
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	logger = logger.WithValues("operationId", azureStackHCIVirtualMachine.GetAnnotations()[infrav1.AzureOperationIDAnnotationKey], "correlationId", azureStackHCIVirtualMachine.GetAnnotations()[infrav1.AzureCorrelationIDAnnotationKey])

	// Create the machine scope
	virtualMachineScope, err := scope.NewVirtualMachineScope(scope.VirtualMachineScopeParams{
		Logger:                      &logger,
		Client:                      r.Client,
		AzureStackHCIVirtualMachine: azureStackHCIVirtualMachine,
	})
	if err != nil {
		r.Recorder.Eventf(azureStackHCIVirtualMachine, corev1.EventTypeWarning, "FailureCreateVMScope", errors.Wrapf(err, "failed to create VM scope").Error())
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any AzureStackHCIVirtualMachine changes.
	defer func() {
		if err := virtualMachineScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted machines
	if !azureStackHCIVirtualMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(virtualMachineScope)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(virtualMachineScope)
}

// findVM queries the AzureStackHCI APIs and retrieves the VM if it exists, returns nil otherwise.
func (r *AzureStackHCIVirtualMachineReconciler) findVM(scope *scope.VirtualMachineScope, ams *azureStackHCIVirtualMachineService) (*infrav1.VM, error) {
	var vm *infrav1.VM

	vm, err := ams.VMIfExists()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query AzureStackHCIVirtualMachine")
	}

	return vm, nil
}

func (r *AzureStackHCIVirtualMachineReconciler) reconcileNormal(virtualMachineScope *scope.VirtualMachineScope) (reconcile.Result, error) {
	virtualMachineScope.Info("Reconciling AzureStackHCIVirtualMachine")
	// If the AzureStackHCIVirtualMachine is in an error state, return early.
	if virtualMachineScope.AzureStackHCIVirtualMachine.Status.FailureReason != nil || virtualMachineScope.AzureStackHCIVirtualMachine.Status.FailureMessage != nil {
		virtualMachineScope.Info("Error state detected, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	// If the AzureStackHCIVirtualMachine doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VirtualMachineFinalizer)
	// Register the finalizer immediately to avoid orphaning resources on delete
	if err := virtualMachineScope.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}

	ams := newAzureStackHCIVirtualMachineService(virtualMachineScope)

	// Get or create the virtual machine.
	vm, err := r.getOrCreate(virtualMachineScope, ams)
	if err != nil {
		return reconcile.Result{}, err
	}

	/*
		// right now validateUpdate seems to be a no-op so skipping this logic for now
		// TODO(ncdc): move this validation logic into a validating webhook
		if errs := r.validateUpdate(&virtualMachineScope.AzureStackHCIMachine.Spec, vm); len(errs) > 0 {
			agg := kerrors.NewAggregate(errs)
			r.Recorder.Eventf(virtualMachineScope.AzureStackHCIMachine, corev1.EventTypeWarning, "InvalidUpdate", "Invalid update: %s", agg.Error())
			return reconcile.Result{}, nil
		} */

	// Proceed to reconcile the AzureStackHCIVirtualMachine state.
	virtualMachineScope.SetVMState(vm.State)

	switch vm.State {
	case infrav1.VMStateSucceeded:
		virtualMachineScope.Info("Machine VM is running", "name", virtualMachineScope.Name())
		virtualMachineScope.SetReady()
		conditions.MarkTrue(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition)
	case infrav1.VMStateUpdating:
		virtualMachineScope.Info("Machine VM is updating", "name", virtualMachineScope.Name())
		conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, infrav1.VMUpdatingReason, clusterv1.ConditionSeverityInfo, "")
	default:
		virtualMachineScope.SetFailureReason(capierrors.UpdateMachineError)
		virtualMachineScope.SetFailureMessage(errors.Errorf("AzureStackHCI VM state %q is unexpected", vm.State))
		r.Recorder.Eventf(virtualMachineScope.AzureStackHCIVirtualMachine, corev1.EventTypeWarning, "UnexpectedVMState", "AzureStackHCIVirtualMachine is in an unexpected state %q", vm.State)
		conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, infrav1.VMProvisionFailedReason, clusterv1.ConditionSeverityWarning, "")
	}

	return reconcile.Result{}, nil
}

func (r *AzureStackHCIVirtualMachineReconciler) getOrCreate(virtualMachineScope *scope.VirtualMachineScope, ams *azureStackHCIVirtualMachineService) (*infrav1.VM, error) {
	virtualMachineScope.Info("Attempting to find VM", "Name", virtualMachineScope.Name())
	vm, err := r.findVM(virtualMachineScope, ams)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "Failed to query for AzureStackHCIVirtualMachine %s/%s", virtualMachineScope.Namespace(), virtualMachineScope.Name())
		r.Recorder.Eventf(virtualMachineScope.AzureStackHCIVirtualMachine, corev1.EventTypeWarning, "FailureQueryForVM", wrappedErr.Error())
		conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, infrav1.VMNotFoundReason, clusterv1.ConditionSeverityError, err.Error())
		return nil, err
	}

	if vm == nil {
		// Create a new AzureStackHCIVirtualMachine if we couldn't find a running VM.
		virtualMachineScope.Info("No VM found, creating VM", "Name", virtualMachineScope.Name())
		vm, err = ams.Create()
		if err != nil {
			switch mocerrors.GetErrorCode(err) {
			case mocerrors.OutOfMemory.Error():
				conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, infrav1.OutOfMemoryReason, clusterv1.ConditionSeverityError, err.Error())
			case mocerrors.OutOfCapacity.Error():
				conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, infrav1.OutOfCapacityReason, clusterv1.ConditionSeverityError, err.Error())
			case mocerrors.OutOfNodeCapacity.Error():
				conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, infrav1.OutOfNodeCapacityReason, clusterv1.ConditionSeverityWarning, err.Error())
			case mocerrors.NotFound.Error(): // "NotFound"
				fallthrough
			// Internally, NotFound is a legacy error and returns the error string instead.
			case moccodes.NotFound.String(): // "Not Found"
				if mocerrors.IsPathNotFound(err) {
					conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, infrav1.PathNotFoundReason, clusterv1.ConditionSeverityError, err.Error())
				} else {
					conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, infrav1.NotFoundReason, clusterv1.ConditionSeverityError, err.Error())
				}
			default:
				conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, infrav1.VMProvisionFailedReason, clusterv1.ConditionSeverityError, err.Error())
			}

			wrappedErr := errors.Wrapf(err, "failed to create AzureStackHCIVirtualMachine")
			r.Recorder.Eventf(virtualMachineScope.AzureStackHCIVirtualMachine, corev1.EventTypeWarning, "FailureCreateVM", wrappedErr.Error())

			return nil, wrappedErr
		}
		r.Recorder.Eventf(virtualMachineScope.AzureStackHCIVirtualMachine, corev1.EventTypeNormal, "SuccessfulCreateVM", "Success creating AzureStackHCIVirtualMachine %s/%s", virtualMachineScope.Namespace(), virtualMachineScope.Name())
	}

	return vm, nil
}

func (r *AzureStackHCIVirtualMachineReconciler) reconcileDelete(virtualMachineScope *scope.VirtualMachineScope) (_ reconcile.Result, reterr error) {
	virtualMachineScope.Info("Handling deleted AzureStackHCIVirtualMachine", "Name", virtualMachineScope.Name())

	conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, clusterv1.DeletingReason, clusterv1.ConditionSeverityInfo, "")

	if err := newAzureStackHCIVirtualMachineService(virtualMachineScope).Delete(); err != nil {
		wrappedErr := errors.Wrapf(err, "error deleting AzureStackHCIVirtualMachine %s/%s", virtualMachineScope.Namespace(), virtualMachineScope.Name())
		r.Recorder.Eventf(virtualMachineScope.AzureStackHCIVirtualMachine, corev1.EventTypeWarning, "FailureDeleteVM", wrappedErr.Error())
		conditions.MarkFalse(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VMRunningCondition, clusterv1.DeletionFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcile.Result{}, wrappedErr
	}
	r.Recorder.Eventf(virtualMachineScope.AzureStackHCIVirtualMachine, corev1.EventTypeNormal, "SuccessfulDeleteVM", "Success deleting AzureStackHCIVirtualMachine %s/%s", virtualMachineScope.Namespace(), virtualMachineScope.Name())

	controllerutil.RemoveFinalizer(virtualMachineScope.AzureStackHCIVirtualMachine, infrav1.VirtualMachineFinalizer)

	return reconcile.Result{}, nil
}

// validateUpdate checks that no immutable fields have been updated and
// returns a slice of errors representing attempts to change immutable state.
func (r *AzureStackHCIVirtualMachineReconciler) validateUpdate(spec *infrav1.AzureStackHCIVirtualMachineSpec, i *infrav1.VM) (errs []error) {
	// TODO: Add comparison logic for immutable fields
	return errs
}
