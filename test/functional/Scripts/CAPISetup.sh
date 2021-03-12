#!/bin/bash
set -e

SCRIPT_DIR=$(dirname "$0")
. $SCRIPT_DIR/Constants.sh

# Set cloud agent URL as the host machine.
# Note: k3d inserts the host's gateway address as "host.k3d.internal" into K8s's DNS.
AZURESTACKHCI_CLOUDAGENT_FQDN="host.k3d.internal"

# Disable TLS on cloud agent.
WSSD_DEBUG_MODE="on"

AZURESTACKHCI_BINARY_LOCATION="TODO"

function to_base64() {
    echo "$(echo -n "$1" | base64 | tr -d '\n')"
}
 
export AZURESTACKHCI_CLOUDAGENT_FQDN_B64="$(to_base64 $AZURESTACKHCI_CLOUDAGENT_FQDN)"
export WSSD_DEBUG_MODE_B64="$(to_base64 $WSSD_DEBUG_MODE)"
export AZURESTACKHCI_BINARY_LOCATION_B64="$(to_base64 $AZURESTACKHCI_BINARY_LOCATION)"

# Initialize CAPI and CAPH.
clusterctl init --infrastructure azurestackhci --bootstrap kubeadm:v0.3.5 --control-plane kubeadm:v0.3.5 --core cluster-api:v0.3.5

echo "Wait for CAPI/CAPH to be ready..."
kubectl wait --for=condition=Ready --timeout=5m -n capi-webhook-system pod -l cluster.x-k8s.io/provider=cluster-api
kubectl wait --for=condition=Ready --timeout=5m -n capi-webhook-system pod -l cluster.x-k8s.io/provider=bootstrap-kubeadm
kubectl wait --for=condition=Ready --timeout=5m -n capi-webhook-system pod -l cluster.x-k8s.io/provider=control-plane-kubeadm
kubectl wait --for=condition=Ready --timeout=5m -n capi-webhook-system pod -l cluster.x-k8s.io/provider=infrastructure-azurestackhci
