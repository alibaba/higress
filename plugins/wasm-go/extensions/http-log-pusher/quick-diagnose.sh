#!/bin/bash
# 快速诊断 log-collector 连通性

# 注意：Gateway 在 ls-test，log-collector 在 higress-system
GATEWAY_NS="ls-test"
COLLECTOR_NS="higress-system"

echo "1. 检查 Service 状态..."
kubectl get svc log-collector -n $COLLECTOR_NS

echo ""
echo "2. 检查 Pod 状态..."
kubectl get pod -n $COLLECTOR_NS -l app=log-collector

echo ""
echo "3. 检查 Endpoints..."
kubectl get endpoints log-collector -n $COLLECTOR_NS

echo ""
echo "4. 获取 Gateway Pod (在 $GATEWAY_NS 命名空间)..."
GATEWAY_POD=$(kubectl get pod -n $GATEWAY_NS -l app=higress-gateway -o jsonpath='{.items[0].metadata.name}')
if [ -z "$GATEWAY_POD" ]; then
    echo "❌ 未找到 Gateway Pod"
    exit 1
fi
echo "✅ Gateway Pod: $GATEWAY_POD"

echo ""
echo "5. 检查 Envoy 集群配置..."
kubectl exec -n $GATEWAY_NS $GATEWAY_POD -c higress-gateway -- curl -s localhost:15000/clusters | grep log-collector

echo ""
echo "6. 检查期望的完整集群名..."
EXPECTED_CLUSTER="outbound|80||log-collector.$COLLECTOR_NS.svc.cluster.local"
echo "期望集群名: $EXPECTED_CLUSTER"
kubectl exec -n $GATEWAY_NS $GATEWAY_POD -c higress-gateway -- curl -s localhost:15000/clusters | grep "$EXPECTED_CLUSTER" || echo "❌ 未找到完全匹配的集群"

echo ""
echo "完成！"
