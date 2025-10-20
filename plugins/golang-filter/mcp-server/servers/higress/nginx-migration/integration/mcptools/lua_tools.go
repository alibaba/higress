//go:build higress_integration
// +build higress_integration

package mcptools

import (
	"fmt"
	"strings"

	"nginx-migration-mcp/tools"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
)

// RegisterLuaPluginTools registers Lua plugin analysis and conversion tools
func RegisterLuaPluginTools(server *common.MCPServer, ctx *MigrationContext) {
	server.RegisterTool(common.NewTool(
		"analyze_lua_plugin",
		"分析 Nginx Lua 插件的兼容性，识别使用的 API 和潜在迁移问题",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"lua_code": map[string]interface{}{
					"type":        "string",
					"description": "Nginx Lua 插件代码",
				},
			},
			"required": []string{"lua_code"},
		},
		func(args map[string]interface{}) (string, error) {
			return analyzeLuaPlugin(args)
		},
	))

	server.RegisterTool(common.NewTool(
		"convert_lua_to_wasm",
		"一键将 Nginx Lua 脚本转换为 Higress WASM 插件，自动生成 Go 代码和配置",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"lua_code": map[string]interface{}{
					"type":        "string",
					"description": "要转换的 Nginx Lua 插件代码",
				},
				"plugin_name": map[string]interface{}{
					"type":        "string",
					"description": "生成的 WASM 插件名称 (小写字母和连字符)",
				},
			},
			"required": []string{"lua_code", "plugin_name"},
		},
		func(args map[string]interface{}) (string, error) {
			return convertLuaToWasm(args)
		},
	))
}

func analyzeLuaPlugin(args map[string]interface{}) (string, error) {
	luaCode, ok := args["lua_code"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid lua_code parameter")
	}

	// Analyze Lua features
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

	warningsText := "无"
	if len(warnings) > 0 {
		warningsText = strings.Join(warnings, "\n")
	}

	result := fmt.Sprintf(`Lua插件兼容性分析

检测特性:
%s

兼容性警告:
%s

兼容性级别: %s

迁移建议:`, strings.Join(features, "\n"), warningsText, compatibility)

	switch compatibility {
	case "full":
		result += "\n- 可直接迁移到WASM插件"
	case "partial":
		result += "\n- 需要部分重构"
	case "manual":
		result += "\n- 需要手动重写"
	}

	return result, nil
}

func convertLuaToWasm(args map[string]interface{}) (string, error) {
	luaCode, ok := args["lua_code"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid lua_code parameter")
	}

	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid plugin_name parameter")
	}

	// 分析Lua脚本
	analyzer := tools.AnalyzeLuaScript(luaCode)

	// 转换为WASM插件
	result, err := tools.ConvertLuaToWasm(analyzer, pluginName)
	if err != nil {
		return "", fmt.Errorf("conversion failed: %w", err)
	}

	warningsText := "无特殊注意事项"
	if len(analyzer.Warnings) > 0 {
		warningsText = strings.Join(analyzer.Warnings, "\n- ")
	}

	response := fmt.Sprintf(`Go 代码:
%s

WasmPlugin 配置:
%s

复杂度: %s, 特性: %d, 警告: %d`,
		result.GoCode,
		result.WasmPluginYAML,
		analyzer.Complexity,
		len(analyzer.Features),
		len(analyzer.Warnings))

	return response, nil
}
