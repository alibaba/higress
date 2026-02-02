#!/bin/bash

# WASM 插件日志级别动态调整脚本
# 用途: 实时调整 Higress Gateway 的 WASM 组件日志级别，无需重启 Pod

set -e

NAMESPACE="himarket-system"
DEFAULT_LEVEL="debug"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}WASM 日志级别调整工具${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 获取 Gateway Pod
echo -e "${YELLOW}正在获取 Higress Gateway Pod...${NC}"
GATEWAY_POD=$(kubectl get pods -n "$NAMESPACE" -l app=higress-gateway --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$GATEWAY_POD" ]; then
    echo -e "${RED}错误: 未找到运行中的 Higress Gateway Pod${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 找到 Gateway Pod: $GATEWAY_POD${NC}"
echo ""

# 解析参数（支持直接传入日志级别）
if [ -n "$1" ]; then
    LOG_LEVEL="$1"
else
    LOG_LEVEL="$DEFAULT_LEVEL"
fi

# 验证日志级别
case "$LOG_LEVEL" in
    trace|debug|info|warning|error|critical|off)
        ;;
    *)
        echo -e "${RED}错误: 无效的日志级别 '$LOG_LEVEL'${NC}"
        echo ""
        echo "支持的日志级别:"
        echo "  - trace    (最详细)"
        echo "  - debug    (调试信息)"
        echo "  - info     (一般信息)"
        echo "  - warning  (警告)"
        echo "  - error    (错误)"
        echo "  - critical (严重错误)"
        echo "  - off      (关闭日志)"
        echo ""
        echo "用法: $0 [日志级别]"
        echo "示例: $0 debug"
        exit 1
        ;;
esac

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}当前操作${NC}"
echo -e "${YELLOW}========================================${NC}"
echo "  Pod:        $GATEWAY_POD"
echo "  Namespace:  $NAMESPACE"
echo "  日志级别:    $LOG_LEVEL"
echo ""

# 执行日志级别调整
echo -e "${YELLOW}正在调整 WASM 日志级别...${NC}"
RESULT=$(kubectl exec -n "$NAMESPACE" "$GATEWAY_POD" -- curl -s "127.0.0.1:15000/logging?wasm=$LOG_LEVEL" -X POST 2>&1 || echo "")

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 日志级别已成功调整为: $LOG_LEVEL${NC}"
    echo ""
    echo "响应:"
    echo "$RESULT"
else
    echo -e "${RED}✗ 调整失败${NC}"
    echo "错误信息:"
    echo "$RESULT"
    exit 1
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}验证日志级别已更新${NC}"
echo -e "${BLUE}========================================${NC}"

# 等待 1 秒让配置生效
sleep 1

# 查询当前日志级别
CURRENT_LEVEL=$(kubectl exec -n "$NAMESPACE" "$GATEWAY_POD" -- curl -s "127.0.0.1:15000/logging" -X POST 2>&1 | grep -i "wasm:" || echo "")

if [ -n "$CURRENT_LEVEL" ]; then
    echo -e "${GREEN}✓ WASM 日志级别:${NC} $CURRENT_LEVEL"
else
    echo -e "${YELLOW}⚠️  无法查询当前日志级别${NC}"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}操作完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}提示:${NC}"
echo "  1. 新的日志级别已生效，无需重启 Pod"
echo "  2. 可以立即查看新的日志输出:"
echo -e "     ${BLUE}kubectl logs -n $NAMESPACE $GATEWAY_POD -c higress-gateway -f | grep 'http-log-pusher'${NC}"
echo ""
echo "  3. 恢复默认级别 (info):"
echo -e "     ${BLUE}$0 info${NC}"
echo ""

# 如果设置为 debug 或 trace，给出额外提示
if [ "$LOG_LEVEL" = "debug" ] || [ "$LOG_LEVEL" = "trace" ]; then
    echo -e "${YELLOW}注意: $LOG_LEVEL 级别会产生大量日志，建议测试完成后恢复为 info${NC}"
    echo ""
fi
