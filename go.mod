module github.com/microsoft/cluster-api-provider-azurestackhci

go 1.14

require (
	github.com/Azure/go-autorest/autorest v0.10.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/containerd/fifo v0.0.0-20200410184934-f15a3290365b // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/microsoft/moc v0.9.0-alpha.4
	github.com/microsoft/moc-sdk-for-go v0.8.1-alpha.1
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20200320181102-891825fb96df
	golang.org/x/text v0.3.2
	google.golang.org/grpc v1.27.1
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200121204235-bf4fb3bd569c // indirect
	k8s.io/utils v0.0.0-20200229041039-0a110f9eb7ab
	sigs.k8s.io/cluster-api v0.3.3
	sigs.k8s.io/controller-runtime v0.5.2
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible
