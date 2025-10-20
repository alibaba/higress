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
			Description: "【工具链 1/4】分析 Nginx Lua 插件的兼容性，并生成 AI 代码生成指令。\n\n工作流程：\n1. 使用规则引擎分析 Lua 代码特性\n2. 返回结构化分析结果\n3. 返回 AI 代码生成上下文和提示\n\n后续操作：AI 可以基于分析结果和指令调用 generate_conversion_hints 工具获取转换建议。",
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
			Description: "【工具链 2/4】基于 Lua 分析结果，生成详细的代码转换提示和映射规则。\n\n输入：analyze_lua_plugin 的结构化分析结果\n输出：\n1. API 映射表（Lua API → Go WASM API）\n2. 详细的代码生成提示词\n3. 最佳实践建议\n4. 示例代码片段\n\n后续操作：AI 根据提示生成 Go WASM 代码。",
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
			Description: "【工具链 3/4】验证生成的 Go WASM 插件代码的正确性。\n\n检查项：\n1. Go 语法正确性\n2. 必要的 import 声明\n3. Higress SDK API 使用规范\n4. 配置结构完整性\n5. 常见错误模式检测\n\n输出：验证报告和改进建议。",
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
			Description: "【工具链 4/4】为验证通过的 WASM 插件生成完整的部署配置。\n\n生成内容：\n1. WasmPlugin YAML 配置\n2. ConfigMap（如需要）\n3. 构建脚本（Makefile/脚本）\n4. 部署说明文档\n5. 测试建议\n\n输出：完整的生产就绪配置包。",
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
			Description: "【一键转换】将 Nginx Lua 脚本自动转换为 Higress WASM 插件。\n\n这是原有的一体化工具，内部会自动调用规则引擎完成转换。\n如果需要更精细的控制和 AI 参与，建议使用工具链：\nanalyze_lua_plugin → generate_conversion_hints → (AI生成代码) → validate_wasm_code → generate_deployment_config",
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
