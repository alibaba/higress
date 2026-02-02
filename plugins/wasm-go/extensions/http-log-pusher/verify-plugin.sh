#!/bin/bash

set -e

NAMESPACE="himarket-system"
GATEWAY_IP="8.153.103.221"
PLUGIN_NAME="http-log-push-plugin"
CONFIG_FILE="/Users/terry/work/higress/plugins/wasm-go/extensions/http-log-pusher/higress-wasm-plugin-improved.yaml"
WASM_GO_DIR="/Users/terry/work/higress/plugins/wasm-go"

echo "=========================================="
echo "Step 0: 构建 WASM 插件"
echo "=========================================="
cd "$WASM_GO_DIR"
DOCKER_HOST=unix://$HOME/.lima/docker/sock/docker.sock PLUGIN_NAME=http-log-pusher make build

if [ ! -f "$WASM_GO_DIR/extensions/http-log-pusher/plugin.wasm" ]; then
    echo "❌ 构建失败：plugin.wasm 文件不存在"
    exit 1
fi

# 生成带时间戳的文件名
BUILD_TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
WASM_WITH_TIMESTAMP="$WASM_GO_DIR/extensions/http-log-pusher/plugin-${BUILD_TIMESTAMP}.wasm"

# 重命名为带时间戳的文件
mv "$WASM_GO_DIR/extensions/http-log-pusher/plugin.wasm" "$WASM_WITH_TIMESTAMP"

echo "✅ 构建成功：plugin-${BUILD_TIMESTAMP}.wasm ($(du -h $WASM_WITH_TIMESTAMP | cut -f1))"
echo ""

# OSS 上的文件名也使用时间戳
OSS_WASM_NAME="plugin-${BUILD_TIMESTAMP}.wasm"
OSS_WASM_URL="https://pysrc-test.oss-cn-beijing.aliyuncs.com/higress-plugin/${OSS_WASM_NAME}"

echo "=========================================="
echo "请上传 WASM 文件到 OSS"
echo "=========================================="
echo "本地文件: $WASM_WITH_TIMESTAMP"
echo "OSS 路径: $OSS_WASM_URL"
echo ""
echo "⚠️  注意: 上传到 OSS 时文件名为 $OSS_WASM_NAME"
echo ""
echo "上传完成后输入 'y' 继续，或按 Ctrl+C 取消"
read -p "是否已上传到 OSS? (y/n): " confirm

if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
    echo "❌ 取消部署"
    rm -f "$WASM_WITH_TIMESTAMP"
    echo "已清理临时文件: $WASM_WITH_TIMESTAMP"
    exit 1
fi

echo "✅ 继续执行部署..."
echo ""

# 上传完成后删除带时间戳的 wasm 文件
rm -f "$WASM_WITH_TIMESTAMP"
echo "✅ 已删除临时文件: plugin-${BUILD_TIMESTAMP}.wasm"
echo ""

echo "=========================================="
echo "Step 1: 更新配置文件并应用"
echo "=========================================="
# 使用相同的 timestamp 用于配置
NEW_TIMESTAMP="$BUILD_TIMESTAMP"
echo "新的 timestamp: $NEW_TIMESTAMP"
echo "新的 OSS URL: $OSS_WASM_URL"

# 更新配置文件中的 timestamp 和 URL
sed -i.bak "s|higress.io/redeploy-timestamp: \"[^\"]*\"|higress.io/redeploy-timestamp: \"$NEW_TIMESTAMP\"|" "$CONFIG_FILE"
sed -i.bak "s|url: https://pysrc-test.oss-cn-beijing.aliyuncs.com/higress-plugin/plugin.*\.wasm|url: $OSS_WASM_URL|" "$CONFIG_FILE"

echo "✅ 已更新配置文件:"
echo "  - redeploy-timestamp: $NEW_TIMESTAMP"
echo "  - url: $OSS_WASM_URL"
echo ""

kubectl apply -f "$CONFIG_FILE"

# echo ""
# echo "=========================================="
# echo "Step 2: 等待并监控插件热加载状态"
# echo "=========================================="
# echo "等待 Gateway 热加载新插件 (最多等待 90 秒)..."

# # 记录当前时间戳，用于后续过滤新日志
# CURRENT_TIME=$(date +%s)
# MAX_WAIT=90
# WAIT_COUNT=0
# RELOAD_SUCCESS=false

# while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
#     # 检查最近的插件加载日志（包含新的 timestamp）
#     RECENT_LOGS=$(kubectl logs -n "$NAMESPACE" -l app=higress-gateway -c higress-gateway --since=60s 2>/dev/null | grep -i "http-log-pusher" || echo "")
    
#     # 检查是否有成功加载的标志
#     if echo "$RECENT_LOGS" | grep -q "loaded successfully\|plugin initializing"; then
#         echo "✅ 检测到插件热加载成功！"
#         echo "加载日志:"
#         echo "$RECENT_LOGS" | grep "http-log-pusher"
#         RELOAD_SUCCESS=true
#         break
#     fi
    
#     # 每 5 秒检查一次
#     if [ $((WAIT_COUNT % 5)) -eq 0 ]; then
#         echo "  等待中... (${WAIT_COUNT}s / ${MAX_WAIT}s)"
#     fi
    
#     sleep 1
#     WAIT_COUNT=$((WAIT_COUNT + 1))
# done

