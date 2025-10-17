.PHONY: all standalone integration clean test help

# 默认目标
all: standalone

# 编译独立模式
standalone:
	@echo "🔨 编译独立模式..."
	@cd cmd/standalone && go build -o ../../nginx-migration-mcp
	@echo "✅ 独立模式编译完成: ./nginx-migration-mcp"

# 清理编译产物
clean:
	@echo "🧹 清理编译产物..."
	@rm -f nginx-migration-mcp
	@echo "✅ 清理完成"

# 运行测试
test:
	@echo "🧪 运行测试..."
	@go test ./...
	@echo "✅ 测试完成"

# 测试独立模式
test-standalone: standalone
	@echo "🧪 测试独立模式..."
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | timeout 2 ./nginx-migration-mcp
	@echo ""
	@echo "✅ 独立模式测试完成"

# 安装依赖
deps:
	@echo "📦 安装依赖..."
	@go mod tidy
	@echo "✅ 依赖安装完成"

# 格式化代码
fmt:
	@echo "💅 格式化代码..."
	@go fmt ./...
	@echo "✅ 代码格式化完成"

# 显示帮助
help:
	@echo "Nginx Migration MCP Server - Makefile"
	@echo ""
	@echo "可用目标:"
	@echo "  make                  - 编译独立模式（默认）"
	@echo "  make standalone       - 编译独立模式"
	@echo "  make test             - 运行测试"
	@echo "  make test-standalone  - 测试独立模式"
	@echo "  make clean            - 清理编译产物"
	@echo "  make deps             - 安装依赖"
	@echo "  make fmt              - 格式化代码"
	@echo "  make help             - 显示此帮助信息"
	@echo ""
	@echo "目录结构:"
	@echo "  cmd/standalone/       - 独立模式入口"
	@echo "  internal/standalone/  - 独立模式实现"
	@echo "  integration/          - Higress MCP 框架集成"
	@echo "  tools/                - 共享核心逻辑"

