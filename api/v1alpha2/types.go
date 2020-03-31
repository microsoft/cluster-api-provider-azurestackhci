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

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AzureStackHCIResourceReference is a reference to a specific AzureStackHCI resource by ID
type AzureStackHCIResourceReference struct {
	// ID of resource
	// +optional
	ID *string `json:"id,omitempty"`
}

// AzureStackHCIMachineProviderConditionType is a valid value for AzureStackHCIMachineProviderCondition.Type
type AzureStackHCIMachineProviderConditionType string

// Valid conditions for an AzureStackHCI machine instance
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

// Network encapsulates AzureStackHCI networking resources.
type Network struct {
	// SecurityGroups is a map from the role/kind of the security group to its unique name, if any.
	SecurityGroups map[SecurityGroupRole]SecurityGroup `json:"securityGroups,omitempty"`

	// APIServerIP is the Kubernetes API server public IP address.
	APIServerIP PublicIP `json:"apiServerIp,omitempty"`
}

// NetworkSpec encapsulates all things related to AzureStackHCI network.
type NetworkSpec struct {
	// Vnet configuration.
	// +optional
	Vnet VnetSpec `json:"vnet,omitempty"`

	// Subnets configuration.
	// +optional
	Subnets Subnets `json:"subnets,omitempty"`
}

// VnetSpec configures an AzureStackHCI virtual network.
type VnetSpec struct {
	// ID is the identifier of the virtual network this provider should use to create resources.
	ID string `json:"id,omitempty"`

	// Name defines a name for the virtual network resource.
	Name string `json:"name"`

	// CidrBlock is the CIDR block to be used when the provider creates a managed virtual network.
	CidrBlock string `json:"cidrBlock,omitempty"`
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

// SecurityGroupRole defines the unique role of a security group.
type SecurityGroupRole string

var (
	// SecurityGroupBastion defines an SSH bastion role
	SecurityGroupBastion = SecurityGroupRole("bastion")

	// SecurityGroupNode defines a Kubernetes workload node role
	SecurityGroupNode = SecurityGroupRole(Node)

	// SecurityGroupControlPlane defines a Kubernetes control plane node role
	SecurityGroupControlPlane = SecurityGroupRole(ControlPlane)
)

// SecurityGroup defines an AzureStackHCI security group.
type SecurityGroup struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	IngressRules IngressRules `json:"ingressRule"`
}

// SecurityGroupProtocol defines the protocol type for a security group rule.
type SecurityGroupProtocol string

var (
	// SecurityGroupProtocolAll is a wildcard for all IP protocols
	SecurityGroupProtocolAll = SecurityGroupProtocol("*")

	// SecurityGroupProtocolTCP represents the TCP protocol in ingress rules
	SecurityGroupProtocolTCP = SecurityGroupProtocol("Tcp")

	// SecurityGroupProtocolUDP represents the UDP protocol in ingress rules
	SecurityGroupProtocolUDP = SecurityGroupProtocol("Udp")
)

// IngressRule defines an AzureStackHCI ingress rule for security groups.
type IngressRule struct {
	Description string                `json:"description"`
	Protocol    SecurityGroupProtocol `json:"protocol"`

	// SourcePorts - The source port or range. Integer or range between 0 and 65535. Asterix '*' can also be used to match all ports.
	SourcePorts *string `json:"sourcePorts,omitempty"`

	// DestinationPorts - The destination port or range. Integer or range between 0 and 65535. Asterix '*' can also be used to match all ports.
	DestinationPorts *string `json:"destinationPorts,omitempty"`

	// Source - The CIDR or source IP range. Asterix '*' can also be used to match all source IPs. Default tags such as 'VirtualNetwork', 'AzureStackHCILoadBalancer' and 'Internet' can also be used. If this is an ingress rule, specifies where network traffic originates from.
	Source *string `json:"source,omitempty"`

	// Destination - The destination address prefix. CIDR or destination IP range. Asterix '*' can also be used to match all source IPs. Default tags such as 'VirtualNetwork', 'AzureStackHCILoadBalancer' and 'Internet' can also be used.
	Destination *string `json:"destination,omitempty"`
}

// IngressRules is a slice of AzureStackHCI ingress rules for security groups.
type IngressRules []*IngressRule

// PublicIP defines an AzureStackHCI public IP address.
// TODO: Remove once load balancer is implemented.
type PublicIP struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
	DNSName   string `json:"dnsName,omitempty"`
}

// VMState describes the state of an AzureStackHCI virtual machine.
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

// VM describes an AzureStackHCI virtual machine.
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

type AvailabilityZone struct {
	ID      *string `json:"id,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
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
}

// APIEndpoint represents a reachable Kubernetes API endpoint.
type APIEndpoint struct {
	// The hostname on which the API server is serving.
	Host string `json:"host"`

	// The port on which the API server is serving.
	Port int `json:"port"`
}

// VMIdentity defines the identity of the virtual machine, if configured.
type VMIdentity string

// TEMP: OSType describes the OS type of a disk.
type OSType string

var (
	// OSTypeLinux
	OSTypeLinux = OSType("Linux")
	// OSTypeWindows
	OSTypeWindows = OSType("Windows")
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

// SubnetSpec configures an AzureStackHCI subnet.
type SubnetSpec struct {
	// ID defines a unique identifier to reference this resource.
	ID string `json:"id,omitempty"`

	// Name defines a name for the subnet resource.
	Name string `json:"name"`

	// VnetID defines the ID of the virtual network this subnet should be built in.
	VnetID string `json:"vnetId"`

	// CidrBlock is the CIDR block to be used when the provider creates a managed Vnet.
	CidrBlock string `json:"cidrBlock,omitempty"`

	// SecurityGroup defines the NSG (network security group) that should be attached to this subnet.
	SecurityGroup SecurityGroup `json:"securityGroup"`
}

const (
	AnnotationClusterInfrastructureReady = "azurestackhci.cluster.sigs.k8s.io/infrastructure-ready"
	ValueReady                           = "true"
	AnnotationControlPlaneReady          = "azurestackhci.cluster.sigs.k8s.io/control-plane-ready"
)
