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

package converters

import (
	"github.com/Azure/go-autorest/autorest/to"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha4"
	"github.com/microsoft/moc-sdk-for-go/services/compute"
)

// SDKToVM converts an SDK VirtualMachine to the provider VM type.
func SDKToVM(v compute.VirtualMachine) (*infrav1.VM, error) {
	vm := &infrav1.VM{
		ID:    to.String(v.ID),
		Name:  to.String(v.Name),
		State: infrav1.VMStateSucceeded, // Hard-coded for now until we expose provisioning state
	}
	return vm, nil
}
