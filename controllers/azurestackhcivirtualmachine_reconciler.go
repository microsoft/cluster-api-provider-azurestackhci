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

package controllers

import (
	"encoding/base64"

	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha4"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/scope"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/disks"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/networkinterfaces"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/secrets"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/services/virtualmachines"
	sdk_compute "github.com/microsoft/moc-sdk-for-go/services/compute"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

// azureStackHCIVirtualMachineService are list of services required by cluster actuator, easy to create a fake
// TODO: We should decide if we want to keep this
type azureStackHCIVirtualMachineService struct {
	vmScope              *scope.VirtualMachineScope
	networkInterfacesSvc azurestackhci.Service
	virtualMachinesSvc   azurestackhci.GetterService
	disksSvc             azurestackhci.GetterService
	secretsSvc           azurestackhci.GetterService
}

// newAzureStackHCIMachineService populates all the services based on input scope
func newAzureStackHCIVirtualMachineService(vmScope *scope.VirtualMachineScope) *azureStackHCIVirtualMachineService {
	return &azureStackHCIVirtualMachineService{
		vmScope:              vmScope,
		networkInterfacesSvc: networkinterfaces.NewService(vmScope),
		virtualMachinesSvc:   virtualmachines.NewService(vmScope),
		disksSvc:             disks.NewService(vmScope),
		secretsSvc:           secrets.NewService(vmScope),
	}
}

// Create creates machine if and only if machine exists, handled by cluster-api
func (s *azureStackHCIVirtualMachineService) Create() (*infrav1.VM, error) {
	nicName := azurestackhci.GenerateNICName(s.vmScope.Name())
	nicErr := s.reconcileNetworkInterface(nicName)
	if nicErr != nil {
		return nil, errors.Wrapf(nicErr, "failed to create nic %s for machine %s", nicName, s.vmScope.Name())
	}

	vm, vmErr := s.createVirtualMachine(nicName)
	if vmErr != nil {
		return nil, errors.Wrapf(vmErr, "failed to create vm %s ", s.vmScope.Name())
	}

	return vm, nil
}

// Delete reconciles all the services in pre determined order
func (s *azureStackHCIVirtualMachineService) Delete() error {
	vmSpec := &virtualmachines.Spec{
		Name: s.vmScope.Name(),
	}

	err := s.virtualMachinesSvc.Delete(s.vmScope.Context, vmSpec)
	if err != nil {
		return errors.Wrapf(err, "failed to delete machine")
	}

	networkInterfaceSpec := &networkinterfaces.Spec{
		Name:     azurestackhci.GenerateNICName(s.vmScope.Name()),
		VnetName: s.vmScope.VnetName(),
	}

	err = s.networkInterfacesSvc.Delete(s.vmScope.Context, networkInterfaceSpec)
	if err != nil {
		return errors.Wrapf(err, "Unable to delete network interface")
	}

	diskSpec := &disks.Spec{
		Name: azurestackhci.GenerateOSDiskName(s.vmScope.Name()),
	}

	err = s.disksSvc.Delete(s.vmScope.Context, diskSpec)
	if err != nil {
		return errors.Wrapf(err, "Unable to delete os disk of machine %s", s.vmScope.Name())
	}

	return nil
}

func (s *azureStackHCIVirtualMachineService) VMIfExists() (*infrav1.VM, error) {

	vmSpec := &virtualmachines.Spec{
		Name: s.vmScope.Name(),
	}
	vmInterface, err := s.virtualMachinesSvc.Get(s.vmScope.Context, vmSpec)

	if err != nil && vmInterface == nil {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "Failed to get vm")
	}

	vm, ok := vmInterface.(*infrav1.VM)
	if !ok {
		return nil, errors.New("returned incorrect vm interface")
	}

	klog.Infof("Found vm for machine %s", s.vmScope.Name())

	return vm, nil
}

// getVirtualMachineZone gets a random availability zones from available set,
// this will hopefully be an input from upstream machinesets so all the vms are balanced
func (s *azureStackHCIVirtualMachineService) getVirtualMachineZone() (string, error) {
	return "", nil
}

func (s *azureStackHCIVirtualMachineService) reconcileDisk(disk infrav1.OSDisk) error {
	diskSpec := &disks.Spec{
		Name:   azurestackhci.GenerateOSDiskName(s.vmScope.Name()), //disk.Name,
		Source: disk.Source,
	}

	err := s.disksSvc.Reconcile(s.vmScope.Context, diskSpec)
	if err != nil {
		return errors.Wrap(err, "unable to create VM OS disk")
	}

	return err
}

