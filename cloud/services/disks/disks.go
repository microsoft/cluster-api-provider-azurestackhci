/*
Copyright 2019 The Kubernetes Authors.

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

package disks

import (
	"context"

	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/wssdcloud-sdk-for-go/services/storage"
	"github.com/pkg/errors"
	"k8s.io/klog"
)

// Spec specification for disk
type Spec struct {
	Name   string
	Source string
}

// Get provides information about a disk.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	diskSpec, ok := spec.(*Spec)
	if !ok {
		return storage.VirtualHardDisk{}, errors.New("Invalid Disk Specification")
	}
	disk, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), "", diskSpec.Name)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		return nil, errors.Wrapf(err, "disk %s not found", diskSpec.Name)
	} else if err != nil {
		return nil, err
	}
	return (*disk)[0], nil
}

// Reconcile gets/creates/updates a disk.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	diskSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid Disk Specification")
	}

	if _, err := s.Get(ctx, diskSpec); err == nil {
		// disk already exists, cannot update since its immutable
		return nil
	}

	klog.V(2).Infof("creating disk %s ", diskSpec.Name)
	_, err := s.Client.CreateOrUpdate(ctx, s.Scope.GetResourceGroup(), "", diskSpec.Name,
		&storage.VirtualHardDisk{
			Name:                      &diskSpec.Name,
			VirtualHardDiskProperties: &storage.VirtualHardDiskProperties{},
		})
	if err != nil {
		return err
	}

	klog.V(2).Infof("successfully created disk %s ", diskSpec.Name)
	return err
}

// Delete deletes the disk associated with a VM.
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	diskSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("Invalid disk specification")
	}
	klog.V(2).Infof("deleting disk %s", diskSpec.Name)
	err := s.Client.Delete(ctx, s.Scope.GetResourceGroup(), "", diskSpec.Name)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		// already deleted
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete disk %s in resource group %s", diskSpec.Name, s.Scope.GetResourceGroup())
	}

	klog.V(2).Infof("successfully deleted disk %s", diskSpec.Name)
	return err
}
