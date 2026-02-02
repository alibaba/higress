#!/bin/bash

# HTTP Log Pusher 日志收集与分析脚本
# 用途: 收集 Higress Gateway 日志并分类保存到本地文件进行详细排查

set -e

NAMESPACE="ls-test"
TARGET_INGRESS="model-api-qwen72b-0"  # 目标 Ingress 名称
TARGET_PATH="/qwen0113/v1/chat/completions"  # 目标路径
LOG_DIR="./logs"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}HTTP Log Pusher 日志收集工具${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 创建日志目录
mkdir -p "$LOG_DIR"

# 获取 Gateway Pod
echo -e "${YELLOW}正在获取 Higress Gateway Pod...${NC}"
GATEWAY_POD=$(kubectl get pods -n "$NAMESPACE" -l app=higress-gateway --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$GATEWAY_POD" ]; then
    echo -e "${RED}错误: 未找到运行中的 Higress Gateway Pod${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 找到 Gateway Pod: $GATEWAY_POD${NC}"
echo ""

# 定义日志文件路径
ALL_LOGS="$LOG_DIR/gateway_all_${TIMESTAMP}.log"
PLUGIN_LOGS="$LOG_DIR/http_log_pusher_${TIMESTAMP}.log"
WASM_LOGS="$LOG_DIR/wasm_all_${TIMESTAMP}.log"
ERROR_LOGS="$LOG_DIR/errors_${TIMESTAMP}.log"
LOAD_LOGS="$LOG_DIR/plugin_load_${TIMESTAMP}.log"
REQUEST_LOGS="$LOG_DIR/http_requests_${TIMESTAMP}.log"
PUSH_LOGS="$LOG_DIR/log_push_${TIMESTAMP}.log"
ANALYSIS_FILE="$LOG_DIR/analysis_${TIMESTAMP}.txt"

echo -e "${YELLOW}正在收集日志... (最近1000行)${NC}"

# 1. 收集完整日志
kubectl logs -n "$NAMESPACE" "$GATEWAY_POD" -c higress-gateway --tail=1000 > "$ALL_LOGS" 2>&1
echo -e "${GREEN}✓ 完整日志已保存: $ALL_LOGS${NC}"

# 2. 提取 http-log-pusher 相关日志（仅目标 Ingress）
grep -i "http-log-pusher" "$ALL_LOGS" > "$PLUGIN_LOGS" 2>/dev/null || echo "未找到 http-log-pusher 相关日志" > "$PLUGIN_LOGS"
echo -e "${GREEN}✓ http-log-pusher 日志已保存: $PLUGIN_LOGS${NC}"

# 2.1 过滤出目标路径的日志（仅包含 path=$TARGET_PATH 的日志）
TARGET_PLUGIN_LOGS="$LOG_DIR/http_log_pusher_${TARGET_INGRESS}_${TIMESTAMP}.log"
if [ -s "$PLUGIN_LOGS" ]; then
    grep "path=$TARGET_PATH" "$PLUGIN_LOGS" > "$TARGET_PLUGIN_LOGS" 2>/dev/null || echo "未找到针对 $TARGET_PATH 的日志" > "$TARGET_PLUGIN_LOGS"
    echo -e "${GREEN}✓ $TARGET_INGRESS 日志已过滤: $TARGET_PLUGIN_LOGS${NC}"
else
    echo "未找到针对 $TARGET_PATH 的日志" > "$TARGET_PLUGIN_LOGS"
fi

# 3. 提取所有 WASM 相关日志
grep -i "wasm\|envoy" "$ALL_LOGS" > "$WASM_LOGS" 2>/dev/null || echo "未找到 WASM 相关日志" > "$WASM_LOGS"
echo -e "${GREEN}✓ WASM 相关日志已保存: $WASM_LOGS${NC}"

# 4. 提取错误日志
grep -iE "error|fatal|fail|exception" "$ALL_LOGS" > "$ERROR_LOGS" 2>/dev/null || echo "未找到错误日志" > "$ERROR_LOGS"
echo -e "${GREEN}✓ 错误日志已保存: $ERROR_LOGS${NC}"

# 5. 提取插件加载日志
grep -iE "load.*wasm|wasm.*load|plugin.*load|filter.*load" "$ALL_LOGS" > "$LOAD_LOGS" 2>/dev/null || echo "未找到插件加载日志" > "$LOAD_LOGS"
echo -e "${GREEN}✓ 插件加载日志已保存: $LOAD_LOGS${NC}"

# 6. 提取 HTTP 请求处理日志
grep -iE "onHttpRequestHeaders|onHttpResponseHeaders|onHttpRequestBody" "$ALL_LOGS" > "$REQUEST_LOGS" 2>/dev/null || echo "未找到 HTTP 请求处理日志" > "$REQUEST_LOGS"
echo -e "${GREEN}✓ HTTP 请求处理日志已保存: $REQUEST_LOGS${NC}"

# 7. 提取日志推送相关日志
grep -iE "push.*log|send.*log|post.*log|sendLogToEndpoint" "$ALL_LOGS" > "$PUSH_LOGS" 2>/dev/null || echo "未找到日志推送相关日志" > "$PUSH_LOGS"
echo -e "${GREEN}✓ 日志推送记录已保存: $PUSH_LOGS${NC}"

echo ""
echo -e "${YELLOW}正在分析日志...${NC}"

# 生成分析报告
{
    echo "=========================================="
    echo "HTTP Log Pusher 日志分析报告"
    echo "生成时间: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Gateway Pod: $GATEWAY_POD"
    echo "目标 Ingress: $TARGET_INGRESS"
    echo "目标路径: $TARGET_PATH"
    echo "=========================================="
    echo ""
    
    echo "【1. 插件加载状态】"
    if grep -q "http-log-pusher" "$LOAD_LOGS"; then
        grep "http-log-pusher" "$LOAD_LOGS" | tail -5
    else
        echo "⚠️ 未找到插件加载日志"
    fi
    echo ""
    
    echo "【２. HTTP 请求处理统计】"
    echo "(a) 所有 Ingress:"
    ONREQUEST_COUNT=$(grep -c "onHttpRequestHeaders" "$PLUGIN_LOGS" 2>/dev/null || echo "0")
    ONRESPONSE_COUNT=$(grep -c "onHttpResponseHeaders" "$PLUGIN_LOGS" 2>/dev/null || echo "0")
    echo "  - onHttpRequestHeaders 调用次数: $ONREQUEST_COUNT"
    echo "  - onHttpResponseHeaders 调用次数: $ONRESPONSE_COUNT"
        
    echo ""
    echo "(b) 目标 Ingress ($TARGET_INGRESS - $TARGET_PATH):"
    TARGET_ONREQUEST_COUNT=$(grep -c "onHttpRequestHeaders" "$TARGET_PLUGIN_LOGS" 2>/dev/null || echo "0")
    TARGET_ONRESPONSE_COUNT=$(grep -c "onHttpResponseHeaders" "$TARGET_PLUGIN_LOGS" 2>/dev/null || echo "0")
    echo "  - onHttpRequestHeaders 调用次数: $TARGET_ONREQUEST_COUNT"
    echo "  - onHttpResponseHeaders 调用次数: $TARGET_ONRESPONSE_COUNT"
        
    if [ "$TARGET_ONREQUEST_COUNT" -eq 0 ]; then
        echo "  ⚠️ 警告: 目标 Ingress 插件未被触发或日志级别过低"
    fi
    echo ""
    
    echo "【3. 日志推送统计】"
    PUSH_COUNT=$(grep -c "sendLogToEndpoint\|push.*log" "$PUSH_LOGS" 2>/dev/null || echo "0")
    echo "- 日志推送尝试次数: $PUSH_COUNT"
    if [ "$PUSH_COUNT" -eq 0 ]; then
        echo "  ⚠️ 警告: 未检测到日志推送操作"
    fi
    echo ""
    
    echo "【4. 错误分析】"
    if [ -s "$ERROR_LOGS" ] && grep -q "http-log-pusher" "$ERROR_LOGS"; then
        echo "发现与插件相关的错误:"
        grep "http-log-pusher" "$ERROR_LOGS" | tail -10
    else
        echo "✓ 未发现插件相关错误"
    fi
    echo ""
    
    echo "【５. 目标 Ingress 最近的插件日志 (最后20条)】"
    echo "Ingress: $TARGET_INGRESS"
    echo "Path: $TARGET_PATH"
    tail -20 "$TARGET_PLUGIN_LOGS"
    echo ""
        
    echo "【６. 所有 Ingress 最近的插件日志 (最后10条)】"
    tail -10 "$PLUGIN_LOGS"
    echo ""
    
    echo "【７. 检查要点】"
    echo "□ 确认插件已加载 (检查 $LOAD_LOGS)"
    echo "□ 确认 log_level 配置正确 (应为 Info 或 Debug)"
    echo "□ 确认 Ingress 配置了正确的 annotation"
    echo "□ 确认请求路径匹配 Ingress 规则: $TARGET_PATH"
    echo "□ 确认日志推送端点可访问"
    echo ""
    
    echo "=========================================="
    echo "详细日志文件列表:"
    echo "  - 完整日志: $ALL_LOGS"
    echo "  - 插件日志(所有): $PLUGIN_LOGS"
    echo "  - 插件日志($TARGET_INGRESS): $TARGET_PLUGIN_LOGS"
    echo "  - WASM日志: $WASM_LOGS"
    echo "  - 错误日志: $ERROR_LOGS"
    echo "  - 加载日志: $LOAD_LOGS"
    echo "  - 请求日志: $REQUEST_LOGS"
    echo "  - 推送日志: $PUSH_LOGS"
    echo "=========================================="
} > "$ANALYSIS_FILE"

cat "$ANALYSIS_FILE"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}日志收集完成!${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "${BLUE}分析报告: $ANALYSIS_FILE${NC}"
echo -e "${BLUE}日志目录: $LOG_DIR${NC}"
echo ""
echo -e "${YELLOW}建议排查步骤:${NC}"
echo -e "1. 查看分析报告: ${BLUE}cat $ANALYSIS_FILE${NC}"
echo -e "2. 检查插件日志: ${BLUE}less $PLUGIN_LOGS${NC}"
echo -e "3. 检查错误日志: ${BLUE}less $ERROR_LOGS${NC}"
echo -e "4. 搜索特定关键字: ${BLUE}grep -i '关键字' $ALL_LOGS${NC}"
