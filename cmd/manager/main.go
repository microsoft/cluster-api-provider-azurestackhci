/*
Copyright 2018 The Kubernetes Authors.

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

package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha2"
	"github.com/microsoft/cluster-api-provider-azurestackhci/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/cluster-api/util/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = infrav1.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	klog.InitFlags(nil)

	var (
		metricsAddr                            string
		enableLeaderElection                   bool
		watchNamespace                         string
		profilerAddress                        string
		azureStackHCIClusterConcurrency        int
		azureStackHCIMachineConcurrency        int
		loadBalancerConcurrency                int
		azureStackHCIVirtualMachineConcurrency int
		syncPeriod                             time.Duration
	)

	flag.StringVar(
		&metricsAddr,
		"metrics-addr",
		":8080",
		"The address the metric endpoint binds to.",
	)

	flag.BoolVar(
		&enableLeaderElection,
		"enable-leader-election",
		false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.",
	)

	flag.StringVar(
		&watchNamespace,
		"namespace",
		"",
		"Namespace that the controller watches to reconcile cluster-api objects. If unspecified, the controller watches for cluster-api objects across all namespaces.",
	)

	flag.StringVar(
		&profilerAddress,
		"profiler-address",
		"",
		"Bind address to expose the pprof profiler (e.g. localhost:6060)",
	)

	flag.IntVar(&azureStackHCIClusterConcurrency,
		"azurestackhcicluster-concurrency",
		10,
		"Number of AzureStackHCIClusters to process simultaneously",
	)

	flag.IntVar(&azureStackHCIMachineConcurrency,
		"azurestackhcimachine-concurrency",
		10,
		"Number of AzureStackHCIMachines to process simultaneously",
	)

	flag.IntVar(&loadBalancerConcurrency,
		"load-balancer-concurrency",
		10,
		"Number of LoadBalancers to process simultaneously",
	)

	flag.IntVar(&azureStackHCIVirtualMachineConcurrency,
		"azurestackhci-virtual-machine-concurrency",
		5,
		"Number of AzureStackHCIVirtualMachines to process simultaneously",
	)

	flag.DurationVar(&syncPeriod,
		"sync-period",
		10*time.Minute,
		"The minimum interval at which watched resources are reconciled (e.g. 15m)",
	)

	flag.Parse()

	if watchNamespace != "" {
		setupLog.Info("Watching cluster-api objects only in namespace for reconciliation", "namespace", watchNamespace)
	}

	if profilerAddress != "" {
		setupLog.Info("Profiler listening for requests", "profiler-address", profilerAddress)
		go func() {
			setupLog.Error(http.ListenAndServe(profilerAddress, nil), "listen and serve error")
		}()
	}

	ctrl.SetLogger(klogr.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "controller-leader-election-caph",
		SyncPeriod:         &syncPeriod,
		Namespace:          watchNamespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize event recorder.
	record.InitFromRecorder(mgr.GetEventRecorderFor("azurestackhci-controller"))

	if err = (&controllers.AzureStackHCIMachineReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("AzureStackHCIMachine"),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: azureStackHCIMachineConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AzureStackHCIMachine")
		os.Exit(1)
	}
	if err = (&controllers.AzureStackHCIClusterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("AzureStackHCICluster"),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: azureStackHCIClusterConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AzureStackHCICluster")
		os.Exit(1)
	}
	if err = (&controllers.LoadBalancerReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("LoadBalancer"),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: loadBalancerConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LoadBalancer")
		os.Exit(1)
	}
	if err = (&controllers.AzureStackHCIVirtualMachineReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("AzureStackHCIVirtualMachine"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: azureStackHCIVirtualMachineConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AzureStackHCIVirtualMachine")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
