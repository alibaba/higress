// Tool Chain implementations for LLM-guided Lua to WASM conversion
package tools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// AnalysisResultForAI 结构化的分析结果，用于 AI 协作
type AnalysisResultForAI struct {
	Features      map[string]bool   `json:"features"`
	Variables     map[string]string `json:"variables"`
	APICalls      []string          `json:"api_calls"`
	Warnings      []string          `json:"warnings"`
	Complexity    string            `json:"complexity"`
	Compatibility string            `json:"compatibility"`
	// OriginalCode 字段已移除，避免返回大量数据
}

// ConversionHints 代码转换提示（简化版）
type ConversionHints struct {
	CodeTemplate string   `json:"code_template"`
	Warnings     []string `json:"warnings"`
}

// ValidationReport 验证报告
type ValidationReport struct {
	Issues         []ValidationIssue `json:"issues"`          // 所有发现的问题
	MissingImports []string          `json:"missing_imports"` // 缺失的 import
	FoundCallbacks []string          `json:"found_callbacks"` // 找到的回调函数
	HasConfig      bool              `json:"has_config"`      // 是否有配置结构
	Summary        string            `json:"summary"`         // 总体评估摘要
}

// ValidationIssue 验证问题
type ValidationIssue struct {
	Category   string `json:"category"`   // required, recommended, optional, best_practice
	Type       string `json:"type"`       // syntax, api_usage, config, error_handling, logging, etc.
	Message    string `json:"message"`    // 问题描述
	Suggestion string `json:"suggestion"` // 改进建议
	Impact     string `json:"impact"`     // 影响说明（为什么重要）
}

// DeploymentPackage 部署配置包
type DeploymentPackage struct {
	WasmPluginYAML string            `json:"wasm_plugin_yaml"`
	Makefile       string            `json:"makefile"`
	Dockerfile     string            `json:"dockerfile"`
	ConfigMap      string            `json:"config_map"`
	README         string            `json:"readme"`
	TestScript     string            `json:"test_script"`
	Dependencies   map[string]string `json:"dependencies"`
}

// AnalyzeLuaPluginForAI 分析 Lua 插件并生成 AI 友好的输出
func AnalyzeLuaPluginForAI(luaCode string) AnalysisResultForAI {
	analyzer := AnalyzeLuaScript(luaCode)

	// 收集所有 API 调用
	apiCalls := []string{}
	for feature := range analyzer.Features {
		apiCalls = append(apiCalls, feature)
	}

	// 确定兼容性级别
	compatibility := "full"
	if len(analyzer.Warnings) > 0 {
		compatibility = "partial"
	}
	if len(analyzer.Warnings) > 2 {
		compatibility = "manual"
	}

	return AnalysisResultForAI{
		Features:      analyzer.Features,
		Variables:     analyzer.Variables,
		APICalls:      apiCalls,
		Warnings:      analyzer.Warnings,
		Complexity:    analyzer.Complexity,
		Compatibility: compatibility,
	}
}

// GenerateConversionHints 生成代码转换提示（简化版）
func GenerateConversionHints(analysis AnalysisResultForAI, pluginName string) ConversionHints {
	return ConversionHints{
		CodeTemplate: generateCodeTemplate(analysis, pluginName),
		Warnings:     analysis.Warnings,
	}
}

// generateCodeTemplate 生成代码模板提示
func generateCodeTemplate(analysis AnalysisResultForAI, pluginName string) string {
	callbacks := generateCallbackSummary(analysis)
	return fmt.Sprintf(`生成 Go WASM 插件 %s，实现回调: %s
参考文档: https://higress.io/zh-cn/docs/user/wasm-go`,
		pluginName, callbacks)
}

// generateCallbackRegistrations 生成回调注册代码
func generateCallbackRegistrations(analysis AnalysisResultForAI) string {
	callbacks := []string{}

	if analysis.Features["ngx.var"] || analysis.Features["request_headers"] || analysis.Features["header_manipulation"] {
		callbacks = append(callbacks, "wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders)")
	}

	if analysis.Features["request_body"] {
		callbacks = append(callbacks, "wrapper.ProcessRequestBodyBy(onHttpRequestBody)")
	}

	if analysis.Features["response_headers"] || analysis.Features["response_control"] {
		callbacks = append(callbacks, "wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders)")
	}

	if len(callbacks) == 0 {
		callbacks = append(callbacks, "wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders)")
	}

	return "\n\t\t" + strings.Join(callbacks, ",\n\t\t")
}

// generateCallbackSummary 生成回调函数摘要
func generateCallbackSummary(analysis AnalysisResultForAI) string {
	callbacks := []string{}

	if analysis.Features["ngx.var"] || analysis.Features["request_headers"] || analysis.Features["header_manipulation"] {
		callbacks = append(callbacks, "onHttpRequestHeaders")
	}
	if analysis.Features["request_body"] {
		callbacks = append(callbacks, "onHttpRequestBody")
	}
	if analysis.Features["response_headers"] || analysis.Features["response_control"] {
		callbacks = append(callbacks, "onHttpResponseHeaders")
	}

	if len(callbacks) == 0 {
		return "onHttpRequestHeaders"
	}
	return strings.Join(callbacks, ", ")
}

