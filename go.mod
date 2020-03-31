module github.com/microsoft/cluster-api-provider-azurestackhci

go 1.12

require (
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/blang/semver v3.5.0+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/microsoft/cluster-api-provider-azurestackhci v0.0.10
	github.com/microsoft/moc v0.8.0-alpha.2
	github.com/microsoft/moc-sdk-for-go v0.8.0-alpha.2
	github.com/pkg/errors v0.9.1
	golang.org/x/crypto v0.0.0-20200320181102-891825fb96df
	golang.org/x/text v0.3.2
	google.golang.org/grpc v1.27.1
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20190809000727-6c36bc71fc4a
	sigs.k8s.io/cluster-api v0.2.9
	sigs.k8s.io/controller-runtime v0.3.0
)

replace (
	github.com/Azure/go-autorest v11.1.2+incompatible => github.com/Azure/go-autorest/autorest v0.10.0
	k8s.io/api => k8s.io/api v0.0.0-20190704095032-f4ca3d3bdf1d
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190704094733-8f6ac2502e51
)
