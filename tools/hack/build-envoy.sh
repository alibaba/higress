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

cd ${ROOT}/external/proxy

CONDITIONAL_HOST_MOUNTS+="--mount type=bind,source=${PWD}/out,destination=/home "

BUILD_WITH_CONTAINER=1 \
    CONDITIONAL_HOST_MOUNTS=${CONDITIONAL_HOST_MOUNTS} \
    BUILD_ENVOY_BINARY_ONLY=1 \
    IMG=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools-proxy:release-1.19-04ab00931b61c082300832a7dd51634e5e3634ad \
    make test_release
