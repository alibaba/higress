//go:build higress_integration
// +build higress_integration

package mcptools

import (
	"fmt"
	"strings"

	"nginx-migration-mcp-final/tools"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
)

// RegisterLuaPluginTools registers Lua plugin analysis and conversion tools
func RegisterLuaPluginTools(server *common.MCPServer, ctx *MigrationContext) {
	// Tool 1: Analyze Lua plugin compatibility
	server.RegisterTool(common.NewTool(
		"analyze_lua_plugin",
		"分析 Nginx Lua 插件的兼容性，评估迁移复杂度和潜在问题",
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

	// Tool 2: Convert Lua to WASM plugin
	server.RegisterTool(common.NewTool(
		"convert_lua_to_wasm",
		"将 Nginx Lua 脚本自动转换为 Higress WASM 插件，生成完整的 Go 代码和配置",
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

	warningsText := "无"
	if len(warnings) > 0 {
		warningsText = strings.Join(warnings, "\n")
	}

	result := fmt.Sprintf(`🔍 Lua插件兼容性分析

📊 检测特性:
%s

⚠️ 兼容性警告:
%s

📈 兼容性级别: %s

💡 迁移建议:`, strings.Join(features, "\n"), warningsText, compatibility)

	switch compatibility {
	case "full":
		result += "\n- 可直接迁移到WASM插件\n- 预计工作量: 1-2天"
	case "partial":
		result += "\n- 需要部分重构\n- 预计工作量: 3-5天"
	case "manual":
		result += "\n- 需要手动重写\n- 预计工作量: 1-2周"
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

	response := fmt.Sprintf(`🚀 Lua脚本转换完成！

📊 转换分析:
- 复杂度: %s
- 检测特性: %d个
- 兼容性警告: %d个

⚠️ 注意事项:
%s

📁 生成的文件:

==== main.go ====
%s

==== WasmPlugin配置 ====
%s

🔧 部署步骤:
1. 创建插件目录: mkdir -p extensions/%s
2. 保存Go代码到: extensions/%s/main.go  
3. 构建插件: PLUGIN_NAME=%s make build
4. 应用配置: kubectl apply -f wasmplugin.yaml

💡 提示:
- 请根据实际需求调整配置
- 测试插件功能后再部署到生产环境
- 如有共享状态需求，请配置Redis等外部存储
`,
		analyzer.Complexity,
		len(analyzer.Features),
		len(analyzer.Warnings),
		warningsText,
		result.GoCode,
		result.WasmPluginYAML,
		pluginName, pluginName, pluginName)

	return response, nil
}
