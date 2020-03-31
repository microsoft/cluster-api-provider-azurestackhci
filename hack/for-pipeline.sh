#!/bin/bash
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
      - image: mocimages.azurecr.io/caphcontroller:latest
        name: manager
        args:
        - "--v=6"
      imagePullSecrets:
      - name: acr-creds
EOF

cat > config/manager/kustomization.yaml <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- manager.yaml
- credentials.yaml
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

rsync -Rr ./ cluster-api-provider-azhci-0.0.x
tar -czvf cluster-api-provider-azhci.tar.gz cluster-api-provider-azhci-0.0.x
cp cluster-api-provider-azhci.tar.gz ./bin
rm -rf cluster-api-provider-azhci-0.0.x