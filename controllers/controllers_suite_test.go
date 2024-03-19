package controllers

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/golang/mock/gomock"
)

var (
	testEnv      *envtest.Environment
	k8sClient    client.Client
	fakeRecorder *record.FakeRecorder
	mockctrl     *gomock.Controller

	azureStackHCIMachineReconciler AzureStackHCIMachineReconciler
)

func TestClusterApiProviderAzureStackHCIControllerSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ClusterApiProviderAzureStackHCIController Suite")
}

var _ = BeforeSuite(func() {
	klog.InitFlags(nil)
	klog.SetOutput(GinkgoWriter)
	ctrl.SetLogger(klog.Background())
	logf.SetLogger(klog.Background())

	// Download the Machine CRD
	resp, err := http.Get("https://raw.githubusercontent.com/kubernetes-sigs/cluster-api/master/config/crd/bases/cluster.x-k8s.io_machines.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()

	// Write the CRD to a temporary file
	tmpfile, err := ioutil.TempFile("", "machine.crd.*.yaml")
	Expect(err).NotTo(HaveOccurred())
	defer os.Remove(tmpfile.Name()) // clean up

	b, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	_, err = tmpfile.Write(b)
	Expect(err).NotTo(HaveOccurred())
	err = tmpfile.Close()
	Expect(err).NotTo(HaveOccurred())

	// Add the path to the temporary file to the CRDDirectoryPaths
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases"), tmpfile.Name()},
		ErrorIfCRDPathMissing: true,
	}

	By("bootstrapping test environment")
	cfg, err := testEnv.Start()
	if err != nil {
		logf.Log.Error(err, "unable to start test environment")
		os.Exit(1)
	}

	err = infrav1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clusterv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	fakeRecorder = record.NewFakeRecorder(100)

	err = (&AzureStackHCIClusterReconciler{
		Client:   k8sClient,
		Log:      ctrl.Log.WithName("controllers").WithName("AzureStackHCICluster"),
		Recorder: fakeRecorder,
	}).SetupWithManager(k8sManager, controller.Options{})
	Expect(err).ToNot(HaveOccurred())

	azureStackHCIMachineReconciler = AzureStackHCIMachineReconciler{
		Client:   k8sClient,
		Log:      ctrl.Log.WithName("controllers").WithName("AzureStackHCIMachine"),
		Recorder: fakeRecorder,
	}
	err = azureStackHCIMachineReconciler.SetupWithManager(k8sManager, controller.Options{})
	Expect(err).ToNot(HaveOccurred())

	// Start the manager/controller
	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		gexec.KillAndWait(4 * time.Second)

		err := testEnv.Stop()
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = BeforeEach(func() {
	mockctrl = gomock.NewController(GinkgoT())
})

var _ = AfterEach(func() {
	GinkgoRecover()

	mockctrl.Finish()
})
