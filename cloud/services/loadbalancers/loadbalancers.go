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

package loadbalancers

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/telemetry"
	"github.com/microsoft/moc-sdk-for-go/services/network"
	"github.com/pkg/errors"
)

// Spec input specification for Get/CreateOrUpdate/Delete calls
type Spec struct {
	Name            string
	BackendPoolName string
	VnetName        string
	FrontendPort    int32
	BackendPort     int32
	Tags            map[string]*string
}

// Get provides information about a load balancer.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	lbSpec, ok := spec.(*Spec)
	if !ok {
		return network.LoadBalancer{}, errors.New("invalid loadbalancer specification")
	}

	lb, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), lbSpec.Name)
	if err != nil {
		return nil, err
	}
	return (*lb)[0], nil
}

// Reconcile gets/creates/updates a load balancer.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	lbSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid loadbalancer specification")
	}

	if lb, err := s.Get(ctx, lbSpec); err == nil {
		// loadbalancer already exists, no update supported for now
		// Sync back to IPAM to ensure claim exists (if IP is available)
		s.syncLoadBalancerIPToIPAM(ctx, lb.(network.LoadBalancer))
		return nil
	}

	networkLB := network.LoadBalancer{
		Name: to.StringPtr(lbSpec.Name),
		LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
			BackendAddressPools: &[]network.BackendAddressPool{
				{
					Name: to.StringPtr(lbSpec.BackendPoolName),
				},
			},
			FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
				{
					FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
						Subnet: &network.Subnet{
							ID: to.StringPtr(lbSpec.VnetName),
						},
					},
				},
			},
			LoadBalancingRules: &[]network.LoadBalancingRule{
				{
					LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
						Protocol:     network.TransportProtocolTCP,
						FrontendPort: to.Int32Ptr(lbSpec.FrontendPort),
						BackendPort:  to.Int32Ptr(lbSpec.BackendPort),
					},
				},
			},
		},
		Tags: lbSpec.Tags,
	}

	// create the load balancer
	logger := s.Scope.GetLogger()
	logger.Info("creating loadbalancer", "name", lbSpec.Name)
	createdLb, err := s.Client.CreateOrUpdate(ctx, s.Scope.GetResourceGroup(), lbSpec.Name, &networkLB)
	telemetry.WriteMocOperationLog(logger, telemetry.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), telemetry.LoadBalancer,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), lbSpec.Name), &networkLB, err)
	if err != nil {
		return err
	}

	logger.Info("successfully created loadbalancer", "name", lbSpec.Name)

	// Try to sync IP to IPAM after creation (best-effort, IP may not be available yet)
	if createdLb != nil {
		s.syncLoadBalancerIPToIPAM(ctx, *createdLb)
	}

	return nil
}

// syncLoadBalancerIPToIPAM syncs the MOC-allocated LB IP to IPAM via SyncIPClaim only.
// We intentionally do not actively allocate IPs in the LB create/reconcile path because
// the CAPH LB path is deprecated (cloud-operator manages LBs via AksHciCluster).
// This is best-effort and non-blocking.
func (s *Service) syncLoadBalancerIPToIPAM(ctx context.Context, lb network.LoadBalancer) {
	if s.IPAMService == nil {
		return
	}

	// Extract IP from the load balancer's frontend configuration
	ip := extractLoadBalancerIP(lb)
	if ip == "" {
		s.Scope.GetLogger().Info("LoadBalancer IP not yet available, skipping IPAM sync")
		return
	}

	mocGroup := s.Scope.GetResourceGroup()
	lbName := ""
	if lb.Name != nil {
		lbName = *lb.Name
	}
	if err := s.IPAMService.SyncLoadBalancerIP(ctx, mocGroup, lbName, ip); err != nil {
		s.Scope.GetLogger().Info("Failed to sync LoadBalancer IP to IPAM", "error", err, "ip", ip)
		// Non-blocking - don't fail LB reconcile
	}
}

// extractLoadBalancerIP extracts the frontend IP from a load balancer.
func extractLoadBalancerIP(lb network.LoadBalancer) string {
	if lb.FrontendIPConfigurations == nil || len(*lb.FrontendIPConfigurations) == 0 {
		return ""
	}
	frontendConfig := (*lb.FrontendIPConfigurations)[0]
	if frontendConfig.FrontendIPConfigurationPropertiesFormat == nil || frontendConfig.IPAddress == nil {
		return ""
	}
	return *frontendConfig.IPAddress
}

// Delete deletes the load balancer with the provided name.
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	lbSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid loadbalancer specification")
	}
	logger := s.Scope.GetLogger()
	logger.Info("deleting loadbalancer", "name", lbSpec.Name)

	err := s.Client.Delete(ctx, s.Scope.GetResourceGroup(), lbSpec.Name)
	telemetry.WriteMocOperationLog(logger, telemetry.Delete, s.Scope.GetCustomResourceTypeWithName(), telemetry.LoadBalancer,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), lbSpec.Name), nil, err)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		// already deleted
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete loadbalancer %s in resource group %s", lbSpec.Name, s.Scope.GetResourceGroup())
	}

	// Delete IPAM claim only after MOC resource is confirmed deleted
	if s.IPAMService != nil {
		if err := s.IPAMService.DeleteLoadBalancerIPClaim(ctx); err != nil {
			logger.Info("Failed to delete LoadBalancer IPClaim", "error", err)
		}
	}

	logger.Info("successfully deleted loadbalancer", "name", lbSpec.Name)
	return nil
}
