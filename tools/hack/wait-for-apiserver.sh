#!/bin/bash

HIGRESS_APISERVER_SVC=${HIGRESS_APISERVER_SVC:-"higress-apiserver"}
HIGRESS_APISERVER_PORT=${HIGRESS_APISERVER_PORT:-"8443"}

if [ -n "$HIGRESS_APISERVER_SVC" ]; then
    # wait for mcp-bridge
    while true; do
        echo "testing higress apiserver is ready to connect..."
        nc -z $HIGRESS_APISERVER_SVC ${HIGRESS_APISERVER_PORT:-8443}
        if [ $? -eq 0 ]; then
            break
        fi
        sleep 1
    done
fi