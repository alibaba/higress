#!/bin/bash

set -e

NAMESPACE="ls-test"
GATEWAY_IP="8.137.23.26"

echo "=========================================="
echo "启动实时日志监控"
echo "=========================================="

# 在后台启动日志监控
kubectl logs -n "$NAMESPACE" -l app=higress-gateway -c higress-gateway -f --tail=10 | grep --line-buffered -E "\[http-log-pusher\]" &
LOG_PID=$!

# 等待 2 秒确保日志开始监控
sleep 2

echo ""
echo "=========================================="
echo "发送测试请求"
echo "=========================================="

# 发送请求
curl -v "http://${GATEWAY_IP}/qwen0113/v1/chat/completions" \
  -H "User-Agent: plugin-test-$(date +%s)" \
  -H "Content-Type: application/json" \
  -d '{"model": "qwen-72b", "messages": [{"role": "user", "content": "test"}]}'

echo ""
echo ""
echo "=========================================="
echo "等待 3 秒收集日志..."
echo "=========================================="
sleep 3

# 停止日志监控
kill $LOG_PID 2>/dev/null || true

echo ""
echo "=========================================="
echo "查看 model-api-qwen72b-0 相关的插件日志"
echo "=========================================="
PLUGIN_LOGS=$(kubectl logs -n "$NAMESPACE" -l app=higress-gateway -c higress-gateway --tail=200 | grep "\[http-log-pusher\]")

if [ -z "$PLUGIN_LOGS" ]; then
    echo "❌ 未找到任何插件日志 - 插件可能未被触发"
    echo ""
    echo "检查 Ingress 路由配置:"
    kubectl get ingress model-api-qwen72b-0 -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].http.paths[0].path}'
    echo ""
else
    echo "✅ 找到插件日志:"
    echo "$PLUGIN_LOGS"
    echo ""
    
    # 统计请求数
    REQUEST_COUNT=$(echo "$PLUGIN_LOGS" | grep -c "onHttpRequestHeaders called" || echo "0")
    SAMPLED_COUNT=$(echo "$PLUGIN_LOGS" | grep -c "request sampled" || echo "0")
    SENT_COUNT=$(echo "$PLUGIN_LOGS" | grep -c "log sent successfully" || echo "0")
    
    echo "📊 统计:"
    echo "  - 请求命中数: $REQUEST_COUNT"
    echo "  - 采样通过数: $SAMPLED_COUNT"  
    echo "  - 日志发送成功数: $SENT_COUNT"
fi
