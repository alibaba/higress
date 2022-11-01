#!/bin/bash

if BUILD_GIT_REVISION=$(git rev-parse HEAD 2> /dev/null); then
  if [[ -z "${IGNORE_DIRTY_TREE}" ]] && [[ -n "$(git status --porcelain 2>/dev/null)" ]]; then
    BUILD_GIT_REVISION=${BUILD_GIT_REVISION}"-dirty"
  fi
else
  BUILD_GIT_REVISION=unknown
fi

# Check for local changes
tree_status="Clean"
if [[ -z "${IGNORE_DIRTY_TREE}" ]] && ! git diff-index --quiet HEAD --; then
  tree_status="Modified"
fi

GIT_DESCRIBE_TAG=$(git describe --tags)
HUB=${HUB:-"higress-registry.cn-hangzhou.cr.aliyuncs.com/higress"}

# used by common/scripts/gobuild.sh
echo "istio.io/pkg/version.buildVersion=${VERSION:-$BUILD_GIT_REVISION}"
echo "istio.io/pkg/version.buildGitRevision=${BUILD_GIT_REVISION}"
echo "istio.io/pkg/version.buildStatus=${tree_status}"
echo "istio.io/pkg/version.buildTag=${GIT_DESCRIBE_TAG}"
echo "istio.io/pkg/version.buildHub=${HUB}"
