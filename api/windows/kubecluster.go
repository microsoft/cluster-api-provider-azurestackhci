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

package windows

type Images struct {
	Pause      string
	Nanoserver string
	ServerCore string
}
type Cri struct {
	Name   string
	Images Images
}

type CniSource struct {
	Name string
	Url  string
}

type Plugin struct {
	Name string
}
type Cni struct {
	Name          string
	Source        CniSource
	Plugin        Plugin
	InterfaceName string
}

type KubernetesSource struct {
	Release string
	Url     string
}

type ControlPlane struct {
	IpAddress     string
	Username      string
	KubeadmToken  string
	KubeadmCAHash string
}

type KubeProxy struct {
	Gates string
}

type Network struct {
	ServiceCidr string
	ClusterCidr string
}
type Kubernetes struct {
	Source       KubernetesSource
	ControlPlane ControlPlane
	KubeProxy    KubeProxy
	Network      Network
}

type Install struct {
	Destination string
}
type KubeCluster struct {
	Cri        Cri
	Cni        Cni
	Kubernetes Kubernetes
	Install    Install
}
