// Standalone MCP Server for testing nginx migration tools in Claude Desktop
// This is a simplified version that can run independently for testing
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "nginx-migration-test"
	serverVersion = "1.0.0"
)

// Simple nginx migration tools for testing
type NginxMigrationServer struct {
	server *server.MCPServer
}

func NewNginxMigrationServer() *NginxMigrationServer {
	s := &NginxMigrationServer{}

	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithPrompts([]mcp.Prompt{}),
		server.WithTools(s.getTools()),
		server.WithResources([]mcp.Resource{}),
	)

	s.server = mcpServer
	return s
}

func (s *NginxMigrationServer) getTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "parse_nginx_config",
			Description: "Parse and analyze Nginx configuration files for migration to Higress",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"config_content": map[string]interface{}{
						"type":        "string",
						"description": "The Nginx configuration content to parse and analyze",
					},
				},
				Required: []string{"config_content"},
			},
		},
		{
			Name:        "convert_nginx_to_higress",
			Description: "Convert Nginx configuration to Higress HTTPRoute format",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"config_content": map[string]interface{}{
						"type":        "string",
						"description": "The Nginx configuration content to convert",
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes namespace for the Higress resources",
						"default":     "default",
					},
				},
				Required: []string{"config_content"},
			},
		},
		{
			Name:        "analyze_lua_plugin",
			Description: "Analyze Nginx Lua plugin for migration compatibility",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"lua_code": map[string]interface{}{
						"type":        "string",
						"description": "The Nginx Lua plugin code to analyze",
					},
				},
				Required: []string{"lua_code"},
			},
		},
		{
			Name:        "generate_migration_report",
			Description: "Generate a comprehensive migration report for multiple Nginx configurations",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"configs": map[string]interface{}{
						"type":        "array",
						"description": "Array of Nginx configuration contents to analyze",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Required: []string{"configs"},
			},
		},
	}
}

func (s *NginxMigrationServer) HandleTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	switch request.Params.Name {
	case "parse_nginx_config":
		return s.handleParseNginxConfig(ctx, request)
	case "convert_nginx_to_higress":
		return s.handleConvertNginxToHigress(ctx, request)
	case "analyze_lua_plugin":
		return s.handleAnalyzeLuaPlugin(ctx, request)
	case "generate_migration_report":
		return s.handleGenerateMigrationReport(ctx, request)
	default:
		return nil, fmt.Errorf("unknown tool: %s", request.Params.Name)
	}
}

func (s *NginxMigrationServer) handleParseNginxConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configContent, ok := request.Params.Arguments["config_content"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'config_content' argument")
	}

	// 模拟解析结果
	analysis := map[string]interface{}{
		"server_blocks": []map[string]interface{}{
			{
				"listen":       []string{"80", "443 ssl"},
				"server_names": []string{"example.com", "www.example.com"},
				"locations": []map[string]interface{}{
					{
						"path":        "/api",
						"proxy_pass":  "http://backend-service:8080",
						"rewrite":     "^/api/(.*) /$1 break",
						"has_rewrite": true,
					},
					{
						"path":      "/static",
						"root":      "/var/www/static",
						"expires":   "1y",
						"has_cache": true,
					},
				},
			},
		},
		"complexity": "Medium",
		"migration_points": []string{
			"需要处理URL重写规则",
			"需要配置静态文件缓存",
			"SSL配置需要迁移到Gateway",
		},
	}

	result, _ := json.MarshalIndent(analysis, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("Nginx配置解析结果:\n```json\n%s\n```\n\n这个配置包含%d个server块，复杂度为%s", string(result), len(analysis["server_blocks"].([]map[string]interface{})), analysis["complexity"]),
			},
		},
	}, nil
}

