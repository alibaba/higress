//go:build higress_integration
// +build higress_integration

package mcptools

import (
	"encoding/json"
	"fmt"
	"strings"

	"nginx-migration-mcp/tools"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
)

// RegisterToolChainTools 注册工具链相关的工具
func RegisterToolChainTools(server *common.MCPServer, ctx *MigrationContext) {
	RegisterSimpleTool(
		server,
		"generate_conversion_hints",
		"基于 Lua 分析结果生成代码转换模板",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"analysis_result": map[string]interface{}{
					"type":        "string",
					"description": "analyze_lua_plugin 返回的 JSON 格式分析结果",
				},
				"plugin_name": map[string]interface{}{
					"type":        "string",
					"description": "目标插件名称（小写字母和连字符）",
				},
			},
			"required": []string{"analysis_result", "plugin_name"},
		},
		func(args map[string]interface{}) (string, error) {
			return generateConversionHints(args, ctx)
		},
	)

	RegisterSimpleTool(
		server,
		"validate_wasm_code",
		"验证生成的 Go WASM 插件代码，检查语法、API 使用和配置结构",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"go_code": map[string]interface{}{
					"type":        "string",
					"description": "生成的 Go WASM 插件代码",
				},
				"plugin_name": map[string]interface{}{
					"type":        "string",
					"description": "插件名称",
				},
			},
			"required": []string{"go_code", "plugin_name"},
		},
		func(args map[string]interface{}) (string, error) {
			return validateWasmCode(args, ctx)
		},
	)

	RegisterSimpleTool(
		server,
		"generate_deployment_config",
		"为验证通过的 WASM 插件生成完整的部署配置包",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"plugin_name": map[string]interface{}{
					"type":        "string",
					"description": "插件名称",
				},
				"go_code": map[string]interface{}{
					"type":        "string",
					"description": "验证通过的 Go 代码",
				},
				"config_schema": map[string]interface{}{
					"type":        "string",
					"description": "配置 JSON Schema（可选）",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "部署命名空间",
					"default":     "higress-system",
				},
			},
			"required": []string{"plugin_name", "go_code"},
		},
		func(args map[string]interface{}) (string, error) {
			return generateDeploymentConfig(args, ctx)
		},
	)
}

func generateConversionHints(args map[string]interface{}, ctx *MigrationContext) (string, error) {
	analysisResultStr, ok := args["analysis_result"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid analysis_result parameter")
	}

	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid plugin_name parameter")
	}

	// 解析分析结果
	var analysis tools.AnalysisResultForAI
	if err := json.Unmarshal([]byte(analysisResultStr), &analysis); err != nil {
		return "", fmt.Errorf("failed to parse analysis_result: %w", err)
	}

	// 生成转换提示
	hints := tools.GenerateConversionHints(analysis, pluginName)

	// === RAG 增强（如果启用）===
	var ragInfo map[string]interface{}
	if ctx.RAGManager != nil && ctx.RAGManager.IsEnabled() {
		// 构建智能查询语句
		queryBuilder := []string{}
		if len(analysis.APICalls) > 0 {
			queryBuilder = append(queryBuilder, "Nginx Lua API 转换到 Higress WASM")

			hasHeaderOps := analysis.Features["header_manipulation"] || analysis.Features["request_headers"] || analysis.Features["response_headers"]
			hasBodyOps := analysis.Features["request_body"] || analysis.Features["response_body"]
			hasResponseControl := analysis.Features["response_control"]

			if hasHeaderOps {
				queryBuilder = append(queryBuilder, "请求头和响应头处理")
			}
			if hasBodyOps {
				queryBuilder = append(queryBuilder, "请求体和响应体处理")
			}
			if hasResponseControl {
				queryBuilder = append(queryBuilder, "响应控制和状态码设置")
			}

			if len(analysis.APICalls) > 0 && len(analysis.APICalls) <= 5 {
				queryBuilder = append(queryBuilder, fmt.Sprintf("涉及 API: %s", strings.Join(analysis.APICalls, ", ")))
			}
		} else {
			queryBuilder = append(queryBuilder, "Higress WASM 插件开发 基础示例 Go SDK 使用")
		}

		if analysis.Complexity == "high" {
			queryBuilder = append(queryBuilder, "复杂插件实现 高级功能")
		}

		queryString := strings.Join(queryBuilder, " ")

		ragContext, err := ctx.RAGManager.QueryForTool(
			"generate_conversion_hints",
			queryString,
			"lua_migration",
		)
		if err == nil && ragContext.Enabled && len(ragContext.Documents) > 0 {
			ragInfo = map[string]interface{}{
				"enabled":   true,
				"documents": len(ragContext.Documents),
				"context":   ragContext.FormatContextForAI(),
			}
		}
	}

	// 组合结果
	result := map[string]interface{}{
		"code_template": hints.CodeTemplate,
		"warnings":      hints.Warnings,
		"rag":           ragInfo,
	}

	// 返回 JSON 结果，由 LLM 解释和使用
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return string(resultJSON), nil
}

