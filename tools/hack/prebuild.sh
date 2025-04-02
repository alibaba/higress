#!/bin/bash
set -e

GO_VERSION=1.22

WORK_DIR=`cd $(dirname "$0")/../..;pwd`

cd $WORK_DIR

mkdir -p external/package

envoy_repos=("go-control-plane" "envoy")

for repo in ${envoy_repos[@]}; do
    if [ -e external/$repo ];then
        continue
    fi
    cp -RP envoy/$repo  external/$repo
    cd external/$repo
    echo "gitdir: /parent/.git/modules/envoy/$repo" > .git
    if [ -f "go.mod" ]; then
        go mod tidy -go=${GO_VERSION}
    fi
    cd $WORK_DIR
done

istio_repos=("api" "client-go" "pkg" "istio" "proxy")

for repo in ${istio_repos[@]}; do
    if [ -e external/$repo ];then
        continue
    fi
    cp -RP istio/$repo external/$repo
    cd external/$repo
    echo "gitdir: /parent/.git/modules/istio/$repo" > .git
    if [ -f "go.mod" ]; then
        go mod tidy -go=${GO_VERSION}
    fi
    cd $WORK_DIR
done
