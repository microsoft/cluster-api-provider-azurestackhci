module github.com/microsoft/cluster-api-provider-azurestackhci

go 1.15

require (
	github.com/Azure/go-autorest/autorest v0.10.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/microsoft/moc v0.10.14-0.20211025231229-16e2917e625a
	github.com/microsoft/moc-sdk-for-go v0.10.14-0.20211025232132-d55a0f131e6c
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20210917221730-978cfadd31cf // indirect
	golang.org/x/sys v0.0.0-20210917161153-d61c044b1678 // indirect
	google.golang.org/grpc v1.27.1
	k8s.io/api v0.17.9
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v0.17.9
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20200619165400-6e3d28b6ed19
	sigs.k8s.io/cluster-api v0.3.11
	sigs.k8s.io/controller-runtime v0.5.11
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible

replace github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2

replace github.com/miekg/dns => github.com/miekg/dns v1.1.25
