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

package v1alpha3

import clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"

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
