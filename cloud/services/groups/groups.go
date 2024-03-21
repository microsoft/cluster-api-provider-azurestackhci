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

package groups

import (
	"context"

	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/cluster-api-provider-azurestackhci/cloud/telemetry"
	"github.com/microsoft/moc-sdk-for-go/services/cloud"
	"github.com/pkg/errors"
)

const (
	TagKeyClusterGroup = "ownedBy"
	TagValClusterGroup = "caph"
)

// Spec specification for group
type Spec struct {
	Name     string
	Location string
}

// Get provides information about a group.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	groupSpec, ok := spec.(*Spec)
	if !ok {
		return cloud.Group{}, errors.New("Invalid group specification")
	}
	group, err := s.Client.Get(ctx, groupSpec.Location, groupSpec.Name)
	if err != nil {
		return nil, err
	}
	return (*group)[0], nil
}

// Reconcile gets/creates/updates a group.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	groupSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid group specification")
	}

	if _, err := s.Get(ctx, groupSpec); err == nil {
		// group already exists, cannot update since its immutable
		return nil
	}

	//adding tag to group
	tag := make(map[string]*string, 1)
	caphVal := TagValClusterGroup
	tag[TagKeyClusterGroup] = &caphVal

	logger := s.Scope.GetLogger()
	logger.Info("creating group", "name", groupSpec.Name, "location", groupSpec.Location)
	_, err := s.Client.CreateOrUpdate(ctx, groupSpec.Location, groupSpec.Name,
		&cloud.Group{
			Name:     &groupSpec.Name,
			Location: &groupSpec.Location,
			Tags:     tag,
		})
	telemetry.WriteMocOperationLog(logger, telemetry.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), telemetry.Group,
		telemetry.GenerateMocResourceName(groupSpec.Location, groupSpec.Name), nil, err)
	if err != nil {
		return err
	}

	logger.Info("successfully created group", "name", groupSpec.Name)
	return err
}

// Delete deletes a group if group is created by caph
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	telemetry.WriteMocInfoLog(ctx, s.Scope)
	groupSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid group specification")
	}
	logger := s.Scope.GetLogger()
	logger.Info("deleting group", "name", groupSpec.Name, "location", groupSpec.Location)

	group, err := s.Client.Get(ctx, groupSpec.Location, groupSpec.Name)
	telemetry.WriteMocOperationLog(logger, telemetry.Delete, s.Scope.GetCustomResourceTypeWithName(), telemetry.Group,
		telemetry.GenerateMocResourceName(groupSpec.Location, groupSpec.Name), nil, err)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		// ignoring the NotFound error, since it might be already deleted
		logger.Info("group not found, skipping deletion", "name", groupSpec.Name)
		return nil
	} else if err != nil {
		return err
	}

	groupObj := (*group)[0]
	value, ok := groupObj.Tags[TagKeyClusterGroup]
	// delete only if created by caph
	if ok && (value != nil && *value == TagValClusterGroup) {
		err := s.Client.Delete(ctx, groupSpec.Location, groupSpec.Name)
		if err != nil && azurestackhci.ResourceNotFound(err) {
			// already deleted
			return nil
		}
		if err != nil {
			return errors.Wrapf(err, "failed to delete group %s", groupSpec.Name)
		}
		logger.Info("successfully deleted group", "name", groupSpec.Name)
	} else {
		logger.Info("skipping group deletion, since it is not created by caph", "name", groupSpec.Name)
	}

	return err
}
