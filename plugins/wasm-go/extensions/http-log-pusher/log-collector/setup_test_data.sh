#!/bin/bash

# 测试数据准备脚本
# 用于向 log-collector 插入测试数据

set -e

COLLECTOR_URL="${COLLECTOR_URL:-http://localhost:8080}"

echo "=========================================="
echo "准备测试数据"
echo "=========================================="
echo ""

# 生成指定时间段的日志数据
generate_time_range_logs() {
    local time_range=$1      # 时间范围: hour, day, week, month
    local count=$2           # 生成数量
    local base_time=$3       # 基准时间戳
    
    echo "生成${time_range}时间段的 $count 条日志数据..."
    
    for i in $(seq 1 $count); do
        # 生成随机数据
        local trace_id="trace-$(printf "%016d" $i)"
        local request_id="req-$(uuidgen | cut -d'-' -f1)"
        
        # 根据时间范围生成不同的时间戳
        local timestamp
        case "$time_range" in
            "hour")
                # 最近1小时内，按分钟递减
                timestamp=$(date -u -v-$((RANDOM % 60))M +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "$((RANDOM % 60)) minutes ago" +"%Y-%m-%dT%H:%M:%SZ")
                ;;
            "day")
                # 最近24小时内，按小时递减
                timestamp=$(date -u -v-$((RANDOM % 1440))M +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "$((RANDOM % 1440)) minutes ago" +"%Y-%m-%dT%H:%M:%SZ")
                ;;
            "week")
                # 最近7天内，按天递减
                timestamp=$(date -u -v-$((RANDOM % 10080))M +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "$((RANDOM % 10080)) minutes ago" +"%Y-%m-%dT%H:%M:%SZ")
                ;;
            "month")
                # 最近30天内，按周递减
                timestamp=$(date -u -v-$((RANDOM % 43200))M +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "$((RANDOM % 43200)) minutes ago" +"%Y-%m-%dT%H:%M:%SZ")
                ;;
            *)
                # 默认使用原来的逻辑
                timestamp=$(date -u -v-${i}M +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "$i minutes ago" +"%Y-%m-%dT%H:%M:%SZ")
                ;;
        esac
        
        # 随机选择服务和方法
        local services=("api-service.default.svc.cluster.local" "web-service.default.svc.cluster.local" "auth-service.default.svc.cluster.local")
        local methods=("GET" "POST" "PUT" "DELETE")
        local paths=("/api/users" "/api/products" "/api/orders" "/api/auth/login" "/health")
        local status_codes=(200 201 400 401 404 500)
        
        local service=${services[$((RANDOM % ${#services[@]}))]}
        local method=${methods[$((RANDOM % ${#methods[@]}))]}
        local path=${paths[$((RANDOM % ${#paths[@]}))]}
        local status_code=${status_codes[$((RANDOM % ${#status_codes[@]}))]}
        
        # 生成随机数值
        local bytes_received=$((RANDOM % 10000 + 100))
        local bytes_sent=$((RANDOM % 50000 + 500))
        local duration=$((RANDOM % 1000 + 10))
        
        # 生成token数据（针对Model API测试）
        local input_tokens=$((RANDOM % 1000 + 50))
        local output_tokens=$((RANDOM % 2000 + 100))
        local total_tokens=$((input_tokens + output_tokens))
        
        # 生成监控元数据
        local models=("qwen-turbo" "gpt-3.5-turbo" "claude-2" "llama-2")
        local consumers=("user-001" "user-002" "admin" "anonymous")
        local routes=("route-1" "route-2" "default-route")
        local mcp_servers=("mcp-server-1" "mcp-server-2")
        local mcp_tools=("calculator" "web-search" "file-reader")
        
        local model=${models[$((RANDOM % ${#models[@]}))]}
        local consumer=${consumers[$((RANDOM % ${#consumers[@]}))]}
        local route=${routes[$((RANDOM % ${#routes[@]}))]}
        local mcp_server=${mcp_servers[$((RANDOM % ${#mcp_servers[@]}))]}
        local mcp_tool=${mcp_tools[$((RANDOM % ${#mcp_tools[@]}))]}
        
        # 构造JSON数据
        local log_data=$(cat <<EOF
{
    "start_time": "$timestamp",
    "authority": "$service",
    "trace_id": "$trace_id",
    "method": "$method",
    "path": "$path",
    "protocol": "HTTP/1.1",
    "request_id": "$request_id",
    "user_agent": "test-client/1.0",
    "x_forwarded_for": "192.168.1.$i",
    "response_code": $status_code,
    "response_flags": "-",
    "response_code_details": "via_upstream",
    "bytes_received": $bytes_received,
    "bytes_sent": $bytes_sent,
    "duration": $duration,
    "upstream_cluster": "outbound|80||$service",
    "upstream_host": "10.0.0.$i:8080",
    "upstream_service_time": "$duration",
    "upstream_transport_failure_reason": "",
    "upstream_local_address": "127.0.0.1:0",
    "downstream_local_address": "127.0.0.1:8080",
    "downstream_remote_address": "192.168.1.$i:0",
    "route_name": "test-route",
    "requested_server_name": "$service",
    "istio_policy_status": "-",
    "ai_log": "{\"model\":\"$model\",\"input_tokens\":$input_tokens,\"output_tokens\":$output_tokens}",
    "instance_id": "test-instance-001",
    "api": "test-api",
    "model": "$model",
    "consumer": "$consumer",
    "route": "$route",
    "service": "$service",
    "mcp_server": "$mcp_server",
    "mcp_tool": "$mcp_tool",
    "input_tokens": $input_tokens,
    "output_tokens": $output_tokens,
    "total_tokens": $total_tokens
}
EOF
)
        
        # 发送到collector
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$log_data" \
            "${COLLECTOR_URL}/ingest")
        
        if [ $? -ne 0 ]; then
            echo "❌ 发送第 $i 条日志失败"
            return 1
        fi
        
        if [ $((i % 50)) -eq 0 ]; then
            echo "已发送 $i/$count 条日志 ($time_range)"
        fi
    done
    
    echo "✅ 成功生成并发送 $count 条$time_range时间段的测试日志"
}

# 生成测试日志数据
generate_test_logs() {
    local total_count=${1:-200}
    echo "生成 $total_count 条测试日志数据（按时间分布）..."
    
    # 按时间分布生成数据
    # 最近1小时: 40%
    # 最近24小时: 30%  
    # 最近7天: 20%
    # 最近30天: 10%
    
    local hour_count=$((total_count * 40 / 100))
    local day_count=$((total_count * 30 / 100))
    local week_count=$((total_count * 20 / 100))
    local month_count=$((total_count - hour_count - day_count - week_count))
    
    echo "时间分布:"
    echo "  - 最近1小时: $hour_count 条"
    echo "  - 最近24小时: $day_count 条"
    echo "  - 最近7天: $week_count 条"
    echo "  - 最近30天: $month_count 条"
    echo ""
    
    # 生成各时间段数据
    generate_time_range_logs "hour" $hour_count
    generate_time_range_logs "day" $day_count
    generate_time_range_logs "week" $week_count
    generate_time_range_logs "month" $month_count
}

# 验证数据插入
verify_data() {
    echo ""
    echo "验证数据插入..."
    
    # 查询总记录数
    response=$(curl -s "${COLLECTOR_URL}/query?page_size=1")
    total=$(echo "$response" | jq -r '.total')
    
    if [ "$total" -gt 0 ]; then
        echo "✅ 数据库中有 $total 条记录"
        
        # 显示一些样本数据
        echo "样本数据:"
        curl -s "${COLLECTOR_URL}/query?page_size=3" | jq '.logs[] | {start_time, trace_id, authority, method, path, response_code, model, consumer}' 2>/dev/null || echo "无法解析JSON响应"
    else
        echo "❌ 数据库中没有记录"
        return 1
    fi
}

# 主函数
main() {
    echo "开始准备测试数据..."
    echo "Collector URL: ${COLLECTOR_URL}"
    echo ""
    
    # 检查依赖
    if ! command -v jq &> /dev/null; then
        echo "❌ jq 未安装，请先安装: brew install jq (macOS) 或 apt-get install jq (Ubuntu)"
        exit 1
    fi
    
    # 检查服务可用性
    if ! curl -s -f "${COLLECTOR_URL}/health" > /dev/null 2>&1; then
        echo "❌ log-collector 服务不可用: ${COLLECTOR_URL}"
        echo "请确保服务正在运行"
        exit 1
    fi
    
    # 生成测试数据
    generate_test_logs 500
    
    # 验证数据
    verify_data
    
    echo ""
    echo "=========================================="
    echo "测试数据准备完成"
    echo "=========================================="
}

# 执行主函数
main "$@"