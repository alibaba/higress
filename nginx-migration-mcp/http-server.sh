#!/bin/bash

# HTTP API服务器启动脚本
# 将MCP服务器运行为HTTP API，支持远程访问

echo "🚀 启动Nginx迁移HTTP API服务器..."

# 检查端口参数
PORT=${1:-8080}
HOST=${2:-"0.0.0.0"}

echo "📋 配置信息:"
echo "  - 监听地址: $HOST:$PORT"
echo "  - API文档: http://localhost:$PORT"
echo "  - 健康检查: http://localhost:$PORT/health"
echo ""

# 构建并启动服务器
echo "📦 构建HTTP服务器..."
go build -o nginx-migration-http-server http-server.go

if [ $? -ne 0 ]; then
    echo "❌ 构建失败，请检查Go环境"
    exit 1
fi

echo "✅ 构建成功"
echo ""

# 启动服务器
echo "🎯 启动服务器 (按Ctrl+C停止)..."
./nginx-migration-http-server $PORT

# 清理
echo ""
echo "🧹 清理临时文件..."
rm -f nginx-migration-http-server
echo "👋 服务器已停止"
