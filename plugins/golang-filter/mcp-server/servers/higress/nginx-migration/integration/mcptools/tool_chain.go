//go:build higress_integration
// +build higress_integration

package mcptools

import (
	"encoding/json"
	"fmt"

	"nginx-migration-mcp/tools"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
)

// RegisterToolChainTools 注册工具链相关的工具
func RegisterToolChainTools(server *common.MCPServer, ctx *MigrationContext) {
	server.RegisterTool(common.NewTool(
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
			return generateConversionHints(args)
		},
	))

	server.RegisterTool(common.NewTool(
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
			return validateWasmCode(args)
		},
	))

	server.RegisterTool(common.NewTool(
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
	))
}

func generateConversionHints(args map[string]interface{}) (string, error) {
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

	// 返回 JSON 结果，由 LLM 解释和使用
	hintsJSON, _ := json.MarshalIndent(hints, "", "  ")
	return string(hintsJSON), nil
}

func validateWasmCode(args map[string]interface{}) (string, error) {
	goCode, ok := args["go_code"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid go_code parameter")
	}

	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid plugin_name parameter")
	}

	// 执行验证并返回 JSON 结果，由 LLM 解释和使用
	report := tools.ValidateWasmCode(goCode, pluginName)
	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	return string(reportJSON), nil
}

func generateDeploymentConfig(args map[string]interface{}, ctx *MigrationContext) (string, error) {
	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid plugin_name parameter")
	}

	goCode, ok := args["go_code"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid go_code parameter")
	}

	namespace := "higress-system"
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	configSchema := ""
	if cs, ok := args["config_schema"].(string); ok {
		configSchema = cs
	}

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