func (s *azureStackHCIVirtualMachineService) reconcileNetworkInterface(nicName string) error {
	networkInterfaceSpec := &networkinterfaces.Spec{
		Name:             nicName,
		VnetName:         s.vmScope.VnetName(),
		SubnetName:       s.vmScope.SubnetName(), // this field is required to be passed from AzureStackHCIMachine
		BackendPoolNames: s.vmScope.BackendPoolNames(),
	}
	err := s.networkInterfacesSvc.Reconcile(s.vmScope.Context, networkInterfaceSpec)
	if err != nil {
		return errors.Wrap(err, "unable to create VM network interface")
	}

	return err
}

func (s *azureStackHCIVirtualMachineService) createVirtualMachine(nicName string) (*infrav1.VM, error) {
	var vm *infrav1.VM
	decodedKeys := []string{}
	decoded, err := base64.StdEncoding.DecodeString(s.vmScope.AzureStackHCIVirtualMachine.Spec.SSHPublicKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode ssh public key")
	}
	decodedKeys = append(decodedKeys, string(decoded))

	for _, key := range s.vmScope.AzureStackHCIVirtualMachine.Spec.AdditionalSSHKeys {
		decoded, err = base64.StdEncoding.DecodeString(key)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode an additional ssh public key")
		}
		decodedKeys = append(decodedKeys, string(decoded))
	}

	vmSpec := &virtualmachines.Spec{
		Name: s.vmScope.Name(),
	}

	vmInterface, err := s.virtualMachinesSvc.Get(s.vmScope.Context, vmSpec)
	if err != nil && vmInterface == nil {
		var vmZone string

		azSupported := s.isAvailabilityZoneSupported()

		if azSupported {
			useAZ := true

			if s.vmScope.AzureStackHCIVirtualMachine.Spec.AvailabilityZone.Enabled != nil {
				useAZ = *s.vmScope.AzureStackHCIVirtualMachine.Spec.AvailabilityZone.Enabled
			}

			if useAZ {
				var zoneErr error
				vmZone, zoneErr = s.getVirtualMachineZone()
				if zoneErr != nil {
					return nil, errors.Wrap(zoneErr, "failed to get availability zone")
				}
			}
		}

		vmType := sdk_compute.Tenant
		if s.vmScope.AzureStackHCILoadBalancerVM() == true {
			vmType = sdk_compute.LoadBalancer
		}

		s.vmScope.Info("VM type is:", "vmType", vmType)

		vmSpec = &virtualmachines.Spec{
			Name:       s.vmScope.Name(),
			NICName:    nicName,
			SSHKeyData: decodedKeys,
			Size:       s.vmScope.AzureStackHCIVirtualMachine.Spec.VMSize,
			OSDisk:     s.vmScope.AzureStackHCIVirtualMachine.Spec.OSDisk,
			Image:      s.vmScope.AzureStackHCIVirtualMachine.Spec.Image,
			CustomData: *s.vmScope.AzureStackHCIVirtualMachine.Spec.BootstrapData,
			Zone:       vmZone,
			VMType:     vmType,
		}

		err = s.virtualMachinesSvc.Reconcile(s.vmScope.Context, vmSpec)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create or get machine")
		}
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get vm")
	}

	newVM, err := s.virtualMachinesSvc.Get(s.vmScope.Context, vmSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get vm")
	}

	vm, ok := newVM.(*infrav1.VM)
	if !ok {
		return nil, errors.New("returned incorrect vm interface")
	}
	if vm.State == "" {
		return nil, errors.Errorf("vm %s is nil provisioning state, reconcile", s.vmScope.Name())
	}

	if vm.State == infrav1.VMStateFailed {
		// If VM failed provisioning, delete it so it can be recreated
		err = s.virtualMachinesSvc.Delete(s.vmScope.Context, vmSpec)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to delete machine")
		}
		return nil, errors.Errorf("vm %s is deleted, retry creating in next reconcile", s.vmScope.Name())
	} else if vm.State != infrav1.VMStateSucceeded {
		return nil, errors.Errorf("vm %s is still in provisioning state %s, reconcile", s.vmScope.Name(), vm.State)
	}

	return vm, nil
}

// isAvailabilityZoneSupported determines if Availability Zones are supported in a selected location
// based on SupportedAvailabilityZoneLocations. Returns true if supported.
func (s *azureStackHCIVirtualMachineService) isAvailabilityZoneSupported() bool {
	azSupported := false

	for _, supportedLocation := range azurestackhci.SupportedAvailabilityZoneLocations {
		if s.vmScope.Location() == supportedLocation {
			azSupported = true

			return azSupported
		}
	}

	s.vmScope.V(2).Info("Availability Zones are not supported in the selected location", "location", s.vmScope.Location())
	return azSupported
}
