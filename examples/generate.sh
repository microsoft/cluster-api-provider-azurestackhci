#!/bin/bash
# Copyright 2019 The Kubernetes Authors.
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

# Directories.
SOURCE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
OUTPUT_DIR=${OUTPUT_DIR:-${SOURCE_DIR}/_out}

# Binaries
ENVSUBST=${ENVSUBST:-envsubst}
command -v "${ENVSUBST}" >/dev/null 2>&1 || echo -v "Cannot find ${ENVSUBST} in path."

RANDOM_STRING=$(date | md5sum | head -c8)

# Cluster.
export TARGET_CLUSTER_NAME="${TARGET_CLUSTER_NAME:-azurestackhci-${RANDOM_STRING}}"
export MANAGEMENT_CLUSTER_NAME="${MANAGEMENT_CLUSTER_NAME:-${TARGET_CLUSTER_NAME}-mgmt}"
export VNET_NAME="${VNET_NAME:-External}"
export KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.16.1}"
export KUBERNETES_SEMVER="${KUBERNETES_VERSION#v}"
export POD_CIDR="${POD_CIDR:-192.168.0.0/16}"
export TARGET_CLUSTER_LB_NAME=$TARGET_CLUSTER_NAME"-load-balancer"
export MANAGEMENT_CLUSTER_LB_NAME=$MANAGEMENT_CLUSTER_NAME"-load-balancer"
export TARGET_CLUSTER_BACKEND_POOL_NAME=$TARGET_CLUSTER_NAME"-backend-pool"
export MANAGEMENT_CLUSTER_BACKEND_POOL_NAME=$MANAGEMENT_CLUSTER_NAME"-backend-pool"
export MANAGEMENT_CLUSTER_RESOURCE_GROUP="${MANAGEMENT_CLUSTER_GROUP_NAME:-${MANAGEMENT_CLUSTER_NAME}}"
export TARGET_CLUSTER_RESOURCE_GROUP="${TARGET_CLUSTER_GROUP_NAME:-${MANAGEMENT_CLUSTER_RESOURCE_GROUP}-target}"

# User.
export CAPH_USER="${CAPH_USER:-clouduser}"

# Debug Mode
export WSSD_DEBUG_MODE="${WSSD_DEBUG_MODE:-off}"

# Disk
export CAPH_DISK_NAME="${CAPH_DISK_NAME:-linux}"

# Machine settings.
export CONTROL_PLANE_REPLICAS="${CONTROL_PLANE_REPLICAS:-1}"
export MACHINE_REPLICAS="${MACHINE_REPLICAS:-2}"
export CONTROL_PLANE_MACHINE_TYPE="${CONTROL_PLANE_MACHINE_TYPE:-Default}"
export NODE_MACHINE_TYPE="${NODE_MACHINE_TYPE:-Default}"

# Outputs.
COMPONENTS_CLUSTER_API_GENERATED_FILE=${SOURCE_DIR}/provider-components/provider-components-cluster-api.yaml
COMPONENTS_KUBEADM_GENERATED_FILE=${SOURCE_DIR}/provider-components/provider-components-kubeadm.yaml
COMPONENTS_CAPH_GENERATED_FILE=${SOURCE_DIR}/provider-components/provider-components-azurestackhci.yaml

PROVIDER_COMPONENTS_GENERATED_FILE=${OUTPUT_DIR}/provider-components.yaml

MANAGEMENT_CLUSTER_GENERATED_FILE=${OUTPUT_DIR}/mgmt-cluster.yaml
MANAGEMENT_CONTROLPLANE_GENERATED_FILE=${OUTPUT_DIR}/mgmt-controlplane.yaml
MANAGEMENT_LOADBALANCER_GENERATED_FILE=${OUTPUT_DIR}/mgmt-loadbalancer.yaml

TARGET_CLUSTER_GENERATED_FILE=${OUTPUT_DIR}/target-cluster.yaml
TARGET_CONTROLPLANE_GENERATED_FILE=${OUTPUT_DIR}/target-controlplane.yaml
TARGET_LOADBALANCER_GENERATED_FILE=${OUTPUT_DIR}/target-loadbalancer.yaml

MACHINEDEPLOYMENT_GENERATED_FILE=${OUTPUT_DIR}/target-machinedeployment.yaml

# Overwrite flag.
OVERWRITE=0

SCRIPT=$(basename "$0")
while test $# -gt 0; do
        case "$1" in
          -h|--help)
            echo "$SCRIPT - generates input yaml files for Cluster API (CAPH)"
            echo " "
            echo "$SCRIPT [options]"
            echo " "
            echo "options:"
            echo "-h, --help                show brief help"
            echo "-f, --force-overwrite     if file to be generated already exists, force script to overwrite it"
            exit 0
            ;;
          -f)
            OVERWRITE=1
            shift
            ;;
          --force-overwrite)
            OVERWRITE=1
            shift
            ;;
          *)
            break
            ;;
        esac
done

if [ $OVERWRITE -ne 1 ] && [ -d "$OUTPUT_DIR" ]; then
  echo "ERR: Folder ${OUTPUT_DIR} already exists. Delete it manually before running this script."
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"

