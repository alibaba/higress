#!/bin/bash

#  Copyright (c) 2022 Alibaba Group Holding Ltd.

#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at

#       http:www.apache.org/licenses/LICENSE-2.0

#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# if has HIGRESS_APISERVER_SVC env, wait for apiserver ready
if [ -n "$HIGRESS_APISERVER_SVC" ]; then
    while true; do
        echo "testing higress apiserver is ready to connect..."
        nc -z "$HIGRESS_APISERVER_SVC" "${HIGRESS_APISERVER_PORT}"
        if [ $? -eq 0 ]; then
            break
        fi
        sleep 1
    done
fi