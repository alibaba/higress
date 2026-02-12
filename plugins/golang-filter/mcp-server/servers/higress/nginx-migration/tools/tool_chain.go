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
参考文档: https://higress.cn/docs/latest/user/wasm-go/`,
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

// ValidateWasmCode 验证生成的 Go WASM 代码
func ValidateWasmCode(goCode, pluginName string) ValidationReport {
	report := ValidationReport{
		Issues:         []ValidationIssue{},
		MissingImports: []string{},
		FoundCallbacks: []string{},
		HasConfig:      false,
	}

	// 移除注释以避免误判
	codeWithoutComments := removeComments(goCode)

	// 检查必要的包声明
	packagePattern := regexp.MustCompile(`(?m)^package\s+main\s*$`)
	if !packagePattern.MatchString(goCode) {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "required",
			Type:       "syntax",
			Message:    "缺少 'package main' 声明",
			Suggestion: "在文件开头添加: package main",
			Impact:     "WASM 插件必须使用 package main，否则无法编译",
		})
	}

	// 检查 main 函数
	mainFuncPattern := regexp.MustCompile(`func\s+main\s*\(\s*\)`)
	if !mainFuncPattern.MatchString(codeWithoutComments) {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "required",
			Type:       "syntax",
			Message:    "缺少 main() 函数",
			Suggestion: "添加空的 main 函数: func main() {}",
			Impact:     "WASM 插件必须有 main 函数，即使是空的",
		})
	}

	// 检查 init 函数
	initFuncPattern := regexp.MustCompile(`func\s+init\s*\(\s*\)`)
	if !initFuncPattern.MatchString(codeWithoutComments) {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "required",
			Type:       "api_usage",
			Message:    "缺少 init() 函数",
			Suggestion: "添加 init() 函数用于注册插件",
			Impact:     "插件需要在 init() 中调用 wrapper.SetCtx 进行注册",
		})
	}

	// 检查 wrapper.SetCtx 调用
	setCtxPattern := regexp.MustCompile(`wrapper\.SetCtx\s*\(`)
	if !setCtxPattern.MatchString(codeWithoutComments) {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "required",
			Type:       "api_usage",
			Message:    "缺少 wrapper.SetCtx 调用",
			Suggestion: "在 init() 函数中调用 wrapper.SetCtx 注册插件上下文",
			Impact:     "没有注册插件上下文将导致插件无法工作",
		})
	}

	// 检查必要的 import
	requiredImports := map[string]string{
		"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types": "定义了 Action 等核心类型",
		"github.com/higress-group/wasm-go/pkg/wrapper":               "提供了 Higress 插件开发的高级封装",
	}

	for importPath, reason := range requiredImports {
		if !containsImport(goCode, importPath) {
			report.MissingImports = append(report.MissingImports, importPath)
			report.Issues = append(report.Issues, ValidationIssue{
				Category:   "required",
				Type:       "imports",
				Message:    fmt.Sprintf("缺少必需的导入: %s", importPath),
				Suggestion: fmt.Sprintf(`添加导入: import "%s"`, importPath),
				Impact:     reason,
			})
		}
	}

	// 检查可选但推荐的 import
	if !containsImport(goCode, "github.com/higress-group/proxy-wasm-go-sdk/proxywasm") {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "optional",
			Type:       "imports",
			Message:    "未导入 proxywasm 包",
			Suggestion: "如需使用日志、HTTP 调用等底层 API，可导入 proxywasm 包",
			Impact:     "proxywasm 提供了日志记录、外部 HTTP 调用等功能",
		})
	}

	// 检查配置结构体
	configPattern := regexp.MustCompile(`type\s+\w+Config\s+struct\s*\{`)
	report.HasConfig = configPattern.MatchString(goCode)

	if !report.HasConfig {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "optional",
			Type:       "config",
			Message:    "未定义配置结构体",
			Suggestion: "如果插件需要配置参数，建议定义配置结构体（如 type MyPluginConfig struct { ... }）",
			Impact:     "配置结构体用于接收和解析插件的配置参数，支持动态配置",
		})
	}

	// 检查 parseConfig 函数
	parseConfigPattern := regexp.MustCompile(`func\s+parseConfig\s*\(`)
	hasParseConfig := parseConfigPattern.MatchString(codeWithoutComments)

	if report.HasConfig && !hasParseConfig {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "recommended",
			Type:       "config",
			Message:    "定义了配置结构体但缺少 parseConfig 函数",
			Suggestion: "实现 parseConfig 函数来解析配置: func parseConfig(json gjson.Result, config *MyPluginConfig, log wrapper.Log) error",
			Impact:     "parseConfig 函数负责将 JSON 配置解析到结构体，是配置系统的核心",
		})
	}

	// 检查回调函数
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

	if len(report.FoundCallbacks) == 0 {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "required",
			Type:       "api_usage",
			Message:    "未找到任何 HTTP 回调函数实现",
			Suggestion: "至少实现一个回调函数，如: func onHttpRequestHeaders(ctx wrapper.HttpContext, config MyPluginConfig, log wrapper.Log) types.Action",
			Impact:     "回调函数是插件逻辑的核心，没有回调函数插件将不会执行任何操作",
		})
	}

	// 检查错误处理
	errHandlingCount := strings.Count(codeWithoutComments, "if err != nil")
	funcCount := strings.Count(codeWithoutComments, "func ")
	if funcCount > 3 && errHandlingCount == 0 {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "best_practice",
			Type:       "error_handling",
			Message:    "代码中缺少错误处理",
			Suggestion: "对可能返回错误的操作添加错误检查: if err != nil { ... }",
			Impact:     "良好的错误处理可以提高插件的健壮性和可调试性",
		})
	}

	// 检查日志记录
	hasLogging := strings.Contains(codeWithoutComments, "proxywasm.Log") ||
		strings.Contains(codeWithoutComments, "log.Error") ||
		strings.Contains(codeWithoutComments, "log.Warn") ||
		strings.Contains(codeWithoutComments, "log.Info") ||
		strings.Contains(codeWithoutComments, "log.Debug")

	if !hasLogging {
		report.Issues = append(report.Issues, ValidationIssue{
			Category:   "best_practice",
			Type:       "logging",
			Message:    "代码中没有日志记录",
			Suggestion: "添加适当的日志记录，如: proxywasm.LogInfo(), log.Errorf() 等",
			Impact:     "日志记录有助于调试、监控和问题排查",
		})
	}

	// 检查回调函数的返回值
	checkCallbackReturnErrors(&report, codeWithoutComments, report.FoundCallbacks)

	// 生成总体评估摘要
	report.Summary = generateValidationSummary(report)

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

