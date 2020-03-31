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

package loadbalancers

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	azurestackhci "github.com/microsoft/cluster-api-provider-azurestackhci/cloud"
	"github.com/microsoft/moc-sdk-for-go/services/network"
	"github.com/pkg/errors"
	"k8s.io/klog"
)

// Spec input specification for Get/CreateOrUpdate/Delete calls
type Spec struct {
	Name            string
	BackendPoolName string
	VnetName        string
}

// Get provides information about a load balancer.
func (s *Service) Get(ctx context.Context, spec interface{}) (interface{}, error) {
	lbSpec, ok := spec.(*Spec)
	if !ok {
		return network.LoadBalancer{}, errors.New("invalid loadbalancer specification")
	}

	lb, err := s.Client.Get(ctx, s.Scope.GetResourceGroup(), lbSpec.Name)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		return nil, errors.Wrapf(err, "loadbalancer %s not found", lbSpec.Name)
	} else if err != nil {
		return nil, err
	}
	return (*lb)[0], nil
}

// Reconcile gets/creates/updates a load balancer.
func (s *Service) Reconcile(ctx context.Context, spec interface{}) error {
	lbSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid loadbalancer specification")
	}

	if _, err := s.Get(ctx, lbSpec); err == nil {
		// loadbalancer already exists, no update supported for now
		return nil
	}

	networkLB := network.LoadBalancer{
		Name: to.StringPtr(lbSpec.Name),
		LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
			BackendAddressPools: &[]network.BackendAddressPool{
				network.BackendAddressPool{
					Name: to.StringPtr(lbSpec.BackendPoolName),
				},
			},
			FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
				network.FrontendIPConfiguration{
					FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
						Subnet: &network.Subnet{
							ID: to.StringPtr(lbSpec.VnetName),
						},
					},
				},
			},
		},
	}

	// create the load balancer
	klog.V(2).Infof("creating loadbalancer %s ", lbSpec.Name)
	_, err := s.Client.CreateOrUpdate(ctx, s.Scope.GetResourceGroup(), lbSpec.Name, &networkLB)
	if err != nil {
		return err
	}

	klog.V(2).Infof("successfully created loadbalancer %s ", lbSpec.Name)
	return err
}

// Delete deletes the load balancer with the provided name.
func (s *Service) Delete(ctx context.Context, spec interface{}) error {
	lbSpec, ok := spec.(*Spec)
	if !ok {
		return errors.New("invalid loadbalancer specification")
	}
	klog.V(2).Infof("deleting loadbalancer %s ", lbSpec.Name)
	err := s.Client.Delete(ctx, s.Scope.GetResourceGroup(), lbSpec.Name)
	if err != nil && azurestackhci.ResourceNotFound(err) {
		// already deleted
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "failed to delete loadbalancer %s in resource group %s", lbSpec.Name, s.Scope.GetResourceGroup())
	}

	klog.V(2).Infof("successfully deleted loadbalancer %s ", lbSpec.Name)
	return err
}
