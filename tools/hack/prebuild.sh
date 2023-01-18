#!/bin/bash
set -e

ENVOY_VERSION="${ENVOY_VERSION:=1.20}"
ISITO_VERSION="${ISTIO_VERSION:=1.12}"
WORK_DIR=`cd $(dirname "$0")/../..;pwd`

cd $WORK_DIR

mkdir -p external/package

envoy_repos=("go-control-plane" "envoy")

for repo in ${envoy_repos[@]}; do
    if [ -e external/$repo ];then
        continue
    fi
    cp -r envoy/${ENVOY_VERSION}/$repo  external/$repo
    for patch in `ls envoy/${ENVOY_VERSION}/patches/$repo/*.patch`; do
        patch -d external/$repo -p1 < $patch
    done
    cd external/$repo
    echo "gitdir: /parent/.git/modules/envoy/${ENVOY_VERSION}/$repo" > .git
    if [ -f "go.mod" ]; then
        go mod tidy
    fi
    cd $WORK_DIR
done

istio_repos=("api" "client-go" "pkg" "istio" "proxy")

for repo in ${istio_repos[@]}; do
    if [ -e external/$repo ];then
        continue
    fi
    cp -r istio/${ISTIO_VERSION}/$repo  external/$repo
    for patch in `ls istio/${ISTIO_VERSION}/patches/$repo/*.patch`; do
        patch -d external/$repo -p1 < $patch
    done
    cd external/$repo
    echo "gitdir: /parent/.git/modules/istio/${ISTIO_VERSION}/$repo" > .git
    if [ -f "go.mod" ]; then
        go mod tidy
    fi
    cd $WORK_DIR
done
