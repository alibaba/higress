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

cd ./plugins/wasm-go/

INNER_PLUGIN_NAME=${PLUGIN_NAME-""}
if [ ! -n "$INNER_PLUGIN_NAME" ]; then
    EXTENSIONS_DIR=$(pwd)"/extensions/"
    echo "build all wasmplugins under folder of $EXTENSIONS_DIR"
    for file in `ls $EXTENSIONS_DIR`                                   
        do
            if [ -d $EXTENSIONS_DIR$file ]; then 
                name=${file##*/}
                echo "build wasmplugin name of $name"
                PLUGIN_NAME=${name} make build
            fi
        done
else
    echo "build wasmplugin name of $INNER_PLUGIN_NAME"
    PLUGIN_NAME=${INNER_PLUGIN_NAME} make build
fi
