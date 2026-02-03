#!/bin/bash

# HTTP Log Pusher - Log Collector 连通性诊断脚本
# 用于排查 "bad argument" 错误

set -e

NAMESPACE="higress-system"
SERVICE_NAME="log-collector"
COLLECTOR_PORT=80
COLLECTOR_PATH="/ingest"

echo "=========================================="
echo "HTTP Log Pusher 诊断脚本"
echo "=========================================="
echo ""

# 1. 检查 Service 和 Pod 状态
echo ">>> 1. 检查 Service 和 Pod 状态"
echo "----------------------------------------"
kubectl get svc,pod -n $NAMESPACE -l app=$SERVICE_NAME
echo ""

# 2. 检查 Endpoints（必须有 IP 地址）
echo ">>> 2. 检查 Endpoints（必须有实际 IP）"
echo "----------------------------------------"
ENDPOINTS=$(kubectl get endpoints $SERVICE_NAME -n $NAMESPACE -o jsonpath='{.subsets[*].addresses[*].ip}')
if [ -z "$ENDPOINTS" ]; then
    echo "❌ ERROR: Endpoints 为空，Service 没有可用的 Pod！"
    echo "   请检查 Pod 是否正常运行"
    exit 1
else
    echo "✅ Endpoints: $ENDPOINTS"
fi
echo ""

# 3. 获取 Gateway Pod 名称
echo ">>> 3. 获取 Higress Gateway Pod"
echo "----------------------------------------"
GATEWAY_POD=$(kubectl get pod -n $NAMESPACE -l app=higress-gateway -o jsonpath='{.items[0].metadata.name}')
if [ -z "$GATEWAY_POD" ]; then
    echo "❌ ERROR: 未找到 higress-gateway Pod"
    exit 1
fi
echo "✅ Gateway Pod: $GATEWAY_POD"
echo ""

# 4. DNS 解析测试
echo ">>> 4. 从 Gateway Pod 测试 DNS 解析"
echo "----------------------------------------"
SERVICE_FQDN="${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local"
echo "测试域名: $SERVICE_FQDN"
kubectl exec -n $NAMESPACE $GATEWAY_POD -c istio-proxy -- \
  nslookup $SERVICE_FQDN || echo "❌ DNS 解析失败"
echo ""

# 5. HTTP 连通性测试
echo ">>> 5. 从 Gateway Pod 测试 HTTP 连通性"
echo "----------------------------------------"
TEST_URL="http://${SERVICE_FQDN}:${COLLECTOR_PORT}${COLLECTOR_PATH}"
echo "测试 URL: $TEST_URL"
kubectl exec -n $NAMESPACE $GATEWAY_POD -c istio-proxy -- \
  curl -v -X POST $TEST_URL \
  -H "Content-Type: application/json" \
  -d '{"test":"connectivity_check","timestamp":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}' \
  --connect-timeout 5 --max-time 10 || echo "❌ HTTP 连接失败"
echo ""

# 6. 检查 Envoy 集群配置
echo ">>> 6. 检查 Envoy 是否有 log-collector 集群"
echo "----------------------------------------"
echo "查找集群名称包含 'log-collector' 或 'outbound|80' 的条目："
CLUSTER_INFO=$(kubectl exec -n $NAMESPACE $GATEWAY_POD -c istio-proxy -- \
  curl -s localhost:15000/clusters | grep -i "log-collector" || echo "")
if [ -z "$CLUSTER_INFO" ]; then
    echo "❌ WARNING: Envoy 中未找到 log-collector 相关集群！"
    echo "   这可能是 'bad argument' 错误的根本原因"
    echo ""
    echo "建议操作："
    echo "  1. 重启 gateway 强制重新同步："
    echo "     kubectl rollout restart deployment higress-gateway -n $NAMESPACE"
    echo "  2. 等待重启完成后重新运行本脚本"
else
    echo "✅ 找到集群配置："
    echo "$CLUSTER_INFO"
fi
echo ""

# 7. 检查期望的集群名称
echo ">>> 7. 检查期望的集群名称"
echo "----------------------------------------"
EXPECTED_CLUSTER="outbound|${COLLECTOR_PORT}||${SERVICE_FQDN}"
echo "期望的集群名: $EXPECTED_CLUSTER"
EXACT_MATCH=$(kubectl exec -n $NAMESPACE $GATEWAY_POD -c istio-proxy -- \
  curl -s localhost:15000/clusters | grep "$EXPECTED_CLUSTER" || echo "")
if [ -z "$EXACT_MATCH" ]; then
    echo "❌ WARNING: Envoy 中没有完全匹配的集群名称！"
else
    echo "✅ 找到完全匹配的集群："
    echo "$EXACT_MATCH"
fi
echo ""

# 8. 总结和建议
echo "=========================================="
echo "诊断总结"
echo "=========================================="
echo ""

if [ -n "$ENDPOINTS" ] && [ -n "$EXACT_MATCH" ]; then
    echo "✅ Service 和 Envoy 配置都正常"
    echo ""
    echo "如果仍然出现 'bad argument' 错误，可能原因："
    echo "  1. WASM 插件配置中的 collector_service 参数格式不对"
    echo "  2. 请求 headers 或 body 格式有问题"
    echo "  3. 查看 WASM 插件日志确认实际使用的参数"
elif [ -z "$EXACT_MATCH" ]; then
    echo "❌ 问题定位: Envoy 没有所需的集群配置"
    echo ""
    echo "解决方案："
    echo "  1. 重启 gateway 强制同步配置："
    echo "     kubectl rollout restart deployment higress-gateway -n $NAMESPACE"
    echo ""
    echo "  2. 或修改 WASM 插件配置，使用 host+port 方式："
    echo "     collector_host: \"$SERVICE_FQDN\""
    echo "     collector_port: $COLLECTOR_PORT"
    echo "     # 移除或注释 collector_service 配置"
fi

echo ""
echo "=========================================="
echo "诊断完成"
echo "=========================================="