// containsImport 检查是否包含特定的 import
func containsImport(code, importPath string) bool {
	// 匹配 import "path" 或 import ("path")
	pattern := regexp.MustCompile(`import\s+(?:\([\s\S]*?)?["` + "`" + `]` +
		regexp.QuoteMeta(importPath) + `["` + "`" + `]`)
	return pattern.MatchString(code)
}

// checkCallbackReturnErrors 检查回调函数的返回值错误
func checkCallbackReturnErrors(report *ValidationReport, code string, foundCallbacks []string) {
	// 检查回调函数内是否有 return nil（应该返回 types.Action）
	for _, callback := range foundCallbacks {
		// 提取回调函数体（简化的检查）
		funcPattern := regexp.MustCompile(
			`func\s+` + callback + `\s*\([^)]*\)\s+types\.Action\s*\{[^}]*return\s+nil[^}]*\}`)
		if funcPattern.MatchString(code) {
			report.Issues = append(report.Issues, ValidationIssue{
				Category:   "required",
				Type:       "api_usage",
				Message:    fmt.Sprintf("回调函数 %s 不应返回 nil", callback),
				Suggestion: "回调函数应返回 types.Action，如: return types.ActionContinue",
				Impact:     "返回 nil 会导致编译错误或运行时异常",
			})
			break // 只报告一次
		}
	}

	// 检查是否正确返回 types.Action
	if len(foundCallbacks) > 0 {
		hasActionReturn := strings.Contains(code, "types.ActionContinue") ||
			strings.Contains(code, "types.ActionPause") ||
			strings.Contains(code, "types.ActionSuspend")

		if !hasActionReturn {
			report.Issues = append(report.Issues, ValidationIssue{
				Category:   "recommended",
				Type:       "api_usage",
				Message:    "未找到明确的 Action 返回值",
				Suggestion: "回调函数应返回明确的 types.Action 值（ActionContinue、ActionPause 等）",
				Impact:     "明确的返回值有助于代码可读性和正确性",
			})
		}
	}
}

// generateValidationSummary 生成验证摘要
func generateValidationSummary(report ValidationReport) string {
	requiredIssues := 0
	recommendedIssues := 0
	optionalIssues := 0
	bestPracticeIssues := 0

	for _, issue := range report.Issues {
		switch issue.Category {
		case "required":
			requiredIssues++
		case "recommended":
			recommendedIssues++
		case "optional":
			optionalIssues++
		case "best_practice":
			bestPracticeIssues++
		}
	}

	if requiredIssues > 0 {
		return fmt.Sprintf("代码存在 %d 个必须修复的问题，%d 个建议修复的问题，%d 个可选优化项，%d 个最佳实践建议。请优先解决必须修复的问题。",
			requiredIssues, recommendedIssues, optionalIssues, bestPracticeIssues)
	}

	if recommendedIssues > 0 {
		return fmt.Sprintf("代码基本结构正确，但有 %d 个建议修复的问题，%d 个可选优化项，%d 个最佳实践建议。",
			recommendedIssues, optionalIssues, bestPracticeIssues)
	}

	if optionalIssues > 0 || bestPracticeIssues > 0 {
		return fmt.Sprintf("代码结构良好，有 %d 个可选优化项和 %d 个最佳实践建议可以考虑。",
			optionalIssues, bestPracticeIssues)
	}

	callbacksInfo := ""
	if len(report.FoundCallbacks) > 0 {
		callbacksInfo = fmt.Sprintf("，实现了 %d 个回调函数", len(report.FoundCallbacks))
	}

	return fmt.Sprintf("代码验证通过，未发现明显问题%s。", callbacksInfo)
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
