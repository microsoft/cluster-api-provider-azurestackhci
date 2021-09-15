module github.com/microsoft/cluster-api-provider-azurestackhci

go 1.15

require (
	github.com/Azure/go-autorest/autorest v0.10.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/microsoft/moc v0.10.12-alpha.6
	github.com/microsoft/moc-sdk-for-go v0.10.9-alpha.4
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
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
