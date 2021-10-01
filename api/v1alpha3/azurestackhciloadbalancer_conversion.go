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
	infrav1alpha4 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1alpha4"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *AzureStackHCILoadBalancer) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1alpha4.AzureStackHCILoadBalancer)
	return Convert_v1alpha3_AzureStackHCILoadBalancer_To_v1alpha4_AzureStackHCILoadBalancer(src, dst, nil)
}

func (dst *AzureStackHCILoadBalancer) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1alpha4.AzureStackHCILoadBalancer)
	return Convert_v1alpha4_AzureStackHCILoadBalancer_To_v1alpha3_AzureStackHCILoadBalancer(src, dst, nil)
}

func (src *AzureStackHCILoadBalancerList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1alpha4.AzureStackHCILoadBalancerList)
	return Convert_v1alpha3_AzureStackHCILoadBalancerList_To_v1alpha4_AzureStackHCILoadBalancerList(src, dst, nil)
}

func (dst *AzureStackHCILoadBalancerList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1alpha4.AzureStackHCILoadBalancerList)
	return Convert_v1alpha4_AzureStackHCILoadBalancerList_To_v1alpha3_AzureStackHCILoadBalancerList(src, dst, nil)
}
