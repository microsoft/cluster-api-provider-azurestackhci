#!/usr/bin/env bash

# Copyright 2014 The Kubernetes Authors.
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
set -o pipefail
#set -o verbose

root=$(dirname "${BASH_SOURCE[0]}")/..
templates_dir="${root}/templates"
flavors_dir="${root}/templates/flavors/"

kustomize build ${flavors_dir}/mgmt > ${templates_dir}/cluster-template-mgmt.yaml
kustomize build ${flavors_dir}/default > ${templates_dir}/cluster-template.yaml
