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

set -o errexit
set -o nounset
set -o pipefail

readonly KIND=${KIND:-tools/bin/kind}
readonly CLUSTER_NAME=${CLUSTER_NAME:-"higress"}

# Docker variables
readonly IMAGE="$1"
readonly TAG="$2"

# Wrap sed to deal with GNU and BSD sed flags.
run::sed() {
  if sed --version </dev/null 2>&1 | grep -q GNU; then
    # GNU sed
    sed -i "$@"
  else
    # assume BSD sed
    sed -i '' "$@"
  fi
}

kind::cluster::exists() {
    ${KIND} get clusters | grep -q "$1"
}

kind::cluster::load() {
    ${KIND} load docker-image \
        --name "${CLUSTER_NAME}" \
        "$@"
}

if ! kind::cluster::exists "$CLUSTER_NAME" ; then
    echo "cluster $CLUSTER_NAME does not exist"
    exit 2
fi

# Push the Higress image to the kind cluster.
echo "Loading image ${IMAGE}:${TAG} to kind cluster ${CLUSTER_NAME}..."
kind::cluster::load "${IMAGE}:${TAG}"
