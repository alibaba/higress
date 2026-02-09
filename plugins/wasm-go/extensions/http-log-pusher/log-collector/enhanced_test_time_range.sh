#!/bin/bash

# 增强版时间范围查询测试脚本
# 包含对新API端点的测试

set -e

COLLECTOR_URL="${COLLECTOR_URL:-http://localhost:8080}"

echo "=========================================="
echo "增强版时间范围查询测试"
echo "=========================================="
echo ""

# 1. 基础健康检查
echo "[1] 健康检查..."
if curl -s -f "${COLLECTOR_URL}/health" > /dev/null; then
    echo "✅ 服务健康检查通过"
else
    echo "❌ 服务不可用: ${COLLECTOR_URL}"
    exit 1
fi
echo ""

# 2. 查询数据库中的时间范围
echo "[2] 查询数据库中现有数据的时间范围..."
response=$(curl -s "${COLLECTOR_URL}/query?page_size=1&sort_by=start_time&sort_order=ASC")
oldest=$(echo "$response" | jq -r '.logs[0].start_time // "N/A"')
echo "最早时间: $oldest"

response=$(curl -s "${COLLECTOR_URL}/query?page_size=1&sort_by=start_time&sort_order=DESC")
latest=$(echo "$response" | jq -r '.logs[0].start_time // "N/A"')
echo "最晚时间: $latest"
echo ""

# 3. 基于实际数据测试时间范围查询
if [ "$oldest" != "N/A" ] && [ "$oldest" != "null" ]; then
    echo "[3] 测试时间范围查询 (基于实际数据)..."
    
    # 提取时间部分（去掉 T 和 Z）
    start_time=$(echo "$oldest" | sed 's/T/ /' | sed 's/Z//')
    end_time=$(echo "$latest" | sed 's/T/ /' | sed 's/Z//')
    
    echo "查询范围: start=$start_time, end=$end_time"
    
    # URL 编码空格为 %20
    start_encoded=$(echo "$start_time" | sed 's/ /%20/g')
    end_encoded=$(echo "$end_time" | sed 's/ /%20/g')
    
    response=$(curl -s "${COLLECTOR_URL}/query?start=${start_encoded}&end=${end_encoded}")
    total=$(echo "$response" | jq -r '.total')
    status=$(echo "$response" | jq -r '.status')
    
    echo "结果: total=$total, status=$status"
    
    if [ "$status" = "success" ] && [ "$total" -gt 0 ]; then
        echo "✅ 时间范围查询成功"
    else
        echo "❌ 时间范围查询失败"
        echo "响应: $response"
    fi
else
    echo "[3] 数据库中没有数据，跳过测试"
fi
echo ""

# 4. 测试不同时间格式
echo "[4] 测试不同时间格式..."
test_formats=(
    "标准格式|2026-02-05 02:00:00|2026-02-05 03:00:00"
    "ISO 8601 格式|2026-02-05T02:00:00|2026-02-05T03:00:00"
    "带时区格式|2026-02-05T02:00:00Z|2026-02-05T03:00:00Z"
)

for test_case in "${test_formats[@]}"; do
    IFS='|' read -r desc start end <<< "$test_case"
    echo "  4.1 $desc: $start"
    
    # URL编码
    start_enc=$(echo "$start" | sed 's/ /%20/g' | sed 's/:/%3A/g')
    end_enc=$(echo "$end" | sed 's/ /%20/g' | sed 's/:/%3A/g')
    
    response=$(curl -s "${COLLECTOR_URL}/query?start=${start_enc}&end=${end_enc}")
    total=$(echo "$response" | jq -r '.total')
    status=$(echo "$response" | jq -r '.status')
    echo "      结果: total=$total, status=$status"
done
echo ""

# 5. 测试新API端点
echo "[5] 测试新API端点..."

