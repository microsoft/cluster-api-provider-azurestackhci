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
	v1beta2 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	corev1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	corev1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// Convert_v1beta1_AzureStackHCIClusterStatus_To_v1beta2_AzureStackHCIClusterStatus converts v1beta1 ClusterStatus to v1beta2.
func Convert_v1beta1_AzureStackHCIClusterStatus_To_v1beta2_AzureStackHCIClusterStatus(in *AzureStackHCIClusterStatus, out *v1beta2.AzureStackHCIClusterStatus, s conversion.Scope) error {
	// Convert all common fields using auto-generated function
	if err := autoConvert_v1beta1_AzureStackHCIClusterStatus_To_v1beta2_AzureStackHCIClusterStatus(in, out, s); err != nil {
		return err
	}

	// v1beta1 has Ready field, v1beta2 uses Initialization.Provisioned
	// This is critical for upgrade scenarios: when a v1beta1 cluster is being upgraded
	// and the v1beta2 controller reads it, the Initialization field must be populated
	// otherwise the machine controller will never reconcile machines (it checks Initialization.Provisioned)
	if in.Ready {
		out.Initialization = &v1beta2.AzureStackHCIClusterInitializationStatus{
			Provisioned: boolPtr(true),
		}
	}

	return nil
}

// Convert_v1beta2_AzureStackHCIClusterStatus_To_v1beta1_AzureStackHCIClusterStatus converts v1beta2 ClusterStatus to v1beta1.
func Convert_v1beta2_AzureStackHCIClusterStatus_To_v1beta1_AzureStackHCIClusterStatus(in *v1beta2.AzureStackHCIClusterStatus, out *AzureStackHCIClusterStatus, s conversion.Scope) error {
	// Convert all common fields using auto-generated function
	if err := autoConvert_v1beta2_AzureStackHCIClusterStatus_To_v1beta1_AzureStackHCIClusterStatus(in, out, s); err != nil {
		return err
	}

	// Set Ready field based on Ready condition in v1beta2
	// Look for "Ready" condition type
	out.Ready = false
	for _, c := range in.Conditions {
		if c.Type == "Ready" && c.Status == metav1.ConditionTrue {
			out.Ready = true
			break
		}
	}

	return nil
}

// Convert_v1beta1_AzureStackHCIMachineStatus_To_v1beta2_AzureStackHCIMachineStatus converts v1beta1 MachineStatus to v1beta2.
func Convert_v1beta1_AzureStackHCIMachineStatus_To_v1beta2_AzureStackHCIMachineStatus(in *AzureStackHCIMachineStatus, out *v1beta2.AzureStackHCIMachineStatus, s conversion.Scope) error {
	// Convert all common fields using auto-generated function
	if err := autoConvert_v1beta1_AzureStackHCIMachineStatus_To_v1beta2_AzureStackHCIMachineStatus(in, out, s); err != nil {
		return err
	}

	// v1beta1 has Ready field, v1beta2 uses Initialization.Provisioned
	// This is needed for upgrade scenarios when v1beta1 machines are read by v1beta2 controller
	if in.Ready {
		out.Initialization = &v1beta2.AzureStackHCIMachineInitializationStatus{
			Provisioned: boolPtr(true),
		}
	}

	return nil
}

// Convert_v1beta2_AzureStackHCIMachineStatus_To_v1beta1_AzureStackHCIMachineStatus converts v1beta2 MachineStatus to v1beta1.
func Convert_v1beta2_AzureStackHCIMachineStatus_To_v1beta1_AzureStackHCIMachineStatus(in *v1beta2.AzureStackHCIMachineStatus, out *AzureStackHCIMachineStatus, s conversion.Scope) error {
	// Convert all common fields using auto-generated function
	if err := autoConvert_v1beta2_AzureStackHCIMachineStatus_To_v1beta1_AzureStackHCIMachineStatus(in, out, s); err != nil {
		return err
	}

	// v1beta2 doesn't have FailureReason/FailureMessage fields
	// Leave them nil as they should be derived from conditions by the controller

	return nil
}

