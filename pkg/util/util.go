package util

import (
	"context"
	"crypto/rand"
	"math/big"

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	charSet = "abcdefghijklmnopqrstuvwxyz0123456789"
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
