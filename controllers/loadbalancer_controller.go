/*
Copyright 2019 The Kubernetes Authors.

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
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha2"
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

// LoadBalancerReconciler reconciles a LoadBalancer object
type LoadBalancerReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	useVIP   bool
}

func (r *LoadBalancerReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	// later we will also want to watch the cluster which owns the LB
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.LoadBalancer{}).
		Watches(
			&source.Kind{Type: &infrav1.AzureStackHCIVirtualMachine{}},
			&handler.EnqueueRequestForOwner{OwnerType: &infrav1.LoadBalancer{}, IsController: false},
		).
		Complete(r)
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=loadbalancers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=loadbalancers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

func (r *LoadBalancerReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.Background()
	logger := r.Log.WithValues("namespace", req.Namespace, "loadBalancer", req.Name)

	// Fetch the LoadBalancer resource.
	loadBalancer := &infrav1.LoadBalancer{}
	err := r.Get(ctx, req.NamespacedName, loadBalancer)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the CAPI Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, loadBalancer.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		logger.Info("AzureStackHCICluster Controller has not set OwnerRef on LoadBalancer")
		return reconcile.Result{}, nil
	}

	azureStackHCICluster := &infrav1.AzureStackHCICluster{}

	azureStackHCIClusterName := client.ObjectKey{
		Namespace: loadBalancer.Namespace,
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
		Logger:               logger.WithValues("loadBalancer", loadBalancer.Name),
		Client:               r.Client,
		LoadBalancer:         loadBalancer,
		AzureStackHCICluster: azureStackHCICluster,
		Cluster:              cluster,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any LoadBalancer changes.
	defer func() {
		if err := loadBalancerScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted LoadBalancers.
	if !loadBalancer.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(loadBalancerScope, clusterScope)
	}

	// Handle non-deleted LoadBalancers.
	return r.reconcileNormal(loadBalancerScope, clusterScope)
}

func (r *LoadBalancerReconciler) reconcileNormal(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	loadBalancerScope.Info("Reconciling LoadBalancer")

	// If the LoadBalancer doesn't have our finalizer, add it.
	if !util.Contains(loadBalancerScope.LoadBalancer.Finalizers, infrav1.LoadBalancerFinalizer) {
		loadBalancerScope.LoadBalancer.Finalizers = append(loadBalancerScope.LoadBalancer.Finalizers, infrav1.LoadBalancerFinalizer)
	}

	vm, err := r.reconcileNormalVirtualMachine(loadBalancerScope, clusterScope)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if vm.Status.VMState == nil {
		loadBalancerScope.Info("Waiting for VM controller to set vm state")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
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
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
		}
	}

	return reconcile.Result{}, nil
}

func (r *LoadBalancerReconciler) reconcileNormalVirtualMachine(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (*infrav1.AzureStackHCIVirtualMachine, error) {
	vm := &infrav1.AzureStackHCIVirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterScope.Namespace(),
			Name:      loadBalancerScope.Name(),
		},
	}

	mutateFn := func() (err error) {
		// Mark the LoadBalancer as the owner of the AzureStackHCIVirtualMachine
		vm.SetOwnerReferences(util.EnsureOwnerRef(
			vm.OwnerReferences,
			metav1.OwnerReference{
				APIVersion: loadBalancerScope.LoadBalancer.APIVersion,
				Kind:       loadBalancerScope.LoadBalancer.Kind,
				Name:       loadBalancerScope.LoadBalancer.Name,
				UID:        loadBalancerScope.LoadBalancer.UID,
			}))

		vm.Spec.ResourceGroup = clusterScope.AzureStackHCICluster.Spec.ResourceGroup
		vm.Spec.VnetName = clusterScope.AzureStackHCICluster.Spec.NetworkSpec.Vnet.Name
		vm.Spec.ClusterName = clusterScope.AzureStackHCICluster.Name
		vm.Spec.SubnetName = azurestackhci.GenerateNodeSubnetName(clusterScope.Name())
		vm.Spec.BootstrapData = r.formatLoadBalancerCloudInit(loadBalancerScope, clusterScope)
		vm.Spec.VMSize = "Default"
		vm.Spec.Image = infrav1.Image{
			Name:      to.StringPtr(loadBalancerScope.LoadBalancer.Spec.ImageReference),
			Offer:     to.StringPtr(azurestackhci.DefaultImageOfferID),
			Publisher: to.StringPtr(azurestackhci.DefaultImagePublisherID),
			SKU:       to.StringPtr(azurestackhci.DefaultImageSKU),
			Version:   to.StringPtr(azurestackhci.LatestVersion),
		}
		vm.Spec.Location = clusterScope.Location()
		vm.Spec.SSHPublicKey = loadBalancerScope.LoadBalancer.Spec.SSHPublicKey

		return nil
	}

	if _, err := controllerutil.CreateOrUpdate(clusterScope.Context, r.Client, vm, mutateFn); err != nil {
		if apierrors.IsAlreadyExists(err) {
			clusterScope.Info("AzureStackHCIVirtualMachine already exists")
			return nil, err
		}
	}

	return vm, nil
}

func (r *LoadBalancerReconciler) reconcileLoadBalancerAddress(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	if r.useVIP {
		loadBalancerScope.Info("Attempting to vip for loadbalancer", "name", loadBalancerScope.LoadBalancer.Name)
		lbSpec := &loadbalancers.Spec{
			Name: loadBalancerScope.LoadBalancer.Name,
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
		loadBalancerScope.Info("Attempting to get network interface information for loadbalancer", "name", loadBalancerScope.LoadBalancer.Name)
		nicInterface, err := networkinterfaces.NewService(clusterScope).Get(clusterScope.Context,
			&networkinterfaces.Spec{
				Name:     azurestackhci.GenerateNICName(loadBalancerScope.Name()),
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

func (r *LoadBalancerReconciler) reconcileLoadBalancer(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	backendPoolName := azurestackhci.GenerateBackendPoolName(clusterScope.Name())
	loadBalancerScope.SetPort(clusterScope.APIServerPort())
	lbSpec := &loadbalancers.Spec{
		Name:            loadBalancerScope.Name(),
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
		return errors.Wrapf(err, "failed to reconcile loadbalancer %s", loadBalancerScope.LoadBalancer.Name)
	}

	return nil
}

func (r *LoadBalancerReconciler) reconcileDelete(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) (_ reconcile.Result, reterr error) {
	loadBalancerScope.Info("Handling deleted LoadBalancer")

	if err := r.reconcileDeleteLoadBalancer(loadBalancerScope, clusterScope); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.reconcileDeleteVirtualMachine(loadBalancerScope, clusterScope); err != nil {
		return reconcile.Result{}, err
	}

	loadBalancerScope.LoadBalancer.Finalizers = util.Filter(loadBalancerScope.LoadBalancer.Finalizers, infrav1.LoadBalancerFinalizer)

	return reconcile.Result{}, nil
}

func (r *LoadBalancerReconciler) reconcileDeleteVirtualMachine(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
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

func (r *LoadBalancerReconciler) reconcileDeleteLoadBalancer(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) error {
	lbSpec := &loadbalancers.Spec{
		Name: loadBalancerScope.LoadBalancer.Name,
	}
	if err := loadbalancers.NewService(clusterScope).Delete(clusterScope.Context, lbSpec); err != nil {
		if !azurestackhci.ResourceNotFound(err) {
			return errors.Wrapf(err, "failed to delete loadbalancer %s", loadBalancerScope.LoadBalancer.Name)
		}
	}

	return nil
}

func (r *LoadBalancerReconciler) formatLoadBalancerCloudInit(loadBalancerScope *scope.LoadBalancerScope, clusterScope *scope.ClusterScope) *string {

	// Temp until lbagent is ready
	binarylocation := os.Getenv("BINARY_LOCATION")
	if binarylocation == "" {
		// Default
		binarylocation = "http://10.231.110.37/AzureEdge/0.7"
		loadBalancerScope.Info("Failed to obtain binary location from env. Using default value.", "binarylocation", binarylocation)
	}

	ret := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`
#cloud-config
packages:
  - keepalived
  - cronie
  - diffutils
  - hyper-v
  - haproxy

write_files:
  - path: /root/crontab.input
    owner: root:root
    permissions: '0640'
    content: |
      * * * * * /root/update.sh

  - path: /root/update.sh
    owner: root:root
    permissions: '0755'
    content: |
      #!/bin/sh
      # TODO - we could make this more generic in the future.  For now, it is tailored to LB.
      # create keepalived.conf and check_apiserver.sh
      # Download keepalived.conf
      export WSSD_DEBUG_MODE=on
      /opt/wssd/k8s/wssdcloudctl security keyvault --cloudFqdn %[1]s --group %[2]s secret --vault-name %[2]s_%[3]s show --name keepalived.conf --query value --output tsv > /root/keepalived.conf.new
      /opt/wssd/k8s/wssdcloudctl security keyvault --cloudFqdn %[1]s --group %[2]s secret --vault-name %[2]s_%[3]s show --name check_apiserver.sh --query value --output tsv > /root/check_apiserver.sh
      /opt/wssd/k8s/wssdcloudctl security keyvault --cloudFqdn %[1]s --group %[2]s secret --vault-name %[2]s_%[3]s show --name haproxy.cfg --query value --output tsv > /root/haproxy.cfg.new
      # if file diff - Restart keepalived (to pick up new conf).
      if [ -f keepalived.conf.new ]
      then
        if ! diff /etc/keepalived/keepalived.conf /root/keepalived.conf.new > /dev/null
        then
          cp /root/keepalived.conf.new /etc/keepalived/keepalived.conf
          systemctl restart keepalived
        fi
      fi

      if [ -f haproxy.cfg.new ]
      then
        if ! diff /etc/haproxy/haproxy.cfg /root/haproxy.cfg.new > /dev/null
        then
          cp /root/haproxy.cfg.new /etc/haproxy/haproxy.cfg
          systemctl restart haproxy
        fi
      fi

runcmd:
- |
  systemctl start hv_kvp_daemon
  # WSSD Setup
  mkdir -p /opt/wssd/k8s
  curl -o /opt/wssd/k8s/wssdcloudctl %[4]s/wssdcloudctl
  chmod 755 /opt/wssd/k8s/wssdcloudctl
  export WSSD_DEBUG_MODE=on
  crontab /root/crontab.input
  systemctl start cron
  systemctl start haproxy
  #TODO: only open up ports that are needed.  This would have to be moved to the cronjob.
  systemctl stop iptables
`, clusterScope.CloudAgentFqdn, clusterScope.GetResourceGroup(), loadBalancerScope.Name(), binarylocation)))
	return &ret
}
