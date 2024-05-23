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

VOLUMES_ROOT="/tmp/higress-apiserver"
RSA_KEY_LENGTH=4096
NAMESPACE="${NAMESPACE:-higress-system}"
CLUSTER_NAME="${CLUSTER_NAME:-higress}"
APISERVER_ADDRESS="${APISERVER_ADDRESS:-https://127.0.0.1:8443}"

initializeApiServer() {
  echo "Initializing API server configurations..."

  mkdir -p "$VOLUMES_ROOT/api" && cd "$_"
  checkExitCode "Creating volume for API server fails with $?"

  if [ ! -f ca.key ] || [ ! -f ca.crt ]; then
    echo "  Generating CA certificate...";
    openssl req -nodes -new -x509 -days 36500 -keyout ca.key -out ca.crt -subj "/CN=higress-root-ca/O=higress" > /dev/null 2>&1
    checkExitCode "  Generating CA certificate for API server fails with $?";
  else
    echo "  CA certificate already exists.";
  fi

  if [ ! -f server.key ] || [ ! -f server.crt ]; then
    echo "  Generating server certificate..."
    openssl req -out server.csr -new -newkey rsa:$RSA_KEY_LENGTH -nodes -keyout server.key -subj "/CN=higress-api-server/O=higress" > /dev/null 2>&1 \
      && openssl x509 -req -days 36500 -in server.csr -CA ca.crt -CAkey ca.key -set_serial 01 -sha256 -out server.crt > /dev/null 2>&1
    checkExitCode "  Generating server certificate fails with $?";
  else
    echo "  Server certificate already exists.";
  fi

  if [ ! -f nacos.key ]; then
    echo "  Generating data encryption key..."
    if [ -z "$NACOS_DATA_ENC_KEY" ]; then
      cat /dev/urandom | tr -dc '[:graph:]' | head -c 32 > nacos.key
    else
      echo -n "$NACOS_DATA_ENC_KEY" > nacos.key
    fi
  else
    echo "  Client certificate already exists.";
  fi

  if [ ! -f client.key ] || [ ! -f client.crt ]; then
    echo "  Generating client certificate..."
    openssl req -out client.csr -new -newkey rsa:$RSA_KEY_LENGTH -nodes -keyout client.key -subj "/CN=higress/O=system:masters" > /dev/null 2>&1 \
      && openssl x509 -req -days 36500 -in client.csr -CA ca.crt -CAkey ca.key -set_serial 02 -sha256 -out client.crt > /dev/null 2>&1
    checkExitCode "  Generating client certificate fails with $?";
  else
    echo "  Client certificate already exists.";
  fi
}

applySecretConfigmap() {
  # create namespace if not exists
  kubectl get namespace $NAMESPACE > /dev/null 2>&1
  if [ $? -ne 0 ]; then
    echo "Creating namespace $NAMESPACE..."
    kubectl create namespace $NAMESPACE
    checkExitCode "Creating namespace fails with $?"
  fi

  echo "Applying secret $NAMESPACE/higress-apiserver..."
  kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: higress-apiserver
  namespace: $NAMESPACE
data:
  ca.crt: $(cat $VOLUMES_ROOT/api/ca.crt | base64 -w 0)
  ca.key: $(cat $VOLUMES_ROOT/api/ca.key | base64 -w 0)
  server.crt: $(cat $VOLUMES_ROOT/api/server.crt | base64 -w 0)
  server.key: $(cat $VOLUMES_ROOT/api/server.key | base64 -w 0)
  client.key: $(cat $VOLUMES_ROOT/api/client.crt | base64 -w 0)
  nacos.key: $(cat $VOLUMES_ROOT/api/nacos.key | base64 -w 0)
EOF

  echo "Applying configmap $NAMESPACE/higress-apiserver..."
  kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: higress-apiserver
  namespace: $NAMESPACE
data:
  kubeconfig: |
    apiVersion: v1
    kind: Config
    clusters:
      - name: $CLUSTER_NAME
        cluster:
          server: $APISERVER_ADDRESS
          insecure-skip-tls-verify: true
    users:
      - name: higress-admin
        user:
          client-certificate-data: $(cat $VOLUMES_ROOT/api/client.crt | base64 -w 0)
          client-key-data: $(cat $VOLUMES_ROOT/api/client.key | base64 -w 0)
    contexts:
      - name: higress
        context:
          cluster: $CLUSTER_NAME
          user: higress-admin
    preferences: {}
    current-context: higress
EOF

    echo "Successfully applied secret and configmap. Now you can start higress-api-server."
}

checkExitCode() {
  # $1 message
  retVal=$?
  if [ $retVal -ne 0 ]; then
    echo ${1:-"  Command fails with $retVal"}
    exit $retVal
  fi
}

initializeApiServer
applySecretConfigmap