#!/bin/bash

# Batch 性能测试脚本
# 测试场景：
# 1. 不同批次大小的写入性能
# 2. 并发写入测试
# 3. 吞吐量测试
# 4. 边界条件测试

set -e

# 配置
COLLECTOR_URL="${COLLECTOR_URL:-http://localhost:8080}"
REPORT_DIR="./benchmark_reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$REPORT_DIR/batch_benchmark_${TIMESTAMP}.txt"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 创建报告目录
mkdir -p "$REPORT_DIR"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$REPORT_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$REPORT_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$REPORT_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$REPORT_FILE"
}

# 分隔线
print_separator() {
    echo "========================================" | tee -a "$REPORT_FILE"
}

# 检查服务可用性
check_service() {
    log_info "检查 log-collector 服务..."
    if curl -s -f "${COLLECTOR_URL}/health" > /dev/null 2>&1; then
        log_success "log-collector 服务正常运行"
        return 0
    else
        log_error "log-collector 服务不可用: ${COLLECTOR_URL}"
        exit 1
    fi
}

# 生成测试日志数据
generate_log() {
    local index=$1
    local path=${2:-"/api/test"}
    local method=${3:-"GET"}
    local status=${4:-200}
    
    cat <<EOF
{
    "start_time": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "authority": "test-service.default.svc.cluster.local",
    "trace_id": "trace-$(printf '%016x' $RANDOM$RANDOM)",
    "method": "$method",
    "path": "$path",
    "protocol": "HTTP/1.1",
    "request_id": "req-$(uuidgen | tr '[:upper:]' '[:lower:]')",
    "user_agent": "benchmark-test/1.0",
    "x_forwarded_for": "192.168.1.100",
    "response_code": $status,
    "response_flags": "-",
    "response_code_details": "via_upstream",
    "bytes_received": 1024,
    "bytes_sent": 2048,
    "duration": $((RANDOM % 1000)),
    "upstream_cluster": "outbound|8080||test-service.default.svc.cluster.local",
    "upstream_host": "10.244.0.5:8080",
    "upstream_service_time": "$((RANDOM % 500))",
    "upstream_transport_failure_reason": "",
    "upstream_local_address": "10.244.0.1:45678",
    "downstream_local_address": "10.244.0.1:80",
    "downstream_remote_address": "192.168.1.100:54321",
    "route_name": "default-route",
    "requested_server_name": "",
    "istio_policy_status": "",
    "ai_log": ""
}
EOF
}

# 发送单条日志
send_log() {
    local log_data="$1"
    curl -s -X POST "${COLLECTOR_URL}/ingest" \
        -H "Content-Type: application/json" \
        -d "$log_data" > /dev/null
}

# 测试1: 单线程批次大小测试
test_batch_sizes() {
    print_separator
    log_info "测试1: 单线程批次大小测试"
    print_separator
    
    local batch_sizes=(1 10 25 50 100 200)
    
    for size in "${batch_sizes[@]}"; do
        log_info "测试批次大小: $size"
        local start_time=$(date +%s.%N)
        
        for i in $(seq 1 $size); do
            local log_data=$(generate_log $i)
            send_log "$log_data"
        done
        
        # 等待flush完成
        sleep 2
        
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc)
        local tps=$(echo "scale=2; $size / $duration" | bc)
        
        log_success "批次大小: $size, 耗时: ${duration}s, TPS: ${tps}/s"
    done
}

# 测试2: 并发写入测试
test_concurrent_writes() {
    print_separator
    log_info "测试2: 并发写入测试"
    print_separator
    
    local concurrent_levels=(1 5 10 20)
    local logs_per_thread=50
    
    for level in "${concurrent_levels[@]}"; do
        log_info "测试并发级别: $level 线程, 每线程 $logs_per_thread 条日志"
        local start_time=$(date +%s.%N)
        
        # 并发发送
        for thread in $(seq 1 $level); do
            (
                for i in $(seq 1 $logs_per_thread); do
                    local log_data=$(generate_log "$thread-$i")
                    send_log "$log_data"
                done
            ) &
        done
        
        # 等待所有后台任务完成
        wait
        
        # 等待flush完成
        sleep 2
        
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc)
        local total_logs=$((level * logs_per_thread))
        local tps=$(echo "scale=2; $total_logs / $duration" | bc)
        
        log_success "并发: $level, 总日志: $total_logs, 耗时: ${duration}s, TPS: ${tps}/s"
    done
}

