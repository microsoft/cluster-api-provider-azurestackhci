# Copyright 2020 The Kubernetes Authors.
# Portions Copyright © Microsoft Corporation.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# If you update this file, please follow
# https://suva.sh/posts/well-documented-makefiles

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL:=/usr/bin/env bash

.DEFAULT_GOAL:=help

## Go environment

# Go version
GO_VERSION := $(shell go env GOVERSION | sed "s/[^[:digit:].-]//g")
ifeq ($(GO_VERSION),)
GO_VERSION := 1.22.4
endif

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

# Go OS and ARCH values
GOOS := $(shell go env GOOS)
ifeq ($(GOOS),)
GOOS := linux
endif

GOARCH := $(shell go env GOARCH)
ifeq ($(GOARCH),)
GOARCH := amd64
endif

# Directories.
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/bin)
BIN_DIR := bin
export PATH := $(TOOLS_BIN_DIR):$(PATH)

# set --output-base used for conversion-gen which needs to be different for in GOPATH and outside GOPATH dev
OUTPUT_BASE := --output-base=$(ROOT_DIR)

# the current cluster API version
CAPI_VERSION := v1.9.4

# Binaries.
GO_INSTALL = ./scripts/go_install.sh

# Binaries.
CLUSTERCTL_VER := $(CAPI_VERSION)
CLUSTERCTL_BIN := clusterctl
CLUSTERCTL := $(TOOLS_BIN_DIR)/$(CLUSTERCTL_BIN)-$(CLUSTERCTL_VER)

CONTROLLER_GEN_VER := v0.17.2
CONTROLLER_GEN_BIN := controller-gen
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/$(CONTROLLER_GEN_BIN)-$(CONTROLLER_GEN_VER)

CONVERSION_GEN_VER := v0.30.0
CONVERSION_GEN_BIN := conversion-gen
CONVERSION_GEN := $(TOOLS_BIN_DIR)/$(CONVERSION_GEN_BIN)-$(CONVERSION_GEN_VER)

ENVSUBST_VER := v2.0.0-20210730161058-179042472c46
ENVSUBST_BIN := envsubst
ENVSUBST := $(TOOLS_BIN_DIR)/$(ENVSUBST_BIN)-$(ENVSUBST_VER)

GOLANGCI_LINT_VER := v1.64.5
GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/$(GOLANGCI_LINT_BIN)-$(GOLANGCI_LINT_VER)

KUSTOMIZE_VER := v5.6.0
KUSTOMIZE_BIN := kustomize
KUSTOMIZE := $(TOOLS_BIN_DIR)/$(KUSTOMIZE_BIN)-$(KUSTOMIZE_VER)

MOCKGEN_VER := v1.6.0
MOCKGEN_BIN := mockgen
MOCKGEN := $(TOOLS_BIN_DIR)/$(MOCKGEN_BIN)-$(MOCKGEN_VER)

RELEASE_NOTES_VER := v0.12.0
RELEASE_NOTES_BIN := release-notes
RELEASE_NOTES := $(TOOLS_BIN_DIR)/$(RELEASE_NOTES_BIN)-$(RELEASE_NOTES_VER)

GO_APIDIFF_VER := v0.8.2
GO_APIDIFF_BIN := go-apidiff
GO_APIDIFF := $(TOOLS_BIN_DIR)/$(GO_APIDIFF_BIN)

GINKGO_VER := v2.19.0
GINKGO_BIN := ginkgo
GINKGO := $(TOOLS_BIN_DIR)/$(GINKGO_BIN)-$(GINKGO_VER)

KUBECTL_VER := v1.31.0
KUBECTL_BIN := kubectl
KUBECTL := $(TOOLS_BIN_DIR)/$(KUBECTL_BIN)-$(KUBECTL_VER)

# ENVTEST is used for running controller tests.
SETUP_ENVTEST_VER :=  v0.0.0-20240522175850-2e9781e9fc60 # updating this to match the version that pulls from controller-tools, details in https://github.com/kubernetes-sigs/cluster-api/pull/10569
SETUP_ENVTEST_BIN := setup-envtest
SETUP_ENVTEST := $(abspath $(TOOLS_BIN_DIR)/$(SETUP_ENVTEST_BIN)-$(SETUP_ENVTEST_VER))

#
# Kubebuilder 
#
export KUBEBUILDER_ENVTEST_KUBERNETES_VERSION ?= 1.26.0
export KUBEBUILDER_CONTROLPLANE_START_TIMEOUT ?= 60s
export KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT ?= 60s

