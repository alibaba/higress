#!/bin/bash

# 增强版Query性能测试脚本
# 包含对新API端点的性能测试

set -e

# 配置
COLLECTOR_URL="${COLLECTOR_URL:-http://localhost:8080}"
REPORT_DIR="./benchmark_reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$REPORT_DIR/query_benchmark_enhanced_${TIMESTAMP}.txt"

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
    local endpoint="$1"
    local query_string="$2"
    local description="$3"
    
    local url="${COLLECTOR_URL}${endpoint}"
    if [ -n "$query_string" ]; then
        url="${url}?${query_string}"
    fi
    
    local start_time=$(date +%s.%N)
    local response=$(curl -s -w "\n%{time_total}" "$url")
    local end_time=$(date +%s.%N)
    
    local curl_time=$(echo "$response" | tail -n1)
    # 使用兼容 macOS 和 Linux 的方法提取除最后一行外的所有内容
    local json_response=$(echo "$response" | sed '$d')
    local total=$(echo "$json_response" | jq -r '.total // 0')
    local returned=$(echo "$json_response" | jq -r '.logs | length // .data | length // 0')
    local response_code=$(echo "$json_response" | jq -r '.status // "unknown"')
    
    local duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "0.001")
    
    if [ "$response_code" = "success" ]; then
        log_success "$description: total=$total, returned=$returned, duration=${duration}s (curl=${curl_time}s)"
    else
        local error=$(echo "$json_response" | jq -r '.error // "unknown error"')
        log_error "$description: FAILED - $error"
    fi
    
    echo "$duration"
}

# 测试传统的 /query 端点
test_traditional_query() {
    print_separator
    log_info "测试传统 /query 端点"
    print_separator
    
    log_info "1.1 基础查询性能"
    query_and_measure "/query" "" "无条件查询"
    
    log_info "1.2 分页查询性能"
    query_and_measure "/query" "page_size=50" "50条记录查询"
    query_and_measure "/query" "page_size=100" "100条记录查询"
    
    log_info "1.3 过滤查询性能"
    query_and_measure "/query" "response_code=200" "状态码200过滤"
    query_and_measure "/query" "method=GET" "GET方法过滤"
    
    log_info "1.4 复合查询性能"
    query_and_measure "/query" "response_code=200&method=GET&page_size=50" "复合条件查询"
}