# 测试3: 吞吐量压测
test_throughput() {
    print_separator
    log_info "测试3: 吞吐量压测 (持续30秒)"
    print_separator
    
    local duration=30
    local concurrent=10
    local counter=0
    
    log_info "启动 $concurrent 个并发线程，持续 $duration 秒..."
    local start_time=$(date +%s)
    local end_target=$((start_time + duration))
    
    for thread in $(seq 1 $concurrent); do
        (
            local thread_counter=0
            while [ $(date +%s) -lt $end_target ]; do
                local log_data=$(generate_log "$thread-$thread_counter")
                send_log "$log_data"
                thread_counter=$((thread_counter + 1))
            done
            echo $thread_counter > /tmp/benchmark_thread_${thread}.count
        ) &
    done
    
    # 等待所有线程完成
    wait
    
    # 统计总数
    local total=0
    for thread in $(seq 1 $concurrent); do
        if [ -f /tmp/benchmark_thread_${thread}.count ]; then
            local count=$(cat /tmp/benchmark_thread_${thread}.count)
            total=$((total + count))
            rm -f /tmp/benchmark_thread_${thread}.count
        fi
    done
    
    # 等待最后的flush
    sleep 2
    
    local avg_tps=$(echo "scale=2; $total / $duration" | bc)
    log_success "吞吐量测试: 总日志 $total 条, 平均 TPS: ${avg_tps}/s"
}

# 测试4: 边界条件测试
test_edge_cases() {
    print_separator
    log_info "测试4: 边界条件测试"
    print_separator
    
    # 4.1 空数据测试
    log_info "4.1 发送空JSON测试"
    local response=$(curl -s -w "\n%{http_code}" -X POST "${COLLECTOR_URL}/ingest" \
        -H "Content-Type: application/json" \
        -d '{}')
    local http_code=$(echo "$response" | tail -n1)
    if [ "$http_code" = "200" ]; then
        log_success "空JSON处理正常: HTTP $http_code"
    else
        log_warning "空JSON处理异常: HTTP $http_code"
    fi
    
    # 4.2 超长字段测试
    log_info "4.2 超长字段测试"
    local long_string=$(python3 -c "print('A' * 10000)")
    local log_data=$(generate_log 1 "$long_string" "GET" 200)
    local response=$(curl -s -w "\n%{http_code}" -X POST "${COLLECTOR_URL}/ingest" \
        -H "Content-Type: application/json" \
        -d "$log_data")
    local http_code=$(echo "$response" | tail -n1)
    if [ "$http_code" = "200" ]; then
        log_success "超长字段处理正常: HTTP $http_code"
    else
        log_warning "超长字段处理异常: HTTP $http_code"
    fi
    
    # 4.3 快速连续发送 (触发条数flush)
    log_info "4.3 快速发送50条日志 (触发条数flush)"
    local start_time=$(date +%s.%N)
    for i in $(seq 1 50); do
        local log_data=$(generate_log $i)
        send_log "$log_data" &
    done
    wait
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    log_success "快速发送50条: 耗时 ${duration}s"
    
    sleep 2
    
    # 4.4 慢速发送 (触发定时flush)
    log_info "4.4 慢速发送10条日志 (间隔200ms, 触发定时flush)"
    local start_time=$(date +%s.%N)
    for i in $(seq 1 10); do
        local log_data=$(generate_log $i)
        send_log "$log_data"
        sleep 0.2
    done
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    log_success "慢速发送10条: 耗时 ${duration}s"
    
    sleep 3
}

# 测试5: 不同HTTP状态码分布测试
test_status_code_distribution() {
    print_separator
    log_info "测试5: 不同HTTP状态码分布测试"
    print_separator
    
    local status_codes=(200 201 400 404 500 503)
    local logs_per_status=20
    
    log_info "发送不同状态码的日志各 $logs_per_status 条"
    local start_time=$(date +%s.%N)
    
    for status in "${status_codes[@]}"; do
        for i in $(seq 1 $logs_per_status); do
            local log_data=$(generate_log "$status-$i" "/api/test" "GET" $status)
            send_log "$log_data"
        done
    done
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    local total=$((${#status_codes[@]} * logs_per_status))
    local tps=$(echo "scale=2; $total / $duration" | bc)
    
    log_success "状态码分布测试: 总 $total 条, 耗时: ${duration}s, TPS: ${tps}/s"
    
    sleep 2
}

# 主函数
main() {
    print_separator
    log_info "Batch 性能测试开始"
    log_info "时间: $(date)"
    log_info "Collector URL: ${COLLECTOR_URL}"
    log_info "报告文件: ${REPORT_FILE}"
    print_separator
    echo ""
    
    # 检查服务
    check_service
    echo ""
    
    # 运行测试
    test_batch_sizes
    echo ""
    
    test_concurrent_writes
    echo ""
    
    test_throughput
    echo ""
    
    test_edge_cases
    echo ""
    
    test_status_code_distribution
    echo ""
    
    # 总结
    print_separator
    log_success "所有测试完成！"
    log_info "详细报告: ${REPORT_FILE}"
    print_separator
    
    # 建议查看collector日志
    echo ""
    log_info "建议操作："
    log_info "1. 查看 log-collector 日志中的 [Batch] 相关输出"
    log_info "2. 检查 MySQL 数据库中的日志记录数"
    log_info "3. 分析不同场景下的 flush 触发方式（条数触发 vs 定时触发）"
    log_info "4. 观察批次大小、耗时、TPS 等性能指标"
}

# 执行主函数
main "$@"