KUBEBUILDER_ASSETS ?= $(shell $(SETUP_ENVTEST) use --use-env -p path $(KUBEBUILDER_ENVTEST_KUBERNETES_VERSION))

# Version
MAJOR_VER ?= 1
MINOR_VER ?= 1
PATCH_VER ?= 3

# Define Docker related variables. Releases should modify and double check these vars.
REGISTRY ?= mocimages.azurecr.io
STAGING_REGISTRY := mocimages.azurecr.io
PROD_REGISTRY := mocimages.azurecr.io
IMAGE_NAME ?= caphcontroller
CONTROLLER_IMG ?= $(REGISTRY)/$(IMAGE_NAME)
TAG := $(MAJOR_VER).$(MINOR_VER).$(PATCH_VER)
ARCH := $(GOARCH)
ALL_ARCH = amd64 arm arm64 ppc64le s390x

# Local repository path for development
export LOCAL_REPOSITORY := $(HOME)/local-repository/infrastructure-azurestackhci/v$(TAG)

# Allow overriding manifest generation destination directory
MANIFEST_ROOT ?= config
CRD_ROOT ?= $(MANIFEST_ROOT)/crd/bases
WEBHOOK_ROOT ?= $(MANIFEST_ROOT)/webhook
RBAC_ROOT ?= $(MANIFEST_ROOT)/rbac

# Allow overriding the imagePullPolicy
PULL_POLICY ?= Always

## --------------------------------------
## Help
## --------------------------------------

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Testing
## --------------------------------------

.PHONY: test
test: generate lint fmt modules vet $(SETUP_ENVTEST) $(GINKGO) ## Run tests
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" \
	ginkgo -r -v -cover -coverprofile cover.out ./...

## --------------------------------------
## Binaries
## --------------------------------------

.PHONY: binaries
binaries: manager ## Builds and installs all binaries

 ## Build manager binary.
.PHONY: manager 
manager: generate lint fmt modules vet ## Build manager binary.
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o bin/manager cmd/manager/main.go

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(CLUSTERCTL): ## Get clusterctl binary.
	mkdir -p $(TOOLS_BIN_DIR)
	rm -f "$(TOOLS_BIN_DIR)/$(CLUSTERCTL_BIN)*"
	curl --retry 3 -fsL https://github.com/kubernetes-sigs/cluster-api/releases/download/$(CLUSTERCTL_VER)/clusterctl-$(GOOS)-$(GOARCH) -o $(CLUSTERCTL)
	ln -sf $(CLUSTERCTL) $(TOOLS_BIN_DIR)/$(CLUSTERCTL_BIN)
	chmod +x $(CLUSTERCTL) $(TOOLS_BIN_DIR)/$(CLUSTERCTL_BIN)

.PHONY: $(CLUSTERCTL_BIN)
$(CLUSTERCTL_BIN): $(CLUSTERCTL)

$(CONTROLLER_GEN): ## Build controller-gen from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) sigs.k8s.io/controller-tools/cmd/controller-gen $(CONTROLLER_GEN_BIN) $(CONTROLLER_GEN_VER)


$(CONVERSION_GEN): ## Build conversion-gen.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) k8s.io/code-generator/cmd/conversion-gen $(CONVERSION_GEN_BIN) $(CONVERSION_GEN_VER)

$(ENVSUBST): ## Build envsubst from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/drone/envsubst/v2/cmd/envsubst $(ENVSUBST_BIN) $(ENVSUBST_VER)

.PHONY: $(ENVSUBST_BIN)
$(ENVSUBST_BIN): $(ENVSUBST)

.PHONY: $(SETUP_ENVTEST_BIN)
$(SETUP_ENVTEST_BIN): $(SETUP_ENVTEST) ## Build a local copy of setup-envtest.

.PHONY: setup-envtest
setup-envtest: $(SETUP_ENVTEST) ## Set up envtest (download kubebuilder assets)
	@echo KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS)

.PHONY: mockgen
mockgen: $(MOCKGEN) ## Generate mocks using mockgen.

$(GOLANGCI_LINT): ## Build golangci-lint from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/golangci/golangci-lint/cmd/golangci-lint $(GOLANGCI_LINT_BIN) $(GOLANGCI_LINT_VER)

$(KUSTOMIZE): ## Build kustomize from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) sigs.k8s.io/kustomize/kustomize/v5 $(KUSTOMIZE_BIN) $(KUSTOMIZE_VER)

$(MOCKGEN): ## Build mockgen from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/golang/mock/mockgen $(MOCKGEN_BIN) $(MOCKGEN_VER)

