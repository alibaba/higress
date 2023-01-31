# Copyright (c) 2022 Alibaba Group Holding Ltd.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#      http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/usr/bin/env bash

set -euo pipefail

# Setup default values
CLUSTER_NAME=${CLUSTER_NAME:-"higress"}
METALLB_VERSION=${METALLB_VERSION:-"v0.13.7"}
KIND_NODE_TAG=${KIND_NODE_TAG:-"v1.25.3"}

## Create kind cluster.
if [[ -z "${KIND_NODE_TAG}" ]]; then
  tools/bin/kind create cluster --name "${CLUSTER_NAME}" --config=tools/hack/cluster.conf
else
  tools/bin/kind create cluster --image "kindest/node:${KIND_NODE_TAG}" --name "${CLUSTER_NAME}" --config=tools/hack/cluster.conf
fi
