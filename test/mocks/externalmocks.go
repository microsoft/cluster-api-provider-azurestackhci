package test

// controller-runtime k8s client

//go:generate mockgen -destination=k8s/client/client.go sigs.k8s.io/controller-runtime/pkg/client Client
