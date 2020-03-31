
# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL:=/usr/bin/env bash

.DEFAULT_GOAL:=help

# Use GOPROXY environment variable if set
GOPROXY := $(shell go env GOPROXY)
ifeq ($(GOPROXY),)
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

# Active module mode, as we use go modules to manage dependencies
export GO111MODULE=on

# Private repo workaround
export GOPRIVATE = github.com/microsoft

# Directories.
BIN_DIR := bin

# Binaries.
CLUSTERCTL := $(BIN_DIR)/clusterctl
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/controller-gen
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
MOCKGEN := $(TOOLS_BIN_DIR)/mockgen
CONVERSION_GEN := $(TOOLS_BIN_DIR)/conversion-gen

# Image URL to use all building/pushing image targets
IMG ?= nwoodmsft/controller:0.14

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## --------------------------------------
## Help
## --------------------------------------

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Binaries
## --------------------------------------

all: manager

.PHONY: manager
manager: generate fmt vet ## Build manager binary.
#	go build -o bin/manager main.go
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o bin/manager cmd/manager/main.go

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(CLUSTERCTL): go.mod ## Build clusterctl binary.
	go build -o $(BIN_DIR)/clusterctl sigs.k8s.io/cluster-api/cmd/clusterctl

## --------------------------------------
## Generate
## --------------------------------------

.PHONY: generate-examples
generate-examples: clean-examples ## Generate examples configurations to run a cluster.
	./examples/generate.sh


# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

## --------------------------------------
## Docker Image
## --------------------------------------

# Build the docker image
docker-build: manager ## Build docker image
	docker build -t ${IMG} .

# Push the docker image
docker-push: ## Push docker image
	docker push ${IMG}

## --------------------------------------
## Development
## --------------------------------------

.PHONY: kind-reset
kind-reset: ## Destroys the "clusterapi" kind cluster.
	kind delete cluster --name=clusterapi || true

.PHONY: create-cluster
create-cluster: kind-reset $(CLUSTERCTL) ## Create a development Kubernetes cluster in a KIND management cluster.
	# Create KIND cluster
	kind create cluster --name=clusterapi
	# Apply provider-components.
	kubectl \
		--kubeconfig=$$(kind get kubeconfig-path --name="clusterapi") \
		create -f examples/_out/provider-components.yaml
	# Create Cluster.
	kubectl \
		--kubeconfig=$$(kind get kubeconfig-path --name="clusterapi") \
		create -f examples/_out/cluster.yaml
	# Create control plane machine.
	kubectl \
		--kubeconfig=$$(kind get kubeconfig-path --name="clusterapi") \
		create -f examples/_out/controlplane.yaml
	# Get KubeConfig using clusterctl.
	# $(CLUSTERCTL) \
	# 	alpha phases get-kubeconfig -v=4 \
	# 	--kubeconfig=$$(kind get kubeconfig-path --name="clusterapi") \
	# 	--namespace=default \
	# 	--cluster-name=$(CLUSTER_NAME)
	# Create a worker node with MachineDeployment.
	kubectl \
		--kubeconfig=$$(kind get kubeconfig-path --name="clusterapi") \
		create -f examples/_out/machinedeployment.yaml

## --------------------------------------
## Cleanup
## --------------------------------------

.PHONY: clean
clean: ## Remove all generated files
	$(MAKE) clean-bin
	$(MAKE) clean-temporary
	$(MAKE) clean-examples

.PHONY: clean-bin
clean-bin:
	rm -rf bin

.PHONY: clean-temporary
clean-temporary:
	rm -f kubeconfig

.PHONY: clean-examples
clean-examples:
	rm -rf examples/_out/
	rm -f examples/provider-components/provider-components-*.yaml