# 5.1 测试 /batch/kpi 端点
echo "  5.1 测试 /batch/kpi 端点"
kpi_payload='[
    {
        "timeRange": {
            "start": "2026-02-05 00:00:00",
            "end": "2026-02-06 00:00:00"
        },
        "bizType": "MODEL_API",
        "filters": {
            "model": "qwen-turbo"
        }
    },
    {
        "timeRange": {
            "start": "2026-02-05 00:00:00",
            "end": "2026-02-06 00:00:00"
        },
        "bizType": "MCP_SERVER",
        "filters": {
            "mcp_server": "mcp-server-1"
        }
    }
]'

response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$kpi_payload" \
    "${COLLECTOR_URL}/batch/kpi")

status=$(echo "$response" | jq -r '.status')
echo "      KPI批量查询状态: $status"

if [ "$status" = "success" ]; then
    echo "      ✅ /batch/kpi 测试通过"
    # 显示第一个查询的结果
    pv=$(echo "$response" | jq -r '.data.query_0.data.pv // 0')
    uv=$(echo "$response" | jq -r '.data.query_0.data.uv // 0')
    echo "      Model API PV: $pv, UV: $uv"
else
    echo "      ❌ /batch/kpi 测试失败"
    echo "      响应: $response"
fi
echo ""

# 5.2 测试 /batch/chart 端点
echo "  5.2 测试 /batch/chart 端点"
chart_payload='[
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

response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$chart_payload" \
    "${COLLECTOR_URL}/batch/chart")

status=$(echo "$response" | jq -r '.status')
echo "      Chart批量查询状态: $status"

if [ "$status" = "success" ]; then
    echo "      ✅ /batch/chart 测试通过"
    # 检查时间戳数据
    timestamps_count=$(echo "$response" | jq -r '.data.query_0.data.timestamps | length')
    echo "      时间点数量: $timestamps_count"
else
    echo "      ❌ /batch/chart 测试失败"
    echo "      响应: $response"
fi
echo ""

# 5.3 测试 /batch/table 端点
echo "  5.3 测试 /batch/table 端点"
table_payload='[
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

response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$table_payload" \
    "${COLLECTOR_URL}/batch/table")

status=$(echo "$response" | jq -r '.status')
echo "      Table批量查询状态: $status"

if [ "$status" = "success" ]; then
    echo "      ✅ /batch/table 测试通过"
    # 显示表格数据条数
    model_stats_count=$(echo "$response" | jq -r '.data.query_0.data.data | length')
    method_dist_count=$(echo "$response" | jq -r '.data.query_1.data.data | length')
    echo "      模型统计条数: $model_stats_count"
    echo "      方法分布条数: $method_dist_count"
else
    echo "      ❌ /batch/table 测试失败"
    echo "      响应: $response"
fi
echo ""

# 6. 边界条件测试
echo "[6] 边界条件测试..."

# 6.1 测试未来时间范围
echo "  6.1 测试未来时间范围"
future_start=$(date -u -v+1d +"%Y-%m-%d %H:%M:%S" 2>/dev/null || date -u -d '1 day' +"%Y-%m-%d %H:%M:%S")
future_end=$(date -u -v+2d +"%Y-%m-%d %H:%M:%S" 2>/dev/null || date -u -d '2 days' +"%Y-%m-%d %H:%M:%S")

start_enc=$(echo "$future_start" | sed 's/ /%20/g')
end_enc=$(echo "$future_end" | sed 's/ /%20/g')

response=$(curl -s "${COLLECTOR_URL}/query?start=${start_enc}&end=${end_enc}")
total=$(echo "$response" | jq -r '.total')
echo "      未来时间查询结果: total=$total (预期为0)"

# 6.2 测试无效时间格式
echo "  6.2 测试无效时间格式"
response=$(curl -s "${COLLECTOR_URL}/query?start=invalid-date&end=another-invalid")
error=$(echo "$response" | jq -r '.error // "no error"')
echo "      无效时间格式错误: $error"

# 6.3 测试重叠时间范围
echo "  6.3 测试重叠时间范围"
response=$(curl -s "${COLLECTOR_URL}/query?start=${end_enc}&end=${start_enc}")
total=$(echo "$response" | jq -r '.total')
echo "      重叠时间范围结果: total=$total"

echo ""
echo "=========================================="
echo "增强版测试完成"
echo "=========================================="