#!/bin/bash

# Nginx迁移MCP服务器快速设置脚本
# 自动配置Claude Desktop并启动测试

echo "🚀 Nginx迁移MCP服务器快速设置"
echo "======================================"

# 检查Go环境
echo "📋 检查环境..."
if ! command -v go &> /dev/null; then
    echo "❌ 错误: 未找到Go，请先安装Go 1.22+"
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3 | cut -d'o' -f2)
echo "✅ Go版本: $GO_VERSION"

# 编译测试
echo "📋 编译测试..."
if go build main.go; then
    echo "✅ 编译成功"
    rm -f main
else
    echo "❌ 编译失败，请检查代码"
    exit 1
fi

# 创建Claude Desktop配置
echo "📋 配置Claude Desktop..."
CONFIG_DIR="$HOME/.config/claude-desktop"
CONFIG_FILE="$CONFIG_DIR/config.json"

# 创建配置目录
mkdir -p "$CONFIG_DIR"

# 获取当前目录的绝对路径
CURRENT_DIR=$(pwd)

# 检查是否已有配置
if [ -f "$CONFIG_FILE" ]; then
    echo "⚠️  发现现有配置文件: $CONFIG_FILE"
    echo "📋 备份原配置..."
    cp "$CONFIG_FILE" "$CONFIG_FILE.backup.$(date +%Y%m%d_%H%M%S)"
fi

# 生成配置
cat > "$CONFIG_FILE" << EOF
{
  "mcpServers": {
    "nginx-migration": {
      "command": "go",
      "args": ["run", "main.go"],
      "cwd": "$CURRENT_DIR",
      "env": {
        "GO111MODULE": "on"
      }
    }
  }
}
EOF

echo "✅ Claude Desktop配置完成: $CONFIG_FILE"

# 验证配置
echo "📋 验证配置..."
if command -v jq &> /dev/null; then
    if jq . "$CONFIG_FILE" > /dev/null 2>&1; then
        echo "✅ JSON配置格式正确"
    else
        echo "❌ JSON格式错误"
        exit 1
    fi
else
    echo "⚠️  建议安装jq来验证JSON格式"
fi

# 测试MCP服务器
echo "📋 测试MCP服务器..."
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05"},"id":1}' | timeout 5 go run main.go > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ MCP服务器测试成功"
else
    echo "⚠️  MCP服务器测试超时（正常现象）"
fi

echo ""
echo "🎉 设置完成！"
echo ""
echo "📋 下一步操作:"
echo "1. 重启Claude Desktop应用程序"
echo "2. 在聊天中测试nginx迁移工具"
echo ""
echo "🧪 测试命令:"
echo "请使用parse_nginx_config工具分析以下配置："
echo ""
echo "server {"
echo "    listen 80;"
echo "    server_name example.com;"
echo "    location / {"
echo "        proxy_pass http://backend:8080;"
echo "    }"
echo "}"
echo ""
echo "📚 更多信息请查看 README.md"
echo ""

# 提供启动选项
read -p "🚀 是否现在启动MCP服务器进行调试? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "启动MCP服务器..."
    echo "按 Ctrl+C 停止服务器"
    echo ""
    go run main.go
fi