$(RELEASE_NOTES): ## Build release notes.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) k8s.io/release/cmd/release-notes $(RELEASE_NOTES_BIN) $(RELEASE_NOTES_VER)

$(GO_APIDIFF): ## Build go-apidiff.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/joelanford/go-apidiff $(GO_APIDIFF_BIN) $(GO_APIDIFF_VER)

$(GINKGO): ## Build ginkgo.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/onsi/ginkgo/v2/ginkgo $(GINKGO_BIN) $(GINKGO_VER)

$(KUBECTL): ## Get kubectl
	mkdir -p $(TOOLS_BIN_DIR)
	rm -f "$(TOOLS_BIN_DIR)/$(KUBECTL_BIN)*"
	curl --retry 3 -fsL https://storage.googleapis.com/kubernetes-release/release/$(KUBECTL_VER)/bin/$(GOOS)/$(GOARCH)/kubectl -o $(KUBECTL)
	ln -sf $(KUBECTL) $(TOOLS_BIN_DIR)/$(KUBECTL_BIN)
	chmod +x $(KUBECTL) $(TOOLS_BIN_DIR)/$(KUBECTL_BIN)

.PHONY: $(KUBECTL_BIN)
$(KUBECTL_BIN): $(KUBECTL)

$(SETUP_ENVTEST): # Build setup-envtest.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) sigs.k8s.io/controller-runtime/tools/setup-envtest $(SETUP_ENVTEST_BIN) $(SETUP_ENVTEST_VER)
	@echo "🛠  setup-envtest installed to $(TOOLS_BIN_DIR)/$(ENVTEST_BIN)"

## --------------------------------------
## Linting
## --------------------------------------

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint codebase
	$(GOLANGCI_LINT) run -v

lint-full: $(GOLANGCI_LINT) ## Run slower linters to detect possible issues
	$(GOLANGCI_LINT) run -v --fast=false

## --------------------------------------
## Generate
## --------------------------------------

.PHONY: modules
modules: ## Runs go mod to ensure proper vendoring.
	go mod tidy
	cd $(TOOLS_DIR); go mod tidy

# Run go fmt against code
.PHONY: fmt
fmt: 
	go fmt ./... 

# Run go vet against code
vet:
	go vet ./...

.PHONY: generate
generate: ## Generate code
	$(MAKE) generate-go
	$(MAKE) generate-manifests
	$(MAKE) generate-flavors

.PHONY: generate-go
generate-go: $(CONTROLLER_GEN) $(MOCKGEN) $(CONVERSION_GEN) ## Runs Go related generate targets
	go generate ./...
	$(CONTROLLER_GEN) \
		paths=./api/v1beta1 \
		object:headerFile=./hack/boilerplate/boilerplate.generatego.txt
	
.PHONY: generate-manifests
generate-manifests: $(CONTROLLER_GEN) ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) \
		paths=./api/v1beta1 \
		crd:crdVersions=v1 \
		rbac:roleName=manager-role \
		output:crd:dir=$(CRD_ROOT) \
		output:webhook:dir=$(WEBHOOK_ROOT) \
		webhook
	$(CONTROLLER_GEN) \
		paths=./controllers/... \
		output:rbac:dir=$(RBAC_ROOT) \
		rbac:roleName=manager-role

.PHONY: generate-flavors ## Generate template flavors
generate-flavors:
	./hack/gen-flavors.sh

## --------------------------------------
## Docker
## --------------------------------------

.PHONY: docker-login
docker-login: ## Login docker to a private registry
	@if [ -z "${DOCKER_USERNAME}" ]; then echo "DOCKER_USERNAME is not set"; exit 1; fi
	@if [ -z "${DOCKER_PASSWORD}" ]; then echo "DOCKER_PASSWORD is not set"; exit 1; fi
	docker login $(STAGING_REGISTRY) -u ${DOCKER_USERNAME} -p ${DOCKER_PASSWORD}

.PHONY: docker-build-img
docker-build-img: manager
	#docker build --pull --build-arg ARCH=$(ARCH) . -t $(CONTROLLER_IMG)-$(ARCH):$(TAG)
	docker build --pull --build-arg ARCH=$(ARCH) . -t $(CONTROLLER_IMG):$(TAG)

.PHONY: docker-build
docker-build: docker-build-img ## Build the docker image for controller-manager
	#MANIFEST_IMG=$(CONTROLLER_IMG)-$(ARCH) MANIFEST_TAG=$(TAG) $(MAKE) set-manifest-image
	MANIFEST_IMG=$(CONTROLLER_IMG) MANIFEST_TAG=$(TAG) $(MAKE) set-manifest-image
	$(MAKE) set-manifest-pull-policy

