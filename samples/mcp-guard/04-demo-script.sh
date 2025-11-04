#!/usr/bin/env bash
set -euo pipefail

# ===== Configurable images (override by env) =====
KIND_NODE_IMAGE=${KIND_NODE_IMAGE:-kindest/node:v1.25.3}
HIGRESS_VER=${HIGRESS_VER:-2.1.9-rc.1}
CTRL_IMAGE=${CTRL_IMAGE:-higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress:${HIGRESS_VER}}
PILOT_IMAGE=${PILOT_IMAGE:-higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/pilot:${HIGRESS_VER}}
GATEWAY_IMAGE=${GATEWAY_IMAGE:-higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway:${HIGRESS_VER}}

echo "Using images:"
echo "  KIND_NODE_IMAGE=${KIND_NODE_IMAGE}"
echo "  CTRL_IMAGE=${CTRL_IMAGE}"
echo "  PILOT_IMAGE=${PILOT_IMAGE}"
echo "  GATEWAY_IMAGE=${GATEWAY_IMAGE}"

# ===== 1) Create kind cluster (with host plugins mount) =====
pushd "$(dirname "$0")/../../higress" >/dev/null
KIND_NODE_IMAGE="${KIND_NODE_IMAGE}" make create-cluster
popd >/dev/null

# ===== 2) Preload images into kind (avoid pulling from internet) =====
kind load docker-image "${CTRL_IMAGE}" --name higress
kind load docker-image "${PILOT_IMAGE}" --name higress
kind load docker-image "${GATEWAY_IMAGE}" --name higress

# ===== 3) Install Higress core via Helm =====
pushd "$(dirname "$0")/../../higress" >/dev/null
helm install higress helm/core -n higress-system --create-namespace \
  --set controller.tag="${HIGRESS_VER}" \
  --set gateway.replicas=1 \
  --set pilot.tag="${HIGRESS_VER}" \
  --set gateway.tag="${HIGRESS_VER}" \
  --set global.local=true \
  --set global.volumeWasmPlugins=true
popd >/dev/null

echo "Waiting for pods to be ready..."
kubectl -n higress-system rollout status deploy/higress-controller --timeout=180s || true
kubectl -n higress-system rollout status deploy/higress-gateway --timeout=180s || true

# ===== 4) Apply demo manifests =====
PROVIDER=${PROVIDER:-openai}
if [[ "${PROVIDER}" == "deepseek" ]]; then
  echo "Using DeepSeek provider manifests"
  kubectl apply -f samples/mcp-guard/02-egress-deepseek.yaml
  kubectl apply -f samples/mcp-guard/01-gw-vs-deepseek.yaml
  kubectl apply -f samples/mcp-guard/03-wasmplugins-deepseek.yaml
else
  echo "Using OpenAI provider manifests"
  kubectl apply -f samples/mcp-guard/02-egress-openai.yaml
  kubectl apply -f samples/mcp-guard/01-gw-vs.yaml
  kubectl apply -f samples/mcp-guard/03-wasmplugins.yaml
fi

cat <<EOF

===== Demo ready =====
1. 将你的 OpenAI Key 写入 03-wasmplugins.yaml 中的 REPLACE_WITH_YOUR_OPENAI_KEY 并重新 kubectl apply
   或者运行： kubectl -n higress-system patch wasmplugin ai-proxy --type merge -p '"spec":{"defaultConfig":{"providers":[{"id":"openai-main","type":"openai","protocol":"openai","apiTokens":["YOUR_KEY_HERE"],"modelMapping":{"*":"gpt-4o-mini"}}]}}'

2. 获取网关地址（NodePort/LB，或本机 docker：localhost）
   - 本机 kind 默认映射 80/443 到主机： 直接使用 http://localhost

3. 试运行：
   - 白金客户（tenantA）访问摘要：
     curl -i -X POST \\
       -H 'Host: api.example.com' \\
       -H 'X-Subject: tenantA' \\
       -H 'X-MCP-Capability: cap.text.summarize' \\
       -H 'Content-Type: application/json' \\
       -d '{"prompt":"用一句话概述：Higress 是什么？"}' \\
       http://localhost/v1/text:summarize

   - 标准客户（tenantB）访问翻译（预期403）：
     curl -i -X POST \\
       -H 'Host: api.example.com' \\
       -H 'X-Subject: tenantB' \\
       -H 'X-MCP-Capability: cap.text.translate' \\
       -H 'Content-Type: application/json' \\
       -d '{"prompt":"请将这句话翻译为英文：你好，世界"}' \\
       http://localhost/v1/text:translate

期望输出：
 - 摘要接口：HTTP/200，返回模型摘要内容
 - 翻译接口：HTTP/403 且 body 含有 "mcp-guard deny"

EOF
