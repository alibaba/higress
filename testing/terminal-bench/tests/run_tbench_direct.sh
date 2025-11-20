#!/bin/bash
#
# Terminal-Bench 直接测试脚本
# Terminal-Bench 直接调用 Higress Gateway，无需额外的 Agent 应用
#

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
HIGRESS_ENDPOINT="${HIGRESS_ENDPOINT:-http://localhost:8080}"
API_KEY="${OPENAI_API_KEY:-your-api-key}"
MODEL="${MODEL:-deepseek-chat}"
RESULTS_DIR="./results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULT_FILE="${RESULTS_DIR}/tbench_${TIMESTAMP}.txt"

# 创建结果目录
mkdir -p "${RESULTS_DIR}"

echo -e "${GREEN}=== Terminal-Bench 直接测试 ===${NC}"
echo "Higress Endpoint: ${HIGRESS_ENDPOINT}"
echo "Model: ${MODEL}"
echo "结果文件: ${RESULT_FILE}"
echo ""

# 检查依赖
echo -e "${YELLOW}检查依赖...${NC}"
if ! command -v tb &> /dev/null; then
    echo -e "${RED}错误: tbench 未安装${NC}"
    echo "请运行: pip install tbench"
    exit 1
fi

# 检查 Higress 是否可访问
echo -e "${YELLOW}检查 Higress 连接...${NC}"
if ! curl -s -o /dev/null -w "%{http_code}" "${HIGRESS_ENDPOINT}/health" | grep -q "200"; then
    echo -e "${YELLOW}警告: 无法访问 ${HIGRESS_ENDPOINT}/health${NC}"
    echo "确保 Higress Gateway 正在运行"
fi

# 检查 Redis
echo -e "${YELLOW}检查 Redis 连接...${NC}"
if command -v redis-cli &> /dev/null; then
    if redis-cli ping &> /dev/null; then
        echo -e "${GREEN}Redis 连接正常${NC}"
        echo "当前 Redis keys:"
        redis-cli KEYS "memory:*" | head -5
    else
        echo -e "${YELLOW}警告: Redis 连接失败${NC}"
    fi
fi

echo ""

# 测试任务列表
EASY_TASKS=(
    "vim-terminal-task"
)

MEDIUM_TASKS=(
    "blind-maze-explorer-5x5"
    "tmux-advanced-workflow"
)

HARD_TASKS=(
    "blind-maze-explorer-algorithm"
)

# 运行测试的函数
run_test() {
    local task=$1
    local difficulty=$2
    
    echo -e "${YELLOW}[${difficulty}] 运行测试: ${task}${NC}" | tee -a "${RESULT_FILE}"
    
    # 运行 tbench（直接调用 Higress）
    if tb run \
        -d terminal-bench-core==0.1.1 \
        -t "${task}" \
        -a "openai:${HIGRESS_ENDPOINT}" \
        -m "${MODEL}" \
        2>&1 | tee -a "${RESULT_FILE}"; then
        echo -e "${GREEN}✓ ${task} 通过${NC}" | tee -a "${RESULT_FILE}"
        return 0
    else
        echo -e "${RED}✗ ${task} 失败${NC}" | tee -a "${RESULT_FILE}"
        return 1
    fi
}

# 运行测试
echo -e "${GREEN}=== 运行 Easy 任务 ===${NC}" | tee -a "${RESULT_FILE}"
EASY_PASSED=0
EASY_FAILED=0
for task in "${EASY_TASKS[@]}"; do
    if run_test "${task}" "EASY"; then
        ((EASY_PASSED++))
    else
        ((EASY_FAILED++))
    fi
    echo "" | tee -a "${RESULT_FILE}"
done

echo -e "${GREEN}=== 运行 Medium 任务 ===${NC}" | tee -a "${RESULT_FILE}"
MEDIUM_PASSED=0
MEDIUM_FAILED=0
for task in "${MEDIUM_TASKS[@]}"; do
    if run_test "${task}" "MEDIUM"; then
        ((MEDIUM_PASSED++))
    else
        ((MEDIUM_FAILED++))
    fi
    echo "" | tee -a "${RESULT_FILE}"
done

echo -e "${GREEN}=== 运行 Hard 任务 ===${NC}" | tee -a "${RESULT_FILE}"
HARD_PASSED=0
HARD_FAILED=0
for task in "${HARD_TASKS[@]}"; do
    if run_test "${task}" "HARD"; then
        ((HARD_PASSED++))
    else
        ((HARD_FAILED++))
    fi
    echo "" | tee -a "${RESULT_FILE}"
done

# 统计结果
TOTAL_PASSED=$((EASY_PASSED + MEDIUM_PASSED + HARD_PASSED))
TOTAL_FAILED=$((EASY_FAILED + MEDIUM_FAILED + HARD_FAILED))
TOTAL_TESTS=$((TOTAL_PASSED + TOTAL_FAILED))

if [ ${TOTAL_TESTS} -eq 0 ]; then
    PASS_RATE=0
else
    PASS_RATE=$(awk "BEGIN {printf \"%.2f\", ($TOTAL_PASSED / $TOTAL_TESTS) * 100}")
fi

# 输出总结
echo "" | tee -a "${RESULT_FILE}"
echo -e "${GREEN}=== 测试总结 ===${NC}" | tee -a "${RESULT_FILE}"
echo "Easy 任务: ${EASY_PASSED}/$((EASY_PASSED + EASY_FAILED)) 通过" | tee -a "${RESULT_FILE}"
echo "Medium 任务: ${MEDIUM_PASSED}/$((MEDIUM_PASSED + MEDIUM_FAILED)) 通过" | tee -a "${RESULT_FILE}"
echo "Hard 任务: ${HARD_PASSED}/$((HARD_PASSED + HARD_FAILED)) 通过" | tee -a "${RESULT_FILE}"
echo "" | tee -a "${RESULT_FILE}"
echo "总计: ${TOTAL_PASSED}/${TOTAL_TESTS} 通过" | tee -a "${RESULT_FILE}"
echo "通过率: ${PASS_RATE}%" | tee -a "${RESULT_FILE}"
echo "" | tee -a "${RESULT_FILE}"
echo "详细结果已保存到: ${RESULT_FILE}" | tee -a "${RESULT_FILE}"

# 检查 Redis 中的压缩统计
if command -v redis-cli &> /dev/null; then
    echo "" | tee -a "${RESULT_FILE}"
    echo -e "${GREEN}=== Redis 压缩统计 ===${NC}" | tee -a "${RESULT_FILE}"
    CONTEXT_COUNT=$(redis-cli KEYS "memory:context:*" | wc -l)
    SUMMARY_COUNT=$(redis-cli KEYS "memory:summary:*" | wc -l)
    echo "压缩的上下文数量: ${CONTEXT_COUNT}" | tee -a "${RESULT_FILE}"
    echo "生成的摘要数量: ${SUMMARY_COUNT}" | tee -a "${RESULT_FILE}"
fi

# 返回状态码
if [ ${TOTAL_FAILED} -eq 0 ]; then
    echo -e "${GREEN}所有测试通过！${NC}"
    exit 0
else
    echo -e "${YELLOW}部分测试失败，通过率: ${PASS_RATE}%${NC}"
    exit 1
fi

