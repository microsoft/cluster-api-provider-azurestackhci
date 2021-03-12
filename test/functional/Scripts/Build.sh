#!/bin/bash
set -e

SCRIPT_DIR=$(dirname "$0")
. $SCRIPT_DIR/Constants.sh

# Check if the container registry already exists.
if ! k3d registry get $REGISTRY_NAME ; then
    # Create container registry.
    k3d registry create $BASE_REGISTRY_NAME --port $REGISTRY_PORT
fi

ROOT_DIR=$SCRIPT_DIR/../../../

# Set a custom docker container registry URL that the Makefile will use.
export REGISTRY=$REGISTRY

# Build the container image and push it to our local registry.
make -C $ROOT_DIR docker-build-img
make -C $ROOT_DIR docker-push

# Build CAPH K8s definitions
make -C $ROOT_DIR dev-manifests

# Publish CAPH build to local CAPI override directory.
make -C $ROOT_DIR create-local-provider-repository
