//go:build higress_integration
// +build higress_integration

package mcptools

import (
	"fmt"
	"log"
	"strings"

	"nginx-migration-mcp/internal/rag"
	"nginx-migration-mcp/tools"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
)

// RegisterLuaPluginTools registers Lua plugin analysis and conversion tools
func RegisterLuaPluginTools(server *common.MCPServer, ctx *MigrationContext) {
	RegisterSimpleTool(
		server,
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
			return analyzeLuaPlugin(args, ctx)
		},
	)

	RegisterSimpleTool(
		server,
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
			return convertLuaToWasm(args, ctx)
		},
	)
}

func analyzeLuaPlugin(args map[string]interface{}, ctx *MigrationContext) (string, error) {
	luaCode, ok := args["lua_code"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid lua_code parameter")
	}

	// Analyze Lua features (基于规则)
	features := []string{}
	warnings := []string{}
	detectedAPIs := []string{}

	if strings.Contains(luaCode, "ngx.var") {
		features = append(features, "- ngx.var - Nginx变量")
		detectedAPIs = append(detectedAPIs, "ngx.var")
	}
	if strings.Contains(luaCode, "ngx.req") {
		features = append(features, "- ngx.req - 请求API")
		detectedAPIs = append(detectedAPIs, "ngx.req")
	}
	if strings.Contains(luaCode, "ngx.exit") {
		features = append(features, "- ngx.exit - 请求终止")
		detectedAPIs = append(detectedAPIs, "ngx.exit")
	}
	if strings.Contains(luaCode, "ngx.shared") {
		features = append(features, "- ngx.shared - 共享字典 (警告)")
		warnings = append(warnings, "共享字典需要外部缓存替换")
		detectedAPIs = append(detectedAPIs, "ngx.shared")
	}
	if strings.Contains(luaCode, "ngx.location.capture") {
		features = append(features, "- ngx.location.capture - 内部请求 (警告)")
		warnings = append(warnings, "需要改为HTTP客户端调用")
		detectedAPIs = append(detectedAPIs, "ngx.location.capture")
	}

	compatibility := "full"
	if len(warnings) > 0 {
		compatibility = "partial"
	}
	if len(warnings) > 2 {
		compatibility = "manual"
	}

	// === RAG 增强：查询知识库获取转换建议 ===
	var ragContext *rag.RAGContext
	if ctx.RAGManager != nil && ctx.RAGManager.IsEnabled() && len(detectedAPIs) > 0 {
		query := fmt.Sprintf("Nginx Lua API %s 在 Higress WASM 中的转换方法和最佳实践", strings.Join(detectedAPIs, ", "))
		var err error
		ragContext, err = ctx.RAGManager.QueryForTool("analyze_lua_plugin", query, "lua_migration")
		if err != nil {
			log.Printf("⚠️  RAG query failed for analyze_lua_plugin: %v", err)
		}
	}

	// 构建结果
	var result strings.Builder

	// RAG 上下文（如果有）
	if ragContext != nil && ragContext.Enabled && len(ragContext.Documents) > 0 {
		result.WriteString("📚 知识库参考资料:\n\n")
		result.WriteString(ragContext.FormatContextForAI())
		result.WriteString("\n")
	}

	// 基于规则的分析
	warningsText := "无"
	if len(warnings) > 0 {
		warningsText = strings.Join(warnings, "\n")
	}

	result.WriteString(fmt.Sprintf(`Lua插件兼容性分析

检测特性:
%s

兼容性警告:
%s

兼容性级别: %s

迁移建议:`, strings.Join(features, "\n"), warningsText, compatibility))

	switch compatibility {
	case "full":
		result.WriteString("\n- 可直接迁移到WASM插件")
	case "partial":
		result.WriteString("\n- 需要部分重构")
	case "manual":
		result.WriteString("\n- 需要手动重写")
	}

	return result.String(), nil
}

func convertLuaToWasm(args map[string]interface{}, ctx *MigrationContext) (string, error) {
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

	// === RAG 增强：查询转换模式和代码示例 ===
	var ragContext *rag.RAGContext
	if ctx.RAGManager != nil && ctx.RAGManager.IsEnabled() && len(analyzer.Features) > 0 {
		// 提取特性列表
		featureList := []string{}
		for feature := range analyzer.Features {
			featureList = append(featureList, feature)
		}

		query := fmt.Sprintf("将使用了 %s 的 Nginx Lua 插件转换为 Higress WASM Go 插件的代码示例",
			strings.Join(featureList, ", "))
		var err error
		ragContext, err = ctx.RAGManager.QueryForTool("convert_lua_to_wasm", query, "lua_to_wasm")
		if err != nil {
			log.Printf("⚠️  RAG query failed for convert_lua_to_wasm: %v", err)
		}
	}

	// 转换为WASM插件
	result, err := tools.ConvertLuaToWasm(analyzer, pluginName)
	if err != nil {
		return "", fmt.Errorf("conversion failed: %w", err)
	}

	// 构建响应
	var response strings.Builder

	// RAG 上下文（如果有）
	if ragContext != nil && ragContext.Enabled && len(ragContext.Documents) > 0 {
		response.WriteString("📚 知识库代码示例:\n\n")
		response.WriteString(ragContext.FormatContextForAI())
		response.WriteString("\n---\n\n")
	}

	response.WriteString(fmt.Sprintf(`Go 代码:
%s

WasmPlugin 配置:
%s

复杂度: %s, 特性: %d, 警告: %d`,
		result.GoCode,
		result.WasmPluginYAML,
		analyzer.Complexity,
		len(analyzer.Features),
		len(analyzer.Warnings)))

	return response.String(), nil
}
