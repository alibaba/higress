# Copyright (c) 2022 Alibaba Group Holding Ltd.

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

INNER_GO_FILTER_NAME=${GO_FILTER_NAME-""}
OUTPUT_PACKAGE_DIR=${OUTPUT_PACKAGE_DIR:-"../../external/package/"}

cd ./plugins/golang-filter
if [ ! -n "$INNER_GO_FILTER_NAME" ]; then
    GO_FILTERS_DIR=$(pwd)
    echo "🚀 Build all Go Filters under folder of $GO_FILTERS_DIR"
    for file in `ls $GO_FILTERS_DIR`
        do
        if [ -d $GO_FILTERS_DIR/$file ]; then
            name=${file##*/}
            echo "🚀 Build Go Filter: $name"
            GO_FILTER_NAME=${name} GOARCH=${TARGET_ARCH} make build
            cp ${GO_FILTERS_DIR}/${file}/${name}_${TARGET_ARCH}.so ${OUTPUT_PACKAGE_DIR}
        fi
    done
else
    echo "🚀 Build Go Filter: $INNER_GO_FILTER_NAME"
    GO_FILTER_NAME=${INNER_GO_FILTER_NAME} make build
fi

