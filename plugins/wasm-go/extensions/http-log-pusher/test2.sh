#!/bin/bash
# 快速诊断 log-collector 连通性（支持 Static 服务和标准 Service）

set -e  # 遇错退出

GATEWAY_NS="himarket-system"
COLLECTOR_NS="higress-system"
COLLECTOR_SVC="log-collector"

echo "=== 1. 检查 Service 状态（Kubernetes 原生） ==="
if kubectl get svc $COLLECTOR_SVC -n $COLLECTOR_NS &>/dev/null; then
    kubectl get svc $COLLECTOR_SVC -n $COLLECTOR_NS
else
    echo "⚠️ 未找到 Kubernetes Service '$COLLECTOR_SVC'，可能是 Static 服务"
fi

echo ""
echo "=== 2. 检查 McpBridge 静态服务（关键！） ==="
if kubectl get mcpservice $COLLECTOR_SVC -n $COLLECTOR_NS &>/dev/null; then
    echo "✅ 发现 Static 服务配置："
    kubectl get mcpservice $COLLECTOR_SVC -n $COLLECTOR_NS -o yaml | grep -A 5 "spec:"
    IS_STATIC=true
else
    echo "⚠️ 未找到 McpService，假设为标准 Kubernetes Service"
    IS_STATIC=false
fi

echo ""
echo "=== 3. 检查 Pod 和 Endpoints ==="
kubectl get pod -n $COLLECTOR_NS -l app=$COLLECTOR_SVC 2>/dev/null || echo "⚠️ 未找到匹配标签的 Pod（标签可能非 'app=log-collector'）"
echo ""
kubectl get endpoints $COLLECTOR_SVC -n $COLLECTOR_NS 2>/dev/null || echo "⚠️ Endpoints 不存在或非标准 Service"

echo ""
echo "=== 4. 获取 Gateway Pod ==="
GATEWAY_POD=$(kubectl get pod -n $GATEWAY_NS -l app=higress-gateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -z "$GATEWAY_POD" ]; then
    echo "❌ 未找到 Gateway Pod（检查标签是否为 'app=higress-gateway'）"
    exit 1
fi
echo "✅ Gateway Pod: $GATEWAY_POD"

# 自动检测容器名
CONTAINER=$(kubectl get pod -n $GATEWAY_NS $GATEWAY_POD -o jsonpath='{.spec.containers[0].name}')
echo "✅ 使用容器: $CONTAINER"

echo ""
echo "=== 5. 检查 Envoy 集群配置 ==="
if kubectl exec -n $GATEWAY_NS $GATEWAY_POD -c $CONTAINER -- curl -sf localhost:15000/clusters &>/dev/null; then
    if [ "$IS_STATIC" = true ]; then
        EXPECTED="outbound|80||$COLLECTOR_SVC.static"
        echo "🔍 检查 Static 服务集群: $EXPECTED"
        kubectl exec -n $GATEWAY_NS $GATEWAY_POD -c $CONTAINER -- curl -s localhost:15000/clusters | grep "$EXPECTED" || echo "❌ 未找到集群"
    else
        EXPECTED="outbound|80||$COLLECTOR_SVC.$COLLECTOR_NS.svc.cluster.local"
        echo "🔍 检查标准 Service 集群: $EXPECTED"
        kubectl exec -n $GATEWAY_NS $GATEWAY_POD -c $CONTAINER -- curl -s localhost:15000/clusters | grep "$EXPECTED" || echo "❌ 未找到集群"
        kubectl exec -n $GATEWAY_NS $GATEWAY_POD -c $CONTAINER -- curl -s localhost:15000/clusters > clusters.json
    fi
else
    echo "⚠️ Envoy Admin 端口 15000 不可用，尝试 pilot-agent..."
    kubectl exec -n $GATEWAY_NS $GATEWAY_POD -c $CONTAINER -- pilot-agent request GET clusters 2>/dev/null | grep $COLLECTOR_SVC || echo "❌ 无法获取集群列表"
fi

echo ""
# echo "=== 6. 【关键】从 Gateway 内部测试连通性 ==="
# if [ "$IS_STATIC" = true ]; then
#     echo "⚠️ Static 服务无 DNS，需通过 IP:Port 直连（检查 McpBridge 配置中的 endpoints）"
#     kubectl get mcpservice $COLLECTOR_SVC -n $COLLECTOR_NS -o jsonpath='{.spec.endpoints}' | jq . 2>/dev/null || echo "无法解析 endpoints"
# else
#     echo "✅ 测试 DNS 解析和 TCP 连通性..."
#     kubectl exec -n $GATEWAY_NS $GATEWAY_POD -c $CONTAINER -- nslookup $COLLECTOR_SVC.$COLLECTOR_NS.svc.cluster.local
#     kubectl exec -n $GATEWAY_NS $GATEWAY_POD -c $CONTAINER -- timeout 3 bash -c "echo > /dev/tcp/$COLLECTOR_SVC.$COLLECTOR_NS.svc.cluster.local/80" && echo "✅ TCP 连通" || echo "❌ TCP 不通"
# fi

echo ""
echo "=== 诊断完成 ==="