#!/bin/bash

export GOPROXY="https://proxy.golang.com.cn,direct"

VERBOSE=${VERBOSE:-"0"}
V=""
if [[ "${VERBOSE}" == "1" ]];then
    V="-x"
    set -x
fi

SCRIPTPATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

OUT=${1:?"output path"}
shift

set -e

BUILD_GOOS=${GOOS:-linux}
BUILD_GOARCH=${GOARCH:-amd64}
GOBINARY=${GOBINARY:-go}
GOPKG="$GOPATH/pkg"
BUILDINFO=${BUILDINFO:-""}
STATIC=${STATIC:-1}
LDFLAGS=${LDFLAGS:--extldflags -static}
GOBUILDFLAGS=${GOBUILDFLAGS:-""}
# Split GOBUILDFLAGS by spaces into an array called GOBUILDFLAGS_ARRAY.
IFS=' ' read -r -a GOBUILDFLAGS_ARRAY <<< "$GOBUILDFLAGS"

GCFLAGS=${GCFLAGS:-}
export CGO_ENABLED=${CGO_ENABLED:-0}

if [[ "${STATIC}" !=  "1" ]];then
    LDFLAGS=""
fi

# BUILD LD_EXTRAFLAGS
LD_EXTRAFLAGS=""

# gather buildinfo if not already provided
# For a release build BUILDINFO should be produced
# at the beginning of the build and used throughout
if [[ -z ${BUILDINFO} ]];then
    BUILDINFO=$(mktemp)
    "${SCRIPTPATH}/report_build_info.sh" > "${BUILDINFO}"
fi

while read -r line; do
    LD_EXTRAFLAGS="${LD_EXTRAFLAGS} -X ${line}"
done < "${BUILDINFO}"

OPTIMIZATION_FLAGS=(-trimpath)
if [ "${DEBUG}" == "1" ]; then
    OPTIMIZATION_FLAGS=()
fi

time GOOS=${BUILD_GOOS} GOARCH=${BUILD_GOARCH} ${GOBINARY} build \
        ${V} "${GOBUILDFLAGS_ARRAY[@]}" ${GCFLAGS:+-gcflags "${GCFLAGS}"} \
        -o "${OUT}" \
        "${OPTIMIZATION_FLAGS[@]}" \
        -pkgdir="${GOPKG}/${BUILD_GOOS}_${BUILD_GOARCH}" \
        -ldflags "${LDFLAGS} ${LD_EXTRAFLAGS}" "${@}"
