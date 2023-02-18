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

package v1beta1

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

// Conditions and condition Reasons for the AzureStackHCIVirtualMachine object

const (
	// VMRunningCondition reports on current status of the AzureStackHCIVirtualMachine.
	VMRunningCondition clusterv1.ConditionType = "VMRunning"
	// VMUpdatingReason used when the vm updating is in progress.
	VMUpdatingReason = "VMUpdating"
	// VMProvisionFailedReason used for failures during vm provisioning.
	VMProvisionFailedReason = "VMProvisionFailed"
	// VMNotFoundReason used when the vm couldn't be retrieved.
	VMNotFoundReason = "VMNotFound"
	// OutOfMemoryReason used when the AzureStackHCI resource is out of memory.
	OutOfMemoryReason = "OutOfMemory"
	// OutOfCapacityReason used when the AzureStackHCI resource is out of capacity.
	OutOfCapacityReason = "OutOfCapacity"
	// NodeOutOfCapacityReason used when the AzureStackHCI node is out of capacity.
	NodeOutOfCapacityReason = "NodeOutOfCapacity"
)

// Conditions and condition Reasons for the AzureStackHCICluster object

const (
	// NetworkInfrastructureReadyCondition reports on current status of the AzureStackHCICluster
	NetworkInfrastructureReadyCondition clusterv1.ConditionType = "NetworkInfrastructureReady"
	// ClusterReconciliationFailedReason used for failures during cluster reconciliation.
	ClusterReconciliationFailedReason = "ClusterReconciliationFailed"
	// LoadBalancerProvisioningReason used for provisioning of lb
	LoadBalancerProvisioningReason = "LoadBalancerProvisioning"
	// LoadBalancerDeletingReason used when waiting on lbs to be deleted
	LoadBalancerDeletingReason = "LoadBalancerDeleting"
	// AzureStackHCIMachinesDeletingReason used when waiting on machines to be deleted
	AzureStackHCIMachinesDeletingReason = "AzureStackHCIMachineDeleting"
)

// Conditions and condition Reasons for the AzureStackHCILoadBalancer object

const (
	// LoadBalancerInfrastructureReadyCondition reports on current status of the AzureStackHCILoadBalancer
	LoadBalancerInfrastructureReadyCondition clusterv1.ConditionType = "LoadBalancerInfrastructureReady"
	// LoadBalancerServiceReconciliationFailedReason used for service failures during loadbalancer reconciliation.
	LoadBalancerServiceReconciliationFailedReason = "ServiceReconciliationFailed"
	// LoadBalancerServiceStatusFailedReason used for service status failures.
	LoadBalancerServiceStatusFailedReason = "ServiceStatusFailed"
	// LoadBalancerMachineReconciliationFailedReason used for machine failures during loadbalancer reconciliation.
	LoadBalancerMachineReconciliationFailedReason = "MachineReconciliationFailed"
	// LoadBalancerAddressUnavailableReason used when waiting for loadbalancer to have an address.
	LoadBalancerAddressUnavailableReason = "AddressUnavailable"
	// LoadBalancerNoReplicasReadyReason used when no replicas are in a ready state.
	LoadBalancerNoReplicasReadyReason = "NoReplicasReady"

	// LoadBalancerReplicasReadyCondition reports on current status of the AzureStackHCILoadBalancer machine replicas
	LoadBalancerReplicasReadyCondition clusterv1.ConditionType = "LoadBalancerReplicasReady"
	// LoadBalancerWaitingForReplicasReadyReason used when we are waiting for replicas to be ready.
	LoadBalancerWaitingForReplicasReadyReason = "WaitingForReplicasToBeReady"
	// LoadBalancerReplicasScalingUpReason used when we are scaling up the replicas.
	LoadBalancerReplicasScalingUpReason = "ScalingUp"
	// LoadBalancerReplicasScalingDownReason used when we are scaling down the replicas.
	LoadBalancerReplicasScalingDownReason = "ScalingDown"
	// LoadBalancerReplicasUpgradingReason used when we are upgrading the replicas.
	LoadBalancerReplicasUpgradingReason = "Upgrading"
	// LoadBalancerReplicasFailedReason used when we have failed replicas.
	LoadBalancerReplicasFailedReason = "FailedReplicas"
)
