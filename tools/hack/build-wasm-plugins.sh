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


TYPE=${PLUGIN_TYPE-""}
INNER_PLUGIN_NAME=${PLUGIN_NAME-""}

if [ $TYPE == "CPP" ]
then
    cd ./plugins/wasm-cpp/
    if [ ! -n "$INNER_PLUGIN_NAME" ]; then
        echo "you must specify which cpp plugin you want to compile"
    else
        echo "build wasmplugin-cpp name of $INNER_PLUGIN_NAME"
        PLUGIN_NAME=${INNER_PLUGIN_NAME} make build
    fi
else
    echo "not specify plugin language, so just compile wasm-go as default"
    cd ./plugins/wasm-go/
    if [ ! -n "$INNER_PLUGIN_NAME" ]; then
        EXTENSIONS_DIR=$(pwd)"/extensions/"
        echo "build all wasmplugins-go under folder of $EXTENSIONS_DIR"
        for file in `ls $EXTENSIONS_DIR`                                   
            do
                # TODO: adjust waf build
                if [ $file == "waf" ]; then 
                    continue
                fi
                if [ -d $EXTENSIONS_DIR$file ]; then 
                    name=${file##*/}
                    echo "build wasmplugin name of $name"
                    PLUGIN_NAME=${name} make build
                fi
            done
    else
        echo "build wasmplugin-go name of $INNER_PLUGIN_NAME"
        PLUGIN_NAME=${INNER_PLUGIN_NAME} make build
    fi
fi


