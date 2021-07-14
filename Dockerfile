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

# Build the manager binary
#FROM golang:1.12.9 as builder
# WORKDIR /workspace

# Run this with docker build --build_arg $(go env GOPROXY) to override the goproxy
#ARG goproxy=https://proxy.golang.org
#ENV GOPROXY=$goproxy

#ENV GOPRIVATE="github.com/microsoft"
#RUN go env GOPRIVATE=github.com/microsoft

# Copy the Go Modules manifests
#COPY go.mod go.mod
#COPY go.sum go.sum
# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
#RUN go mod download

# Copy the sources
#COPY ./ ./
#COPY ./bin/manager ./

# Build
#ARG ARCH
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} GO111MODULE=on  \
#    go build -a -ldflags '-extldflags "-static"' \
#    -o manager .


# NOTE: Approach above is not used while we still have a couple of private git repo's.
# Can be uncommented later.

# Copy the controller-manager into a thin image
#FROM alpine:3.11
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY bin/staging/manager ./
USER nobody
ENTRYPOINT ["/manager"]