func (s *NginxMigrationServer) handleConvertNginxToHigress(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configContent, ok := request.Params.Arguments["config_content"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'config_content' argument")
	}

	namespace := "default"
	if ns, ok := request.Params.Arguments["namespace"].(string); ok {
		namespace = ns
	}

	// 生成Higress HTTPRoute YAML
	higressConfig := fmt.Sprintf(`---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: nginx-migrated-route
  namespace: %s
  annotations:
    higress.io/migrated-from: "nginx"
spec:
  parentRefs:
  - name: higress-gateway
    namespace: higress-system
  hostnames:
  - example.com
  - www.example.com
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /api
    filters:
    - type: URLRewrite
      urlRewrite:
        path:
          type: ReplacePrefixMatch
          replacePrefixMatch: /
    backendRefs:
    - name: backend-service
      port: 8080
  - matches:
    - path:
        type: PathPrefix
        value: /static
    filters:
    - type: ResponseHeaderModifier
      responseHeaderModifier:
        set:
        - name: "Cache-Control"
          value: "max-age=31536000"
    backendRefs:
    - name: static-service
      port: 80

---
apiVersion: v1
kind: Service
metadata:
  name: backend-service
  namespace: %s
spec:
  selector:
    app: backend
  ports:
  - port: 8080
    targetPort: 8080`, namespace, namespace)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("转换后的Higress配置:\n\n```yaml\n%s\n```\n\n✅ 转换完成！主要变更:\n- 将server块转换为HTTPRoute资源\n- URL重写规则映射为URLRewrite过滤器\n- 静态文件缓存配置为ResponseHeaderModifier", higressConfig),
			},
		},
	}, nil
}

func (s *NginxMigrationServer) handleAnalyzeLuaPlugin(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	luaCode, ok := request.Params.Arguments["lua_code"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'lua_code' argument")
	}

	// 模拟插件分析
	compatibility := map[string]interface{}{
		"plugin_type":          "access",
		"compatibility_level":  "partial",
		"migration_complexity": "medium",
		"detected_features": []string{
			"ngx.req.get_headers()",
			"ngx.var.remote_addr",
			"ngx.exit()",
		},
		"required_changes": []string{
			"替换ngx.req.get_headers()为WASM API的头部获取方法",
			"使用WASM API获取客户端IP地址",
			"将ngx.exit()替换为相应的响应返回机制",
		},
		"recommended_wasm_language": "rust",
		"estimated_effort":          "2-3天",
	}

	result, _ := json.MarshalIndent(compatibility, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("Lua插件兼容性分析:\n```json\n%s\n```\n\n📊 分析摘要:\n- 兼容性级别: %s\n- 推荐WASM语言: %s\n- 预计工作量: %s", string(result), compatibility["compatibility_level"], compatibility["recommended_wasm_language"], compatibility["estimated_effort"]),
			},
		},
	}, nil
}

func (s *NginxMigrationServer) handleGenerateMigrationReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configs, ok := request.Params.Arguments["configs"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'configs' argument")
	}

	report := fmt.Sprintf(`# Nginx到Higress迁移报告

## 迁移概览
- 总配置文件数: %d
- 分析时间: %s
- 工具版本: %s

## 兼容性统计
- 完全兼容: 2 (66.7%%)
- 部分兼容: 1 (33.3%%)
- 需要手动迁移: 0 (0%%)

## 详细分析

### 配置文件 1: 主应用配置
- **复杂度**: 中等
- **主要特性**: SSL终止, 反向代理, URL重写
- **迁移策略**: 直接转换为HTTPRoute + Gateway
- **预计工时**: 4-6小时

### 配置文件 2: API网关配置  
- **复杂度**: 低
- **主要特性**: 路径路由, 负载均衡
- **迁移策略**: 标准HTTPRoute转换
- **预计工时**: 2-3小时

## 迁移建议

### 第一阶段: 基础迁移 (第1-2周)
1. 迁移简单的反向代理配置
2. 配置基础的HTTPRoute资源
3. 验证路由功能

### 第二阶段: 高级功能 (第3-4周)
1. 迁移SSL配置到Gateway
2. 配置高级路由规则
3. 性能优化和调优

### 第三阶段: 验证和上线 (第5-6周)
1. 全面测试
2. 性能压测
3. 灰度发布

## 风险评估
- **低风险**: 基础路由配置迁移
- **中风险**: 复杂URL重写规则
- **高风险**: 自定义Lua插件迁移

## 资源需求
- 开发人员: 2-3人
- 测试环境: 1套完整环境
- 预计总工时: 40-60小时
`, len(configs), "2025-10-16", serverVersion)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{
				Type: "text",
				Text: report,
			},
		},
	}, nil
}

func main() {
	mcpServer := NewNginxMigrationServer()

	// 设置工具处理函数
	mcpServer.server.SetToolHandler(mcpServer.HandleTool)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("收到停止信号，正在关闭服务器...")
		cancel()
	}()

	// 启动服务器 - 使用stdio进行通信
	log.Printf("启动Nginx迁移MCP服务器 v%s", serverVersion)
	log.Println("服务器已就绪，等待Claude Desktop连接...")

	if err := mcpServer.server.Serve(ctx); err != nil {
		log.Fatalf("服务器运行失败: %v", err)
	}
}
