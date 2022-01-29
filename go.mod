module github.com/microsoft/cluster-api-provider-azurestackhci

go 1.15

require (
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/microsoft/moc v0.10.14
	github.com/microsoft/moc-sdk-for-go v0.10.14
	github.com/onsi/gomega v1.14.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914 // indirect
	google.golang.org/grpc v1.39.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210722164352-7f3ee0f31471
	sigs.k8s.io/cluster-api v0.4.2
	sigs.k8s.io/controller-runtime v0.9.6
	sigs.k8s.io/controller-tools v0.6.2 // indirect
)

replace github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2

replace github.com/miekg/dns => github.com/miekg/dns v1.1.25
