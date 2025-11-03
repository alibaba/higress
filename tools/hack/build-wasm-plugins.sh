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

if [ "$TYPE" == "CPP" ]
then
    cd ./plugins/wasm-cpp/
    if [ ! -n "$INNER_PLUGIN_NAME" ]; then
        echo "You must specify which cpp plugin you want to compile"
    else
        echo "🚀 Build CPP WasmPlugin: $INNER_PLUGIN_NAME"
        PLUGIN_NAME=${INNER_PLUGIN_NAME} make build
    fi
elif [ "$TYPE" == "RUST" ]
then
    cd ./plugins/wasm-rust/
    make lint-base
    make test-base
    if [ ! -n "$INNER_PLUGIN_NAME" ]; then
        EXTENSIONS_DIR=$(pwd)"/extensions/"
        echo "🚀 Build all Rust WasmPlugins under folder of $EXTENSIONS_DIR"
        for file in `ls $EXTENSIONS_DIR`                                   
            do
                if [ -d $EXTENSIONS_DIR$file ]; then 
                    name=${file##*/}
                    echo "🚀 Build Rust WasmPlugin: $name"
                    PLUGIN_NAME=${name} make lint 
                    PLUGIN_NAME=${name} make test 
                    PLUGIN_NAME=${name} make build
                fi
            done
    else
        echo "🚀 Build Rust WasmPlugin: $INNER_PLUGIN_NAME"
        PLUGIN_NAME=${INNER_PLUGIN_NAME} make lint 
        PLUGIN_NAME=${INNER_PLUGIN_NAME} make build
    fi
else
    echo "Not specify plugin language, so just compile wasm-go as default"
    cd ./plugins/wasm-go/
    if [ ! -n "$INNER_PLUGIN_NAME" ]; then
        EXTENSIONS_DIR=$(pwd)"/extensions/"
        echo "🚀 Build all Go WasmPlugins under folder of $EXTENSIONS_DIR"
        for file in `ls $EXTENSIONS_DIR`                                   
            do
                # : adjust waf build
                if [ "$file" == "" ]; then
                    continue
                fi
                if [ -d $EXTENSIONS_DIR$file ]; then
                    name=${file##*/}
                    version_file="$EXTENSIONS_DIR$file/VERSION"
                    if [ -f "$version_file" ]; then
                        version=$(cat "$version_file")
                        if [[ "$version" =~ -alpha$ ]]; then
                            echo "🚀 Build Go WasmPlugin: $name (version $version)"
                            # Load .buildrc file
                            buildrc_file="$EXTENSIONS_DIR$file/.buildrc"
                            if [ -f "$buildrc_file" ]; then
                                echo "Found .buildrc file, sourcing it..."
                                . "$buildrc_file"
                            else
                                echo ".buildrc file not found"
                            fi
                            echo "EXTRA_TAGS=${EXTRA_TAGS:-}"
                            # Build plugin
                            PLUGIN_NAME=${name} EXTRA_TAGS=${EXTRA_TAGS:-} make build
                            # Clean up EXTRA_TAGS environment variable
                            unset EXTRA_TAGS
                        else
                            echo "Plugin version $version not ends with '-alpha', skipping compilation for $name."
                        fi
                    else
                        echo "VERSION file not found for plugin $name, skipping compilation."
                    fi
                fi
            done
    else
        echo "🚀 Build Go WasmPlugin: $INNER_PLUGIN_NAME"
        PLUGIN_NAME=${INNER_PLUGIN_NAME} make build
    fi
fi
