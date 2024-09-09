#!/usr/bin/env bash

# Copyright (c) 2023 Alibaba Group Holding Ltd.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#      http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

source "$(dirname -- "$0")/setup-istio-env.sh"

cd ${ROOT}/external/istio
rm -rf out/linux_${TARGET_ARCH}; 

GOOS_LOCAL=linux TARGET_OS=linux TARGET_ARCH=${TARGET_ARCH} \
    ISTIO_ENVOY_LINUX_RELEASE_URL=${ISTIO_ENVOY_LINUX_RELEASE_URL} \
    BUILD_WITH_CONTAINER=1 \
    CONDITIONAL_HOST_MOUNTS=${CONDITIONAL_HOST_MOUNTS} \
    ISTIO_BASE_REGISTRY="${HUB}" \
    BASE_VERSION="${HIGRESS_BASE_VERSION}" \
    DOCKER_RUN_OPTIONS="--user root -e HTTP_PROXY -e HTTPS_PROXY" \
    IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools:release-1.19-ef344298e65eeb2d9e2d07b87eb4e715c2def613 \
    make build-linux