.PHONY: docker-push
docker-push: ## Push the docker image
	docker push $(CONTROLLER_IMG):$(TAG)

## --------------------------------------
## Docker — All ARCH
## --------------------------------------

.PHONY: docker-build-all ## Build all the architecture docker images
docker-build-all: $(addprefix docker-build-,$(ALL_ARCH))

docker-build-%:
	$(MAKE) ARCH=$* docker-build

.PHONY: docker-push-all ## Push all the architecture docker images
docker-push-all: $(addprefix docker-push-,$(ALL_ARCH))
	$(MAKE) docker-push-manifest

docker-push-%:
	$(MAKE) ARCH=$* docker-push

.PHONY: docker-push-manifest
docker-push-manifest: ## Push the fat manifest docker image.
	## Minimum docker version 18.06.0 is required for creating and pushing manifest images.
	docker manifest create --amend $(CONTROLLER_IMG):$(TAG) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(CONTROLLER_IMG)\-&:$(TAG)~g")
	@for arch in $(ALL_ARCH); do docker manifest annotate --arch $${arch} ${CONTROLLER_IMG}:${TAG} ${CONTROLLER_IMG}-$${arch}:${TAG}; done
	docker manifest push --purge ${CONTROLLER_IMG}:${TAG}
	MANIFEST_IMG=$(CONTROLLER_IMG) MANIFEST_TAG=$(TAG) $(MAKE) set-manifest-image
	$(MAKE) set-manifest-pull-policy

.PHONY: set-manifest-image
set-manifest-image:
	$(info Updating kustomize image patch file for manager resource)
	sed -i'' -e 's@image: .*@image: '"${MANIFEST_IMG}:$(MANIFEST_TAG)"'@' ./config/default/manager_image_patch.yaml

.PHONY: set-manifest-pull-policy
set-manifest-pull-policy:
	$(info Updating kustomize pull policy file for manager resource)
	sed -i'' -e 's@imagePullPolicy: .*@imagePullPolicy: '"$(PULL_POLICY)"'@' ./config/default/manager_pull_policy.yaml

## --------------------------------------
## Release
## --------------------------------------

#RELEASE_TAG := $(shell git describe --abbrev=0 2>/dev/null)
RELEASE_TAG ?= $(TAG)
RELEASE_DIR := out

$(RELEASE_DIR):
	mkdir -p $(RELEASE_DIR)/

.PHONY: release
release: clean-release  ## Builds and push container images using the latest git tag for the commit.
	@if [ -z "${RELEASE_TAG}" ]; then echo "RELEASE_TAG is not set"; exit 1; fi
	#@if ! [ -z "$$(git status --porcelain)" ]; then echo "Your local git repository contains uncommitted changes, use git clean before proceeding."; exit 1; fi
	#git checkout "${RELEASE_TAG}"
	# Set the manifest image to the production bucket.
	$(MAKE) set-manifest-image MANIFEST_IMG=$(PROD_REGISTRY)/$(IMAGE_NAME) MANIFEST_TAG=$(RELEASE_TAG)
	#$(MAKE) set-manifest-pull-policy PULL_POLICY=IfNotPresent
	$(MAKE) release-manifests

.PHONY: release-manifests
release-manifests: $(KUSTOMIZE) $(RELEASE_DIR) ## Builds the manifests to publish with a release
	$(KUSTOMIZE) build config/default > $(RELEASE_DIR)/infrastructure-components.yaml

.PHONY: release-binary
release-binary: $(RELEASE_DIR)
	docker run \
		--rm \
		-e CGO_ENABLED=0 \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-v "$$(pwd):/workspace" \
		-w /workspace \
		golang:$(GO_VERSION) \
		go build -a -ldflags '-extldflags "-static"' \
		-o $(RELEASE_DIR)/$(notdir $(RELEASE_BINARY))-$(GOOS)-$(GOARCH) $(RELEASE_BINARY)

.PHONY: release-staging
release-staging: ## Builds and push container images to the staging bucket.
	REGISTRY=$(STAGING_REGISTRY) $(MAKE) docker-build-all docker-push-all release-alias-tag

.PHONY: release-pipelines
release-pipelines: $(KUSTOMIZE) $(RELEASE_DIR) ## Builds the manifests to publish with a release
	$(MAKE) set-manifest-image MANIFEST_IMG=$(PROD_REGISTRY)/$(IMAGE_NAME) MANIFEST_TAG=$(RELEASE_TAG)
	$(KUSTOMIZE) build config/manager > $(RELEASE_DIR)/deployment.yaml

