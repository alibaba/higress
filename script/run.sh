#!/bin/bash

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
