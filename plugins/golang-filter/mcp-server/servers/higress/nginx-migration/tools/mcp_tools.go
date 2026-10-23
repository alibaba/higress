// MCP Tools Definitions
// 定义所有可用的MCP工具及其描述信息
package tools

import (
	"encoding/json"
	"os"
)

// MCPTool represents a tool definition in MCP protocol
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content []Content `json:"content"`
}

// Content represents content within a tool result
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCPServer is an interface for server methods needed by tool handlers
type MCPServer interface {
	ParseNginxConfig(args map[string]interface{}) ToolResult
	ConvertToHigress(args map[string]interface{}) ToolResult
	AnalyzeLuaPlugin(args map[string]interface{}) ToolResult
	ConvertLuaToWasm(args map[string]interface{}) ToolResult
	// 新增工具链方法
	GenerateConversionHints(args map[string]interface{}) ToolResult
	ValidateWasmCode(args map[string]interface{}) ToolResult
	GenerateDeploymentConfig(args map[string]interface{}) ToolResult
}

// MCPToolsConfig 工具配置文件结构
type MCPToolsConfig struct {
	Version     string    `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tools       []MCPTool `json:"tools"`
}

// LoadToolsFromFile 从 JSON 文件加载工具定义
func LoadToolsFromFile(filename string) ([]MCPTool, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		// 文件不存在时使用默认配置
		return GetMCPToolsDefault(), nil
	}

	var config MCPToolsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config.Tools, nil
}

// GetMCPTools 返回所有可用的 MCP 工具定义
// 优先从 mcp-tools.json 加载，失败时使用默认定义
func GetMCPTools() []MCPTool {
	tools, err := LoadToolsFromFile("mcp-tools.json")
	if err != nil {
		return GetMCPToolsDefault()
	}
	return tools
}

// GetMCPToolsDefault 返回默认的工具定义（包含完整工具链）
func GetMCPToolsDefault() []MCPTool {
	return []MCPTool{
		{
			Name:        "parse_nginx_config",
			Description: "解析和分析 Nginx 配置文件，识别配置结构和复杂度",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"config_content": {
						"type": "string",
						"description": "Nginx 配置文件内容"
					}
				},
				"required": ["config_content"]
			}`),
		},
		{
			Name:        "convert_to_higress",
			Description: "将 Nginx 配置转换为 Higress HTTPRoute 和 Service 资源",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"config_content": {
						"type": "string",
						"description": "Nginx 配置文件内容"
					},
					"namespace": {
						"type": "string",
						"description": "目标 Kubernetes 命名空间",
						"default": "default"
					}
				},
				"required": ["config_content"]
			}`),
		},
		{
			Name:        "analyze_lua_plugin",
			Description: "分析 Nginx Lua 插件的兼容性，识别使用的 API 和潜在迁移问题，返回结构化分析结果",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"lua_code": {
						"type": "string",
						"description": "Nginx Lua 插件代码"
					}
				},
				"required": ["lua_code"]
			}`),
		},
		{
			Name:        "generate_conversion_hints",
			Description: "基于 Lua 分析结果生成代码转换模板",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"analysis_result": {
						"type": "string",
						"description": "analyze_lua_plugin 返回的 JSON 格式分析结果"
					},
					"plugin_name": {
						"type": "string",
						"description": "目标插件名称（小写字母和连字符）"
					}
				},
				"required": ["analysis_result", "plugin_name"]
			}`),
		},
		{
			Name:        "validate_wasm_code",
			Description: "验证生成的 Go WASM 插件代码，检查语法、API 使用、配置结构等，输出验证报告和改进建议",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"go_code": {
						"type": "string",
						"description": "生成的 Go WASM 插件代码"
					},
					"plugin_name": {
						"type": "string",
						"description": "插件名称"
					}
				},
				"required": ["go_code", "plugin_name"]
			}`),
		},
		{
			Name:        "generate_deployment_config",
			Description: "为验证通过的 WASM 插件生成完整的部署配置包，包括 WasmPlugin YAML、Makefile、Dockerfile、README 和测试脚本",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"plugin_name": {
						"type": "string",
						"description": "插件名称"
					},
					"go_code": {
						"type": "string",
						"description": "验证通过的 Go 代码"
					},
					"config_schema": {
						"type": "string",
						"description": "配置 JSON Schema（可选）"
					},
					"namespace": {
						"type": "string",
						"description": "部署命名空间",
						"default": "higress-system"
					}
				},
				"required": ["plugin_name", "go_code"]
			}`),
		},
		{
			Name:        "convert_lua_to_wasm",
			Description: "一键将 Nginx Lua 脚本转换为 Higress WASM 插件，自动生成 Go 代码和 WasmPlugin 配置。适合简单插件快速转换",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"lua_code": {
						"type": "string",
						"description": "要转换的 Nginx Lua 插件代码"
					},
					"plugin_name": {
						"type": "string",
						"description": "生成的 WASM 插件名称 (小写字母和连字符)"
					}
				},
				"required": ["lua_code", "plugin_name"]
			}`),
		},
	}
}

// ToolHandler 定义工具处理函数的类型
type ToolHandler func(args map[string]interface{}) ToolResult

// GetToolHandlers 返回工具名称到处理函数的映射
func GetToolHandlers(s MCPServer) map[string]ToolHandler {
	return map[string]ToolHandler{
		"parse_nginx_config":  s.ParseNginxConfig,
		"convert_to_higress":  s.ConvertToHigress,
		"analyze_lua_plugin":  s.AnalyzeLuaPlugin,
		"convert_lua_to_wasm": s.ConvertLuaToWasm,
		// 新增工具链处理器
		"generate_conversion_hints":  s.GenerateConversionHints,
		"validate_wasm_code":         s.ValidateWasmCode,
		"generate_deployment_config": s.GenerateDeploymentConfig,
	}
}