// ValidateWasmCode 验证生成的 Go WASM 代码（AI 驱动）
// 该函数提取代码的基本信息，由 AI 进行智能分析
func ValidateWasmCode(goCode, pluginName string) ValidationReport {
	report := ValidationReport{
		Issues:         []ValidationIssue{},
		MissingImports: []string{},
		FoundCallbacks: []string{},
		HasConfig:      false,
	}

	// 移除注释以避免误判
	codeWithoutComments := removeComments(goCode)

	// === 基础结构检测（提供给 AI 作为上下文）===

	// 检测回调函数
	callbacks := map[string]*regexp.Regexp{
		"onHttpRequestHeaders":  regexp.MustCompile(`func\s+onHttpRequestHeaders\s*\(`),
		"onHttpRequestBody":     regexp.MustCompile(`func\s+onHttpRequestBody\s*\(`),
		"onHttpResponseHeaders": regexp.MustCompile(`func\s+onHttpResponseHeaders\s*\(`),
		"onHttpResponseBody":    regexp.MustCompile(`func\s+onHttpResponseBody\s*\(`),
	}

	for name, pattern := range callbacks {
		if pattern.MatchString(codeWithoutComments) {
			report.FoundCallbacks = append(report.FoundCallbacks, name)
		}
	}

	// 检测配置结构体
	configPattern := regexp.MustCompile(`type\s+\w+Config\s+struct\s*\{`)
	report.HasConfig = configPattern.MatchString(goCode)

	// === 生成 AI 友好的总结 ===
	report.Summary = fmt.Sprintf(`请作为 Higress WASM 插件开发专家，分析以下 Go 代码：

插件名称: %s
代码行数: %d
检测到的回调函数: %v
是否有配置结构: %v

请检查以下方面并给出专业建议：

1. **必须修复的问题** (required):
   - 是否有 package main 声明
   - 是否有 main() 函数（即使为空）
   - 是否有 init() 函数
   - 是否在 init() 中调用了 wrapper.SetCtx 注册插件
   - 导入的包是否完整（wrapper、types、proxywasm 等）
   - 回调函数是否正确返回 types.Action
   - 是否至少实现了一个回调函数

2. **建议修复的问题** (recommended):
   - 配置解析函数 parseConfig 是否正确实现
   - 错误处理是否完善
   - 回调函数签名是否正确

3. **可选优化** (optional):
   - 代码结构是否清晰
   - 变量命名是否规范

4. **最佳实践** (best_practice):
   - 日志记录是否充分
   - 性能考虑（避免不必要的内存分配）
   - 安全性考虑

请以结构化格式返回问题列表，每个问题包括：
- category: required/recommended/optional/best_practice
- type: syntax/api_usage/config/error_handling/logging/performance/security
- message: 问题描述
- suggestion: 具体的修复建议
- impact: 为什么这个问题重要

代码内容：
%s`, pluginName, len(strings.Split(goCode, "\n")), report.FoundCallbacks, report.HasConfig, goCode)

	return report
}

// removeComments 移除 Go 代码中的注释
func removeComments(code string) string {
	// 移除单行注释
	singleLineComment := regexp.MustCompile(`//.*`)
	code = singleLineComment.ReplaceAllString(code, "")

	// 移除多行注释
	multiLineComment := regexp.MustCompile(`(?s)/\*.*?\*/`)
	code = multiLineComment.ReplaceAllString(code, "")

	return code
}

// GenerateDeploymentPackage 生成部署配置提示（由 LLM 根据代码生成具体内容）
func GenerateDeploymentPackage(pluginName, goCode, configSchema, namespace string) DeploymentPackage {
	// 所有配置文件都由 LLM 根据实际代码生成，不使用固定模板
	return DeploymentPackage{
		WasmPluginYAML: "", // LLM 生成
		Makefile:       "", // LLM 生成
		Dockerfile:     "", // LLM 生成
		ConfigMap:      "", // LLM 生成
		README:         "", // LLM 生成
		TestScript:     "", // LLM 生成
		Dependencies:   make(map[string]string),
	}
}

// FormatToolResultWithAIContext 格式化工具结果
func FormatToolResultWithAIContext(userMessage, aiInstructions string, structuredData interface{}) ToolResult {
	jsonData, _ := json.MarshalIndent(structuredData, "", "  ")
	output := fmt.Sprintf("%s\n\n%s\n\n%s", userMessage, aiInstructions, string(jsonData))
	return ToolResult{
		Content: []Content{{Type: "text", Text: output}},
	}
}
