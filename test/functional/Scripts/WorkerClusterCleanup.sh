#!/bin/bash
set -e

SCRIPT_DIR=$(dirname "$0")
. $SCRIPT_DIR/Constants.sh

kubectl delete -f $SCRIPT_DIR/tmp/testcluster.yaml
