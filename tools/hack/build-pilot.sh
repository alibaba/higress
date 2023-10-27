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

#!/usr/bin/env bash

set -euo pipefail

TARGET_ARCH=${TARGET_ARCH-"amd64"}

cd external/istio
rm -rf out/linux_${TARGET_ARCH}; 

BUILDINFO=$(mktemp)
"${PWD}/common/scripts/report_build_info.sh" > "${BUILDINFO}"

TAG=$(git rev-parse --verify HEAD)

GOOS_LOCAL=linux TARGET_OS=linux TARGET_ARCH=${TARGET_ARCH} \
    ISTIO_ENVOY_LINUX_RELEASE_URL="${ENVOY_PACKAGE_URL}" \
    BUILDINFO="${BUILDINFO}" \
    TAG="${TAG}" \
    BUILD_WITH_CONTAINER=1 \
    CONDITIONAL_HOST_MOUNTS="--mount type=bind,source=${BUILDINFO},destination=${BUILDINFO},readonly " \
    make build-linux
