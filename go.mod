module github.com/microsoft/cluster-api-provider-azurestackhci

go 1.16

require (
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/caddyserver/caddy v1.0.3 // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0 // indirect
	github.com/drone/envsubst v1.0.3-0.20200709223903-efdb65b94e5a // indirect
	github.com/go-logr/logr v0.4.0
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/gophercloud/gophercloud v0.1.0 // indirect
	github.com/microsoft/moc v0.10.9
	github.com/microsoft/moc-sdk-for-go v0.10.9-alpha.4
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	gonum.org/v1/netlib v0.0.0-20190331212654-76723241ea4e // indirect
	google.golang.org/grpc v1.39.0
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210722164352-7f3ee0f31471
	sigs.k8s.io/cluster-api v0.4.2
	sigs.k8s.io/controller-runtime v0.9.6
	sigs.k8s.io/kind v0.7.1-0.20200303021537-981bd80d3802 // indirect
	sigs.k8s.io/structured-merge-diff/v2 v2.0.1 // indirect
)

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v0.4.2

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible

replace github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2

replace github.com/miekg/dns => github.com/miekg/dns v1.1.25
