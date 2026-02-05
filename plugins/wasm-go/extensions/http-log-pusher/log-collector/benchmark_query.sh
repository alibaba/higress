#!/bin/bash

# Query 性能测试脚本
# 测试场景：
# 1. 不同查询条件的性能对比
# 2. 分页查询性能
# 3. 排序查询性能
# 4. 索引效果验证
# 5. 并发查询测试

set -e

# 配置
COLLECTOR_URL="${COLLECTOR_URL:-http://localhost:8080}"
REPORT_DIR="./benchmark_reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$REPORT_DIR/query_benchmark_${TIMESTAMP}.txt"

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

# 执行查询并测量性能
query_and_measure() {
    local query_string="$1"
    local description="$2"
    
    local start_time=$(date +%s.%N)
    local response=$(curl -s -w "\n%{time_total}" "${COLLECTOR_URL}/query?${query_string}")
    local end_time=$(date +%s.%N)
    
    local curl_time=$(echo "$response" | tail -n1)
    # 使用兼容 macOS 和 Linux 的方法提取除最后一行外的所有内容
    local json_response=$(echo "$response" | sed '$d')
    local total=$(echo "$json_response" | jq -r '.total // 0')
    local returned=$(echo "$json_response" | jq -r '.logs | length')
    local response_code=$(echo "$json_response" | jq -r '.status // "unknown"')
    
    local duration=$(echo "$end_time - $start_time" | bc)
    
    if [ "$response_code" = "success" ]; then
        log_success "$description: total=$total, returned=$returned, duration=${duration}s (curl=${curl_time}s)"
    else
        local error=$(echo "$json_response" | jq -r '.error // "unknown error"')
        log_error "$description: FAILED - $error"
    fi
    
    echo "$duration"
}

# 测试1: 无条件查询（全表扫描）
test_full_scan() {
    print_separator
    log_info "测试1: 全表扫描查询"
    print_separator
    
    log_info "1.1 查询所有记录 (page=1, page_size=10)"
    query_and_measure "" "无条件查询"
    
    log_info "1.2 查询所有记录 (page=1, page_size=50)"
    query_and_measure "page_size=50" "大页面查询"
    
    log_info "1.3 查询所有记录 (page=1, page_size=100)"
    query_and_measure "page_size=100" "最大页面查询"
}

# 测试2: 索引字段查询
test_indexed_queries() {
    print_separator
    log_info "测试2: 索引字段查询 (验证索引效果)"
    print_separator
    
    # 假设有索引的字段
    log_info "2.1 按 trace_id 精确查询"
    query_and_measure "trace_id=trace-0000000000000001" "trace_id查询"
    
    log_info "2.2 按 start_time 范围查询"
    local start_date=$(date -u -v-1H +"%Y-%m-%d %H:%M:%S" 2>/dev/null || date -u -d '1 hour ago' +"%Y-%m-%d %H:%M:%S")
    local end_date=$(date -u +"%Y-%m-%d %H:%M:%S")
    query_and_measure "start=${start_date}&end=${end_date}" "时间范围查询"
    
    log_info "2.3 按 response_code 查询"
    query_and_measure "response_code=200" "状态码查询(200)"
    query_and_measure "response_code=404" "状态码查询(404)"
    query_and_measure "response_code=500" "状态码查询(500)"
    
    log_info "2.4 按 authority 查询"
    query_and_measure "service=test-service.default.svc.cluster.local" "服务名查询"
}

# 测试3: 非索引字段查询
test_non_indexed_queries() {
    print_separator
    log_info "测试3: 非索引字段查询 (对比索引效果)"
    print_separator
    
    log_info "3.1 按 path 模糊查询"
    query_and_measure "path=/api/test" "路径模糊查询"
    
    log_info "3.2 按 method 查询"
    query_and_measure "method=GET" "HTTP方法查询(GET)"
    query_and_measure "method=POST" "HTTP方法查询(POST)"
    
    log_info "3.3 组合条件查询"
    query_and_measure "method=GET&response_code=200&path=/api" "多条件组合查询"
}

# 测试4: 分页性能
test_pagination() {
    print_separator
    log_info "测试4: 分页查询性能"
    print_separator
    
    local page_sizes=(10 20 50 100)
    
    for size in "${page_sizes[@]}"; do
        log_info "4.1 Page size=$size, page=1"
        query_and_measure "page_size=$size&page=1" "首页(size=$size)"
        
        log_info "4.2 Page size=$size, page=5"
        query_and_measure "page_size=$size&page=5" "第5页(size=$size)"
        
        log_info "4.3 Page size=$size, page=10"
        query_and_measure "page_size=$size&page=10" "第10页(size=$size)"
    done
}

# 测试5: 排序性能
test_sorting() {
    print_separator
    log_info "测试5: 排序查询性能"
    print_separator
    
    local sort_fields=("start_time" "response_code" "duration" "bytes_sent")
    
    for field in "${sort_fields[@]}"; do
        log_info "5.1 按 $field 升序排序"
        query_and_measure "sort_by=$field&sort_order=ASC&page_size=50" "排序($field ASC)"
        
        log_info "5.2 按 $field 降序排序"
        query_and_measure "sort_by=$field&sort_order=DESC&page_size=50" "排序($field DESC)"
    done
}

