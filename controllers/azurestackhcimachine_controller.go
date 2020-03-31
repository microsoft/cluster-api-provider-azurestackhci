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
	"encoding/json"
	"time"

	"golang.org/x/text/encoding/unicode"

	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-logr/logr"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha2"
	winapi "github.com/microsoft/cluster-api-provider-azurestackhci/api/windows"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/secrets"
	"github.com/microsoft/moc-sdk-for-go/services/security/keyvault"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apitypes "k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
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

// AzureStackHCIMachineReconciler reconciles a AzureStackHCIMachine object
type AzureStackHCIMachineReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

func (r *AzureStackHCIMachineReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.AzureStackHCIMachine{}).
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("AzureStackHCIMachine")),
			},
		).
		Watches(
			&source.Kind{Type: &infrav1.AzureStackHCICluster{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(r.AzureStackHCIClusterToAzureStackHCIMachines)},
		).
		Watches(
			&source.Kind{Type: &infrav1.AzureStackHCIVirtualMachine{}},
			&handler.EnqueueRequestForOwner{OwnerType: &infrav1.AzureStackHCIMachine{}, IsController: false},
		).
		Complete(r)
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhcimachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=azurestackhcimachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

func (r *AzureStackHCIMachineReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.TODO()
	logger := r.Log.WithValues("namespace", req.Namespace, "azureStackHCIMachine", req.Name)

	// Fetch the AzureStackHCIMachine VM.
	azureStackHCIMachine := &infrav1.AzureStackHCIMachine{}
	err := r.Get(ctx, req.NamespacedName, azureStackHCIMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

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
		Logger:               logger,
		Cluster:              cluster,
		AzureStackHCICluster: azureStackHCICluster,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Logger:               logger,
		Client:               r.Client,
		Cluster:              cluster,
		Machine:              machine,
		AzureStackHCICluster: azureStackHCICluster,
		AzureStackHCIMachine: azureStackHCIMachine,
	})
	if err != nil {
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
	// If the AzureStackHCIMachine is in an error state, return early.
	if machineScope.AzureStackHCIMachine.Status.ErrorReason != nil || machineScope.AzureStackHCIMachine.Status.ErrorMessage != nil {
		machineScope.Info("Error state detected, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	// If the AzureStackHCIMachine doesn't have our finalizer, add it.
	// with controller-runtime 0.4.0 you can do this with AddFinalizer
	if !util.Contains(machineScope.AzureStackHCIMachine.Finalizers, infrav1.MachineFinalizer) {
		machineScope.AzureStackHCIMachine.Finalizers = append(machineScope.AzureStackHCIMachine.Finalizers, infrav1.MachineFinalizer)
	}

	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		return reconcile.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.Data == nil {
		machineScope.Info("Bootstrap data is not yet available")
		return reconcile.Result{}, nil
	}

	if machineScope.AzureStackHCIMachine.Spec.OSDisk.OSType == "Windows" {
		//populate bootstrap data
		windowsBootstrap, err := r.getWindowsBootstrapData(clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		machineScope.Machine.Spec.Bootstrap.Data = &windowsBootstrap
	}

	vm, err := r.reconcileVirtualMachineNormal(machineScope, clusterScope)

	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// TODO(ncdc): move this validation logic into a validating webhook
	if errs := r.validateUpdate(&machineScope.AzureStackHCIMachine.Spec, vm); len(errs) > 0 {
		agg := kerrors.NewAggregate(errs)
		r.Recorder.Eventf(machineScope.AzureStackHCIMachine, corev1.EventTypeWarning, "InvalidUpdate", "Invalid update: %s", agg.Error())
		return reconcile.Result{}, nil
	}

	// Make sure Spec.ProviderID is always set.
	machineScope.SetProviderID(fmt.Sprintf("azurestackhci:////%s", vm.Name))

	// TODO(vincepri): Remove this annotation when clusterctl is no longer relevant.
	machineScope.SetAnnotation("cluster-api-provider-azurestackhci", "true")

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
	case infrav1.VMStateUpdating:
		machineScope.Info("Machine VM is updating", "name", vm.Name)
	default:
		machineScope.SetErrorReason(capierrors.UpdateMachineError)
		machineScope.SetErrorMessage(errors.Errorf("AzureStackHCI VM state %q is unexpected", *machineScope.GetVMState()))
	}

	return reconcile.Result{}, nil
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

		switch role := machineScope.Role(); role {
		case infrav1.Node:
			vm.Spec.SubnetName = azurestackhci.GenerateNodeSubnetName(clusterScope.Name())
		case infrav1.ControlPlane:
			vm.Spec.SubnetName = azurestackhci.GenerateControlPlaneSubnetName(clusterScope.Name())
			backendPoolName, err := r.GetBackendPoolName(machineScope, clusterScope)
			if err != nil {
				return err
			} else if backendPoolName != "" {
				vm.Spec.BackendPoolName = backendPoolName
			}

		default:
			return errors.Errorf("unknown value %s for label `set` on machine %s, unable to create virtual machine resource", role, machineScope.Name())
		}

		image, err := getVMImage(machineScope)
		if err != nil {
			return errors.Wrap(err, "failed to get VM image")
		}
		image.DeepCopyInto(&vm.Spec.Image)

		vm.Spec.VMSize = machineScope.AzureStackHCIMachine.Spec.VMSize
		machineScope.AzureStackHCIMachine.Spec.AvailabilityZone.DeepCopyInto(&vm.Spec.AvailabilityZone)
		machineScope.AzureStackHCIMachine.Spec.OSDisk.DeepCopyInto(&vm.Spec.OSDisk)
		vm.Spec.Location = machineScope.AzureStackHCIMachine.Spec.Location
		vm.Spec.SSHPublicKey = machineScope.AzureStackHCIMachine.Spec.SSHPublicKey
		vm.Spec.BootstrapData = machineScope.Machine.Spec.Bootstrap.Data

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

func (r *AzureStackHCIMachineReconciler) reconcileDelete(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (_ reconcile.Result, reterr error) {
	machineScope.Info("Handling deleted AzureStackHCIMachine")

	if err := r.reconcileVirtualMachineDelete(machineScope, clusterScope); err != nil {
		return reconcile.Result{}, err
	}

	machineScope.AzureStackHCIMachine.Finalizers = util.Filter(machineScope.AzureStackHCIMachine.Finalizers, infrav1.MachineFinalizer)
	// can use this method in controller runtime v0.4.0
	// controllerutil.RemoveFinalizer(machineScope.AzureStackHCIMachine, infrav1.MachineFinalizer)

	return reconcile.Result{}, nil
}

func (r *AzureStackHCIMachineReconciler) reconcileVirtualMachineDelete(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	// use Get to find VM
	vm := &infrav1.AzureStackHCIVirtualMachine{}
	vmName := apitypes.NamespacedName{
		Namespace: clusterScope.Namespace(),
		Name:      machineScope.Name(),
	}

	// Use Delete to delete it
	if err := r.Client.Get(clusterScope.Context, vmName, vm); err != nil {
		// if the VM resource is not found, it was already deleted
		// otherwise return the error
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get AzureStackHCIVirtualMachine %s", vmName)
		}
	} else if vm.GetDeletionTimestamp().IsZero() {
		// this means the VM resource was found and has not been deleted
		// is this a synchronous call?
		if err := r.Client.Delete(clusterScope.Context, vm); err != nil {
			if !apierrors.IsNotFound(err) {
				return errors.Wrapf(err, "failed to get AzureStackHCIVirtualMachine %s", vmName)
			}
		}
	}

	return nil
}

// validateUpdate checks that no immutable fields have been updated and
// returns a slice of errors representing attempts to change immutable state.
func (r *AzureStackHCIMachineReconciler) validateUpdate(spec *infrav1.AzureStackHCIMachineSpec, i *infrav1.AzureStackHCIVirtualMachine) (errs []error) {
	// TODO: Add comparison logic for immutable fields
	return errs
}

// AzureStackHCIClusterToAzureStackHCIMachines is a handler.ToRequestsFunc to be used to enqueue requests for reconciliation
// of AzureStackHCIMachines.
func (r *AzureStackHCIMachineReconciler) AzureStackHCIClusterToAzureStackHCIMachines(o handler.MapObject) []ctrl.Request {
	result := []ctrl.Request{}

	c, ok := o.Object.(*infrav1.AzureStackHCICluster)
	if !ok {
		r.Log.Error(errors.Errorf("expected a AzureStackHCICluster but got a %T", o.Object), "failed to get AzureStackHCIMachine for AzureStackHCICluster")
		return nil
	}
	log := r.Log.WithValues("AzureStackHCICluster", c.Name, "Namespace", c.Namespace)

	cluster, err := util.GetOwnerCluster(context.TODO(), r.Client, c.ObjectMeta)
	switch {
	case apierrors.IsNotFound(err) || cluster == nil:
		return result
	case err != nil:
		log.Error(err, "failed to get owning cluster")
		return result
	}

	labels := map[string]string{clusterv1.MachineClusterLabelName: cluster.Name}
	machineList := &clusterv1.MachineList{}
	if err := r.List(context.TODO(), machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
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

// GetBackendPoolName finds the clusters load balancer and gets its backend pool name
func (r *AzureStackHCIMachineReconciler) GetBackendPoolName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (string, error) {
	if clusterScope.LoadBalancerRef() == nil {
		clusterScope.Info("Omitting backend pool name for control plane machine nic since AzureStackHCICluster.Spec.LoadBalancerRef is nil")
		return "", nil
	}

	// find the load balancer
	loadBalancer := &infrav1.LoadBalancer{}
	loadBalancerNamespacedName := types.NamespacedName{
		Namespace: clusterScope.AzureStackHCICluster.Spec.LoadBalancerRef.Namespace,
		Name:      clusterScope.AzureStackHCICluster.Spec.LoadBalancerRef.Name,
	}
	if err := r.Get(clusterScope.Context, loadBalancerNamespacedName, loadBalancer); err != nil {
		return "", err
	}

	return loadBalancer.Spec.BackendPoolName, nil
}

// Pick image from the machine configuration, or use a default one.
func getVMImage(scope *scope.MachineScope) (*infrav1.Image, error) {
	// Use custom image if provided
	if scope.AzureStackHCIMachine.Spec.Image.Name != nil {
		scope.Info("Using custom image name for machine", "machine", scope.AzureStackHCIMachine.GetName(), "imageName", scope.AzureStackHCIMachine.Spec.Image.Name)
		return &scope.AzureStackHCIMachine.Spec.Image, nil
	}

	return azurestackhci.GetDefaultLinuxImage(to.String(scope.Machine.Spec.Version))
}

func (r *AzureStackHCIMachineReconciler) getWindowsBootstrapData(clusterScope *scope.ClusterScope) (string, error) {

	secretsSvc := secrets.NewService(clusterScope)

	secretInterface, err := secretsSvc.Get(clusterScope.Context, &secrets.Spec{Name: "kubeconf", VaultName: clusterScope.Name()})
	if err != nil {
		return "", errors.Wrap(err, "error retrieving 'conf' secret")
	}
	conf, ok := secretInterface.(keyvault.Secret)
	if !ok {
		return "", errors.New("error retrieving 'conf' secret")
	}

	//Temp until CABPK work is complete
	secretInterface, err = secretsSvc.Get(clusterScope.Context, &secrets.Spec{Name: "joincommand", VaultName: clusterScope.Name()})
	if err != nil {
		return "", errors.Wrap(err, "error retrieving 'joincommand' secret")
	}
	joinCmd, ok := secretInterface.(keyvault.Secret)
	if !ok {
		return "", errors.New("error retrieving 'joincommand' secret")
	}

	joinArray := strings.Fields(*joinCmd.Value)

	//Temp: Replace with clusterScope.Cluster.Spec.ApiEndoints[0] ?
	masterIP := strings.Split(joinArray[2], ":")[0]

	//dummy not needed
	username := "masteruser"
	token := joinArray[4]
	hash := joinArray[6]

	clusterCidr := clusterScope.Cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0]
	//The following line is broken
	//serviceCidr := clusterScope.Cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0]
	serviceCidr := "10.96.0.0/12"

	kubecluster := winapi.KubeCluster{
		Cri: winapi.Cri{
			Name: "dockerd",
			Images: winapi.Images{
				Pause:      "kubeletwin/pause",
				Nanoserver: "microsoft/nanoserver",
				ServerCore: "microsoft/windowsservercore",
			},
		},
		Cni: winapi.Cni{
			Name: "flannel",
			Source: winapi.CniSource{
				Name: "flanneld",
				Url:  "https://github.com/coreos/flannel/releases/download/v0.11.0/flanneld.exe",
			},
			Plugin: winapi.Plugin{
				Name: "vxlan",
			},
			//TODO: Fill out with expected interface name, probably will change the KubeCluster scripts to do this
			InterfaceName: "Ethernet 2",
		},
		Kubernetes: winapi.Kubernetes{
			Source: winapi.KubernetesSource{
				Release: "1.16.2",
				Url:     "https://dl.k8s.io/v1.16.2/kubernetes-node-windows-amd64.tar.gz",
			},
			ControlPlane: winapi.ControlPlane{
				IpAddress:     masterIP,
				Username:      username,
				KubeadmToken:  token,
				KubeadmCAHash: hash,
			},
			KubeProxy: winapi.KubeProxy{
				Gates: "WinOverlay=true",
			},
			Network: winapi.Network{
				ServiceCidr: serviceCidr,
				ClusterCidr: clusterCidr,
			},
		},
		Install: winapi.Install{
			Destination: "C:\\ProgramData\\Kubernetes",
		},
	}

	kubeclusterJSON, err := json.Marshal(kubecluster)
	if err != nil {
		return "", err
	}

	kubeconfig := *conf.Value
	psScript := `
				$cmd = $cmd = (Get-Service docker -ErrorAction SilentlyContinue).Status -eq "Running"
				while (!$cmd)
				{
					Start-Sleep -s 1
					$cmd = (Get-Service docker -ErrorAction SilentlyContinue).Status -eq "Running"
				}
				$BaseDir = "$env:ALLUSERSPROFILE\Kubernetes"
				mkdir $BaseDir
				$jsonString = '` + string(kubeclusterJSON) + `'
				Set-Content -Path $BaseDir/kubecluster.json -Value $jsonString
				$kubeconfig = '` + kubeconfig + `'
				Set-Content -Path $BaseDir/config -Value $kubeconfig

				$secureProtocols = @()
				$insecureProtocols = @([System.Net.SecurityProtocolType]::SystemDefault, [System.Net.SecurityProtocolType]::Ssl3)
				foreach ($protocol in [System.Enum]::GetValues([System.Net.SecurityProtocolType]))
				{
					if ($insecureProtocols -notcontains $protocol)
					{
						$secureProtocols += $protocol
					}
				}
				[System.Net.ServicePointManager]::SecurityProtocol = $secureProtocols

				$Url = "https://raw.githubusercontent.com/ksubrmnn/sig-windows-tools/bootstrap/kubeadm/KubeClusterHelper.psm1"
				$Destination = "$BaseDir/KubeClusterHelper.psm1"
				try {
					(New-Object System.Net.WebClient).DownloadFile($Url,$Destination)
					Write-Host "Downloaded [$Url] => [$Destination]"
				} catch {
					Write-Error "Failed to download $Url"
					throw
				}
				ipmo $BaseDir/KubeClusterHelper.psm1
				DownloadFile -Url "https://raw.githubusercontent.com/ksubrmnn/sig-windows-tools/bootstrap/kubeadm/KubeCluster.ps1" -Destination "$BaseDir/KubeCluster.ps1"
				docker tag microsoft/nanoserver:latest mcr.microsoft.com/windows/nanoserver:latest
				Write-Host "Building kubeletwin/pause image"
				pushd
				cd $Global:BaseDir
				DownloadFile -Url "https://github.com/madhanrm/SDN/raw/kubeadm/Kubernetes/windows/Dockerfile" -Destination $BaseDir\Dockerfile
				docker build -t kubeletwin/pause .
				

				popd
				
				$scriptPath = [io.Path]::Combine($BaseDir, "KubeCluster.ps1")
				$configPath = [io.Path]::Combine($BaseDir, "kubecluster.json")
				.$scriptPath -install -ConfigFile  $configPath
				.$scriptPath -join -ConfigFile $configPath
				`

	Utf16leEncoding := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	if err != nil {
		return "", err
	}

	psScriptEncodedUtf16, err := Utf16leEncoding.NewEncoder().String(psScript)
	psScriptEncoded64, err := base64.StdEncoding.EncodeToString([]byte(psScriptEncodedUtf16)), nil
	if err != nil {
		return "", err
	}

	cmdScript := "mkdir %WINDIR%\\Setup\\Scripts &&  powershell.exe echo 'powershell.exe -encoded " + psScriptEncoded64 + " > C:\\logs.txt 2>&1' > %WINDIR%\\Setup\\Scripts\\SetupComplete.cmd"

	cmdScriptEncoded, err := base64.StdEncoding.EncodeToString([]byte(cmdScript)), nil
	if err != nil {
		return "", err
	}

	return cmdScriptEncoded, nil
}
