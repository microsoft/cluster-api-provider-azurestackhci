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
	AffinityType = "weak"
)

type Spec struct {
	Name     string
	Location string
}

func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	logger := s.Scope.GetLogger()

	availabilitysetSpec, ok := spec.(*Spec)
	if !ok {
		return compute.AvailabilitySet{}, errors.New("invalid availibility set specification")
	}
	logger.Info("finding availability set", "name", availabilitysetSpec.Name)
	availabilityset, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), availabilitysetSpec.Name)
	if err != nil {
		if azurestackhci.ResourceNotFound(err) {
			logger.Info("availability set doesn't exists", "name", availabilitysetSpec.Name)
			return nil, nil
		}
		logger.Info("Error in finding availability set", "name", availabilitysetSpec.Name)
		return nil, err
	}
	logger.Info("successfully found availability set", "name", availabilitysetSpec.Name)

	return (*availabilityset)[0], nil
}

func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	availabilitysetSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid availibility set specification")
	}
	logger := s.Scope.GetLogger()

	logger.Info("creating availability set", "name", availabilitysetSpec.Name, "location", availabilitysetSpec.Location)

	nodeCount, err := s.GetNodeCount(ctx, availabilitysetSpec.Location)

	if err != nil {
		logger.Error(err, "error in getting node count")
		return err
	}

	if nodeCount == 0 {
		return errors.New("Node count is zero")
	}

	// Availability set is not supported on 1 node.
	if nodeCount == 1 {
		logger.Info("availability set is not supported on 1 node")
		return nil
	}

	existingSet, err := s.Get(ctx, spec)
	if err != nil {
		logger.Info("error in getting availability set", "name", availabilitysetSpec.Name)
		return err
	}
	if existingSet != nil {
		logger.Info("availability set exists", "name", availabilitysetSpec.Name)
		return nil
	}

	newAvailbilitySet := compute.AvailabilitySet{
		Name:                     to.StringPtr(availabilitysetSpec.Name),
		Type:                     to.StringPtr(AffinityType),
		PlatformFaultDomainCount: to.Int32Ptr(int32(nodeCount)),
	}

	_, err = s.Client.Create(ctx, s.Scope.GetResourceGroup(), availabilitysetSpec.Name, &newAvailbilitySet)

	telemetry.WriteMocOperationLog(logger, telemetry.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), telemetry.VirtualMachine,
		telemetry.GenerateMocResourceName(s.Scope.GetResourceGroup(), availabilitysetSpec.Name), nil, err)
	if err != nil {
		return errors.Wrapf(err, "cannot create availability set %s", availabilitysetSpec.Name)
	}

	logger.Info("successfully created availability set", "name", availabilitysetSpec.Name)
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
	if err != nil {
		logger.Info("error in getting availability set", "name", availabilitysetSpec.Name)
		return err
	}

	if existingSet == nil {
		logger.Info("availability set not found", "name", availabilitysetSpec.Name)
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
			logger.Info("error in deleting availability set", "name", availabilitysetSpec.Name)
			return err
		}
		logger.Info("successfully deleted availability set", "name", availabilitysetSpec.Name)
	} else {
		logger.Info("availability set has vms associated. skip deletion", "name", availabilitysetSpec.Name)
	}

	return nil
}

func (s *Service) GetNodeCount(ctx context.Context, location string) (int, error) {
	logger := s.Scope.GetLogger()
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