# 测试新的批量API端点
test_batch_apis() {
    print_separator
    log_info "测试新的批量API端点"
    print_separator
    
    # 准备测试数据
    local kpi_payload='[
        {
            "timeRange": {
                "start": "2026-02-05 00:00:00",
                "end": "2026-02-06 00:00:00"
            },
            "bizType": "MODEL_API",
            "filters": {}
        },
        {
            "timeRange": {
                "start": "2026-02-05 00:00:00",
                "end": "2026-02-06 00:00:00"
            },
            "bizType": "MCP_SERVER",
            "filters": {}
        }
    ]'
    
    local chart_payload='[
        {
            "timeRange": {
                "start": "2026-02-05 00:00:00",
                "end": "2026-02-06 00:00:00"
            },
            "interval": "60s",
            "scenario": "success_rate",
            "bizType": "MODEL_API",
            "filters": {}
        },
        {
            "timeRange": {
                "start": "2026-02-05 00:00:00",
                "end": "2026-02-06 00:00:00"
            },
            "interval": "60s",
            "scenario": "qps_total_simple",
            "bizType": "MCP_SERVER",
            "filters": {}
        }
    ]'
    
    local table_payload='[
        {
            "timeRange": {
                "start": "2026-02-05 00:00:00",
                "end": "2026-02-06 00:00:00"
            },
            "tableType": "model_token_stats",
            "bizType": "MODEL_API",
            "filters": {}
        },
        {
            "timeRange": {
                "start": "2026-02-05 00:00:00",
                "end": "2026-02-06 00:00:00"
            },
            "tableType": "method_distribution",
            "bizType": "MCP_SERVER",
            "filters": {}
        }
    ]'
    
    log_info "2.1 测试 /batch/kpi 端点"
    local start_time=$(date +%s.%N)
    local response=$(curl -s -w "\n%{time_total}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$kpi_payload" \
        "${COLLECTOR_URL}/batch/kpi")
    local end_time=$(date +%s.%N)
    
    local curl_time=$(echo "$response" | tail -n1)
    local json_response=$(echo "$response" | sed '$d')
    local status=$(echo "$json_response" | jq -r '.status // "unknown"')
    local duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "0.001")
    
    if [ "$status" = "success" ]; then
        log_success "/batch/kpi: 批量KPI查询成功, duration=${duration}s (curl=${curl_time}s)"
    else
        log_error "/batch/kpi: 查询失败"
    fi
    
    log_info "2.2 测试 /batch/chart 端点"
    start_time=$(date +%s.%N)
    response=$(curl -s -w "\n%{time_total}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$chart_payload" \
        "${COLLECTOR_URL}/batch/chart")
    end_time=$(date +%s.%N)
    
    curl_time=$(echo "$response" | tail -n1)
    json_response=$(echo "$response" | sed '$d')
    status=$(echo "$json_response" | jq -r '.status // "unknown"')
    duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "0.001")
    
    if [ "$status" = "success" ]; then
        log_success "/batch/chart: 批量图表查询成功, duration=${duration}s (curl=${curl_time}s)"
    else
        log_error "/batch/chart: 查询失败"
    fi
    
    log_info "2.3 测试 /batch/table 端点"
    start_time=$(date +%s.%N)
    response=$(curl -s -w "\n%{time_total}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$table_payload" \
        "${COLLECTOR_URL}/batch/table")
    end_time=$(date +%s.%N)
    
    curl_time=$(echo "$response" | tail -n1)
    json_response=$(echo "$response" | sed '$d')
    status=$(echo "$json_response" | jq -r '.status // "unknown"')
    duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "0.001")
    
    if [ "$status" = "success" ]; then
        log_success "/batch/table: 批量表格查询成功, duration=${duration}s (curl=${curl_time}s)"
    else
        log_error "/batch/table: 查询失败"
    fi
}

# 测试并发性能
test_concurrent_performance() {
    print_separator
    log_info "测试并发性能"
    print_separator
    
    local concurrent_levels=(1 5 10)
    local queries_per_thread=5
    
    for level in "${concurrent_levels[@]}"; do
        log_info "测试并发级别: $level 线程, 每线程 $queries_per_thread 次查询"
        local start_time=$(date +%s.%N)
        
        # 并发查询传统端点
        for thread in $(seq 1 $level); do
            (
                for i in $(seq 1 $queries_per_thread); do
                    curl -s "${COLLECTOR_URL}/query?page_size=20&page=$i" > /dev/null
                done
            ) &
        done
        
        # 等待所有后台任务完成
        wait
        
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "0.001")
        local total_queries=$((level * queries_per_thread))
        local qps=$(echo "scale=2; $total_queries / $duration" | bc 2>/dev/null || echo "0")
        
        log_success "传统查询并发: $level 线程, 总查询: $total_queries, 耗时: ${duration}s, QPS: ${qps}/s"
        
        # 测试批量API并发
        start_time=$(date +%s.%N)
        local kpi_payload='{"timeRange":{"start":"2026-02-05 00:00:00","end":"2026-02-06 00:00:00"},"bizType":"MODEL_API","filters":{}}'
        
        for thread in $(seq 1 $level); do
            (
                for i in $(seq 1 $queries_per_thread); do
                    curl -s -X POST \
                        -H "Content-Type: application/json" \
                        -d "[$kpi_payload]" \
                        "${COLLECTOR_URL}/batch/kpi" > /dev/null
                done
            ) &
        done
        
        wait
        
        end_time=$(date +%s.%N)
        duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "0.001")
        qps=$(echo "scale=2; $total_queries / $duration" | bc 2>/dev/null || echo "0")
        
        log_success "批量API并发: $level 线程, 总查询: $total_queries, 耗时: ${duration}s, QPS: ${qps}/s"
    done
}