// Convert_v1beta2_APIEndpoint_To_v1beta1_APIEndpoint converts CAPI v1beta2 APIEndpoint to v1beta1.
func Convert_v1beta2_APIEndpoint_To_v1beta1_APIEndpoint(in *corev1beta2.APIEndpoint, out *corev1beta1.APIEndpoint, s conversion.Scope) error {
	out.Host = in.Host
	out.Port = in.Port
	return nil
}

// Convert_v1beta1_APIEndpoint_To_v1beta2_APIEndpoint converts CAPI v1beta1 APIEndpoint to v1beta2.
func Convert_v1beta1_APIEndpoint_To_v1beta2_APIEndpoint(in *corev1beta1.APIEndpoint, out *corev1beta2.APIEndpoint, s conversion.Scope) error {
	out.Host = in.Host
	out.Port = in.Port
	return nil
}

// Convert_Slice_v1_Condition_To_Slice_Pointer_v1beta1_Condition converts []metav1.Condition to []*corev1beta1.Condition.
func Convert_Slice_v1_Condition_To_Slice_Pointer_v1beta1_Condition(in *[]metav1.Condition, out *[]*corev1beta1.Condition, s conversion.Scope) error {
	if *in == nil {
		return nil
	}
	*out = make([]*corev1beta1.Condition, len(*in))
	for i := range *in {
		(*out)[i] = &corev1beta1.Condition{}
		if err := Convert_v1_Condition_To_v1beta1_Condition(&(*in)[i], (*out)[i], s); err != nil {
			return err
		}
	}
	return nil
}

