#!/bin/bash
set -e

SCRIPT_DIR=$(dirname "$0")
. $SCRIPT_DIR/Constants.sh

k3d cluster delete $CLUSTER_NAME