# if [ "$RELOAD_SUCCESS" = false ]; then
#     echo "❌ 错误: 在 ${MAX_WAIT} 秒内未检测到插件热加载日志"
#     echo ""
#     echo "可能原因:"
#     echo "  1. Gateway 正在拉取 WASM 文件（网络慢）- 建议手动检查后重试"
#     echo "  2. redeploy-timestamp 或 URL 未更新 - 检查 YAML 文件"
#     echo "  3. WASM 文件拉取失败 - 检查 OSS URL 是否可访问"
#     echo "  4. WASM 文件编译错误 - 检查构建日志"
#     echo ""
#     echo "调试命令:"
#     echo "  kubectl logs -n $NAMESPACE -l app=higress-gateway -c higress-gateway --tail=300 | grep -iE 'http-log-pusher|wasm|error'"
#     echo ""
#     exit 1
# fi

# echo ""

echo "=========================================="
echo "Step 3: 验证插件版本和配置"
echo "=========================================="
# 动态获取 Gateway Pod 名称
GATEWAY_POD=$(kubectl get pod -n "$NAMESPACE" -l app=higress-gateway -o jsonpath='{.items[0].metadata.name}')
echo "Gateway Pod: $GATEWAY_POD"
echo ""

echo "(a) 检查 WasmPlugin 配置是否正确更新:"
ACTUAL_URL=$(kubectl get wasmplugin -n "$NAMESPACE" "$PLUGIN_NAME" -o jsonpath='{.spec.url}')
ACTUAL_TIMESTAMP=$(kubectl get wasmplugin -n "$NAMESPACE" "$PLUGIN_NAME" -o jsonpath='{.metadata.annotations.higress\.io/redeploy-timestamp}')

echo "  当前 URL: $ACTUAL_URL"
echo "  当前 Timestamp: $ACTUAL_TIMESTAMP"
echo "  预期 URL: $OSS_WASM_URL"
echo "  预期 Timestamp: $NEW_TIMESTAMP"

if [ "$ACTUAL_URL" = "$OSS_WASM_URL" ] && [ "$ACTUAL_TIMESTAMP" = "$NEW_TIMESTAMP" ]; then
    echo "  ✅ 配置已正确更新"
else
    echo "  ❌ 配置更新失败！"
    echo "  建议执行: kubectl get wasmplugin -n $NAMESPACE $PLUGIN_NAME -o yaml"
    exit 1
fi
echo ""

echo "(b) 检查最近的插件加载日志 (最近 20 行):"
LOAD_LOGS=$(kubectl logs -n "$NAMESPACE" "$GATEWAY_POD" -c higress-gateway --tail=20 | grep -i "http-log-pusher" || echo "")
if [ -n "$LOAD_LOGS" ]; then
    echo "  ✅ 找到插件日志："
    echo "$LOAD_LOGS" | tail -10
else
    echo "  ⚠️  未找到插件加载日志"
fi
echo ""

echo "(c) 检查 WASM 文件拉取日志:"
WASM_FETCH_LOGS=$(kubectl logs -n "$NAMESPACE" "$GATEWAY_POD" -c higress-gateway --tail=30 | grep -iE "wasm.*fetch|wasm.*download|wasm.*load.*$OSS_WASM_NAME" | tail -5 || echo "")
if [ -n "$WASM_FETCH_LOGS" ]; then
    echo "  ✅ WASM 拉取日志："
    echo "$WASM_FETCH_LOGS"
else
    echo "  ⚠️  未找到 WASM 拉取日志（可能已缓存或拉取中）"
fi

echo ""
echo "=========================================="
echo "Step 4: 检查 WasmPlugin 状态"
echo "=========================================="
kubectl get wasmplugin -n "$NAMESPACE" "$PLUGIN_NAME" -o yaml | grep -A 10 "status:" || echo "无 status 字段"

echo ""
echo "=========================================="
echo "Step 5: 发送测试请求触发插件 (测试 model-api-qwen72b-0 Ingress)"
echo "=========================================="
echo "发送请求到: http://$GATEWAY_IP/test-ai/v1/chat/completions"
RESPONSE=$(curl -s -X POST "http://$GATEWAY_IP/test-ai/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "User-Agent: verify-plugin-test-$(date +%s)" \
  -d '{
    "stream": true,
    "model": "qwen3-max",
    "messages": [
      { "role": "system", "content": "You are a helpful assistant." },
      { "role": "user", "content": "who are you?" }
    ],
    "temperature": 1,
    "top_p": 0.95,
    "max_tokens": 1024,
    "presence_penalty": 0,
    "frequency_penalty": 0
  }' \
  -w "\nHTTP Status: %{http_code}\n" || echo "请求失败")
echo "$RESPONSE"

echo ""
echo "=========================================="
echo "Step 6: 等待 3 秒后查看请求处理日志"
echo "=========================================="
sleep 3

REQUEST_LOGS=$(kubectl logs -n "$NAMESPACE" "$GATEWAY_POD" -c higress-gateway --tail=100 | grep "http-log-pusher" || echo "")
if [ -n "$REQUEST_LOGS" ]; then
    echo "✅ 找到请求处理日志："
    echo "$REQUEST_LOGS"
else
    echo "⚠️  未找到请求处理日志"
fi

echo ""
echo "=========================================="
echo "验证完成"
echo "=========================================="
