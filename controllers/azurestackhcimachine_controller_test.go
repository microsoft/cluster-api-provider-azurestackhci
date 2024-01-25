package controllers

import (
	"context"
	"time"

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	mock8sclient "github.com/microsoft/cluster-api-provider-azurestackhci/test/mocks/k8s/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("AzureStackHCIMachine Controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		AzureStackHCIMachineName = "test-cluster-control-plane-0"
		AzureStackHCIClusterName = "test-cluster"
		MachineNamespace         = "default"
		ClusterNamespace         = "default"
	)

	var (
		cluster              *clusterv1.Cluster
		machine              *clusterv1.Machine
		azureStackHCICluster *infrav1.AzureStackHCICluster
		azureStackHCIMachine *infrav1.AzureStackHCIMachine
		clusterScope         *scope.ClusterScope
		machineScope         *scope.MachineScope
	)

	ctx := context.Background()

	Context("Unit tests for reconcileVirtualMachineCreate", func() {

		It("should return no requeue and no error when AzureStackHCIVirtualMachine is not found", func() {
			logger := log.FromContext(ctx)

			// Create a test cluster resource
			cluster = &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AzureStackHCIClusterName,
					Namespace: ClusterNamespace,
				},
			}

			// Create a test AzureStackHCICluster resource
			azureStackHCICluster = &infrav1.AzureStackHCICluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AzureStackHCIClusterName,
					Namespace: ClusterNamespace,
				},
			}

			// Create a test machine resource
			machine = &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AzureStackHCIMachineName,
					Namespace: MachineNamespace,
				},
			}

			// Create a test AzureStackHCIMachine resource
			azureStackHCIMachine = &infrav1.AzureStackHCIMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      AzureStackHCIMachineName,
					Namespace: MachineNamespace,
				},
			}

			// Create a new scope for the cluster(creating this way to skip some of the logic in the scope constructor)
			clusterScope = &scope.ClusterScope{
				Logger:               logger,
				Cluster:              cluster,
				AzureStackHCICluster: azureStackHCICluster,
				Context:              ctx,
			}

			// Create a new scope for the machine
			machineScope = &scope.MachineScope{
				Logger:               logger,
				Cluster:              cluster,
				Machine:              machine,
				AzureStackHCICluster: azureStackHCICluster,
				AzureStackHCIMachine: azureStackHCIMachine,
			}

			reconcileResult, reconcileErr := azureStackHCIMachineReconciler.reconcileVirtualMachineDelete(machineScope, clusterScope)
			Expect(reconcileResult).To(Equal(ctrl.Result{}))
			Expect(reconcileErr).ToNot(HaveOccurred())
		})

		It("should return no requeue and error if there is error in getting the AzureStackHCIVirtualMachine resource", func() {
			// Create mocks client
			mockClient := mock8sclient.NewMockClient(mockctrl)

			vmName := apitypes.NamespacedName{
				Namespace: clusterScope.Namespace(),
				Name:      machineScope.Name(),
			}

			// When it asks for the AzureStackHCIVirtualMachine, return some error
			mockClient.EXPECT().Get(ctx, vmName, &infrav1.AzureStackHCIVirtualMachine{}).Return(errors.New("test error"))
			azureStackHCIMachineReconciler.Client = mockClient

			reconcileResult, reconcileErr := azureStackHCIMachineReconciler.reconcileVirtualMachineDelete(machineScope, clusterScope)
			Expect(reconcileResult).To(Equal(ctrl.Result{}))
			Expect(reconcileErr).To(HaveOccurred())
			Expect(reconcileErr.Error()).To(ContainSubstring("test error"))
		})

		It("should return requeue and no error if AzureStackHCIVirtualMachine is found and deletion timestamp is not zero", func() {
			// Create mocks client
			mockClient := mock8sclient.NewMockClient(mockctrl)

			vmName := apitypes.NamespacedName{
				Namespace: clusterScope.Namespace(),
				Name:      machineScope.Name(),
			}

			// Set the deletion timestamp to nil
			azureStackHCIVirtualMachine := &infrav1.AzureStackHCIVirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:              machineScope.Name(),
					Namespace:         machineScope.Namespace(),
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			}

			// When it asks for the AzureStackHCIVirtualMachine, return the test resource
			mockClient.EXPECT().Get(ctx, vmName, &infrav1.AzureStackHCIVirtualMachine{}).Return(nil).SetArg(2, *azureStackHCIVirtualMachine)
			azureStackHCIMachineReconciler.Client = mockClient

			reconcileResult, reconcileErr := azureStackHCIMachineReconciler.reconcileVirtualMachineDelete(machineScope, clusterScope)
			Expect(reconcileResult).To(Equal(ctrl.Result{RequeueAfter: 15 * time.Second}))
			Expect(reconcileErr).ToNot(HaveOccurred())
		})

		It("should return no requeue and error if AzureStackHCIVirtualMachine is found, deletion timestamp is zero and update returns error", func() {
			// Create mocks client
			mockClient := mock8sclient.NewMockClient(mockctrl)

			vmName := apitypes.NamespacedName{
				Namespace: clusterScope.Namespace(),
				Name:      machineScope.Name(),
			}

			// Set the deletion timestamp to nil
			azureStackHCIVirtualMachine := &infrav1.AzureStackHCIVirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:              machineScope.Name(),
					Namespace:         machineScope.Namespace(),
					DeletionTimestamp: nil,
				},
			}

			// When it asks for the AzureStackHCIVirtualMachine, return the test resource
			mockClient.EXPECT().Get(ctx, vmName, &infrav1.AzureStackHCIVirtualMachine{}).Return(nil).SetArg(2, *azureStackHCIVirtualMachine)

			// When it asks to update the AzureStackHCIVirtualMachine, return some error
			mockClient.EXPECT().Update(ctx, azureStackHCIVirtualMachine).Return(errors.New("test error"))
			azureStackHCIMachineReconciler.Client = mockClient

			reconcileResult, reconcileErr := azureStackHCIMachineReconciler.reconcileVirtualMachineDelete(machineScope, clusterScope)
			Expect(reconcileResult).To(Equal(ctrl.Result{}))
			Expect(reconcileErr).To(HaveOccurred())
			Expect(reconcileErr.Error()).To(ContainSubstring("test error"))
		})

		It("should return no requeue and error if AzureStackHCIVirtualMachine is found, deletion timestamp is zero, update returns no error and delete returns error", func() {
			// Create mocks client
			mockClient := mock8sclient.NewMockClient(mockctrl)

			vmName := apitypes.NamespacedName{
				Namespace: clusterScope.Namespace(),
				Name:      machineScope.Name(),
			}

			// Set the deletion timestamp to nil
			azureStackHCIVirtualMachine := &infrav1.AzureStackHCIVirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:              machineScope.Name(),
					Namespace:         machineScope.Namespace(),
					DeletionTimestamp: nil,
				},
			}

			// When it asks for the AzureStackHCIVirtualMachine, return the test resource
			mockClient.EXPECT().Get(ctx, vmName, &infrav1.AzureStackHCIVirtualMachine{}).Return(nil).SetArg(2, *azureStackHCIVirtualMachine)

			// When it asks to update the AzureStackHCIVirtualMachine, return no error
			mockClient.EXPECT().Update(ctx, azureStackHCIVirtualMachine).Return(nil)

			// When it asks to delete the AzureStackHCIVirtualMachine, return some error
			mockClient.EXPECT().Delete(ctx, azureStackHCIVirtualMachine).Return(errors.New("test error"))
			azureStackHCIMachineReconciler.Client = mockClient

			reconcileResult, reconcileErr := azureStackHCIMachineReconciler.reconcileVirtualMachineDelete(machineScope, clusterScope)
			Expect(reconcileResult).To(Equal(ctrl.Result{}))
			Expect(reconcileErr).To(HaveOccurred())
			Expect(reconcileErr.Error()).To(ContainSubstring("test error"))
		})

		It("should return requeue and no error if AzureStackHCIVirtualMachine is found, deletion timestamp is zero, update returns no error and delete returns no error", func() {
			// Create mocks client
			mockClient := mock8sclient.NewMockClient(mockctrl)

			vmName := apitypes.NamespacedName{
				Namespace: clusterScope.Namespace(),
				Name:      machineScope.Name(),
			}

			// Set the deletion timestamp to nil
			azureStackHCIVirtualMachine := &infrav1.AzureStackHCIVirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:              machineScope.Name(),
					Namespace:         machineScope.Namespace(),
					DeletionTimestamp: nil,
				},
			}

			// When it asks for the AzureStackHCIVirtualMachine, return the test resource
			mockClient.EXPECT().Get(ctx, vmName, &infrav1.AzureStackHCIVirtualMachine{}).Return(nil).SetArg(2, *azureStackHCIVirtualMachine)

			// When it asks to update the AzureStackHCIVirtualMachine, return no error
			mockClient.EXPECT().Update(ctx, azureStackHCIVirtualMachine).Return(nil)

			// When it asks to delete the AzureStackHCIVirtualMachine, return no error
			mockClient.EXPECT().Delete(ctx, azureStackHCIVirtualMachine).Return(nil)
			azureStackHCIMachineReconciler.Client = mockClient

			reconcileResult, reconcileErr := azureStackHCIMachineReconciler.reconcileVirtualMachineDelete(machineScope, clusterScope)
			Expect(reconcileResult).To(Equal(ctrl.Result{RequeueAfter: 15 * time.Second}))
			Expect(reconcileErr).ToNot(HaveOccurred())
		})

	})
})
