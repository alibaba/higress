#!/bin/bash

# WARNING: DO NOT EDIT, THIS FILE IS PROBABLY A COPY
#
# The original version of this file is located in the https://github.com/istio/common-files repo.
# If you're looking at this file in a different repo and want to make a change, please go to the
# common-files repo, make the change there and check it in. Then come back to this repo and run
# "make update-common".

# Copyright Istio Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

WD=$(dirname "$0")
WD=$(cd "$WD"; pwd)

# shellcheck disable=SC1090
source "${WD}/setup_env.sh"

# Override variables with container specific
export TARGET_OUT=${CONTAINER_TARGET_OUT}
export TARGET_OUT_LINUX=${CONTAINER_TARGET_OUT_LINUX}
export REPO_ROOT=/work

HUB="${HUB:-higress-registry.cn-hangzhou.cr.aliyuncs.com/higress}"
MOUNT_SOURCE="${MOUNT_SOURCE:-${PWD}}"
MOUNT_DEST="${MOUNT_DEST:-/work}"

read -ra DOCKER_RUN_OPTIONS <<< "${DOCKER_RUN_OPTIONS:-}"


[[ -t 1 ]] && DOCKER_RUN_OPTIONS+=("-it")

# $CONTAINER_OPTIONS becomes an empty arg when quoted, so SC2086 is disabled for the
# following command only
# shellcheck disable=SC2086
"${CONTAINER_CLI}" run \
    --rm \
    "${DOCKER_RUN_OPTIONS[@]}" \
    --init \
    --sig-proxy=true \
    ${DOCKER_SOCKET_MOUNT:--v /var/run/docker.sock:/var/run/docker.sock} \
    $CONTAINER_OPTIONS \
    --env-file <(env | grep -v ${ENV_BLOCKLIST}) \
    -e IN_BUILD_CONTAINER=1 \
    -e TZ="${TIMEZONE:-$TZ}" \
    -e HUB="${HUB}" \
    --mount "type=bind,source=${MOUNT_SOURCE},destination=/work" \
    --mount "type=volume,source=go,destination=/go" \
    --mount "type=volume,source=gocache,destination=/gocache" \
    --mount "type=volume,source=cache,destination=/home/.cache" \
    ${CONDITIONAL_HOST_MOUNTS} \
    -w "${MOUNT_DEST}" "${IMG}" "$@"
