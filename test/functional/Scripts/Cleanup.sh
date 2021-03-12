#!/bin/bash
set -e

SCRIPT_DIR=$(dirname "$0")
. $SCRIPT_DIR/Constants.sh

k3d registry delete $REGISTRY_NAME
