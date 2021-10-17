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

DOCKER_DELIM=":"
DOCKER_AUTH=$DOCKER_USERNAME$DOCKER_DELIM$DOCKER_PASSWORD
DOCKER_AUTH_B64="$(echo -n "$DOCKER_AUTH" | base64 | tr -d '\n')"

DOCKER_SECRET="{\"auths\":{\"https:\/\/mocimages.azurecr.io\":{\"username\":\"${DOCKER_USERNAME}\",\"password\":\"${DOCKER_PASSWORD}\",\"auth\":\"${DOCKER_AUTH_B64}\"}}}"
DOCKER_SECRET_B64="$(echo -n "$DOCKER_SECRET" | base64 | tr -d '\n')"

cat > config/default/manager_image_patch.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
spec:
  template:
    spec:
      containers:
      - image: mocimages.azurecr.io/caphcontroller-staging:latest
        name: manager
      imagePullSecrets:
      - name: acr-creds
EOF

cat > config/manager/kustomization.yaml <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - manager.yaml
  - dockercredentials.yaml
EOF

cat > config/manager/dockercredentials.yaml <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: acr-creds
  namespace: default
data:
  .dockerconfigjson: ${DOCKER_SECRET_B64}
type: kubernetes.io/dockerconfigjson
EOF
