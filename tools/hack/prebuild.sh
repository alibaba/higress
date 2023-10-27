#!/bin/bash
set -e

GO_VERSION=1.20

WORK_DIR=`cd $(dirname "$0")/../..;pwd`

cd $WORK_DIR

mkdir -p external/package

envoy_repos=("go-control-plane" "envoy")

for repo in ${envoy_repos[@]}; do
    cd external/$repo
    if [ -f "go.mod" ]; then
        go mod tidy -go=${GO_VERSION}
    fi
    cd $WORK_DIR
done

istio_repos=("api" "client-go" "pkg" "istio" "proxy")

for repo in ${istio_repos[@]}; do
    cd external/$repo
    if [ -f "go.mod" ]; then
        go mod tidy -go=${GO_VERSION}
    fi
    cd $WORK_DIR
done
