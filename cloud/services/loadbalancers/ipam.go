/*
Copyright 2024 The Kubernetes Authors.
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

package loadbalancers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/telemetry"
	ipam "github.com/microsoft/cluster-api-provider-azurestackhci/pkg/ipam"
)

// Annotation for marking LoadBalancer IP claims
const AnnotationLegacyLoadBalancerIP = ipam.AzstackhciAPIGroup + "/legacy-loadbalancer-ip"

// CAPHTelemetryWriter implements ipam.IPAMTelemetryWriter for CAPH LoadBalancer.
type CAPHTelemetryWriter struct {
	clusterScope *scope.ClusterScope
}

// WriteIPAMOperationLog implements ipam.IPAMTelemetryWriter.
func (w *CAPHTelemetryWriter) WriteIPAMOperationLog(logger logr.Logger, operation ipam.IPAMOperation, claimName string, params map[string]string, err error) {
	var telemetryOp telemetry.Operation
	switch operation {
	case ipam.OperationCreate:
		telemetryOp = telemetry.Create
	case ipam.OperationSync:
		telemetryOp = telemetry.CreateOrUpdate
	case ipam.OperationDelete:
		telemetryOp = telemetry.Delete
	case ipam.OperationGet:
		telemetryOp = telemetry.Get
	default:
		telemetryOp = telemetry.Create
	}

	resource := fmt.Sprintf("IPAddressClaim/%s/%s", ipam.IPClaimNamespace, claimName)
	telemetry.RecordHybridAKSCRDChange(
		logger,
		w.clusterScope.GetCustomResourceTypeWithName(),
		resource,
		telemetryOp,
		telemetry.CRD,
		params,
		err,
	)
}

// IPAMService wraps ipam.IPAMService for CAPH LoadBalancer functionality.
type IPAMService struct {
	*ipam.IPAMService
	clusterName string
}

// NewIPAMService creates a new IPAM service instance for CAPH LoadBalancer.
// Returns nil if IPAM is not supported on this environment (e.g., 22H2).
func NewIPAMService(clusterScope *scope.ClusterScope, lbScope *scope.LoadBalancerScope) *IPAMService {
	logger := clusterScope.GetLogger().WithName("LB-IPAMService")

	logger.Info("Initializing LB IPAM service",
		"cluster", clusterScope.Name(),
		"namespace", clusterScope.Namespace(),
		"vnet", clusterScope.Vnet().Name)

	if !ipam.IsIPAMSupported(clusterScope.Context, clusterScope.Client) {
		logger.Info("IPAM not supported on this environment, skipping LB IPAM service initialization")
		return nil
	}

	config := ipam.IPAMServiceConfig{
		Client:               clusterScope.Client,
		Logger:               logger,
		Namespace:            clusterScope.Namespace(),
		VnetName:             clusterScope.Vnet().Name,
		CloudFqdn:            clusterScope.GetCloudAgentFqdn(),
		Authorizer:           clusterScope.GetAuthorizer(),
		TelemetryWriter:      &CAPHTelemetryWriter{clusterScope: clusterScope},
		ClusterName:          clusterScope.Name(),
		CreatorID:            ipam.IPClaimCreatorCAPH,
		Owner:                lbScope.AzureStackHCILoadBalancer,
		ClusterResourceGroup: clusterScope.GetResourceGroup(),
	}

	logger.Info("LB IPAM service initialized successfully")
	return &IPAMService{
		IPAMService: ipam.NewIPAMService(config),
		clusterName: clusterScope.Name(),
	}
}

// generateLegacyLoadBalancerIPClaimName creates a deterministic IPClaim name for legacy LB IP sync.
func generateLegacyLoadBalancerIPClaimName(clusterName string) string {
	return fmt.Sprintf("ipclaim-%s-legacy-lb-ip", clusterName)
}

// SyncLoadBalancerIP syncs the MOC-allocated LB IP to IPAM.
// This is best-effort and non-blocking - it creates an IPClaim with a static IP annotation
// to record the allocation in the K8s-based IPAM system.
func (s *IPAMService) SyncLoadBalancerIP(ctx context.Context, mocGroup, lbName, mocAllocatedIP string) error {
	claimName := generateLegacyLoadBalancerIPClaimName(s.clusterName)
	lbAnnotations := map[string]string{
		AnnotationLegacyLoadBalancerIP: "true",
	}
	lbLabels := map[string]string{
		ipam.LabelMocGroupName:    mocGroup,
		ipam.LabelMocResourceName: lbName,
		ipam.LabelMocResourceType: ipam.MocResourceTypeLoadBalancer,
	}
	return s.IPAMService.SyncIPClaim(ctx, claimName, mocAllocatedIP, lbAnnotations, lbLabels)
}

// DeleteLoadBalancerIPClaim deletes the legacy LB IP claim (used during cleanup).
func (s *IPAMService) DeleteLoadBalancerIPClaim(ctx context.Context) error {
	claimName := generateLegacyLoadBalancerIPClaimName(s.clusterName)
	return s.IPAMService.DeleteIPClaim(ctx, claimName)
}

// GetLoadBalancerIPClaimName returns the claim name for external use.
func (s *IPAMService) GetLoadBalancerIPClaimName() string {
	return generateLegacyLoadBalancerIPClaimName(s.clusterName)
}