# Verify the required Environment Variables are present.
: "${CLOUDAGENT_FQDN:?Environment variable empty or not defined.}"
: "${SSH_PUBLIC_KEY:?Environment variable empty or not defined.}"

# If requested, adjust control plane kustomization to point to the HA (3 node) yaml.
# This is temporary until we move to alpha3 and truly support user specified replica counts for the control plane.
if [ ${CONTROL_PLANE_REPLICAS} -gt 1 ]; then
    sed  -ri 's/- controlplane.yaml/- controlplane-ha.yaml/' "${SOURCE_DIR}/controlplane/kustomization.yaml"
else
  sed  -ri 's/- controlplane-ha.yaml/- controlplane.yaml/' "${SOURCE_DIR}/controlplane/kustomization.yaml"
fi

# Cloudagent FQDN is passed through to the manager pod via secret
export CLOUDAGENT_FQDN_B64="$(echo -n "$CLOUDAGENT_FQDN" | base64 | tr -d '\n')"
export WSSD_DEBUG_MODE_B64="$(echo -n "$WSSD_DEBUG_MODE" | base64 | tr -d '\n')"

# Prepare environment for generation of management cluster yamls
export CLUSTER_NAME="${MANAGEMENT_CLUSTER_NAME}"
export LOAD_BALANCER_NAME=${MANAGEMENT_CLUSTER_LB_NAME}
export BACKEND_POOL_NAME=${MANAGEMENT_CLUSTER_BACKEND_POOL_NAME}
export CLUSTER_RESOURCE_GROUP=${MANAGEMENT_CLUSTER_RESOURCE_GROUP}

# Generate management cluster resources.
kustomize build "${SOURCE_DIR}/cluster" | envsubst > "${MANAGEMENT_CLUSTER_GENERATED_FILE}"
echo "Generated ${MANAGEMENT_CLUSTER_GENERATED_FILE}"

# Generate management controlplane resources.
kustomize build "${SOURCE_DIR}/controlplane" | envsubst > "${MANAGEMENT_CONTROLPLANE_GENERATED_FILE}"
echo "Generated ${MANAGEMENT_CONTROLPLANE_GENERATED_FILE}"

# Generate loadbalancer resources.
kustomize build "${SOURCE_DIR}/loadbalancer" | envsubst >> "${MANAGEMENT_LOADBALANCER_GENERATED_FILE}"
echo "Generated ${MANAGEMENT_LOADBALANCER_GENERATED_FILE}"

# Prepare environment for generation of target cluster yamls
# If target cluster LB is not specified (e.g. converged cluster) then management LB is used.
export CLUSTER_NAME="${TARGET_CLUSTER_NAME}"
export LOAD_BALANCER_NAME=${TARGET_CLUSTER_LB_NAME}
export BACKEND_POOL_NAME=${TARGET_CLUSTER_BACKEND_POOL_NAME}
export CLUSTER_RESOURCE_GROUP=${TARGET_CLUSTER_RESOURCE_GROUP}

# Generate target cluster resources.
kustomize build "${SOURCE_DIR}/cluster" | envsubst > "${TARGET_CLUSTER_GENERATED_FILE}"
echo "Generated ${TARGET_CLUSTER_GENERATED_FILE}"

# Generate target controlplane resources.
kustomize build "${SOURCE_DIR}/controlplane" | envsubst > "${TARGET_CONTROLPLANE_GENERATED_FILE}"
echo "Generated ${TARGET_CONTROLPLANE_GENERATED_FILE}"

# Generate loadbalancer resources.
kustomize build "${SOURCE_DIR}/loadbalancer" | envsubst >> "${TARGET_LOADBALANCER_GENERATED_FILE}"
echo "Generated ${TARGET_LOADBALANCER_GENERATED_FILE}"

# Generate machinedeployment resources.
kustomize build "${SOURCE_DIR}/machinedeployment" | envsubst >> "${MACHINEDEPLOYMENT_GENERATED_FILE}"
echo "Generated ${MACHINEDEPLOYMENT_GENERATED_FILE}"

# Generate Cluster API provider components file.
curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v0.2.4/cluster-api-components.yaml > "${COMPONENTS_CLUSTER_API_GENERATED_FILE}"
echo "Downloaded ${COMPONENTS_CLUSTER_API_GENERATED_FILE}"

# Generate Kubeadm Bootstrap Provider components file.
curl -L https://github.com/kubernetes-sigs/cluster-api-bootstrap-provider-kubeadm/releases/download/v0.1.2/bootstrap-components.yaml > "${COMPONENTS_KUBEADM_GENERATED_FILE}"
echo "Downloaded ${COMPONENTS_KUBEADM_GENERATED_FILE}"

# Generate AzureStackHCI Infrastructure Provider components file.
kustomize build "${SOURCE_DIR}/../config/default" | envsubst > "${COMPONENTS_CAPH_GENERATED_FILE}"
echo "Generated ${COMPONENTS_CAPH_GENERATED_FILE}"

# Generate a single provider components file.
kustomize build "${SOURCE_DIR}/provider-components" | envsubst > "${PROVIDER_COMPONENTS_GENERATED_FILE}"
echo "Generated ${PROVIDER_COMPONENTS_GENERATED_FILE}"