# 测试大数据量性能
test_large_dataset() {
    print_separator
    log_info "测试大数据量性能"
    print_separator
    
    # 测试不同数据量的查询
    local page_sizes=(10 50 100)
    
    for size in "${page_sizes[@]}"; do
        log_info "测试页面大小: $size"
        query_and_measure "/query" "page_size=$size" "大页面查询($size条)"
    done
    
    # 测试时间范围查询性能
    log_info "测试时间范围查询性能"
    local time_ranges=(
        "1小时|$(date -u -v-1H +"%Y-%m-%d %H:%M:%S" 2>/dev/null || date -u -d '1 hour ago' +"%Y-%m-%d %H:%M:%S")|$(date -u +"%Y-%m-%d %H:%M:%S")"
        "1天|$(date -u -v-1d +"%Y-%m-%d %H:%M:%S" 2>/dev/null || date -u -d '1 day ago' +"%Y-%m-%d %H:%M:%S")|$(date -u +"%Y-%m-%d %H:%M:%S")"
        "7天|$(date -u -v-7d +"%Y-%m-%d %H:%M:%S" 2>/dev/null || date -u -d '7 days ago' +"%Y-%m-%d %H:%M:%S")|$(date -u +"%Y-%m-%d %H:%M:%S")"
    )
    
    for range_spec in "${time_ranges[@]}"; do
        IFS='|' read -r desc start end <<< "$range_spec"
        start_enc=$(echo "$start" | sed 's/ /%20/g')
        end_enc=$(echo "$end" | sed 's/ /%20/g')
        query_and_measure "/query" "start=${start_enc}&end=${end_enc}" "时间范围查询(${desc})"
    done
}

# 生成性能报告
generate_performance_report() {
    print_separator
    log_info "性能测试总结报告"
    print_separator
    
    log_info "测试项目:"
    log_info "1. 传统 /query 端点性能测试"
    log_info "2. 新增批量API端点性能测试 (/batch/kpi, /batch/chart, /batch/table)"
    log_info "3. 并发查询性能对比"
    log_info "4. 大数据量查询性能"
    log_info "5. 不同查询条件的性能表现"
    
    log_info ""
    log_info "关键性能指标:"
    log_info "- 响应时间 (Response Time)"
    log_info "- 吞吐量 (Throughput/QPS)"
    log_info "- 并发处理能力"
    log_info "- 内存使用情况"
    log_info "- CPU使用情况"
    
    log_info ""
    log_info "优化建议:"
    log_info "1. 对频繁查询的字段建立数据库索引"
    log_info "2. 考虑查询结果缓存机制"
    log_info "3. 优化JSON序列化/反序列化性能"
    log_info "4. 实施查询限流和熔断机制"
    log_info "5. 监控慢查询并进行优化"
}

# 主函数
main() {
    print_separator
    log_info "增强版Query性能测试开始"
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
    
    if ! command -v bc &> /dev/null; then
        log_error "bc 未安装，请先安装: brew install bc (macOS) 或 apt-get install bc (Ubuntu)"
        exit 1
    fi
    
    # 检查服务
    check_service
    echo ""
    
    # 运行测试
    test_traditional_query
    echo ""
    
    test_batch_apis
    echo ""
    
    test_concurrent_performance
    echo ""
    
    test_large_dataset
    echo ""
    
    generate_performance_report
    echo ""
    
    # 总结
    print_separator
    log_success "增强版性能测试完成！"
    log_info "详细报告: ${REPORT_FILE}"
    print_separator
    
    # 建议
    echo ""
    log_info "建议操作："
    log_info "1. 查看 log-collector 日志中的性能相关信息"
    log_info "2. 分析传统查询与批量查询的性能差异"
    log_info "3. 监控高并发场景下的系统资源使用"
    log_info "4. 根据测试结果优化数据库索引和查询逻辑"
    log_info "5. 考虑实施缓存策略提升查询性能"
}

# 执行主函数
main "$@"