RELEASE_ALIAS_TAG=$(PULL_BASE_REF)

.PHONY: release-alias-tag
release-alias-tag: # Adds the tag to the last build tag.
	gcloud container images add-tag $(CONTROLLER_IMG):$(TAG) $(CONTROLLER_IMG):$(RELEASE_ALIAS_TAG)

.PHONY: release-notes
release-notes: $(RELEASE_NOTES)
	$(RELEASE_NOTES)

## --------------------------------------
## Development
## --------------------------------------

.PHONY: dev-release
dev-release:
	#$(MAKE) generate
	$(MAKE) docker-build
	$(MAKE) docker-push
	@if [ "${DOCKER_USERNAME}" ]; then ./hack/for-pipeline.sh; fi
	$(MAKE) release

.PHONY: create-local-provider-repository
create-local-provider-repository: $(ENVSUBST) generate-flavors
	# Create the required directories
	mkdir -p $(LOCAL_REPOSITORY)/
	mkdir -p $(HOME)/.cluster-api/
	# Prepare configuration file for clusterctl
	cat hack/clusterctl.yaml | $(ENVSUBST) > $(HOME)/.cluster-api/clusterctl.yaml
	# Prepare metadata yaml for clusterctl
	sed -i'' -e 's@major: .*@major: '"$(MAJOR_VER)"'@' ./metadata.yaml
	sed -i'' -e 's@minor: .*@minor: '"$(MINOR_VER)"'@' ./metadata.yaml
	# Populate the local repository
	cp metadata.yaml $(LOCAL_REPOSITORY)
	cp out/infrastructure-components.yaml $(LOCAL_REPOSITORY)
	cp templates/cluster-template.yaml $(LOCAL_REPOSITORY)

.PHONY: create-cluster
create-cluster:
	./hack/create-dev-cluster.sh

.PHONY: deployment
deployment: dev-release create-cluster  ## Build and deploy caph in a kind management cluster.

## --------------------------------------
## Development: Local/private registry
## --------------------------------------

# Create patch files to override container image registry.
.PHONY: local-dev-set-manifest-image
local-dev-set-manifest-image:
	cp ./hack/config/manager_image_patch_template.yaml ./hack/config/manager_image_patch.yaml
	sed -i'' -e 's@value:.*@value: '"${CONTROLLER_IMG}:$(TAG)"'@' ./hack/config/manager_image_patch.yaml

# Build config.
.PHONY: local-dev-release-manifests
local-dev-release-manifests: $(KUSTOMIZE) $(RELEASE_DIR)
	$(KUSTOMIZE) build ./hack/config > $(RELEASE_DIR)/infrastructure-components.yaml

.PHONY: local-dev-release
local-dev-release:
	$(MAKE) docker-build-img
	$(MAKE) docker-push
	$(MAKE) local-dev-set-manifest-image
	$(MAKE) local-dev-release-manifests

## --------------------------------------
## Kind
## --------------------------------------

.PHONY: kind-create
kind-create: ## create caph kind cluster if needed
	kind create cluster --name=caph-$(USER)

.PHONY: kind-reset
kind-reset: ## Attempts to delete a caph kind cluster
	kind delete cluster --name=caph-$(USER) || true

## --------------------------------------
## Cleanup / Verification
## --------------------------------------

.PHONY: clean
clean: ## Remove all generated files
	$(MAKE) clean-bin
	$(MAKE) clean-temporary

.PHONY: clean-bin
clean-bin: ## Remove all generated binaries
	rm -rf bin
	rm -rf hack/tools/bin

.PHONY: clean-temporary
clean-temporary: ## Remove all temporary files and folders
	rm -f minikube.kubeconfig
	rm -f kubeconfig

.PHONY: clean-release
clean-release: ## Remove the release folder
	rm -rf $(RELEASE_DIR)

.PHONY: verify
verify: verify-boilerplate verify-modules verify-gen

.PHONY: verify-boilerplate
verify-boilerplate:
	./hack/verify-boilerplate.sh

.PHONY: verify-modules
verify-modules: modules
	@if !(git diff --quiet HEAD -- go.sum go.mod hack/tools/go.mod hack/tools/go.sum); then \
		echo "go module files are out of date"; exit 1; \
	fi

.PHONY: verify-gen
verify-gen: generate
	@if !(git diff --quiet HEAD); then \
		echo "generated files are out of date, run make generate"; exit 1; \
	fi