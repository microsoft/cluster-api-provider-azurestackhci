package util

import (
	"context"
	"crypto/rand"
	"math/big"
	"strings"
	"time"

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/kubeconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	charSet       = "abcdefghijklmnopqrstuvwxyz0123456789"
	diskCsiDriver = "disk.csi.akshci.com"
)

// GetAzureStackHCIMachinesInCluster gets a cluster's AzureStackHCIMachines resources.
func GetAzureStackHCIMachinesInCluster(ctx context.Context, controllerClient client.Client, namespace, clusterName string) ([]*infrav1.AzureStackHCIMachine, error) {
	labels := map[string]string{clusterv1.ClusterNameLabel: clusterName}
	machineList := &infrav1.AzureStackHCIMachineList{}

	if err := controllerClient.List(
		ctx, machineList,
		client.InNamespace(namespace),
		client.MatchingLabels(labels)); err != nil {
		return nil, err
	}

	machines := make([]*infrav1.AzureStackHCIMachine, len(machineList.Items))
	for i := range machineList.Items {
		machines[i] = &machineList.Items[i]
	}

	return machines, nil
}

// Create a target cluster config based on the secret in the management cluster
func NewTargetClusterConfig(ctx context.Context, controllerClient client.Reader, clusterKey client.ObjectKey) (*rest.Config, error) {
	kubeconfig, err := kubeconfig.FromSecret(ctx, controllerClient, clusterKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve kubeconfig secret for cluster %s:%s", clusterKey.Namespace, clusterKey.Name)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create client configuration for cluster %s:%s", clusterKey.Namespace, clusterKey.Name)
	}

	return restConfig, nil
}

func NewTargetClusterClient(ctx context.Context, controllerClient client.Client, clusterKey client.ObjectKey) (*kubernetes.Clientset, error) {
	restConfig, err := NewTargetClusterConfig(ctx, controllerClient, clusterKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create client configuration for cluster %s:%s", clusterKey.Namespace, clusterKey.Name)
	}

	// sets the timeout, otherwise this will default to 0 (i.e. no timeout)
	restConfig.Timeout = 10 * time.Second

	targetClusterClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to the cluster %s:%s", clusterKey.Namespace, clusterKey.Name)
	}

	return targetClusterClient, err
}

// GetNodeName returns the Node Name from the resource's owning CAPI machine object.
func GetNodeName(ctx context.Context, client client.Client, obj metav1.ObjectMeta) (string, error) {
	machine, err := capiutil.GetOwnerMachine(ctx, client, obj)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get owner machine for %s.%s", obj.Namespace, obj.Name)
	}
	if machine == nil {
		return "", errors.Errorf("resource %s.%s has no owning machine", obj.Namespace, obj.Name)
	}
	if machine.Status.NodeRef == nil {
		return "", errors.Errorf("machine %s.%s has no node ref", machine.Namespace, machine.Name)
	}
	return machine.Status.NodeRef.Name, nil
}

func ListVolumeAttachmentOnNode(ctx context.Context, client *kubernetes.Clientset, clusterKey client.ObjectKey, nodeName string) ([]string, error) {
	volumeAttachmentList, err := client.StorageV1().VolumeAttachments().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list VolumeAttachments for Cluster %s:%s", clusterKey.Namespace, clusterKey.Name)
	}

	res := []string{}
	if volumeAttachmentList != nil && len(volumeAttachmentList.Items) > 0 {
		for _, va := range volumeAttachmentList.Items {
			if va.Spec.Attacher == diskCsiDriver && strings.EqualFold(va.Spec.NodeName, nodeName) {
				res = append(res, pointer.StringDeref(va.Spec.Source.PersistentVolumeName, ""))
			}
		}
	}
	return res, nil
}

// RandomAlphaNumericString returns a random alphanumeric string.
func RandomAlphaNumericString(n int) (string, error) {
	result := make([]byte, n)
	for i := range result {
		val, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
		if err != nil {
			return "", err
		}
		result[i] = charSet[val.Int64()]
	}
	return string(result), nil
}

func GetReconcileID(ctx context.Context) types.UID {
	reconcileID := controller.ReconcileIDFromContext(ctx)
	if len(reconcileID) == 0 {
		reconcileID = uuid.NewUUID()
	}
	return reconcileID
}

func CopyCorrelationID(source, target client.Object) {
	sourceAnnotations := source.GetAnnotations()
	if len(sourceAnnotations) == 0 {
		return
	}

	annotations := target.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[infrav1.AzureCorrelationIDAnnotationKey] = sourceAnnotations[infrav1.AzureCorrelationIDAnnotationKey]
	annotations[infrav1.AzureOperationIDAnnotationKey] = sourceAnnotations[infrav1.AzureOperationIDAnnotationKey]
	target.SetAnnotations(annotations)

}
