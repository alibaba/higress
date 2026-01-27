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

echo "✅ 构建成功：plugin.wasm ($(du -h $WASM_GO_DIR/extensions/http-log-pusher/plugin.wasm | cut -f1))"
echo ""

echo "=========================================="
echo "请上传 plugin.wasm 到 OSS"
echo "=========================================="
echo "文件路径: $WASM_GO_DIR/extensions/http-log-pusher/plugin.wasm"
echo "OSS 地址: https://pysrc-test.oss-cn-beijing.aliyuncs.com/higress-plugin/plugin.wasm"
echo ""
echo "上传完成后输入 'y' 继续，或按 Ctrl+C 取消"
read -p "是否已上传到 OSS? (y/n): " confirm

if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
    echo "❌ 取消部署"
    exit 1
fi

echo "✅ 继续执行部署..."
echo ""

echo "=========================================="
echo "Step 1: 更新 redeploy-timestamp 并应用配置"
echo "=========================================="
# 生成新的 timestamp
NEW_TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
echo "新的 timestamp: $NEW_TIMESTAMP"

# 更新配置文件中的 timestamp
sed -i.bak "s/higress.io\/redeploy-timestamp: \"[^\"]*\"/higress.io\/redeploy-timestamp: \"$NEW_TIMESTAMP\"/" "$CONFIG_FILE"

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
