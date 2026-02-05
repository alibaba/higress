#!/bin/bash

# 时间范围查询测试脚本

set -e

COLLECTOR_URL="${COLLECTOR_URL:-http://localhost:8080}"

echo "=========================================="
echo "时间范围查询功能测试"
echo "=========================================="
echo ""

# 1. 查询数据库中的时间范围
echo "[1] 查询数据库中现有数据的时间范围..."
response=$(curl -s "${COLLECTOR_URL}/query?page_size=1&sort_by=start_time&sort_order=ASC")
oldest=$(echo "$response" | jq -r '.logs[0].start_time // "N/A"')
echo "最早时间: $oldest"

response=$(curl -s "${COLLECTOR_URL}/query?page_size=1&sort_by=start_time&sort_order=DESC")
latest=$(echo "$response" | jq -r '.logs[0].start_time // "N/A"')
echo "最晚时间: $latest"
echo ""

# 2. 基于实际数据测试时间范围查询
if [ "$oldest" != "N/A" ] && [ "$oldest" != "null" ]; then
    echo "[2] 测试时间范围查询 (基于实际数据)..."
    
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
    echo "[2] 数据库中没有数据，跳过测试"
fi

echo ""
echo "[3] 测试不同时间格式..."

# 测试 1: 标准格式 YYYY-MM-DD HH:MM:SS
echo "  3.1 标准格式: 2026-02-05 02:00:00"
response=$(curl -s "${COLLECTOR_URL}/query?start=2026-02-05%2002:00:00&end=2026-02-05%2003:00:00")
total=$(echo "$response" | jq -r '.total')
status=$(echo "$response" | jq -r '.status')
echo "      结果: total=$total, status=$status"

# 测试 2: ISO 8601 格式
echo "  3.2 ISO 8601 格式: 2026-02-05T02:00:00"
response=$(curl -s "${COLLECTOR_URL}/query?start=2026-02-05T02:00:00&end=2026-02-05T03:00:00")
total=$(echo "$response" | jq -r '.total')
status=$(echo "$response" | jq -r '.status')
echo "      结果: total=$total, status=$status"

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
