#!/bin/bash

# if has HIGRESS_APISERVER_SVC env, wait for apiserver ready
if [ -n "$HIGRESS_APISERVER_SVC" ]; then
    while true; do
        echo "testing higress apiserver is ready to connect..."
        nc -z "$HIGRESS_APISERVER_SVC" "${HIGRESS_APISERVER_PORT:-8443}"
        if [ $? -eq 0 ]; then
            break
        fi
        sleep 1
    done
fi