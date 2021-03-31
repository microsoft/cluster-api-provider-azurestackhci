module github.com/microsoft/cluster-api-provider-azurestackhci

go 1.14

require (
	github.com/Azure/go-autorest/autorest v0.10.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/microsoft/moc v0.10.7-alpha.5
	github.com/microsoft/moc-sdk-for-go v0.10.1-alpha.1
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
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

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible
	github.com/microsoft/moc => ../moc
	github.com/microsoft/moc-sdk-for-go => ../moc-sdk-for-go
)
