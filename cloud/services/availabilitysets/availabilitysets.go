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

package availabilitysets

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/telemetry"
	"github.com/microsoft/moc-sdk-for-go/services/compute"
	"github.com/pkg/errors"
)

const (
	FaultDomainCount = 2
	AffinityType     = "weak"
)

type Spec struct {
	Name     string
	Location string
}

func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	availabilitysetSpec, ok := spec.(*Spec)
	if !ok {
		return compute.AvailabilitySet{}, errors.New("Invalid Availibility Set Specification")
	}
	availabilityset, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), availabilitysetSpec.Name)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		return nil, errors.Wrapf(err, "Availability Set %s not found", availabilitysetSpec.Name)
	} else if err != nil {
		return nil, err
	}
	return (*availabilityset)[0], nil
}

func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	availabilitysetSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid Availibility Set Specification")
	}

	// TODO: nodeCount is failing with error "Authentication failed. Roles not found for [GET] operation"
	/*
		nodeCount, err := s.GetNodeCount(ctx, availabilitysetSpec.Location)

		if err != nil {
			return errors.Wrapf(err, "Error getting node count")
		}

		// TODO: Confirm if node resources are created
		if nodeCount == 0 {
			return errors.New("Node count is zero")
		}

		// Availability Set is not supported on 1 Node cluster
		// TODO: uncomment when mock client is removed

			if nodeCount == 1 {
				return nil
			}
	*/

	logger := s.Scope.GetLogger()
	existingSet, err := s.Get(ctx, spec)
	if err != nil && !azurestackhci.ResourceNotFound(err) {
		return err
	}
	if existingSet != nil {
		logger.Info("Availability Set exists", "name", availabilitysetSpec.Name)
		return nil
	}

	logger.Info("creating availability set", "name", availabilitysetSpec.Name)

	newAvailbilitySet := compute.AvailabilitySet{
		Name:                     to.StringPtr(availabilitysetSpec.Name),
		Type:                     to.StringPtr(AffinityType),
		PlatformFaultDomainCount: to.Int32Ptr(int32(FaultDomainCount)),
	}

	_, err = s.Client.Create(ctx, s.Scope.GetResourceGroup(), availabilitysetSpec.Name, &newAvailbilitySet)

	telemetry.WriteMocOperationLog(logger, telemetry.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), telemetry.VirtualMachine,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), availabilitysetSpec.Name), nil, err)
	if err != nil {
		return errors.Wrapf(err, "cannot create availability set %s", availabilitysetSpec.Name)
	}

	logger.Info("successfully availability set", "name", availabilitysetSpec.Name)
	return err
}

func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	availabilitysetSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid availibility set specification")
	}

	logger := s.Scope.GetLogger()
	logger.Info("deleting availability set", "name", availabilitysetSpec.Name)

	existingSet, err := s.Get(ctx, spec)
	// TODO: error for NotFound is not matching ResourceNotFound. Skip for testing
	/*
		if err != nil {

			if azurestackhci.ResourceNotFound(err) {
				// Already deleted or not created
				logger.Info("availability set not found", availabilitysetSpec.Name)
				return nil
			}
			return err

		}
	*/
	if err != nil || existingSet == nil {
		return nil
	}

	availabilitySet, ok := existingSet.(compute.AvailabilitySet)
	if !ok {
		return errors.New("error in converting to compute.AvailabilitySet")
	}

	if len(availabilitySet.VirtualMachines) == 0 {
		err = s.Client.Delete(ctx, s.Scope.GetResourceGroup(), availabilitysetSpec.Name)
		telemetry.WriteMocOperationLog(s.Scope.GetLogger(), telemetry.Delete, s.Scope.GetCustomResourceTypeWithName(), telemetry.VirtualMachine,
			telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), availabilitysetSpec.Name), nil, err)
		if err != nil {
			if !azurestackhci.ResourceNotFound(err) {
				return errors.Wrapf(err, "error in deleting availability set %s", availabilitysetSpec.Name)
			}
		}
		logger.Info("successfully deleted availability set", "name", availabilitysetSpec.Name)
	} else {
		logger.Info("availability set has vms associated. skip deletion", "name", availabilitysetSpec.Name)
	}

	return nil
}

func (s *Service) GetNodeCount(ctx context.Context, location string) (int, error) {
	logger := s.Scope.GetLogger()
	// TODO: Location is not populated in AzureStackHCIVirtualMachine CR. It needs to be popualted correctly or fetched from AzureStackHCICluster CR.
	//       Remove hard-coded value once above issue is resolved.
	location = "MocLocation"
	nodes, err := s.NodeClient.Get(ctx, location, "")
	if err != nil {
		return 0, err
	}

	if nodes == nil {
		logger.Info("Empty node resources")
		return 0, nil
	}

	return len(*nodes), nil
}
