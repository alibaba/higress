// Simple MCP Server for Nginx Migration Tools
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// MCP Protocol structures
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ToolResult struct {
	Content []Content `json:"content"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type MCPServer struct {
	config *ServerConfig
}

func (s *MCPServer) handleMessage(msg MCPMessage) MCPMessage {
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
	tools := []Tool{
		{
			Name:        "parse_nginx_config",
			Description: "Parse and analyze Nginx configuration files",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"config_content": {
						"type": "string",
						"description": "Nginx configuration content"
					}
				},
				"required": ["config_content"]
			}`),
		},
		{
			Name:        "convert_to_higress",
			Description: "Convert Nginx config to Higress HTTPRoute",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"config_content": {
						"type": "string",
						"description": "Nginx configuration content"
					},
					"namespace": {
						"type": "string",
						"description": "Target namespace",
						"default": "default"
					}
				},
				"required": ["config_content"]
			}`),
		},
		{
			Name:        "analyze_lua_plugin",
			Description: "Analyze Nginx Lua plugin compatibility",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"lua_code": {
						"type": "string",
						"description": "Lua plugin code"
					}
				},
				"required": ["lua_code"]
			}`),
		},
	}

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *MCPServer) handleToolsCall(msg MCPMessage) MCPMessage {
	var params CallToolParams
	paramsBytes, _ := json.Marshal(msg.Params)
	json.Unmarshal(paramsBytes, &params)

	var result ToolResult

	switch params.Name {
	case "parse_nginx_config":
		result = s.parseNginxConfig(params.Arguments)
	case "convert_to_higress":
		result = s.convertToHigress(params.Arguments)
	case "analyze_lua_plugin":
		result = s.analyzeLuaPlugin(params.Arguments)
	default:
		return s.errorResponse(msg.ID, -32601, "Unknown tool")
	}

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

func (s *MCPServer) parseNginxConfig(args map[string]interface{}) ToolResult {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return ToolResult{Content: []Content{{Type: "text", Text: "Error: Missing config_content"}}}
	}

	// Simple analysis
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

	analysis := fmt.Sprintf(`🔍 Nginx配置分析结果

📊 基础信息:
- Server块: %d个
- Location块: %d个  
- SSL配置: %t
- 反向代理: %t
- URL重写: %t

📈 复杂度: %s

🎯 迁移建议:`, serverCount, locationCount, hasSSL, hasProxy, hasRewrite, complexity)

	if hasProxy {
		analysis += "\n✓ 反向代理将转换为HTTPRoute backendRefs"
	}
	if hasRewrite {
		analysis += "\n✓ URL重写将使用URLRewrite过滤器"
	}
	if hasSSL {
		analysis += "\n✓ SSL配置需要迁移到Gateway资源"
	}

	return ToolResult{Content: []Content{{Type: "text", Text: analysis}}}
}

func (s *MCPServer) convertToHigress(args map[string]interface{}) ToolResult {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return ToolResult{Content: []Content{{Type: "text", Text: "Error: Missing config_content"}}}
	}

	namespace := s.config.Defaults.Namespace
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	// Extract hostname
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

	yamlConfig := fmt.Sprintf(`🚀 转换后的Higress配置

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

✅ 转换完成！

📋 应用步骤:
1. 保存为 higress-config.yaml
2. 执行: kubectl apply -f higress-config.yaml
3. 验证: kubectl get httproute -n %s`,
		s.config.GenerateRouteName(hostname), namespace,
		s.config.Gateway.Name, s.config.Gateway.Namespace, hostname, s.config.Defaults.PathPrefix,
		s.config.GenerateServiceName(hostname), s.config.Service.DefaultPort,
		s.config.GenerateServiceName(hostname), namespace,
		s.config.Service.DefaultPort, s.config.Service.DefaultTarget, namespace)

	return ToolResult{Content: []Content{{Type: "text", Text: yamlConfig}}}
}

func (s *MCPServer) analyzeLuaPlugin(args map[string]interface{}) ToolResult {
	luaCode, ok := args["lua_code"].(string)
	if !ok {
		return ToolResult{Content: []Content{{Type: "text", Text: "Error: Missing lua_code"}}}
	}

	// Analyze Lua features
	features := []string{}
	warnings := []string{}

	if strings.Contains(luaCode, "ngx.var") {
		features = append(features, "✓ ngx.var - Nginx变量")
	}
	if strings.Contains(luaCode, "ngx.req") {
		features = append(features, "✓ ngx.req - 请求API")
	}
	if strings.Contains(luaCode, "ngx.exit") {
		features = append(features, "✓ ngx.exit - 请求终止")
	}
	if strings.Contains(luaCode, "ngx.shared") {
		features = append(features, "⚠️ ngx.shared - 共享字典")
		warnings = append(warnings, "共享字典需要外部缓存替换")
	}
	if strings.Contains(luaCode, "ngx.location.capture") {
		features = append(features, "⚠️ ngx.location.capture - 内部请求")
		warnings = append(warnings, "需要改为HTTP客户端调用")
	}

	compatibility := "full"
	if len(warnings) > 0 {
		compatibility = "partial"
	}
	if len(warnings) > 2 {
		compatibility = "manual"
	}

	result := fmt.Sprintf(`🔍 Lua插件兼容性分析

📊 检测特性:
%s

⚠️ 兼容性警告:
%s

📈 兼容性级别: %s

💡 迁移建议:`, strings.Join(features, "\n"), strings.Join(warnings, "\n"), compatibility)

	switch compatibility {
	case "full":
		result += "\n- 可直接迁移到WASM插件\n- 预计工作量: 1-2天"
	case "partial":
		result += "\n- 需要部分重构\n- 预计工作量: 3-5天"
	case "manual":
		result += "\n- 需要手动重写\n- 预计工作量: 1-2周"
	}

	return ToolResult{Content: []Content{{Type: "text", Text: result}}}
}

func main() {
	config := LoadConfig()
	server := &MCPServer{config: config}

	log.Println("🚀 Nginx迁移MCP服务器启动...")
	log.Println("🔗 等待MCP客户端连接...")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("JSON解析错误: %v", err)
			continue
		}

		response := server.handleMessage(msg)

		responseBytes, _ := json.Marshal(response)
		fmt.Println(string(responseBytes))
	}
}
