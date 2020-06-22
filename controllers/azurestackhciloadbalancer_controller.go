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
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha3"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/groups"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/loadbalancers"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/networkinterfaces"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/vippools"
	"github.com/microsoft/moc-sdk-for-go/services/cloud"
	"github.com/microsoft/moc-sdk-for-go/services/network"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
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
	useVIP   bool
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
	logger := r.Log.WithValues("namespace", req.Namespace, "azurestackhciloadBalancer", req.Name)

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

	azureStackHCICluster := &infrav1.AzureStackHCICluster{}

	azureStackHCIClusterName := client.ObjectKey{
		Namespace: azureStackHCILoadBalancer.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, azureStackHCIClusterName, azureStackHCICluster); err != nil {
		logger.Info("AzureStackHCICluster is not available yet")
		return reconcile.Result{}, nil
	}

	// create a cluster scope for the request.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:               r.Client,
		Logger:               logger.WithValues("cluster", cluster.Name),
		Cluster:              cluster,
		AzureStackHCICluster: azureStackHCICluster,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// create a lb scope for this request.
	loadBalancerScope, err := scope.NewLoadBalancerScope(scope.LoadBalancerScopeParams{
		Logger:                    logger.WithValues("azureStackHCILoadBalancer", azureStackHCILoadBalancer.Name),
		Client:                    r.Client,
		AzureStackHCILoadBalancer: azureStackHCILoadBalancer,
		AzureStackHCICluster:      azureStackHCICluster,
		Cluster:                   cluster,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any AzureStackHCILoadBalancer changes.
	defer func() {
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

func (r *AzureStackHCILoadBalancerReconciler) reconcileNormal(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	loadBalancerScope.Info("Reconciling AzureStackHCILoadBalancer")

	// If the AzureStackHCILoadBalancer doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(loadBalancerScope.AzureStackHCILoadBalancer, infrav1.AzureStackHCILoadBalancerFinalizer)
	// Register the finalizer immediately to avoid orphaning resources on delete
	if err := loadBalancerScope.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}

	vm, err := r.reconcileNormalVirtualMachine(loadBalancerScope, clusterScope)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			clusterScope.Info("AzureStackHCIVirtualMachine already exists")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if vm.Status.VMState == nil {
		loadBalancerScope.Info("Waiting for VM controller to set vm state")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Minute}, nil
	}

	// changed to avoid using dereference in function param for deep copying
	loadBalancerScope.SetVMState(vm.Status.VMState)

	switch *loadBalancerScope.GetVMState() {
	case infrav1.VMStateSucceeded:
		loadBalancerScope.Info("Machine VM is running", "name", vm.Name)
		loadBalancerScope.SetReady()
	case infrav1.VMStateUpdating:
		loadBalancerScope.Info("Machine VM is updating", "name", vm.Name)
	default:
		loadBalancerScope.SetErrorReason(capierrors.UpdateMachineError)
		loadBalancerScope.SetErrorMessage(errors.Errorf("AzureStackHCI VM state %q is unexpected", *loadBalancerScope.GetVMState()))
	}

	// reconcile the loadbalancer
	err = r.reconcileLoadBalancer(loadBalancerScope, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	// wait for ip address to be exposed
	if loadBalancerScope.Address() == "" {
		err := r.reconcileLoadBalancerAddress(loadBalancerScope, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		if loadBalancerScope.Address() == "" {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 20}, nil
		}
	}

	return reconcile.Result{}, nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileNormalVirtualMachine(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (*infrav1.AzureStackHCIVirtualMachine, error) {
	vm := &infrav1.AzureStackHCIVirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterScope.Namespace(),
			Name:      loadBalancerScope.Name(),
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

		vm.Spec.ResourceGroup = clusterScope.AzureStackHCICluster.Spec.ResourceGroup
		vm.Spec.VnetName = clusterScope.AzureStackHCICluster.Spec.NetworkSpec.Vnet.Name
		vm.Spec.ClusterName = clusterScope.AzureStackHCICluster.Name
		vm.Spec.SubnetName = azurestackhci.GenerateNodeSubnetName(clusterScope.Name())
		vm.Spec.BootstrapData = r.formatLoadBalancerCloudInit(loadBalancerScope, clusterScope)
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

func (r *AzureStackHCILoadBalancerReconciler) reconcileLoadBalancerAddress(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	if r.useVIP {
		loadBalancerScope.Info("Attempting to vip for azurestackhciloadbalancer", "name", loadBalancerScope.AzureStackHCILoadBalancer.Name)
		lbSpec := &loadbalancers.Spec{
			Name: loadBalancerScope.AzureStackHCILoadBalancer.Name,
		}
		lbInterface, err := loadbalancers.NewService(clusterScope).Get(clusterScope.Context, lbSpec)
		if err != nil {
			return err
		}

		lb, ok := lbInterface.(network.LoadBalancer)
		if !ok {
			return errors.New("error getting load balancer")
		}

		loadBalancerScope.SetAddress(*((*lb.FrontendIPConfigurations)[0].IPAddress))
	} else {
		loadBalancerScope.Info("Attempting to get network interface information for loadbalancer", "name", loadBalancerScope.AzureStackHCILoadBalancer.Name)
		nicInterface, err := networkinterfaces.NewService(clusterScope).Get(clusterScope.Context,
			&networkinterfaces.Spec{
				Name:     azurestackhci.GenerateNICName(loadBalancerScope.AzureStackHCILoadBalancer.Name),
				VnetName: clusterScope.AzureStackHCICluster.Spec.NetworkSpec.Vnet.Name,
			})
		if err != nil {
			return err
		}

		nic, ok := nicInterface.(network.Interface)
		if !ok {
			return errors.New("error getting network interface")
		}

		if nic.IPConfigurations != nil && len(*nic.IPConfigurations) > 0 && (*nic.IPConfigurations)[0].PrivateIPAddress != nil && *((*nic.IPConfigurations)[0].PrivateIPAddress) != "" {
			loadBalancerScope.SetAddress(*((*nic.IPConfigurations)[0].PrivateIPAddress))
			loadBalancerScope.Info("Load balancer address is available", "address", loadBalancerScope.Address())
		} else {
			loadBalancerScope.Info("Load balancer address is not yet available")
		}
	}
	return nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileLoadBalancer(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	backendPoolName := azurestackhci.GenerateControlPlaneBackendPoolName(clusterScope.Name())
	loadBalancerScope.SetPort(clusterScope.APIServerPort())
	lbSpec := &loadbalancers.Spec{
		Name:            loadBalancerScope.AzureStackHCILoadBalancer.Name,
		BackendPoolName: backendPoolName,
		FrontendPort:    loadBalancerScope.GetPort(),
		BackendPort:     clusterScope.APIServerPort(),
	}

	// Currently, CAPI doesn't have correct location.
	loadBalancerScope.Info("Attempting to get location for group", "group", clusterScope.GetResourceGroup())
	groupInterface, err := groups.NewService(clusterScope).Get(clusterScope.Context, &groups.Spec{Name: clusterScope.GetResourceGroup()})
	if err != nil {
		return err
	}

	group, ok := groupInterface.(cloud.Group)
	if !ok {
		return errors.New("error getting group")
	}
	location := *group.Location

	// If vippool does not exists, specify vnetname.
	loadBalancerScope.Info("Attempting to get vippool for location", "location", location)
	vippool, err := vippools.NewService(clusterScope).Get(clusterScope.Context, &vippools.Spec{Location: location})
	if err == nil && vippool != nil {
		loadBalancerScope.Info("Using vippool", "vippool", vippool)
		r.useVIP = true
	} else {
		r.useVIP = false
		loadBalancerScope.Info("Vippool does not exist at location. Using the ip address of the virtual machine as the frontend", "location", location)
		lbSpec.VnetName = clusterScope.AzureStackHCICluster.Spec.NetworkSpec.Vnet.Name
	}

	if err := loadbalancers.NewService(clusterScope).Reconcile(clusterScope.Context, lbSpec); err != nil {
		return errors.Wrapf(err, "failed to reconcile loadbalancer %s", loadBalancerScope.AzureStackHCILoadBalancer.Name)
	}

	return nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileDelete(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (_ reconcile.Result, reterr error) {
	loadBalancerScope.Info("Handling deleted AzureStackHCILoadBalancer")

	if err := r.reconcileDeleteLoadBalancer(loadBalancerScope, clusterScope); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileDeleteVirtualMachine(loadBalancerScope, clusterScope); err != nil {
		return reconcile.Result{}, err
	}

	defer func() {
		if reterr == nil {
			// VM is deleted so remove the finalizer.
			controllerutil.RemoveFinalizer(loadBalancerScope.AzureStackHCILoadBalancer, infrav1.AzureStackHCILoadBalancerFinalizer)
		}
	}()

	return reconcile.Result{}, nil
}

func (r *AzureStackHCILoadBalancerReconciler) reconcileDeleteVirtualMachine(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	// use Get to find VM
	vm := &infrav1.AzureStackHCIVirtualMachine{}
	vmName := apitypes.NamespacedName{
		Namespace: clusterScope.Namespace(),
		Name:      loadBalancerScope.Name(),
	}

	// Use Delete to delete it
	if err := r.Client.Get(loadBalancerScope.Context, vmName, vm); err != nil {
		// if the VM resource is not found, it was already deleted
		// otherwise return the error
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get AzureStackHCIVirtualMachine %s", vmName)
		}
	} else if vm.GetDeletionTimestamp().IsZero() {
		// this means the VM resource was found and has not been deleted
		if err := r.Client.Delete(clusterScope.Context, vm); err != nil {
			if !apierrors.IsNotFound(err) {
				return errors.Wrapf(err, "failed to get AzureStackHCIVirtualMachine %s", vmName)
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

func (r *AzureStackHCILoadBalancerReconciler) formatLoadBalancerCloudInit(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) *string {

	// Temp until lbagent is ready
	binarylocation := os.Getenv("AZURESTACKHCI_BINARY_LOCATION")
	if binarylocation == "" {
		// Default
		binarylocation = "http://10.231.110.37/AzureEdge/0.8"
		loadBalancerScope.Info("Failed to obtain binary location from env. Using default value.", "binarylocation", binarylocation)
	}

	ret := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`
#cloud-config
write_files:
  - path: /lib/systemd/system/lbagent.service
    owner: root:root
    permissions: '0644'
    content: |
      [Unit]
      Description=AzEdge lbagent service
      After=syslog.target network-online.target
      Wants=network-online.target
      
      [Service]
      Environment="LB_DEBUG_MODE=on"
      Type=simple
      PIDFile=/var/run/lbagent.pid
      KillMode=process
      ExecStart=/usr/sbin/lbagent
      ExecReload=/bin/kill -HUP $MAINPID
      Restart=always
      
      [Install]
      WantedBy=multi-user.target

runcmd:
- |
  curl -o /usr/sbin/lbagent %s/lbagent
  chmod 755 /usr/sbin/lbagent
  systemctl start lbagent
  sysctl -w net.ipv4.ip_nonlocal_bind=1
  systemctl reload haproxy
  systemctl stop iptables
`, binarylocation)))
	return &ret
}

func (r *AzureStackHCILoadBalancerReconciler) getVMImage(loadBalancerScope *scope.LoadBalancerScope) (*infrav1.Image, error) {
	// Use custom image if provided
	if loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image.Name != nil {
		loadBalancerScope.Info("Using custom image name for loadbalancer", "loadbalancer", loadBalancerScope.AzureStackHCILoadBalancer.GetName(), "imageName", loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image.Name)
		return &loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image, nil
	}

	return azurestackhci.GetDefaultImage(loadBalancerScope.AzureStackHCILoadBalancer.Spec.Image.OSType, to.String(loadBalancerScope.AzureStackHCICluster.Spec.Version))
}