// Convert_Slice_Pointer_v1beta1_Condition_To_Slice_v1_Condition converts []*corev1beta1.Condition to []metav1.Condition.
func Convert_Slice_Pointer_v1beta1_Condition_To_Slice_v1_Condition(in *[]*corev1beta1.Condition, out *[]metav1.Condition, s conversion.Scope) error {
	if *in == nil {
		return nil
	}
	*out = make([]metav1.Condition, len(*in))
	for i := range *in {
		if (*in)[i] != nil {
			if err := Convert_v1beta1_Condition_To_v1_Condition((*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	}
	return nil
}

// Convert_v1_Condition_To_v1beta1_Condition converts metav1.Condition to corev1beta1.Condition.
func Convert_v1_Condition_To_v1beta1_Condition(in *metav1.Condition, out *corev1beta1.Condition, s conversion.Scope) error {
	out.Type = corev1beta1.ConditionType(in.Type)
	out.Status = corev1.ConditionStatus(in.Status)
	out.LastTransitionTime = in.LastTransitionTime
	out.Reason = in.Reason
	out.Message = in.Message
	// ObservedGeneration and Severity don't exist in metav1.Condition
	return nil
}

// Convert_v1beta1_Condition_To_v1_Condition converts corev1beta1.Condition to metav1.Condition.
func Convert_v1beta1_Condition_To_v1_Condition(in *corev1beta1.Condition, out *metav1.Condition, s conversion.Scope) error {
	out.Type = string(in.Type)
	out.Status = metav1.ConditionStatus(in.Status)
	out.LastTransitionTime = in.LastTransitionTime
	out.Reason = in.Reason
	out.Message = in.Message
	// corev1beta1.Condition doesn't have ObservedGeneration field
	return nil
}

// Convert_v1beta1_OSDisk_To_v1beta2_OSDisk converts v1beta1 OSDisk to v1beta2.
// Manual conversion needed because v1beta2 ManagedDisk is a pointer type.
func Convert_v1beta1_OSDisk_To_v1beta2_OSDisk(in *OSDisk, out *v1beta2.OSDisk, s conversion.Scope) error {
	out.Name = in.Name
	out.Source = in.Source
	out.OSType = v1beta2.OSType(in.OSType)
	out.DiskSizeGB = in.DiskSizeGB
	// Convert value type to pointer - only set if non-empty
	if in.ManagedDisk.StorageAccountType != "" {
		out.ManagedDisk = &v1beta2.ManagedDisk{
			StorageAccountType: in.ManagedDisk.StorageAccountType,
		}
	}
	return nil
}

// Convert_v1beta2_OSDisk_To_v1beta1_OSDisk converts v1beta2 OSDisk to v1beta1.
// Manual conversion needed because v1beta2 ManagedDisk is a pointer type.
func Convert_v1beta2_OSDisk_To_v1beta1_OSDisk(in *v1beta2.OSDisk, out *OSDisk, s conversion.Scope) error {
	out.Name = in.Name
	out.Source = in.Source
	out.OSType = OSType(in.OSType)
	out.DiskSizeGB = in.DiskSizeGB
	// Convert pointer type to value type
	if in.ManagedDisk != nil {
		out.ManagedDisk = ManagedDisk{
			StorageAccountType: in.ManagedDisk.StorageAccountType,
		}
	}
	return nil
}

// Convert_v1beta1_Image_To_v1beta2_Image converts v1beta1 Image to v1beta2.
func Convert_v1beta1_Image_To_v1beta2_Image(in *Image, out *v1beta2.Image, s conversion.Scope) error {
	out.Publisher = in.Publisher
	out.Offer = in.Offer
	out.SKU = in.SKU
	out.ID = in.ID
	out.SubscriptionID = in.SubscriptionID
	out.ResourceGroup = in.ResourceGroup
	out.Gallery = in.Gallery
	out.Name = in.Name
	out.Version = in.Version
	out.OSType = v1beta2.OSType(in.OSType)
	return nil
}

// Convert_v1beta2_Image_To_v1beta1_Image converts v1beta2 Image to v1beta1.
func Convert_v1beta2_Image_To_v1beta1_Image(in *v1beta2.Image, out *Image, s conversion.Scope) error {
	out.Publisher = in.Publisher
	out.Offer = in.Offer
	out.SKU = in.SKU
	out.ID = in.ID
	out.SubscriptionID = in.SubscriptionID
	out.ResourceGroup = in.ResourceGroup
	out.Gallery = in.Gallery
	out.Name = in.Name
	out.Version = in.Version
	out.OSType = OSType(in.OSType)
	return nil
}

// Convert_v1beta1_AvailabilityZone_To_v1beta2_AvailabilityZone converts v1beta1 AvailabilityZone to v1beta2.
func Convert_v1beta1_AvailabilityZone_To_v1beta2_AvailabilityZone(in *AvailabilityZone, out *v1beta2.AvailabilityZone, s conversion.Scope) error {
	out.ID = in.ID
	out.Enabled = in.Enabled
	return nil
}

// Convert_v1beta2_AvailabilityZone_To_v1beta1_AvailabilityZone converts v1beta2 AvailabilityZone to v1beta1.
func Convert_v1beta2_AvailabilityZone_To_v1beta1_AvailabilityZone(in *v1beta2.AvailabilityZone, out *AvailabilityZone, s conversion.Scope) error {
	out.ID = in.ID
	out.Enabled = in.Enabled
	return nil
}

// Convert_v1beta1_AzureStackHCIMachineSpec_To_v1beta2_AzureStackHCIMachineSpec converts v1beta1 MachineSpec to v1beta2.
// Manual conversion needed because v1beta2 uses pointer types for Image, OSDisk, and AvailabilityZone.
func Convert_v1beta1_AzureStackHCIMachineSpec_To_v1beta2_AzureStackHCIMachineSpec(in *AzureStackHCIMachineSpec, out *v1beta2.AzureStackHCIMachineSpec, s conversion.Scope) error {
	out.ProviderID = in.ProviderID
	out.VMSize = in.VMSize
	out.Location = in.Location
	out.SSHPublicKey = in.SSHPublicKey
	out.StorageContainer = in.StorageContainer
	out.GpuCount = in.GpuCount
	out.AllocatePublicIP = in.AllocatePublicIP
	out.AdditionalSSHKeys = in.AdditionalSSHKeys
	out.AvailabilitySetName = in.AvailabilitySetName
	out.PlacementGroupName = in.PlacementGroupName

	// Convert NetworkInterfaces (slice of pointers, should work automatically)
	if in.NetworkInterfaces != nil {
		out.NetworkInterfaces = make(v1beta2.NetworkInterfaces, len(in.NetworkInterfaces))
		for i, nic := range in.NetworkInterfaces {
			if nic != nil {
				outNic := &v1beta2.NetworkInterfaceSpec{Name: nic.Name}
				if nic.IPConfigurations != nil {
					outNic.IPConfigurations = make(v1beta2.IpConfigurations, len(nic.IPConfigurations))
					for j, ipconfig := range nic.IPConfigurations {
						if ipconfig != nil {
							outNic.IPConfigurations[j] = &v1beta2.IpConfigurationSpec{
								Name:         ipconfig.Name,
								Primary:      ipconfig.Primary,
								Allocation:   v1beta2.IPAllocationMethod(ipconfig.Allocation),
								IpAddress:    ipconfig.IpAddress,
								PrefixLength: ipconfig.PrefixLength,
								SubnetId:     ipconfig.SubnetId,
								Gateway:      ipconfig.Gateway,
							}
						}
					}
				}
				out.NetworkInterfaces[i] = outNic
			}
		}
	}

	// Convert value types to pointers - only if non-zero
	if in.AvailabilityZone.ID != nil || in.AvailabilityZone.Enabled != nil {
		out.AvailabilityZone = &v1beta2.AvailabilityZone{
			ID:      in.AvailabilityZone.ID,
			Enabled: in.AvailabilityZone.Enabled,
		}
	}

	// Only set Image if it has meaningful content
	if in.Image.OSType != "" || in.Image.ID != nil || in.Image.Publisher != nil {
		out.Image = &v1beta2.Image{}
		if err := Convert_v1beta1_Image_To_v1beta2_Image(&in.Image, out.Image, s); err != nil {
			return err
		}
	}

	// Only set OSDisk if it has meaningful content
	if in.OSDisk.Name != "" || in.OSDisk.DiskSizeGB != 0 {
		out.OSDisk = &v1beta2.OSDisk{}
		if err := Convert_v1beta1_OSDisk_To_v1beta2_OSDisk(&in.OSDisk, out.OSDisk, s); err != nil {
			return err
		}
	}

	return nil
}

// Convert_v1beta2_AzureStackHCIMachineSpec_To_v1beta1_AzureStackHCIMachineSpec converts v1beta2 MachineSpec to v1beta1.
// Manual conversion needed because v1beta2 uses pointer types for Image, OSDisk, and AvailabilityZone.
func Convert_v1beta2_AzureStackHCIMachineSpec_To_v1beta1_AzureStackHCIMachineSpec(in *v1beta2.AzureStackHCIMachineSpec, out *AzureStackHCIMachineSpec, s conversion.Scope) error {
	out.ProviderID = in.ProviderID
	out.VMSize = in.VMSize
	out.Location = in.Location
	out.SSHPublicKey = in.SSHPublicKey
	out.StorageContainer = in.StorageContainer
	out.GpuCount = in.GpuCount
	out.AllocatePublicIP = in.AllocatePublicIP
	out.AdditionalSSHKeys = in.AdditionalSSHKeys
	out.AvailabilitySetName = in.AvailabilitySetName
	out.PlacementGroupName = in.PlacementGroupName

	// Convert NetworkInterfaces
	if in.NetworkInterfaces != nil {
		out.NetworkInterfaces = make(NetworkInterfaces, len(in.NetworkInterfaces))
		for i, nic := range in.NetworkInterfaces {
			if nic != nil {
				outNic := &NetworkInterfaceSpec{Name: nic.Name}
				if nic.IPConfigurations != nil {
					outNic.IPConfigurations = make(IpConfigurations, len(nic.IPConfigurations))
					for j, ipconfig := range nic.IPConfigurations {
						if ipconfig != nil {
							outNic.IPConfigurations[j] = &IpConfigurationSpec{
								Name:         ipconfig.Name,
								Primary:      ipconfig.Primary,
								Allocation:   IPAllocationMethod(ipconfig.Allocation),
								IpAddress:    ipconfig.IpAddress,
								PrefixLength: ipconfig.PrefixLength,
								SubnetId:     ipconfig.SubnetId,
								Gateway:      ipconfig.Gateway,
							}
						}
					}
				}
				out.NetworkInterfaces[i] = outNic
			}
		}
	}

	// Convert pointer types to value types
	if in.AvailabilityZone != nil {
		out.AvailabilityZone = AvailabilityZone{
			ID:      in.AvailabilityZone.ID,
			Enabled: in.AvailabilityZone.Enabled,
		}
	}

	if in.Image != nil {
		if err := Convert_v1beta2_Image_To_v1beta1_Image(in.Image, &out.Image, s); err != nil {
			return err
		}
	}

	if in.OSDisk != nil {
		if err := Convert_v1beta2_OSDisk_To_v1beta1_OSDisk(in.OSDisk, &out.OSDisk, s); err != nil {
			return err
		}
	}

	return nil
}

// Convert_v1beta1_AzureStackHCIVirtualMachineSpec_To_v1beta2_AzureStackHCIVirtualMachineSpec converts v1beta1 to v1beta2.
func Convert_v1beta1_AzureStackHCIVirtualMachineSpec_To_v1beta2_AzureStackHCIVirtualMachineSpec(in *AzureStackHCIVirtualMachineSpec, out *v1beta2.AzureStackHCIVirtualMachineSpec, s conversion.Scope) error {
	out.VMSize = in.VMSize
	out.BootstrapData = in.BootstrapData
	out.Identity = v1beta2.VMIdentity(in.Identity)
	out.Location = in.Location
	out.SSHPublicKey = in.SSHPublicKey
	out.StorageContainer = in.StorageContainer
	out.GpuCount = in.GpuCount
	out.ResourceGroup = in.ResourceGroup
	out.VnetName = in.VnetName
	out.ClusterName = in.ClusterName
	out.SubnetName = in.SubnetName
	out.BackendPoolNames = in.BackendPoolNames
	out.AdditionalSSHKeys = in.AdditionalSSHKeys
	out.AvailabilitySetName = in.AvailabilitySetName
	out.PlacementGroupName = in.PlacementGroupName

	// Convert NetworkInterfaces
	if in.NetworkInterfaces != nil {
		out.NetworkInterfaces = make(v1beta2.NetworkInterfaces, len(in.NetworkInterfaces))
		for i, nic := range in.NetworkInterfaces {
			if nic != nil {
				outNic := &v1beta2.NetworkInterfaceSpec{Name: nic.Name}
				if nic.IPConfigurations != nil {
					outNic.IPConfigurations = make(v1beta2.IpConfigurations, len(nic.IPConfigurations))
					for j, ipconfig := range nic.IPConfigurations {
						if ipconfig != nil {
							outNic.IPConfigurations[j] = &v1beta2.IpConfigurationSpec{
								Name:         ipconfig.Name,
								Primary:      ipconfig.Primary,
								Allocation:   v1beta2.IPAllocationMethod(ipconfig.Allocation),
								IpAddress:    ipconfig.IpAddress,
								PrefixLength: ipconfig.PrefixLength,
								SubnetId:     ipconfig.SubnetId,
								Gateway:      ipconfig.Gateway,
							}
						}
					}
				}
				out.NetworkInterfaces[i] = outNic
			}
		}
	}

	// Convert value types to pointers
	if in.AvailabilityZone.ID != nil || in.AvailabilityZone.Enabled != nil {
		out.AvailabilityZone = &v1beta2.AvailabilityZone{
			ID:      in.AvailabilityZone.ID,
			Enabled: in.AvailabilityZone.Enabled,
		}
	}

	// Always set Image for VirtualMachine since it was required
	out.Image = &v1beta2.Image{}
	if err := Convert_v1beta1_Image_To_v1beta2_Image(&in.Image, out.Image, s); err != nil {
		return err
	}

	if in.OSDisk.Name != "" || in.OSDisk.DiskSizeGB != 0 {
		out.OSDisk = &v1beta2.OSDisk{}
		if err := Convert_v1beta1_OSDisk_To_v1beta2_OSDisk(&in.OSDisk, out.OSDisk, s); err != nil {
			return err
		}
	}

	return nil
}

// Convert_v1beta2_AzureStackHCIVirtualMachineSpec_To_v1beta1_AzureStackHCIVirtualMachineSpec converts v1beta2 to v1beta1.
func Convert_v1beta2_AzureStackHCIVirtualMachineSpec_To_v1beta1_AzureStackHCIVirtualMachineSpec(in *v1beta2.AzureStackHCIVirtualMachineSpec, out *AzureStackHCIVirtualMachineSpec, s conversion.Scope) error {
	out.VMSize = in.VMSize
	out.BootstrapData = in.BootstrapData
	out.Identity = VMIdentity(in.Identity)
	out.Location = in.Location
	out.SSHPublicKey = in.SSHPublicKey
	out.StorageContainer = in.StorageContainer
	out.GpuCount = in.GpuCount
	out.ResourceGroup = in.ResourceGroup
	out.VnetName = in.VnetName
	out.ClusterName = in.ClusterName
	out.SubnetName = in.SubnetName
	out.BackendPoolNames = in.BackendPoolNames
	out.AdditionalSSHKeys = in.AdditionalSSHKeys
	out.AvailabilitySetName = in.AvailabilitySetName
	out.PlacementGroupName = in.PlacementGroupName

	// Convert NetworkInterfaces
	if in.NetworkInterfaces != nil {
		out.NetworkInterfaces = make(NetworkInterfaces, len(in.NetworkInterfaces))
		for i, nic := range in.NetworkInterfaces {
			if nic != nil {
				outNic := &NetworkInterfaceSpec{Name: nic.Name}
				if nic.IPConfigurations != nil {
					outNic.IPConfigurations = make(IpConfigurations, len(nic.IPConfigurations))
					for j, ipconfig := range nic.IPConfigurations {
						if ipconfig != nil {
							outNic.IPConfigurations[j] = &IpConfigurationSpec{
								Name:         ipconfig.Name,
								Primary:      ipconfig.Primary,
								Allocation:   IPAllocationMethod(ipconfig.Allocation),
								IpAddress:    ipconfig.IpAddress,
								PrefixLength: ipconfig.PrefixLength,
								SubnetId:     ipconfig.SubnetId,
								Gateway:      ipconfig.Gateway,
							}
						}
					}
				}
				out.NetworkInterfaces[i] = outNic
			}
		}
	}

	// Convert pointer types to value types
	if in.AvailabilityZone != nil {
		out.AvailabilityZone = AvailabilityZone{
			ID:      in.AvailabilityZone.ID,
			Enabled: in.AvailabilityZone.Enabled,
		}
	}

	if in.Image != nil {
		if err := Convert_v1beta2_Image_To_v1beta1_Image(in.Image, &out.Image, s); err != nil {
			return err
		}
	}

	if in.OSDisk != nil {
		if err := Convert_v1beta2_OSDisk_To_v1beta1_OSDisk(in.OSDisk, &out.OSDisk, s); err != nil {
			return err
		}
	}

	return nil
}

// Convert_v1beta1_AzureStackHCILoadBalancerSpec_To_v1beta2_AzureStackHCILoadBalancerSpec converts v1beta1 to v1beta2.
func Convert_v1beta1_AzureStackHCILoadBalancerSpec_To_v1beta2_AzureStackHCILoadBalancerSpec(in *AzureStackHCILoadBalancerSpec, out *v1beta2.AzureStackHCILoadBalancerSpec, s conversion.Scope) error {
	out.SSHPublicKey = in.SSHPublicKey
	out.VMSize = in.VMSize
	out.StorageContainer = in.StorageContainer
	out.Replicas = in.Replicas

	// Convert value type to pointer
	out.Image = &v1beta2.Image{}
	if err := Convert_v1beta1_Image_To_v1beta2_Image(&in.Image, out.Image, s); err != nil {
		return err
	}

	return nil
}

// Convert_v1beta2_AzureStackHCILoadBalancerSpec_To_v1beta1_AzureStackHCILoadBalancerSpec converts v1beta2 to v1beta1.
func Convert_v1beta2_AzureStackHCILoadBalancerSpec_To_v1beta1_AzureStackHCILoadBalancerSpec(in *v1beta2.AzureStackHCILoadBalancerSpec, out *AzureStackHCILoadBalancerSpec, s conversion.Scope) error {
	out.SSHPublicKey = in.SSHPublicKey
	out.VMSize = in.VMSize
	out.StorageContainer = in.StorageContainer
	out.Replicas = in.Replicas

	// Convert pointer type to value type
	if in.Image != nil {
		if err := Convert_v1beta2_Image_To_v1beta1_Image(in.Image, &out.Image, s); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to create a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}
