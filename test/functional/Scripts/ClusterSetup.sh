#!/bin/bash
set -e

SCRIPT_DIR=$(dirname "$0")
. $SCRIPT_DIR/Constants.sh

# Create management cluster.
k3d cluster create $CLUSTER_NAME --registry-use $REGISTRY -v /dev/mapper:/dev/mapper

# Print gateway IP address.
# This is useful when manually testing, to know which IP address the cloud agent should listen on.
GATEWAY_IP_ADDR=$(docker network inspect $FULL_CLUSTER_NAME | grep Gateway | grep -o '[0-9,\.]*' --color=never)
echo "Cluster gateway: $GATEWAY_IP_ADDR"
