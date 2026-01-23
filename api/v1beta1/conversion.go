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

	// v1beta1 has Ready field, v1beta2 uses conditions
	// The Ready condition should be managed by the controller, so we don't convert it here
	// v1beta2 has Initialization field which doesn't exist in v1beta1, leave it nil

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

	// v1beta1 has FailureReason/FailureMessage, v1beta2 uses conditions
	// These are managed by the controller and reflected in conditions, so we don't convert them
	// v1beta2 has Initialization field which doesn't exist in v1beta1, leave it nil

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
