#!/bin/bash

set -e

NAMESPACE="ls-test"
GATEWAY_IP="8.137.23.26"
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

echo ""
echo "=========================================="
echo "Step 2: 等待 10 秒让 Gateway 拉取新配置"
echo "=========================================="
sleep 10

echo ""
echo "=========================================="
echo "Step 3: 检查插件加载日志"
echo "=========================================="
# 动态获取 Gateway Pod 名称
GATEWAY_POD=$(kubectl get pod -n "$NAMESPACE" -l app=higress-gateway -o jsonpath='{.items[0].metadata.name}')
echo "Gateway Pod: $GATEWAY_POD"
echo ""

LOAD_LOGS=$(kubectl logs -n "$NAMESPACE" "$GATEWAY_POD" -c higress-gateway --tail=200 | grep -i "http-log-pusher" || echo "")
if [ -n "$LOAD_LOGS" ]; then
    echo "✅ 找到插件日志："
    echo "$LOAD_LOGS"
else
    echo "⚠️  未找到插件加载日志"
fi

echo ""
echo "=========================================="
echo "Step 4: 检查 WasmPlugin 状态"
echo "=========================================="
kubectl get wasmplugin -n "$NAMESPACE" "$PLUGIN_NAME" -o yaml | grep -A 10 "status:" || echo "无 status 字段"

echo ""
echo "=========================================="
echo "Step 5: 发送测试请求触发插件"
echo "=========================================="
echo "发送请求到: http://$GATEWAY_IP/test"
RESPONSE=$(curl -s -X POST "http://$GATEWAY_IP/test" -d '{"test":"data"}' -w "\nHTTP Status: %{http_code}\n" || echo "请求失败")
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
