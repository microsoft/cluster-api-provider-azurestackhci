// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
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
