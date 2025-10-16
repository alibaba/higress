// MCP-HTTP Bridge Server
// 这个MCP服务器作为HTTP API的客户端，让MCP客户端可以通过MCP协议调用HTTP API
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// MCP Protocol structures (复用之前的定义)
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

// HTTP API客户端结构
type HTTPAPIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPAPIClient(baseURL string) *HTTPAPIClient {
	return &HTTPAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HTTP API请求结构
type APIRequest struct {
	ConfigContent  string `json:"config_content,omitempty"`
	LuaCode        string `json:"lua_code,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	TargetLanguage string `json:"target_language,omitempty"`
}

type APIResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// MCP服务器结构
type MCPHTTPBridge struct {
	apiClient *HTTPAPIClient
	config    *ServerConfig
}

func NewMCPHTTPBridge(apiURL string, config *ServerConfig) *MCPHTTPBridge {
	return &MCPHTTPBridge{
		apiClient: NewHTTPAPIClient(apiURL),
		config:    config,
	}
}

func (bridge *MCPHTTPBridge) handleMessage(msg MCPMessage) MCPMessage {
	switch msg.Method {
	case "initialize":
		return bridge.handleInitialize(msg)
	case "tools/list":
		return bridge.handleToolsList(msg)
	case "tools/call":
		return bridge.handleToolsCall(msg)
	default:
		return bridge.errorResponse(msg.ID, -32601, "Method not found")
	}
}

func (bridge *MCPHTTPBridge) handleInitialize(msg MCPMessage) MCPMessage {
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
				"name":    bridge.config.Server.Name + "-http-bridge",
				"version": bridge.config.Server.Version,
			},
		},
	}
}

func (bridge *MCPHTTPBridge) handleToolsList(msg MCPMessage) MCPMessage {
	tools := []Tool{
		{
			Name:        "parse_nginx_config",
			Description: "通过HTTP API解析和分析Nginx配置文件",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"config_content": {
						"type": "string",
						"description": "要分析的Nginx配置内容"
					}
				},
				"required": ["config_content"]
			}`),
		},
		{
			Name:        "convert_to_higress",
			Description: "通过HTTP API将Nginx配置转换为Higress HTTPRoute格式",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"config_content": {
						"type": "string",
						"description": "要转换的Nginx配置内容"
					},
					"namespace": {
						"type": "string",
						"description": "Kubernetes命名空间",
						"default": "default"
					}
				},
				"required": ["config_content"]
			}`),
		},
		{
			Name:        "analyze_lua_plugin",
			Description: "通过HTTP API分析Nginx Lua插件兼容性",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"lua_code": {
						"type": "string",
						"description": "要分析的Lua插件代码"
					}
				},
				"required": ["lua_code"]
			}`),
		},
		{
			Name:        "check_api_status",
			Description: "检查HTTP API服务器状态",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"required": []
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

func (bridge *MCPHTTPBridge) handleToolsCall(msg MCPMessage) MCPMessage {
	var params CallToolParams
	paramsBytes, _ := json.Marshal(msg.Params)
	json.Unmarshal(paramsBytes, &params)

	var result ToolResult

	switch params.Name {
	case "parse_nginx_config":
		result = bridge.callParseNginxAPI(params.Arguments)
	case "convert_to_higress":
		result = bridge.callConvertAPI(params.Arguments)
	case "analyze_lua_plugin":
		result = bridge.callAnalyzeLuaAPI(params.Arguments)
	case "check_api_status":
		result = bridge.checkAPIStatus()
	default:
		return bridge.errorResponse(msg.ID, -32601, "Unknown tool: "+params.Name)
	}

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  result,
	}
}

func (bridge *MCPHTTPBridge) callParseNginxAPI(args map[string]interface{}) ToolResult {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return bridge.errorResult("❌ 缺少config_content参数")
	}

	apiReq := APIRequest{
		ConfigContent: configContent,
	}

	response, err := bridge.callHTTPAPI("/api/parse-nginx", apiReq)
	if err != nil {
		return bridge.errorResult(fmt.Sprintf("❌ API调用失败: %v", err))
	}

	if !response.Success {
		return bridge.errorResult(fmt.Sprintf("❌ API返回错误: %s", response.Error))
	}

	return ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("🔗 **通过HTTP API调用结果**\n\n%s\n\n📡 **API服务器**: %s", response.Data, bridge.apiClient.baseURL),
		}},
	}
}

