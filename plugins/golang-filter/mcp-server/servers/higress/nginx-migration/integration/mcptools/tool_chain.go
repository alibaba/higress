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
	// Tool 3: Generate conversion hints
	server.RegisterTool(common.NewTool(
		"generate_conversion_hints",
		"【工具链 2/4】基于 Lua 分析结果，生成详细的代码转换提示和映射规则",
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

	// Tool 4: Validate WASM code
	server.RegisterTool(common.NewTool(
		"validate_wasm_code",
		"【工具链 3/4】验证生成的 Go WASM 插件代码的正确性",
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

	// Tool 5: Generate deployment config
	server.RegisterTool(common.NewTool(
		"generate_deployment_config",
		"【工具链 4/4】为验证通过的 WASM 插件生成完整的部署配置",
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

	// 格式化输出
	hintsJSON, _ := json.MarshalIndent(hints, "", "  ")

	result := fmt.Sprintf(`🎯 代码转换提示已生成

## 📚 API 映射表

为你准备了 %d 个 Lua API 到 Go WASM 的映射规则。

## 📝 代码生成模板

已生成针对插件 **%s** 的完整代码模板。

## ✨ 最佳实践

提供了 %d 条最佳实践建议。

## 💡 示例代码片段

准备了 %d 个常用场景的示例代码。

---

详细信息（JSON 格式）：
%s

---

**现在你可以**：
1. 基于这些提示开始编写 Go WASM 代码
2. 参考 API 映射表进行精确转换
3. 遵循最佳实践建议
4. 使用示例代码片段作为参考

生成代码后，建议调用 validate_wasm_code 工具进行验证。
`,
		len(hints.APIMappings),
		pluginName,
		len(hints.BestPractices),
		len(hints.ExampleSnippets),
		string(hintsJSON),
	)

	return result, nil
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

	// 执行验证
	report := tools.ValidateWasmCode(goCode, pluginName)

	// 格式化输出
	statusEmoji := "✅"
	statusText := "通过"
	if !report.IsValid {
		statusEmoji = "❌"
		statusText = "未通过"
	}

	errorsText := "无"
	if len(report.Errors) > 0 {
		errList := []string{}
		for _, e := range report.Errors {
			errList = append(errList, fmt.Sprintf("- [%s] %s\n  建议: %s", e.Severity, e.Message, e.Suggestion))
		}
		errorsText = "\n" + fmt.Sprint(errList)
	}

	warningsText := "无"
	if len(report.Warnings) > 0 {
		warningsText = "\n- " + fmt.Sprint(report.Warnings)
	}

	suggestionsText := "无"
	if len(report.Suggestions) > 0 {
		suggestionsText = "\n- " + fmt.Sprint(report.Suggestions)
	}

	result := fmt.Sprintf(`%s 代码验证结果：%s

## 📊 验证评分：%d/100

### 错误 (%d 个)
%s

### 警告 (%d 个)
%s

### 改进建议 (%d 个)
%s

### 缺失的导入包
%v

---

`,
		statusEmoji,
		statusText,
		report.Score,
		len(report.Errors),
		errorsText,
		len(report.Warnings),
		warningsText,
		len(report.Suggestions),
		suggestionsText,
		report.MissingImports,
	)

	if report.IsValid {
		result += "🎉 **代码验证通过！**\n\n"
		result += "**下一步**：调用 `generate_deployment_config` 工具生成部署配置。"
	} else {
		result += "⚠️ **请修复上述错误后重新验证。**"
	}

	return result, nil
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

	// 生成部署包
	pkg := tools.GenerateDeploymentPackage(pluginName, goCode, configSchema, namespace)

	// 格式化输出
	result := fmt.Sprintf(`🎉 部署配置生成完成！

已为插件 **%s** 生成完整的部署配置包。

## 📦 生成的文件

### 1. WasmPlugin 配置
文件名：wasmplugin.yaml
%s

### 2. Makefile
%s

### 3. Dockerfile
%s

### 4. README.md
（略，见完整输出）

### 5. 测试脚本 (test.sh)
%s

---

## 🚀 快速部署

`+"```bash"+`
# 1. 构建插件
make build

# 2. 构建并推送镜像
make docker-build docker-push

# 3. 部署到 Kubernetes
make deploy

# 4. 验证部署
kubectl get wasmplugin -n %s
`+"```"+`
`,
		pluginName,
		pkg.WasmPluginYAML,
		pkg.Makefile,
		pkg.Dockerfile,
		pkg.TestScript,
		namespace,
	)

	return result, nil
}
