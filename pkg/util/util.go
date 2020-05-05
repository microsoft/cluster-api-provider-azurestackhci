package util

import (
	"context"

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetAzureStackHCIMachinesInCluster gets a cluster's AzureStackHCIMachines resources.
func GetAzureStackHCIMachinesInCluster(ctx context.Context, controllerClient client.Client, namespace, clusterName string) ([]*infrav1.AzureStackHCIMachine, error) {
	labels := map[string]string{clusterv1.ClusterLabelName: clusterName}
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
