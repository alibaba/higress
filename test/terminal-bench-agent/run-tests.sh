#!/bin/bash
# Terminal-Bench 测试脚本

set -e

# 配置
HIGRESS_ENDPOINT="${HIGRESS_ENDPOINT:-http://localhost:8080/v1/chat/completions}"
HIGRESS_API_KEY="${HIGRESS_API_KEY:-your-api-key}"
MODEL="${MODEL:-deepseek-chat}"
RESULT_DIR="./test-results"

# 创建结果目录
mkdir -p "$RESULT_DIR"

echo "==================================="
echo "Terminal-Bench Integration Testing"
echo "==================================="
echo "Endpoint: $HIGRESS_ENDPOINT"
echo "Model: $MODEL"
echo "==================================="

# 测试函数
run_task() {
    local task_name=$1
    local description=$2
    
    echo ""
    echo ">>> Testing: $task_name"
    echo "    Description: $description"
    
    tb run \
        -d terminal-bench-core==0.1.1 \
        -t "$task_name" \
        -a "higress-agent" \
        -m "$MODEL" \
        --agent-config agent-config.json \
        --output "$RESULT_DIR/$task_name.json" \
        || echo "Task $task_name failed"
}

# 基础测试集
echo ""
echo "=== Phase 1: Basic Tasks ==="
run_task "vim-terminal-task" "File operations with vim"

echo ""
echo "=== Phase 2: Multi-turn Interaction ==="
run_task "blind-maze-explorer-5x5" "Maze exploration with multiple turns"

echo ""
echo "=== Phase 3: Data Processing ==="
run_task "train-fasttext" "Model training with large data"

echo ""
echo "=== Phase 4: Software Engineering ==="
run_task "swe-bench-langcodes" "Code debugging task"

echo ""
echo "=== Phase 5: Full Test Suite (Optional) ==="
read -p "Run all 80 tasks? This will take a long time (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    tb run \
        -d terminal-bench-core==0.1.1 \
        -a "higress-agent" \
        -m "$MODEL" \
        --agent-config agent-config.json \
        --output "$RESULT_DIR/full-results.json"
fi

# 生成报告
echo ""
echo "=== Generating Report ==="
python generate-report.py "$RESULT_DIR"

echo ""
echo "==================================="
echo "Testing completed!"
echo "Results saved to: $RESULT_DIR"
echo "==================================="

