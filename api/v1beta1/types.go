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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AzureStackHCIResourceReference is a reference to a specific Azure resource by ID
type AzureStackHCIResourceReference struct {
	// ID of resource
	// +optional
	ID *string `json:"id,omitempty"`
}

// AzureStackHCIMachineProviderConditionType is a valid value for AzureStackHCIMachineProviderCondition.Type
type AzureStackHCIMachineProviderConditionType string

// Valid conditions for an Azure machine instance
const (
	// MachineCreated indicates whether the machine has been created or not. If not,
	// it should include a reason and message for the failure.
	MachineCreated AzureStackHCIMachineProviderConditionType = "MachineCreated"
)

// AzureStackHCIMachineProviderCondition is a condition in a AzureStackHCIMachineProviderStatus
type AzureStackHCIMachineProviderCondition struct {
	// Type is the type of the condition.
	Type AzureStackHCIMachineProviderConditionType `json:"type"`
	// Status is the status of the condition.
	Status corev1.ConditionStatus `json:"status"`
	// LastProbeTime is the last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime"`
	// LastTransitionTime is the last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// Reason is a unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason"`
	// Message is a human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message"`
}

const (
	// ControlPlane machine label
	ControlPlane string = "control-plane"
	// Node machine label
	Node string = "node"
)

// NetworkSpec encapsulates all things related to Azure network.
type NetworkSpec struct {
	// Vnet is the configuration for the Azure virtual network.
	// +optional
	Vnet VnetSpec `json:"vnet,omitempty"`

	// Subnets is the configuration for the control-plane subnet and the node subnet.
	// +optional
	Subnets Subnets `json:"subnets,omitempty"`
}

// VnetSpec configures an Azure virtual network.
type VnetSpec struct {
	// ID is the identifier of the virtual network this provider should use to create resources.
	ID string `json:"id,omitempty"`

	// Name defines a name for the virtual network resource.
	Name string `json:"name"`

	// CidrBlock is the CIDR block to be used when the provider creates a managed virtual network.
	CidrBlock string `json:"cidrBlock,omitempty"`

	// Group is the resource group the vnet should use.
	Group string `json:"group,omitempty"`
}

// Subnets is a slice of Subnet.
type Subnets []*SubnetSpec

// ToMap returns a map from id to subnet.
func (s Subnets) ToMap() map[string]*SubnetSpec {
	res := make(map[string]*SubnetSpec)
	for _, x := range s {
		res[x.ID] = x
	}
	return res
}

type IPAllocationMethod int32

const (
	IPAllocationMethod_Invalid IPAllocationMethod = 0
	IPAllocationMethod_Dynamic IPAllocationMethod = 1
	IPAllocationMethod_Static  IPAllocationMethod = 2
)

type IpConfigurationSpec struct {
	Name string `json:"name,omitempty"`
	// +optional
	Primary bool `json:"primary,omitempty"`
	// +optional
	Allocation IPAllocationMethod `json:"allocation,omitempty"`
	// below fields are unused, but adding for completeness
	// +optional
	IpAddress string `json:"ipAddress,omitempty"`
	// +optional
	PrefixLength string `json:"prefixLength,omitempty"`
	// +optional
	SubnetId string `json:"subnetId,omitempty"`
	// +optional
	Gateway string `json:"gateway,omitempty"`
}
type IpConfigurations []*IpConfigurationSpec

type NetworkInterfaceSpec struct {
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	IPConfigurations IpConfigurations `json:"ipConfigurations,omitempty"`
}

type NetworkInterfaces []*NetworkInterfaceSpec

const (
	// OSVersionLabelName is the label set on resources to identify their os version
	OSVersionLabelName = "msft.microsoft/os-version"
	// LoadBalancerLabel is the label set on load balancer replica machines
	LoadBalancerLabel = "msft.microsoft/load-balancer"
)

// VMState describes the state of an Azure virtual machine.
type VMState string

var (
	// VMStateCreating ...
	VMStateCreating = VMState("Creating")
	// VMStateDeleting ...
	VMStateDeleting = VMState("Deleting")
	// VMStateFailed ...
	VMStateFailed = VMState("Failed")
	// VMStateMigrating ...
	VMStateMigrating = VMState("Migrating")
	// VMStateSucceeded ...
	VMStateSucceeded = VMState("Succeeded")
	// VMStateUpdating ...
	VMStateUpdating = VMState("Updating")
)

// VM describes an Azure virtual machine.
type VM struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`

	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// Hardware profile
	VMSize string `json:"vmSize,omitempty"`

	// Storage profile
	Image  Image  `json:"image,omitempty"`
	OSDisk OSDisk `json:"osDisk,omitempty"`

	BootstrapData string `json:"bootstrapData,omitempty"`

	// State - The provisioning state, which only appears in the response.
	State    VMState    `json:"vmState,omitempty"`
	Identity VMIdentity `json:"identity,omitempty"`
}

// Image defines information about the image to use for VM creation.
// There are three ways to specify an image: by ID, by publisher, or by Shared Image Gallery.
// If specifying an image by ID, only the ID field needs to be set.
// If specifying an image by publisher, the Publisher, Offer, SKU, and Version fields must be set.
// If specifying an image from a Shared Image Gallery, the SubscriptionID, ResourceGroup,
// Gallery, Name, and Version fields must be set.
type Image struct {
	Publisher *string `json:"publisher,omitempty"`
	Offer     *string `json:"offer,omitempty"`
	SKU       *string `json:"sku,omitempty"`

	ID *string `json:"id,omitempty"`

	SubscriptionID *string `json:"subscriptionID,omitempty"`
	ResourceGroup  *string `json:"resourceGroup,omitempty"`
	Gallery        *string `json:"gallery,omitempty"`
	Name           *string `json:"name,omitempty"`

	Version *string `json:"version,omitempty"`
	OSType  OSType  `json:"osType"`
}

type AvailabilityZone struct {
	ID      *string `json:"id,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

// VMIdentity defines the identity of the virtual machine, if configured.
type VMIdentity string

// OSType describes the OS type of a disk.
type OSType string

const (
	// OSTypeLinux
	OSTypeLinux = OSType("Linux")
	// OSTypeWindows
	OSTypeWindows = OSType("Windows")
	// OSTypeWindows2022
	OSTypeWindows2022 = OSType("Windows2022")
)

type OSDisk struct {
	Name        string      `json:"name"`
	Source      string      `json:"source"`
	OSType      OSType      `json:"osType"`
	DiskSizeGB  int32       `json:"diskSizeGB"`
	ManagedDisk ManagedDisk `json:"managedDisk"`
}

type ManagedDisk struct {
	StorageAccountType string `json:"storageAccountType"`
}

// SubnetSpec configures an Azure subnet.
type SubnetSpec struct {
	// ID defines a unique identifier to reference this resource.
	ID string `json:"id,omitempty"`

	// Name defines a name for the subnet resource.
	Name string `json:"name"`

	// VnetID defines the ID of the virtual network this subnet should be built in.
	VnetID string `json:"vnetId"`

	// CidrBlock is the CIDR block to be used when the provider creates a managed Vnet.
	CidrBlock string `json:"cidrBlock,omitempty"`
}

const (
	AnnotationClusterInfrastructureReady = "azurestackhci.cluster.sigs.k8s.io/infrastructure-ready"
	ValueReady                           = "true"
	AnnotationControlPlaneReady          = "azurestackhci.cluster.sigs.k8s.io/control-plane-ready"
	AzureOperationIDAnnotationKey        = "management.azure.com/operationId"
	AzureCorrelationIDAnnotationKey      = "management.azure.com/correlationId"
)
