#!/bin/bash
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

set -o errexit
set -o nounset
set -o pipefail

# Verify the required Environment Variables are present.
: "${AZURESTACKHCI_CLOUDAGENT_FQDN:?Environment variable empty or not defined.}"
: "${AZURESTACKHCI_BINARY_LOCATION:?Environment variable empty or not defined.}"

# Cluster settings.
export AZURESTACKHCI_CLUSTER_RESOURCE_GROUP="${AZURESTACKHCI_CLUSTER_RESOURCE_GROUP:-nickgroup}"
export CLUSTER_NAME="${CLUSTER_NAME:-${AZURESTACKHCI_CLUSTER_RESOURCE_GROUP}-caph-test}"
export KUBERNETES_VERSION=${KUBERNETES_VERSION:-v1.16.2}

# AzureStackHCI settings.
export AZURESTACKHCI_CLOUDAGENT_FQDN_B64="$(echo -n "$AZURESTACKHCI_CLOUDAGENT_FQDN" | base64 | tr -d '\n')"
export WSSD_DEBUG_MODE_B64="$(echo -n "on" | base64 | tr -d '\n')"

# Temp until lbagent work is complete.
export AZURESTACKHCI_BINARY_LOCATION_B64="$(echo -n "$AZURESTACKHCI_BINARY_LOCATION" | base64 | tr -d '\n')"

# Machine settings.
export CONTROL_PLANE_MACHINE_COUNT=${CONTROL_PLANE_MACHINE_COUNT:-1}
export WORKER_MACHINE_COUNT=${WORKER_MACHINE_COUNT:-2}
export AZURESTACKHCI_CONTROL_PLANE_MACHINE_TYPE="${AZURESTACKHCI_CONTROL_PLANE_MACHINE_TYPE:-Standard_K8S_v1}"
export AZURESTACKHCI_NODE_MACHINE_TYPE="${AZURESTACKHCI_NODE_MACHINE_TYPE:-Standard_K8S_v1}"
export AZURESTACKHCI_LOAD_BALANCER_MACHINE_TYPE="${AZURESTACKHCI_LOAD_BALANCER_MACHINE_TYPE:-Default}"
export AZURESTACKHCI_LOAD_BALANCER_NAME="${AZURESTACKHCI_LOAD_BALANCER_NAME:-${CLUSTER_NAME}-load-balancer}"
export AZURESTACKHCI_POD_CIDR="${AZURESTACKHCI_POD_CIDR:-10.244.0.0/16}"
export AZURESTACKHCI_VNET_NAME="${AZURESTACKHCI_VNET_NAME:-External}"
export AZURESTACKHCI_BACKEND_POOL_NAME="${AZURESTACKHCI_BACKEND_POOL_NAME:-${CLUSTER_NAME}-backend-pool}"

#Generate SSH key.
SSH_KEY_FILE=${SSH_KEY_FILE:-""}
if ! [ -n "$SSH_KEY_FILE" ]; then
    SSH_KEY_FILE=.sshkey
    rm -f "${SSH_KEY_FILE}" 2>/dev/null
    ssh-keygen -t rsa -b 2048 -f "${SSH_KEY_FILE}" -N '' 1>/dev/null
    echo "Machine SSH key generated in ${SSH_KEY_FILE}"
fi
export AZURESTACKHCI_SSH_PUBLIC_KEY=$(cat "${SSH_KEY_FILE}.pub" | base64 | tr -d '\r\n')

# Helpers
GREEN='\e[92m'
NC='\033[0m'

function print_banner ()
{
    printf "\n${GREEN}====== $1 ======${NC}\n"
}

# Main Steps
print_banner "Create Local Provider Repository"
make create-local-provider-repository

print_banner "Make Clean"
make clean

print_banner "Create KIND Cluster"
make kind-reset
make kind-create

print_banner "ClusterCTL Init"
clusterctl init --infrastructure azurestackhci

print_banner "Wait For CAPI Pods To Be Ready"
kubectl wait --for=condition=Ready --timeout=5m -n capi-system pod -l cluster.x-k8s.io/provider=cluster-api
kubectl wait --for=condition=Ready --timeout=5m -n capi-kubeadm-bootstrap-system pod -l cluster.x-k8s.io/provider=bootstrap-kubeadm
kubectl wait --for=condition=Ready --timeout=5m -n capi-kubeadm-control-plane-system pod -l cluster.x-k8s.io/provider=control-plane-kubeadm
kubectl wait --for=condition=Ready --timeout=5m -n capi-webhook-system pod -l cluster.x-k8s.io/provider=cluster-api
kubectl wait --for=condition=Ready --timeout=5m -n capi-webhook-system pod -l cluster.x-k8s.io/provider=bootstrap-kubeadm
kubectl wait --for=condition=Ready --timeout=5m -n capi-webhook-system pod -l cluster.x-k8s.io/provider=control-plane-kubeadm

print_banner "Wait For CAPH Pods To Be Ready"
kubectl wait --for=condition=Ready --timeout=5m -n caph-system pod -l cluster.x-k8s.io/provider=infrastructure-azurestackhci
kubectl wait --for=condition=Ready --timeout=5m -n capi-webhook-system pod -l cluster.x-k8s.io/provider=infrastructure-azurestackhci

print_banner "ClusterCTL Config Cluster"
clusterctl config cluster ${CLUSTER_NAME} --kubernetes-version ${KUBERNETES_VERSION} --control-plane-machine-count=${CONTROL_PLANE_MACHINE_COUNT} --worker-machine-count=${WORKER_MACHINE_COUNT} | kubectl apply -f -