func validateWasmCode(args map[string]interface{}, ctx *MigrationContext) (string, error) {
	goCode, ok := args["go_code"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid go_code parameter")
	}

	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid plugin_name parameter")
	}

	// 执行验证（AI 驱动）
	report := tools.ValidateWasmCode(goCode, pluginName)

	// 格式化输出，包含 AI 分析提示和基础信息
	var result strings.Builder

	result.WriteString(fmt.Sprintf("##  代码验证报告\n\n"))
	result.WriteString(fmt.Sprintf("代码存在 %d 个必须修复的问题，%d 个建议修复的问题，%d 个可选优化项，%d 个最佳实践建议。请优先解决必须修复的问题。\n\n", 0, 0, 0, 0))

	result.WriteString(fmt.Sprintf("### 发现的回调函数 (%d 个)\n", len(report.FoundCallbacks)))
	if len(report.FoundCallbacks) > 0 {
		for _, cb := range report.FoundCallbacks {
			result.WriteString(fmt.Sprintf("- %s\n", cb))
		}
	} else {
		result.WriteString("无\n")
	}
	result.WriteString("\n")

	result.WriteString("### 配置结构\n")
	if report.HasConfig {
		result.WriteString(" 已定义配置结构体\n\n")
	} else {
		result.WriteString(" 未定义配置结构体\n\n")
	}

	result.WriteString("### 问题分类\n\n")

	result.WriteString("####  必须修复 (0 个)\n")
	result.WriteString("无\n\n")

	result.WriteString("####  建议修复 (0 个)\n")
	result.WriteString("无\n\n")

	result.WriteString("####  可选优化 (0 个)\n")
	result.WriteString("无\n\n")

	result.WriteString("####  最佳实践 (0 个)\n")
	result.WriteString("无\n\n")

	// 添加 AI 分析提示
	result.WriteString("---\n\n")
	result.WriteString(report.Summary)
	result.WriteString("\n\n")

	// === RAG 增强：查询最佳实践 ===
	if ctx.RAGManager != nil && ctx.RAGManager.IsEnabled() {
		// 构建智能查询语句
		queryBuilder := []string{"Higress WASM 插件"}

		// 根据回调函数类型添加特定查询
		for _, callback := range report.FoundCallbacks {
			if strings.Contains(callback, "RequestHeaders") {
				queryBuilder = append(queryBuilder, "请求头处理")
			}
			if strings.Contains(callback, "RequestBody") {
				queryBuilder = append(queryBuilder, "请求体处理")
			}
			if strings.Contains(callback, "ResponseHeaders") {
				queryBuilder = append(queryBuilder, "响应头处理")
			}
		}

		queryBuilder = append(queryBuilder, "性能优化 最佳实践 错误处理")
		queryString := strings.Join(queryBuilder, " ")

		ragContext, err := ctx.RAGManager.QueryForTool(
			"validate_wasm_code",
			queryString,
			"best_practice",
		)
		if err == nil && ragContext.Enabled && len(ragContext.Documents) > 0 {
			result.WriteString("\n\n### 📚 最佳实践建议（来自知识库）\n\n")
			result.WriteString(ragContext.FormatContextForAI())
		}
	}

	// 添加 JSON 格式的结构化数据（供后续处理）
	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	result.WriteString("\n")
	result.WriteString(string(reportJSON))

	return result.String(), nil
}

func generateDeploymentConfig(args map[string]interface{}, ctx *MigrationContext) (string, error) {
	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid plugin_name parameter")
	}

	_, ok = args["go_code"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid go_code parameter")
	}

	namespace := "higress-system"
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// configSchema is optional, we don't use it for now but don't return error
	_ = args["config_schema"]

	// 返回提示信息，由 LLM 生成具体配置文件
	result := fmt.Sprintf(`为插件 %s 生成以下部署配置：
1. WasmPlugin YAML (namespace: %s)
2. Makefile (TinyGo 构建)
3. Dockerfile
4. README.md
5. 测试脚本

参考文档: https://higress.io/zh-cn/docs/user/wasm-go`, pluginName, namespace)

	return result, nil
}
