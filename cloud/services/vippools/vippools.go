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

package vippools

import (
	"context"
	"fmt"

	azhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/moc-sdk-for-go/services/network"
	"github.com/pkg/errors"
	"k8s.io/klog"
)

// Spec input specification for Get/CreateOrUpdate/Delete calls
type Spec struct {
	Name     string
	Location string
}

// Get provides information about a vip pool.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	vpSpec, ok := spec.(*Spec)
	if !ok {
		return network.VipPool{}, errors.New("invalid vippool specification")
	}

	vp, err := s.Client.Get(ctx, vpSpec.Location, vpSpec.Name)
	if err != nil && azhci.ResourceNotFound(err) {
		return nil, errors.Wrapf(err, "vippool %s not found", vpSpec.Name)
	} else if err != nil {
		return nil, err
	}
	//If the user wants to get all the vippools, but none exist, cloudagent will return
	//a 0 length array.
	if vp == nil || len(*vp) == 0 {
		return nil, nil
	}
	return (*vp)[0], nil
}

// Reconcile gets/creates/updates a vip pool.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	vpSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid vippool specification")
	}

	if _, err := s.Get(ctx, vpSpec); err == nil {
		// vippool already exists, no update supported for now
		return nil
	}
	klog.V(2).Infof("creating a vippool is not supported")
	return fmt.Errorf("creating a vippool is not supported")
}

// Delete deletes the vip pool with the provided name.
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	klog.V(2).Infof("deleting a vippool is not supported")
	return fmt.Errorf("deleting a vippool is not supported")
}
