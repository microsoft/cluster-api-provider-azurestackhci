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
	"github.com/microsoft/moc-sdk-for-go/services/cloud"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
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
	azurestackhci.WriteMocDeploymentIdLog(ctx, s.Scope.GetCloudAgentFqdn(), s.Scope.GetAuthorizer())
	groupSpec, ok := spec.(*Spec)
	if !ok {
		return cloud.Group{}, errors.New("Invalid group specification")
	}
	group, err := s.Client.Get(ctx, groupSpec.Location, groupSpec.Name)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		return nil, errors.Wrapf(err, "group %s not found in location %s", groupSpec.Name, groupSpec.Location)
	} else if err != nil {
		return nil, err
	}
	return (*group)[0], nil
}

// Reconcile gets/creates/updates a group.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	azurestackhci.WriteMocDeploymentIdLog(ctx, s.Scope.GetCloudAgentFqdn(), s.Scope.GetAuthorizer())
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

	klog.V(2).Infof("creating group %s in location %s", groupSpec.Name, groupSpec.Location)
	_, err := s.Client.CreateOrUpdate(ctx, groupSpec.Location, groupSpec.Name,
		&cloud.Group{
			Name:     &groupSpec.Name,
			Location: &groupSpec.Location,
			Tags:     tag,
		})
	azurestackhci.WriteMocOperationLog(azurestackhci.CreateOrUpdate, s.Scope.GetCustomResourceTypeWithName(), azurestackhci.Group,
		azurestackhci.GenerateMocResourceName(groupSpec.Location, groupSpec.Name), nil, err)
	if err != nil {
		return err
	}

	klog.V(2).Infof("successfully created group %s", groupSpec.Name)
	return err
}

// Delete deletes a group if group is created by caph
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	azurestackhci.WriteMocDeploymentIdLog(ctx, s.Scope.GetCloudAgentFqdn(), s.Scope.GetAuthorizer())
	groupSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid group specification")
	}
	klog.V(2).Infof("deleting group %s in location %s", groupSpec.Name, groupSpec.Location)

	group, err := s.Client.Get(ctx, groupSpec.Location, groupSpec.Name)
	azurestackhci.WriteMocOperationLog(azurestackhci.Delete, s.Scope.GetCustomResourceTypeWithName(), azurestackhci.Group,
		azurestackhci.GenerateMocResourceName(groupSpec.Location, groupSpec.Name), nil, err)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		// ignoring the NotFound error, since it might be already deleted
		klog.V(2).Infof("group %s not found in location %s", groupSpec.Name, groupSpec.Location)
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
		klog.V(2).Infof("successfully deleted group %s", groupSpec.Name)
	} else {
		klog.V(2).Infof("skipping group %s deletion, since it is not created by caph", groupSpec.Name)
	}

	return err
}
