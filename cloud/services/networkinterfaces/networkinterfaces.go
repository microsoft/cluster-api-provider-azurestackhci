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
	"os"

	"github.com/Azure/go-autorest/autorest/to"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/moc-sdk-for-go/services/network"
	"github.com/pkg/errors"
	"k8s.io/klog"
)

// Spec specification for network interface
type Spec struct {
	Name             string
	SubnetName       string
	VnetName         string
	StaticIPAddress  string
	MacAddress       string
	BackendPoolNames []string
}

// Get provides information about a network interface.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	nicSpec, ok := spec.(*Spec)
	if !ok {
		return network.Interface{}, errors.New("invalid network interface specification")
	}
	nic, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), nicSpec.Name)
	if err != nil {
		if azurestackhci.TransportUnavailable(err) {
			klog.Error("Communication with cloud agent failed. Exiting Process.")
			os.Exit(1)
		}

		if azurestackhci.ResourceNotFound(err) {
			return nil, errors.Wrapf(err, "network interface %s not found", nicSpec.Name)
		}

		return nil, err
	}

	return (*nic)[0], nil
}

// Reconcile gets/creates/updates a network interface.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	nicSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid network interface specification")
	}

	if _, err := s.Get(ctx, nicSpec); err == nil {
		// Nic already exists, no update supported for now
		return nil
	}

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

	_, err := s.Client.CreateOrUpdate(ctx,
		s.Scope.GetResourceGroup(),
		nicSpec.Name,
		&network.Interface{
			Name: &nicSpec.Name,
			InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
				EnableIPForwarding: to.BoolPtr(true),
				EnableMACSpoofing:  to.BoolPtr(true),
				MacAddress:         &nicSpec.MacAddress,
				IPConfigurations: &[]network.InterfaceIPConfiguration{
					{
						Name:                                     to.StringPtr("pipConfig"),
						InterfaceIPConfigurationPropertiesFormat: nicConfig,
					},
				},
			},
		})
	if err != nil {
		if azurestackhci.TransportUnavailable(err) {
			klog.Error("Communication with cloud agent failed. Exiting Process.")
			os.Exit(1)
		}

		return errors.Wrapf(err, "failed to create network interface %s in resource group %s", nicSpec.Name, s.Scope.GetResourceGroup())
	}

	klog.V(2).Infof("successfully created network interface %s", nicSpec.Name)
	return err
}

// Delete deletes the network interface with the provided name.
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	nicSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid network interface Specification")
	}
	klog.V(2).Infof("deleting nic %s", nicSpec.Name)
	err := s.Client.Delete(ctx, s.Scope.GetResourceGroup(), nicSpec.Name)
	if err != nil {
		if azurestackhci.TransportUnavailable(err) {
			klog.Error("Communication with cloud agent failed. Exiting Process.")
			os.Exit(1)
		}

		if azurestackhci.ResourceNotFound(err) {
			// already deleted
			return nil
		}

		return errors.Wrapf(err, "failed to delete network interface %s in resource group %s", nicSpec.Name, s.Scope.GetResourceGroup())
	}

	klog.V(2).Infof("successfully deleted nic %s", nicSpec.Name)
	return nil
}