func (bridge *MCPHTTPBridge) callConvertAPI(args map[string]interface{}) ToolResult {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return bridge.errorResult("❌ 缺少config_content参数")
	}

	namespace := bridge.config.Defaults.Namespace
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	apiReq := APIRequest{
		ConfigContent: configContent,
		Namespace:     namespace,
	}

	response, err := bridge.callHTTPAPI("/api/convert-to-higress", apiReq)
	if err != nil {
		return bridge.errorResult(fmt.Sprintf("❌ API调用失败: %v", err))
	}

	if !response.Success {
		return bridge.errorResult(fmt.Sprintf("❌ API返回错误: %s", response.Error))
	}

	return ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("🔗 **通过HTTP API转换结果**\n\n%s\n\n📡 **API服务器**: %s", response.Data, bridge.apiClient.baseURL),
		}},
	}
}

func (bridge *MCPHTTPBridge) callAnalyzeLuaAPI(args map[string]interface{}) ToolResult {
	luaCode, ok := args["lua_code"].(string)
	if !ok {
		return bridge.errorResult("❌ 缺少lua_code参数")
	}

	apiReq := APIRequest{
		LuaCode: luaCode,
	}

	response, err := bridge.callHTTPAPI("/api/analyze-lua", apiReq)
	if err != nil {
		return bridge.errorResult(fmt.Sprintf("❌ API调用失败: %v", err))
	}

	if !response.Success {
		return bridge.errorResult(fmt.Sprintf("❌ API返回错误: %s", response.Error))
	}

	return ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("🔗 **通过HTTP API分析结果**\n\n%s\n\n📡 **API服务器**: %s", response.Data, bridge.apiClient.baseURL),
		}},
	}
}

func (bridge *MCPHTTPBridge) checkAPIStatus() ToolResult {
	resp, err := bridge.apiClient.httpClient.Get(bridge.apiClient.baseURL + "/health")
	if err != nil {
		return bridge.errorResult(fmt.Sprintf("❌ 无法连接到API服务器: %v", err))
	}
	defer resp.Body.Close()

	var healthData map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&healthData)

	status := "未知"
	if statusStr, ok := healthData["status"].(string); ok {
		status = statusStr
	}

	return ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("🔗 **HTTP API服务器状态检查**\n\n📡 **服务器地址**: %s\n📊 **状态**: %s\n🚥 **HTTP状态码**: %d\n\n✅ API服务器运行正常！", bridge.apiClient.baseURL, status, resp.StatusCode),
		}},
	}
}

func (bridge *MCPHTTPBridge) callHTTPAPI(endpoint string, request APIRequest) (*APIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("JSON序列化失败: %w", err)
	}

	resp, err := bridge.apiClient.httpClient.Post(
		bridge.apiClient.baseURL+endpoint,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}

	return &apiResp, nil
}

func (bridge *MCPHTTPBridge) errorResponse(id interface{}, code int, message string) MCPMessage {
	return MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
}

func (bridge *MCPHTTPBridge) errorResult(message string) ToolResult {
	return ToolResult{
		Content: []Content{{
			Type: "text",
			Text: message,
		}},
	}
}

func main() {
	config := LoadConfig()

	// 从命令行参数、环境变量或配置获取API服务器地址
	apiURL := config.Server.APIBaseURL
	if len(os.Args) > 1 {
		apiURL = os.Args[1]
	}
	if envURL := os.Getenv("NGINX_MIGRATION_API_URL"); envURL != "" {
		apiURL = envURL
	}

	bridge := NewMCPHTTPBridge(apiURL, config)

	log.Printf("🌉 MCP-HTTP桥接服务器启动...")
	log.Printf("🔗 连接到API服务器: %s", apiURL)
	log.Printf("📡 等待MCP客户端连接...")

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

		response := bridge.handleMessage(msg)

		responseBytes, _ := json.Marshal(response)
		fmt.Println(string(responseBytes))
	}
}