# 测试6: 并发查询
test_concurrent_queries() {
    print_separator
    log_info "测试6: 并发查询测试"
    print_separator
    
    local concurrent_levels=(1 5 10 20)
    local queries_per_thread=10
    
    for level in "${concurrent_levels[@]}"; do
        log_info "测试并发级别: $level 线程, 每线程 $queries_per_thread 次查询"
        local start_time=$(date +%s.%N)
        
        # 并发查询
        for thread in $(seq 1 $level); do
            (
                for i in $(seq 1 $queries_per_thread); do
                    curl -s "${COLLECTOR_URL}/query?page_size=10&page=$i" > /dev/null
                done
            ) &
        done
        
        # 等待所有后台任务完成
        wait
        
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc)
        local total_queries=$((level * queries_per_thread))
        local qps=$(echo "scale=2; $total_queries / $duration" | bc)
        
        log_success "并发: $level, 总查询: $total_queries, 耗时: ${duration}s, QPS: ${qps}/s"
    done
}

# 测试7: 复杂查询场景
test_complex_queries() {
    print_separator
    log_info "测试7: 复杂查询场景"
    print_separator
    
    log_info "7.1 时间范围 + 状态码 + 分页 + 排序"
    local start_date=$(date -u -v-1H +"%Y-%m-%d %H:%M:%S" 2>/dev/null || date -u -d '1 hour ago' +"%Y-%m-%d %H:%M:%S")
    local end_date=$(date -u +"%Y-%m-%d %H:%M:%S")
    query_and_measure "start=${start_date}&end=${end_date}&response_code=200&page_size=50&sort_by=duration&sort_order=DESC" "复杂查询1"
    
    log_info "7.2 服务名 + 方法 + 路径 + 分页"
    query_and_measure "service=test-service.default.svc.cluster.local&method=GET&path=/api&page_size=20" "复杂查询2"
    
    log_info "7.3 时间范围 + 方法 + 状态码 + 排序"
    query_and_measure "start=${start_date}&end=${end_date}&method=POST&response_code=500&sort_by=start_time&sort_order=DESC" "复杂查询3"
}

# 测试8: 压力测试
test_stress() {
    print_separator
    log_info "测试8: 查询压力测试 (持续30秒)"
    print_separator
    
    local duration=30
    local concurrent=10
    
    log_info "启动 $concurrent 个并发线程，持续 $duration 秒..."
    local start_time=$(date +%s)
    local end_target=$((start_time + duration))
    
    for thread in $(seq 1 $concurrent); do
        (
            local thread_counter=0
            while [ $(date +%s) -lt $end_target ]; do
                local page=$((thread_counter % 10 + 1))
                curl -s "${COLLECTOR_URL}/query?page=$page&page_size=20" > /dev/null
                thread_counter=$((thread_counter + 1))
            done
            echo $thread_counter > /tmp/query_thread_${thread}.count
        ) &
    done
    
    # 等待所有线程完成
    wait
    
    # 统计总数
    local total=0
    for thread in $(seq 1 $concurrent); do
        if [ -f /tmp/query_thread_${thread}.count ]; then
            local count=$(cat /tmp/query_thread_${thread}.count)
            total=$((total + count))
            rm -f /tmp/query_thread_${thread}.count
        fi
    done
    
    local avg_qps=$(echo "scale=2; $total / $duration" | bc)
    log_success "压力测试: 总查询 $total 次, 平均 QPS: ${avg_qps}/s"
}

# 测试9: 边界条件
test_edge_cases() {
    print_separator
    log_info "测试9: 边界条件测试"
    print_separator
    
    log_info "9.1 无效参数测试"
    query_and_measure "page=-1" "负数页码"
    query_and_measure "page_size=0" "零页面大小"
    query_and_measure "page_size=1000" "超大页面"
    
    log_info "9.2 不存在的数据查询"
    query_and_measure "trace_id=nonexistent-trace-id" "不存在的trace_id"
    query_and_measure "response_code=999" "不存在的状态码"
    
    log_info "9.3 特殊字符查询"
    query_and_measure "path=%2F%3F%26%3D" "URL编码路径"
}

# 生成性能对比表
generate_summary() {
    print_separator
    log_info "性能测试总结"
    print_separator
    
    log_info "建议关注指标："
    log_info "1. COUNT 查询耗时 - 反映索引效果"
    log_info "2. SELECT 查询耗时 - 反映数据读取性能"
    log_info "3. Rows 扫描耗时 - 反映数据解析性能"
    log_info "4. 总耗时分布 - count + query + scan"
    log_info "5. 不同查询条件的性能差异"
    log_info "6. 分页和排序对性能的影响"
}

# 主函数
main() {
    print_separator
    log_info "Query 性能测试开始"
    log_info "时间: $(date)"
    log_info "Collector URL: ${COLLECTOR_URL}"
    log_info "报告文件: ${REPORT_FILE}"
    print_separator
    echo ""
    
    # 检查依赖
    if ! command -v jq &> /dev/null; then
        log_error "jq 未安装，请先安装: brew install jq (macOS) 或 apt-get install jq (Ubuntu)"
        exit 1
    fi
    
    # 检查服务
    check_service
    echo ""
    
    # 运行测试
    test_full_scan
    echo ""
    
    test_indexed_queries
    echo ""
    
    test_non_indexed_queries
    echo ""
    
    test_pagination
    echo ""
    
    test_sorting
    echo ""
    
    test_concurrent_queries
    echo ""
    
    test_complex_queries
    echo ""
    
    test_stress
    echo ""
    
    test_edge_cases
    echo ""
    
    generate_summary
    echo ""
    
    # 总结
    print_separator
    log_success "所有测试完成！"
    log_info "详细报告: ${REPORT_FILE}"
    print_separator
    
    # 建议
    echo ""
    log_info "建议操作："
    log_info "1. 查看 log-collector 日志中的 [Query] 相关输出"
    log_info "2. 分析 COUNT、SELECT、Scan 各阶段的耗时"
    log_info "3. 对比索引字段和非索引字段的查询性能"
    log_info "4. 检查是否需要添加索引优化查询性能"
    log_info "5. 分析分页和排序对性能的影响"
}

# 执行主函数
main "$@"
