#!/usr/bin/env bash

# Copyright (c) 2026 Alibaba Group Holding Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
MAKE_LOG=$(mktemp)
trap 'rm -f "$MAKE_LOG"' EXIT
export MAKE_LOG

# 用 make 桩记录调用，避免测试触发真实的 Rust、WASM 或镜像构建。
make() {
    printf '%s:%s\n' "${PLUGIN_NAME-}" "$*" >> "$MAKE_LOG"
}
export -f make

(
    cd "$REPO_ROOT"
    PLUGIN_TYPE=RUST PLUGIN_NAME=test-plugin \
        bash tools/hack/build-wasm-plugins.sh >/dev/null
)

actual=$(<"$MAKE_LOG")
expected=$'test-plugin:lint-base\ntest-plugin:test-base\ntest-plugin:lint\ntest-plugin:test\ntest-plugin:build'
if [[ "$actual" != "$expected" ]]; then
    printf 'unexpected make call sequence:\n%s\n' "$actual" >&2
    printf 'expected:\n%s\n' "$expected" >&2
    exit 1
fi
