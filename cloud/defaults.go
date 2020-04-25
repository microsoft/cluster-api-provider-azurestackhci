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

package azurestackhci

import (
	"fmt"

	"github.com/blang/semver"
	infrav1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha3"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"
)

const (
	// DefaultUserName is the default username for created vm
	DefaultUserName = "clouduser"
	// DefaultVnetCIDR is the default Vnet CIDR
	DefaultVnetCIDR = "10.0.0.0/8"
	// DefaultVnetRouteDestinationPrefix is the destination prefix of the default Vnet route
	DefaultVnetRouteDestinationPrefix = "0.0.0.0/0"
	// DefaultVnetRouteNextHop is the next hop of the default Vnet route
	DefaultVnetRouteNextHop = "10.0.0.1"
	// DefaultControlPlaneSubnetCIDR is the default Control Plane Subnet CIDR
	DefaultControlPlaneSubnetCIDR = "10.0.0.0/16"
	// DefaultNodeSubnetCIDR is the default Node Subnet CIDR
	DefaultNodeSubnetCIDR = "10.1.0.0/16"
	// DefaultInternalLBIPAddress is the default internal load balancer ip address
	DefaultInternalLBIPAddress = "10.0.0.100"
	// DefaultAzureStackHCIDNSZone is the default provided azurestackhci dns zone
	DefaultAzureStackHCIDNSZone = "cloudapp.azurestackhci.com"
	// UserAgent used for communicating with azurestackhci
	UserAgent = "cluster-api-azurestackhci-services"
)

const (
	// DefaultImageOfferID is the default image offer ID
	DefaultImageOfferID = "linux"
	// DefaultImageSKU is the default image SKU
	DefaultImageSKU = "linux"
	// DefaultImagePublisherID is the default publisher ID
	DefaultImagePublisherID = "na"
	// LatestVersion is the image version latest
	LatestVersion = "latest"
)

// SupportedAvailabilityZoneLocations is a slice of the locations where Availability Zones are supported.
// This is used to validate whether a virtual machine should leverage an Availability Zone.
// Based on the Availability Zones listed in https://docs.microsoft.com/en-us/azure/availability-zones/az-overview
var SupportedAvailabilityZoneLocations = []string{
	// Americas
	"centralus",
	"eastus",
	"eastus2",
	"westus2",

	// Europe
	"francecentral",
	"northeurope",
	"uksouth",
	"westeurope",

	// Asia Pacific
	"japaneast",
	"southeastasia",
}

// GenerateVnetName generates a virtual network name, based on the cluster name.
func GenerateVnetName(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, "vnet")
}

// GenerateControlPlaneSecurityGroupName generates a control plane security group name, based on the cluster name.
func GenerateControlPlaneSecurityGroupName(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, "controlplane-nsg")
}

// GenerateNodeSecurityGroupName generates a node security group name, based on the cluster name.
func GenerateNodeSecurityGroupName(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, "node-nsg")
}

// GenerateNodeRouteTableName generates a node route table name, based on the cluster name.
func GenerateNodeRouteTableName(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, "node-routetable")
}

// GenerateControlPlaneSubnetName generates a node subnet name, based on the cluster name.
func GenerateControlPlaneSubnetName(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, "controlplane-subnet")
}

// GenerateNodeSubnetName generates a node subnet name, based on the cluster name.
func GenerateNodeSubnetName(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, "node-subnet")
}

// GenerateFQDN generates a fully qualified domain name, based on the public IP name and cluster location.
func GenerateFQDN(publicIPName, location string) string {
	return fmt.Sprintf("%s.%s.%s", publicIPName, location, DefaultAzureStackHCIDNSZone)
}

// GenerateNICName generates the name of a network interface based on the name of a VM.
func GenerateNICName(machineName string) string {
	return fmt.Sprintf("%s-nic", machineName)
}

// GenerateOSDiskName generates the name of an OS disk based on the name of a VM.
func GenerateOSDiskName(machineName string) string {
	return fmt.Sprintf("%s_OSDisk", machineName)
}

// GenerateLoadBalancerName generates the name of a load balancer based on the name of a cluster.
func GenerateLoadBalancerName(clusterName string) string {
	return fmt.Sprintf("%s-load-balancer", clusterName)
}

// GenerateBackendPoolName generates the name of a backend pool based on the name of a cluster.
func GenerateBackendPoolName(clusterName string) string {
	return fmt.Sprintf("%s-backend-pool", clusterName)
}

// GetDefaultImageName gets the name of the image to use for the provided OS and version of Kubernetes.
func getDefaultImageName(osType infrav1.OSType, k8sVersion string) (string, error) {
	version, err := semver.ParseTolerant(k8sVersion)
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse Kubernetes version \"%s\" in spec, expected valid SemVer string", k8sVersion)
	}
	return fmt.Sprintf("%s_k8s_%d-%d-%d", osType, version.Major, version.Minor, version.Patch), nil
}

// GetDefaultImage returns the default image spec for the provided OS and version of Kubernetes.
func GetDefaultImage(osType infrav1.OSType, k8sVersion string) (*infrav1.Image, error) {
	imageName, err := getDefaultImageName(osType, k8sVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get default image name")
	}

	defaultImage := &infrav1.Image{
		Name:      &imageName,
		OSType:    osType,
		Publisher: pointer.StringPtr(DefaultImagePublisherID),
		Offer:     pointer.StringPtr(DefaultImageOfferID),
		SKU:       pointer.StringPtr(DefaultImageSKU),
		Version:   pointer.StringPtr(LatestVersion),
	}

	return defaultImage, nil
}
