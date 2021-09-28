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

package v1alpha4

type AzureStackHCIClusterPhase string

const (
	// AzureStackHCIClusterPhasePending is the first state a Cluster is assigned by
	// Cluster API Cluster controller after being created.
	AzureStackHCIClusterPhasePending = AzureStackHCIClusterPhase("pending")

	// AzureStackHCIClusterPhaseProvisioning is the state when the Cluster has a provider infrastructure
	// object associated and can start provisioning.
	AzureStackHCIClusterPhaseProvisioning = AzureStackHCIClusterPhase("provisioning")

	// AzureStackHCIClusterPhaseProvisioned is the state when its
	// infrastructure has been created and configured.
	AzureStackHCIClusterPhaseProvisioned = AzureStackHCIClusterPhase("provisioned")

	// AzureStackHCIClusterPhaseDeleting is the Cluster state when a delete
	// request has been sent to the API Server,
	// but its infrastructure has not yet been fully deleted.
	AzureStackHCIClusterPhaseDeleting = AzureStackHCIClusterPhase("deleting")

	// AzureStackHCIClusterPhaseFailed is the Cluster state when the system
	// might require user intervention.
	AzureStackHCIClusterPhaseFailed = AzureStackHCIClusterPhase("failed")

	// AzureStackHCIClusterPhaseUpgrading is the Cluster state when the system
	// is in the middle of a update.
	AzureStackHCIClusterPhaseUpgrading = AzureStackHCIClusterPhase("upgrading")

	// AzureStackHCIClusterPhaseUnknown is returned if the Cluster state cannot be determined.
	AzureStackHCIClusterPhaseUnknown = AzureStackHCIClusterPhase("")
)

type AzureStackHCILoadBalancerPhase string

const (
	// AzureStackHCILoadBalancerPhasePending is the first state a LoadBalancer is assigned by
	// the controller after being created.
	AzureStackHCILoadBalancerPhasePending = AzureStackHCILoadBalancerPhase("pending")

	// AzureStackHCILoadBalancerPhaseProvisioning is the state when the LoadBalancer is waiting for the
	// first replica to be ready.
	AzureStackHCILoadBalancerPhaseProvisioning = AzureStackHCILoadBalancerPhase("provisioning")

	// AzureStackHCILoadBalancerPhaseProvisioned is the state when its infrastructure has been created
	// and configured. All replicas are ready and we have the desired number of replicas.
	AzureStackHCILoadBalancerPhaseProvisioned = AzureStackHCILoadBalancerPhase("provisioned")

	// AzureStackHCILoadBalancerPhaseScaling is the state when replicas are being scaled.
	AzureStackHCILoadBalancerPhaseScaling = AzureStackHCILoadBalancerPhase("scaling")

	// AzureStackHCILoadBalancerPhaseUpgrading is the state when the system is in the middle of a update.
	AzureStackHCILoadBalancerPhaseUpgrading = AzureStackHCILoadBalancerPhase("upgrading")

	// AzureStackHCILoadBalancerPhaseDeleting is the state when a delete request has been sent to
	// the API Server, but its infrastructure has not yet been fully deleted.
	AzureStackHCILoadBalancerPhaseDeleting = AzureStackHCILoadBalancerPhase("deleting")

	// AzureStackHCILoadBalancerPhaseFailed is the state when the system might require user intervention.
	AzureStackHCILoadBalancerPhaseFailed = AzureStackHCILoadBalancerPhase("failed")

	// AzureStackHCILoadBalancerPhaseUnknown is returned if the state cannot be determined.
	AzureStackHCILoadBalancerPhaseUnknown = AzureStackHCILoadBalancerPhase("")
)
