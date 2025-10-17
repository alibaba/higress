// MCP Server implementation for Nginx Migration Tools - Standalone Mode
package standalone

import (
	"encoding/json"
	"fmt"
	"strings"

	"nginx-migration-mcp/tools"
)

// NewMCPServer creates a new MCP server instance
func NewMCPServer(config *ServerConfig) *MCPServer {
	return &MCPServer{config: config}
}

// HandleMessage processes an incoming MCP message
func (s *MCPServer) HandleMessage(msg MCPMessage) MCPMessage {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "tools/list":
		return s.handleToolsList(msg)
	case "tools/call":
		return s.handleToolsCall(msg)
	default:
		return s.errorResponse(msg.ID, -32601, "Method not found")
	}
}

func (s *MCPServer) handleInitialize(msg MCPMessage) MCPMessage {
	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": true,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    s.config.Server.Name,
				"version": s.config.Server.Version,
			},
		},
	}
}

func (s *MCPServer) handleToolsList(msg MCPMessage) MCPMessage {
	toolsList := tools.GetMCPTools()

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"tools": toolsList,
		},
	}
}

func (s *MCPServer) handleToolsCall(msg MCPMessage) MCPMessage {
	var params CallToolParams
	paramsBytes, _ := json.Marshal(msg.Params)
	json.Unmarshal(paramsBytes, &params)

	handlers := tools.GetToolHandlers(s)
	handler, exists := handlers[params.Name]

	if !exists {
		return s.errorResponse(msg.ID, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
	}

	result := handler(params.Arguments)

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  result,
	}
}

func (s *MCPServer) errorResponse(id interface{}, code int, message string) MCPMessage {
	return MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
}

// Tool implementations

func (s *MCPServer) parseNginxConfig(args map[string]interface{}) tools.ToolResult {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing config_content"}}}
	}

	serverCount := strings.Count(configContent, "server {")
	locationCount := strings.Count(configContent, "location")
	hasSSL := strings.Contains(configContent, "ssl")
	hasProxy := strings.Contains(configContent, "proxy_pass")
	hasRewrite := strings.Contains(configContent, "rewrite")

	complexity := "Simple"
	if serverCount > 1 || (hasRewrite && hasSSL) {
		complexity = "Complex"
	} else if hasRewrite || hasSSL {
		complexity = "Medium"
	}

	analysis := fmt.Sprintf(`Nginx配置分析结果

基础信息:
- Server块: %d个
- Location块: %d个  
- SSL配置: %t
- 反向代理: %t
- URL重写: %t

复杂度: %s

迁移建议:`, serverCount, locationCount, hasSSL, hasProxy, hasRewrite, complexity)

	if hasProxy {
		analysis += "\n- 反向代理将转换为HTTPRoute backendRefs"
	}
	if hasRewrite {
		analysis += "\n- URL重写将使用URLRewrite过滤器"
	}
	if hasSSL {
		analysis += "\n- SSL配置需要迁移到Gateway资源"
	}

	return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: analysis}}}
}

func (s *MCPServer) convertToHigress(args map[string]interface{}) tools.ToolResult {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing config_content"}}}
	}

	namespace := s.config.Defaults.Namespace
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	hostname := s.config.Defaults.Hostname
	lines := strings.Split(configContent, "\n")
	for _, line := range lines {
		if strings.Contains(line, "server_name") && !strings.Contains(line, "#") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				hostname = strings.TrimSuffix(parts[1], ";")
				break
			}
		}
	}

	yamlConfig := fmt.Sprintf(`转换后的Higress配置

apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s
  namespace: %s
  annotations:
    higress.io/migrated-from: "nginx"
spec:
  parentRefs:
  - name: %s
    namespace: %s
  hostnames:
  - %s
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: %s
    backendRefs:
    - name: %s
      port: %d

---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    app: backend
  ports:
  - port: %d
    targetPort: %d

转换完成

应用步骤:
1. 保存为 higress-config.yaml
2. 执行: kubectl apply -f higress-config.yaml
3. 验证: kubectl get httproute -n %s`,
		s.config.GenerateRouteName(hostname), namespace,
		s.config.Gateway.Name, s.config.Gateway.Namespace, hostname, s.config.Defaults.PathPrefix,
		s.config.GenerateServiceName(hostname), s.config.Service.DefaultPort,
		s.config.GenerateServiceName(hostname), namespace,
		s.config.Service.DefaultPort, s.config.Service.DefaultTarget, namespace)

	return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: yamlConfig}}}
}

func (s *MCPServer) analyzeLuaPlugin(args map[string]interface{}) tools.ToolResult {
	luaCode, ok := args["lua_code"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing lua_code"}}}
	}

	features := []string{}
	warnings := []string{}

	if strings.Contains(luaCode, "ngx.var") {
		features = append(features, "- ngx.var - Nginx变量")
	}
	if strings.Contains(luaCode, "ngx.req") {
		features = append(features, "- ngx.req - 请求API")
	}
	if strings.Contains(luaCode, "ngx.exit") {
		features = append(features, "- ngx.exit - 请求终止")
	}
	if strings.Contains(luaCode, "ngx.shared") {
		features = append(features, "- ngx.shared - 共享字典 (警告)")
		warnings = append(warnings, "共享字典需要外部缓存替换")
	}
	if strings.Contains(luaCode, "ngx.location.capture") {
		features = append(features, "- ngx.location.capture - 内部请求 (警告)")
		warnings = append(warnings, "需要改为HTTP客户端调用")
	}

	compatibility := "full"
	if len(warnings) > 0 {
		compatibility = "partial"
	}
	if len(warnings) > 2 {
		compatibility = "manual"
	}

	result := fmt.Sprintf(`Lua插件兼容性分析

检测特性:
%s

兼容性警告:
%s

兼容性级别: %s

迁移建议:`, strings.Join(features, "\n"), strings.Join(warnings, "\n"), compatibility)

	switch compatibility {
	case "full":
		result += "\n- 可直接迁移到WASM插件\n- 预计工作量: 1-2天"
	case "partial":
		result += "\n- 需要部分重构\n- 预计工作量: 3-5天"
	case "manual":
		result += "\n- 需要手动重写\n- 预计工作量: 1-2周"
	}

	return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: result}}}
}

func (s *MCPServer) convertLuaToWasm(args map[string]interface{}) tools.ToolResult {
	luaCode, ok := args["lua_code"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing lua_code"}}}
	}

	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing plugin_name"}}}
	}

	analyzer := tools.AnalyzeLuaScript(luaCode)
	result, err := tools.ConvertLuaToWasm(analyzer, pluginName)
	if err != nil {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}}}
	}

	response := fmt.Sprintf(`Lua脚本转换完成

转换分析:
- 复杂度: %s
- 检测特性: %d个
- 兼容性警告: %d个

注意事项:
%s

生成的文件:

==== main.go ====
%s

==== WasmPlugin配置 ====
%s

部署步骤:
1. 创建插件目录: mkdir -p extensions/%s
2. 保存Go代码到: extensions/%s/main.go  
3. 构建插件: PLUGIN_NAME=%s make build
4. 应用配置: kubectl apply -f wasmplugin.yaml

提示:
- 请根据实际需求调整配置
- 测试插件功能后再部署到生产环境
- 如有共享状态需求，请配置Redis等外部存储
`,
		analyzer.Complexity,
		len(analyzer.Features),
		len(analyzer.Warnings),
		strings.Join(analyzer.Warnings, "\n- "),
		result.GoCode,
		result.WasmPluginYAML,
		pluginName, pluginName, pluginName)

	return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: response}}}
}
