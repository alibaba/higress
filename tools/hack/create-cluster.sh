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
# Two ways to choose cluster node image:
# - KIND_NODE_IMAGE: full image reference, e.g. docker.m.daocloud.io/kindest/node:v1.25.3
# - KIND_NODE_TAG:   short tag used with default image "kindest/node:<tag>" (kept for compatibility)
KIND_NODE_IMAGE=${KIND_NODE_IMAGE:-""}
KIND_NODE_TAG=${KIND_NODE_TAG:-"v1.25.3"}
PROJECT_DIR=$(pwd)

echo "KIND_NODE_TAG=${KIND_NODE_TAG}"
echo "KIND_NODE_IMAGE=${KIND_NODE_IMAGE}"
echo "CLUSTER_NAME=${CLUSTER_NAME}"

# Resolve final node image, prefer KIND_NODE_IMAGE if set
NODE_IMAGE=""
if [[ -n "${KIND_NODE_IMAGE}" ]]; then
  NODE_IMAGE="${KIND_NODE_IMAGE}"
elif [[ -n "${KIND_NODE_TAG}" ]]; then
  NODE_IMAGE="kindest/node:${KIND_NODE_TAG}"
fi

# If ONLY_PRINT_NODE_IMAGE=1, just print and exit (for dry-run/CI param check)
if [[ "${ONLY_PRINT_NODE_IMAGE:-}" == "1" ]]; then
  echo "Resolved NODE_IMAGE=${NODE_IMAGE}"
  exit 0
fi

cat <<EOF > "tools/hack/cluster.conf"
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

# cluster.conf
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  ipFamily: dual
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  extraMounts:
    - hostPath: ${PROJECT_DIR}/plugins
      containerPath: /opt/plugins
EOF

## Create kind cluster.
if [[ -n "${NODE_IMAGE}" ]]; then
  echo "Creating kind cluster with image: ${NODE_IMAGE}"
  tools/bin/kind create cluster --image "${NODE_IMAGE}" --name "${CLUSTER_NAME}" --config=tools/hack/cluster.conf
else
  echo "Creating kind cluster with default kindest/node image"
  tools/bin/kind create cluster --name "${CLUSTER_NAME}" --config=tools/hack/cluster.conf
fi
