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

package main

import (
	"flag"
	"net/http"

	//nolint:gosec
	_ "net/http/pprof"
	"os"
	"time"

	infrav1beta1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"

	// +kubebuilder:scaffold:imports

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	cgrecord "k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/feature"
	"sigs.k8s.io/cluster-api/util/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/microsoft/cluster-api-provider-azurestackhci/controllers"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	klog.InitFlags(nil)

	_ = clientgoscheme.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	_ = infrav1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

var (
	metricsAddr                            string
	enableLeaderElection                   bool
	watchNamespace                         string
	profilerAddress                        string
	azureStackHCIClusterConcurrency        int
	azureStackHCIMachineConcurrency        int
	azureStackHCIloadBalancerConcurrency   int
	azureStackHCIVirtualMachineConcurrency int
	syncPeriod                             time.Duration
	healthAddr                             string
	webhookPort                            int
)

func InitFlags(fs *pflag.FlagSet) {
	flag.StringVar(
		&metricsAddr,
		"metrics-bind-addr",
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

	flag.IntVar(&azureStackHCIloadBalancerConcurrency,
		"azurestackhciload-balancer-concurrency",
		10,
		"Number of AzureStackHCILoadBalancers to process simultaneously",
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

	fs.StringVar(&healthAddr,
		"health-addr",
		":9440",
		"The address the health endpoint binds to.",
	)

	fs.IntVar(&webhookPort,
		"webhook-port",
		9443,
		"Webhook Server port, disabled by default. When enabled, the manager will only work as webhook server, no reconcilers are installed.",
	)

	feature.MutableGates.AddFlag(fs)
}

func main() {
	InitFlags(pflag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	if watchNamespace != "" {
		setupLog.Info("Watching cluster-api objects only in namespace for reconciliation", "namespace", watchNamespace)
	}

	if profilerAddress != "" {
		setupLog.Info("Profiler listening for requests", "profiler-address", profilerAddress)
		go func() {
			//nolint:gosec
			setupLog.Error(http.ListenAndServe(profilerAddress, nil), "listen and serve error")
		}()
	}

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Machine and cluster operations can create enough events to trigger the event recorder spam filter
	// Setting the burst size higher ensures all events will be recorded and submitted to the API
	broadcaster := cgrecord.NewBroadcasterWithCorrelatorOptions(cgrecord.CorrelatorOptions{
		BurstSize: 100,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "controller-leader-election-caph",
		SyncPeriod:             &syncPeriod,
		Namespace:              watchNamespace,
		HealthProbeBindAddress: healthAddr,
		Port:                   webhookPort,
		EventBroadcaster:       broadcaster,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize event recorder.
	record.InitFromRecorder(mgr.GetEventRecorderFor("azurestackhci-controller"))

	if err = (&controllers.AzureStackHCIMachineReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("AzureStackHCIMachine"),
		Recorder: mgr.GetEventRecorderFor("azurestackhcimachine-reconciler"),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: azureStackHCIMachineConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AzureStackHCIMachine")
		os.Exit(1)
	}
	if err = (&controllers.AzureStackHCIClusterReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("AzureStackHCICluster"),
		Recorder: mgr.GetEventRecorderFor("azurestackhcicluster-reconciler"),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: azureStackHCIClusterConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AzureStackHCICluster")
		os.Exit(1)
	}
	if err = (&controllers.AzureStackHCILoadBalancerReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("AzureStackHCILoadBalancer"),
		Recorder: mgr.GetEventRecorderFor("azurestackhciloadbalancer-reconciler"),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: azureStackHCIloadBalancerConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AzureStackHCILoadBalancer")
		os.Exit(1)
	}
	if err = (&controllers.AzureStackHCIVirtualMachineReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("AzureStackHCIVirtualMachine"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("azurestackhcivirtualmachine-reconciler"),
	}).SetupWithManager(mgr, controller.Options{MaxConcurrentReconciles: azureStackHCIVirtualMachineConcurrency}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AzureStackHCIVirtualMachine")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := (&infrav1beta1.AzureStackHCICluster{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AzureStackHCICluster")
		os.Exit(1)
	}

	if err := (&infrav1beta1.AzureStackHCIMachine{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AzureStackHCIMachine")
		os.Exit(1)
	}

	if err := (&infrav1beta1.AzureStackHCIMachineTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AzureStackHCIMachineTemplate")
		os.Exit(1)
	}

	if err := (&infrav1beta1.AzureStackHCIVirtualMachine{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AzureStackHCIVirtualMachine")
		os.Exit(1)
	}

	if err := (&infrav1beta1.AzureStackHCILoadBalancer{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AzureStackHCILoadBalancer")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to create ready check")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to create health check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
