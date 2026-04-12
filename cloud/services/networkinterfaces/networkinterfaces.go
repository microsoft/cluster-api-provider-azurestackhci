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

package networkinterfaces

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/telemetry"
	"github.com/microsoft/moc-sdk-for-go/services/network"
	mocerrors "github.com/microsoft/moc/pkg/errors"
	"github.com/pkg/errors"
)

// Spec specification for ip configuration
type IPConfiguration struct {
	Name    string
	Primary bool
}

type IPConfigurations []*IPConfiguration

// Spec specification for network interface
type Spec struct {
	Name             string
	SubnetName       string
	VnetName         string
	StaticIPAddress  string
	MacAddress       string
	BackendPoolNames []string
	IPConfigurations IPConfigurations
	IPAMService      *IPAMService
}

// Get provides information about a network interface.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	nicSpec, ok := spec.(*Spec)
	if !ok {
		return network.Interface{}, errors.New("invalid network interface specification")
	}
	nic, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), nicSpec.Name)
	if err != nil {
		return nil, err
	}
	return (*nic)[0], nil
}

// Reconcile gets/creates/updates a network interface.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	nicSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid network interface specification")
	}

	if nic, err := s.Get(ctx, nicSpec); err == nil {
		// Nic already exists, no update supported for now
		// Sync back to IPAM to ensure claim exists
		mocNic := nic.(network.Interface)
		if s.IPAMService != nil {
			if err := s.IPAMService.SyncNicIPClaim(ctx, s.Scope.GetResourceGroup(), mocNic); err != nil {
				s.Scope.GetLogger().Info("Failed to sync IPClaim during reconcile", "error", err)
				// Non-blocking - don't fail NIC reconcile
			}
		}
		return nil
	}

	logger := s.Scope.GetLogger()
	nicConfig := &network.InterfaceIPConfigurationPropertiesFormat{}
	nicConfig.Subnet = &network.APIEntityReference{
		ID: to.StringPtr(nicSpec.VnetName),
	}
	backendAddressPools := []network.BackendAddressPool{}
	for _, backendpoolname := range nicSpec.BackendPoolNames {
		name := backendpoolname
		backendAddressPools = append(backendAddressPools, network.BackendAddressPool{Name: &name})
	}
	nicConfig.LoadBalancerBackendAddressPools = &backendAddressPools

	if nicSpec.StaticIPAddress != "" {
		nicConfig.PrivateIPAddress = to.StringPtr(nicSpec.StaticIPAddress)
	}

	networkInterface := network.Interface{
		Name: &nicSpec.Name,
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			EnableIPForwarding: to.BoolPtr(true),
			EnableMACSpoofing:  to.BoolPtr(true),
			MacAddress:         &nicSpec.MacAddress,
			IPConfigurations:   &[]network.InterfaceIPConfiguration{},
		},
	}

	if len(nicSpec.IPConfigurations) > 0 {
		logger.Info("Adding ipconfigurations to nic ", "len", len(nicSpec.IPConfigurations), "name", nicSpec.Name)
		for _, ipconfig := range nicSpec.IPConfigurations {
			networkIPConfig := network.InterfaceIPConfiguration{
				Name: &ipconfig.Name,
				InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
					Primary: &ipconfig.Primary,
					Subnet: &network.APIEntityReference{
						ID: to.StringPtr(nicSpec.VnetName),
					},
				},
			}

			if ipconfig.Primary {
				networkIPConfig.LoadBalancerBackendAddressPools = &backendAddressPools
			}

			*networkInterface.IPConfigurations = append(*networkInterface.IPConfigurations, networkIPConfig)
		}
	} else {
		networkIPConfig := network.InterfaceIPConfiguration{
			Name:                                     to.StringPtr("pipConfig"),
			InterfaceIPConfigurationPropertiesFormat: nicConfig,
		}

		*networkInterface.IPConfigurations = append(*networkInterface.IPConfigurations, networkIPConfig)
	}

	// assign ipam IP to the moc nic object.
	if s.IPAMService != nil {
		if err := s.IPAMService.AllocateNicIPClaim(ctx, s.Scope.GetResourceGroup(), networkInterface, nicSpec.StaticIPAddress); err != nil {
			if s.IPAMService.IsIPAMSoleAllocator(ctx) {
				// IPAM is the sole allocator (azlocal-overlay): propagate error, do not fall back to MOC
				return errors.Wrapf(err, "IPAM sole allocator: failed to allocate IP for network interface %s", nicSpec.Name)
			}
			logger.Error(err, "Failed to allocate IPClaim for network interface", "name", nicSpec.Name)
			// Best-effort - continue with NIC creation (MOC will allocate)
		}
	}

	logger.Info("creating network interface ", "name", nicSpec.Name)

	createdNic, err := s.Client.CreateOrUpdate(ctx,
		s.Scope.GetResourceGroup(),
		nicSpec.Name,
		&networkInterface)
	telemetry.WriteMocOperationLog(s.Scope.GetLogger(), telemetry.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), telemetry.NetworkInterface,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), nicSpec.Name), &networkInterface, err)
	if err != nil {
		if isIPConflictError(err) && s.shouldRetryIfIPConflict(ctx, nicSpec) {
			if createdNic, err = s.handleIPAddressConflictRetry(ctx, nicSpec, &networkInterface); err != nil {
				return errors.Wrapf(err, "failed to retry create with network interface %s in resource group %s", nicSpec.Name, s.Scope.GetResourceGroup())
			}
		} else {
			// Clean up IPAM allocation on non-conflict failure to avoid leaking reserved IPs
			if s.IPAMService != nil {
				if delErr := s.IPAMService.DeleteNicIPClaim(ctx, nicSpec); delErr != nil {
					logger.Error(delErr, "Failed to clean up IPClaim after NIC creation failure")
				}
			}
			return errors.Wrapf(err, "failed to create network interface %s in resource group %s", nicSpec.Name, s.Scope.GetResourceGroup())
		}
	}

	if s.IPAMService != nil {
		if err := s.IPAMService.SyncNicIPClaim(ctx, s.Scope.GetResourceGroup(), *createdNic); err != nil {
			logger.Info("Failed to sync IPClaim after NIC creation", "error", err)
			// Non-blocking - don't fail NIC reconcile
		}
	}

	logger.Info("successfully created network interface ", "name", nicSpec.Name)
	return nil
}

