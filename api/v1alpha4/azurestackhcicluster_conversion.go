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

package v1alpha4

import (
	infrav1beta1 "github.com/microsoft/cluster-api-provider-azurestackhci/api/v1beta1"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	clusterv1alpha4 "sigs.k8s.io/cluster-api/api/v1alpha4"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *AzureStackHCICluster) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta1.AzureStackHCICluster)
	return Convert_v1alpha4_AzureStackHCICluster_To_v1beta1_AzureStackHCICluster(src, dst, nil)
}

func (dst *AzureStackHCICluster) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta1.AzureStackHCICluster)
	return Convert_v1beta1_AzureStackHCICluster_To_v1alpha4_AzureStackHCICluster(src, dst, nil)
}

func Convert_v1alpha4_APIEndpoint_To_v1beta1_APIEndpoint(in *clusterv1alpha4.APIEndpoint, out *clusterv1beta1.APIEndpoint, s apiconversion.Scope) error {
	return clusterv1alpha4.Convert_v1alpha4_APIEndpoint_To_v1beta1_APIEndpoint(in, out, s)
}

func Convert_v1beta1_APIEndpoint_To_v1alpha4_APIEndpoint(in *clusterv1beta1.APIEndpoint, out *clusterv1alpha4.APIEndpoint, s apiconversion.Scope) error {
	return clusterv1alpha4.Convert_v1beta1_APIEndpoint_To_v1alpha4_APIEndpoint(in, out, s)
}

func (src *AzureStackHCIClusterList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta1.AzureStackHCIClusterList)
	return Convert_v1alpha4_AzureStackHCIClusterList_To_v1beta1_AzureStackHCIClusterList(src, dst, nil)
}

func (dst *AzureStackHCIClusterList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta1.AzureStackHCIClusterList)
	return Convert_v1beta1_AzureStackHCIClusterList_To_v1alpha4_AzureStackHCIClusterList(src, dst, nil)
}
