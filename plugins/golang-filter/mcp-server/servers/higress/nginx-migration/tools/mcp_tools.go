// MCP Tools Definitions
// 定义所有可用的MCP工具及其描述信息
package tools

import (
	"encoding/json"
	"log"
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
}

// MCPToolsConfig 工具配置文件结构
type MCPToolsConfig struct {
	Version     string    `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tools       []MCPTool `json:"tools"`
}

// isDebugMode 检查是否启用调试模式
func isDebugMode() bool {
	debug := os.Getenv("DEBUG")
	return debug == "true" || debug == "1"
}

// LoadToolsFromFile 从JSON文件加载工具定义
func LoadToolsFromFile(filename string) ([]MCPTool, error) {
	// 调试模式下输出详细日志
	if isDebugMode() {
		cwd, _ := os.Getwd()
		log.Printf("📂 当前工作目录: %s", cwd)
		log.Printf("📄 尝试加载配置文件: %s", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		if isDebugMode() {
			log.Printf("⚠️  无法读取 %s: %v，使用默认配置", filename, err)
		}
		// 如果文件不存在，返回默认工具
		return GetMCPToolsDefault(), nil
	}

	var config MCPToolsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		if isDebugMode() {
			log.Printf("❌ 解析 %s 失败: %v，使用默认配置", filename, err)
		}
		return nil, err
	}

	if isDebugMode() {
		log.Printf("✅ 成功从 %s 加载了 %d 个工具", filename, len(config.Tools))
	}
	return config.Tools, nil
}

// GetMCPTools 返回所有可用的MCP工具定义
// 优先从 mcp-tools.json 文件加载，如果文件不存在则使用默认定义
func GetMCPTools() []MCPTool {
	tools, err := LoadToolsFromFile("mcp-tools.json")
	if err != nil {
		// 加载失败，使用默认定义
		return GetMCPToolsDefault()
	}
	return tools
}

// GetMCPToolsDefault 返回默认的工具定义
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
			Description: "分析 Nginx Lua 插件的兼容性，评估迁移复杂度和潜在问题",
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
			Name:        "convert_lua_to_wasm",
			Description: "将 Nginx Lua 脚本自动转换为 Higress WASM 插件，生成完整的 Go 代码和配置",
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
	}
}

// ToolMetadata 包含工具的元数据信息
type ToolMetadata struct {
	Category     string   // 工具分类
	Tags         []string // 标签
	Version      string   // 版本
	Complexity   string   // 复杂度: simple, medium, complex
	Experimental bool     // 是否为实验性功能
}

// GetToolMetadata 返回工具的元数据
func GetToolMetadata() map[string]ToolMetadata {
	return map[string]ToolMetadata{
		"parse_nginx_config": {
			Category:   "analysis",
			Tags:       []string{"nginx", "config", "parser"},
			Version:    "1.0.0",
			Complexity: "simple",
		},
		"convert_to_higress": {
			Category:   "conversion",
			Tags:       []string{"nginx", "higress", "k8s"},
			Version:    "1.0.0",
			Complexity: "medium",
		},
		"analyze_lua_plugin": {
			Category:   "analysis",
			Tags:       []string{"lua", "compatibility", "assessment"},
			Version:    "1.0.0",
			Complexity: "simple",
		},
		"convert_lua_to_wasm": {
			Category:   "conversion",
			Tags:       []string{"lua", "wasm", "codegen"},
			Version:    "1.0.0",
			Complexity: "complex",
		},
	}
}
