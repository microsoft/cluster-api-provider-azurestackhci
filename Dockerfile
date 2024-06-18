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

# First stage: Use an image that includes shell and utilities
FROM mcr.microsoft.com/cbl-mariner/base/core:2.0 AS builder

# Set the working directory and copy the 'manager' binary
WORKDIR /
COPY bin/manager .

# Set the executable permission on the 'manager' binary
RUN chmod +x /manager

# Use distroless as minimal base image to package the manager binary
FROM  mcr.microsoft.com/cbl-mariner/distroless/debug:2.0
WORKDIR /

# Copy the 'manager' binary from the first stage with the correct permissions
COPY --from=builder --chown=65532:65532 /manager .

# Set the user ID for the container process to 65532 (nonroot user)
USER 65532:65532

# Specify the command to run when the container starts

ENTRYPOINT ["/manager"]