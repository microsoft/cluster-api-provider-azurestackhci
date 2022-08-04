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

package v1alpha3

import (
	infrav1beta1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *AzureStackHCIMachine) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta1.AzureStackHCIMachine)
	return Convert_v1alpha3_AzureStackHCIMachine_To_v1beta1_AzureStackHCIMachine(src, dst, nil)
}

func (dst *AzureStackHCIMachine) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta1.AzureStackHCIMachine)
	return Convert_v1beta1_AzureStackHCIMachine_To_v1alpha3_AzureStackHCIMachine(src, dst, nil)
}

func (src *AzureStackHCIMachineList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta1.AzureStackHCIMachineList)
	return Convert_v1alpha3_AzureStackHCIMachineList_To_v1beta1_AzureStackHCIMachineList(src, dst, nil)
}

func (dst *AzureStackHCIMachineList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta1.AzureStackHCIMachineList)
	return Convert_v1beta1_AzureStackHCIMachineList_To_v1alpha3_AzureStackHCIMachineList(src, dst, nil)
}
