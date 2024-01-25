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
}

// Get provides information about a network interface.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	nicSpec, ok := spec.(*Spec)
	if !ok {
		return network.Interface{}, errors.New("invalid network interface specification")
	}
	nic, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), nicSpec.Name)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		return nil, errors.Wrapf(err, "network interface %s not found", nicSpec.Name)
	} else if err != nil {
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

	if _, err := s.Get(ctx, nicSpec); err == nil {
		// Nic already exists, no update supported for now
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

	_, err := s.Client.CreateOrUpdate(ctx,
		s.Scope.GetResourceGroup(),
		nicSpec.Name,
		&networkInterface)
	telemetry.WriteMocOperationLog(s.Scope.GetLogger(), telemetry.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), telemetry.NetworkInterface,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), nicSpec.Name), &networkInterface, err)
	if err != nil {
		return errors.Wrapf(err, "failed to create network interface %s in resource group %s", nicSpec.Name, s.Scope.GetResourceGroup())
	}

	logger.Info("successfully created network interface ", "name", nicSpec.Name)
	return err
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
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete network interface %s in resource group %s", nicSpec.Name, s.Scope.GetResourceGroup())
	}

	logger.Info("successfully deleted nic", "name", nicSpec.Name)
	return err
}