// isIPConflictError checks if the error indicates an IP address conflict.
func isIPConflictError(err error) bool {
	return mocerrors.IsInUse(err) || mocerrors.IsAlreadySet(err)
}

// shouldRetryIfIPConflict determines whether a NIC creation failure due to an IP address conflict
// should trigger a retry with MOC auto-allocation. This handles an edge case where a race condition
// between IPAM state and MOC state causes the IPAM-assigned IP to already be in use in MOC. The retry path clears the IPAM IP and
// recreates the NIC with an empty PrivateIPAddress, letting MOC auto-allocate a non-conflicting IP.
//
// Returns false (no retry) when:
//   - The user specified a static IP (not managed by IPAM).
//   - IPAM is the sole allocator (azlocal-overlay scenario): MOC fallback is not available, so
//     retrying with MOC auto-allocation is not an option. The error propagates and the reconciler
//     will retry the full IPAM allocation flow. IPClaims are cleaned up on cluster deletion.
func (s *Service) shouldRetryIfIPConflict(ctx context.Context, nicSpec *Spec) bool {
	if nicSpec.StaticIPAddress != "" {
		return false
	}

	// When IPAM is the sole allocator (azlocal-overlay), MOC auto-allocation fallback is not
	// available. The error propagates so the reconciler retries the full IPAM allocation flow.
	if s.IPAMService != nil && s.IPAMService.IsIPAMSoleAllocator(ctx) {
		return false
	}

	return true
}

func (s *Service) handleIPAddressConflictRetry(ctx context.Context, vnicSpec *Spec, networkInterface *network.Interface) (*network.Interface, error) {
	logger := s.Scope.GetLogger()
	var conflictedIP string
	if networkInterface.IPConfigurations != nil && len(*networkInterface.IPConfigurations) > 0 {
		ipConfig := (*networkInterface.IPConfigurations)[0]
		if ipConfig.InterfaceIPConfigurationPropertiesFormat != nil && ipConfig.InterfaceIPConfigurationPropertiesFormat.PrivateIPAddress != nil {
			conflictedIP = *ipConfig.InterfaceIPConfigurationPropertiesFormat.PrivateIPAddress
		}
	}
	logger.Info("IP allocated by IPAM is already taken in Moc, retrying", "Conflicted IP", conflictedIP)

	// Remove the failed mocnetworkinterface (this also cleans up the IPClaim via defer in Delete)
	if err := s.Delete(ctx, vnicSpec); err != nil && !azurestackhci.ResourceNotFound(err) {
		return nil, err
	}

	for i := range *networkInterface.IPConfigurations {
		if (*networkInterface.IPConfigurations)[i].InterfaceIPConfigurationPropertiesFormat != nil {
			(*networkInterface.IPConfigurations)[i].InterfaceIPConfigurationPropertiesFormat.PrivateIPAddress = nil
		}
	}

	logger.Info("Creating network interface with empty PrivateIPAddress")
	// Recreate the mocnetworkinterface without the IPAM allocated IP
	createdNic, err := s.Client.CreateOrUpdate(ctx,
		s.Scope.GetResourceGroup(),
		vnicSpec.Name,
		networkInterface)

	telemetry.WriteMocOperationLog(s.Scope.GetLogger(), telemetry.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), telemetry.NetworkInterface,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), vnicSpec.Name), &networkInterface, err)

	return createdNic, err
}

// Delete deletes the network interface with the provided name.
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	nicSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid network interface Specification")
	}
	logger := s.Scope.GetLogger()
	logger.Info("deleting nic", "name", nicSpec.Name)

	err := s.Client.Delete(ctx, s.Scope.GetResourceGroup(), nicSpec.Name)
	telemetry.WriteMocOperationLog(logger, telemetry.Delete, s.Scope.GetCustomResourceTypeWithName(), telemetry.NetworkInterface,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), nicSpec.Name), nil, err)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		// already deleted
	} else if err != nil {
		return errors.Wrapf(err, "failed to delete network interface %s in resource group %s", nicSpec.Name, s.Scope.GetResourceGroup())
	}

	// Delete IPAM claim only after MOC resource is confirmed deleted
	if s.IPAMService != nil {
		if err := s.IPAMService.DeleteNicIPClaim(ctx, nicSpec); err != nil {
			logger.Error(err, "failed to delete IPAM claim for nic", "name", nicSpec.Name)
		}
	}

	logger.Info("successfully deleted nic", "name", nicSpec.Name)
	return nil
}
