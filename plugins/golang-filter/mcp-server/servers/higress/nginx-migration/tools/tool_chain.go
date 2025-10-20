// Tool Chain implementations for AI-guided Lua to WASM conversion
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

// ConversionHints 代码转换提示
type ConversionHints struct {
	APIMappings     map[string]APIMappingDetail `json:"api_mappings"`
	CodeTemplate    string                      `json:"code_template"`
	BestPractices   []string                    `json:"best_practices"`
	ExampleSnippets map[string]string           `json:"example_snippets"`
	Warnings        []string                    `json:"warnings"`
}

// APIMappingDetail API 映射详情
type APIMappingDetail struct {
	LuaAPI         string   `json:"lua_api"`
	GoEquivalent   string   `json:"go_equivalent"`
	Description    string   `json:"description"`
	ExampleCode    string   `json:"example_code"`
	RequiresImport []string `json:"requires_import"`
	Notes          string   `json:"notes"`
}

// ValidationReport 验证报告
type ValidationReport struct {
	IsValid        bool              `json:"is_valid"`
	Errors         []ValidationError `json:"errors"`
	Warnings       []string          `json:"warnings"`
	Suggestions    []string          `json:"suggestions"`
	Score          int               `json:"score"` // 0-100
	MissingImports []string          `json:"missing_imports"`
}

// ValidationError 验证错误
type ValidationError struct {
	Type       string `json:"type"`     // syntax, api_usage, config, etc.
	Severity   string `json:"severity"` // error, warning, info
	Message    string `json:"message"`
	LineNumber int    `json:"line_number"` // 如果能检测到
	Suggestion string `json:"suggestion"`
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

// GenerateConversionHints 生成详细的转换提示
func GenerateConversionHints(analysis AnalysisResultForAI, pluginName string) ConversionHints {
	hints := ConversionHints{
		APIMappings:     make(map[string]APIMappingDetail),
		BestPractices:   []string{},
		ExampleSnippets: make(map[string]string),
		Warnings:        analysis.Warnings,
	}

	// 生成 API 映射表
	hints.APIMappings = generateAPIMappingTable()

	// 生成代码模板
	hints.CodeTemplate = generateCodeTemplate(analysis, pluginName)

	// 生成最佳实践建议
	hints.BestPractices = generateBestPractices(analysis)

	// 生成示例代码片段
	hints.ExampleSnippets = generateExampleSnippets(analysis)

	return hints
}

// generateAPIMappingTable 生成完整的 API 映射表
func generateAPIMappingTable() map[string]APIMappingDetail {
	return map[string]APIMappingDetail{
		"ngx.var.uri": {
			LuaAPI:       "ngx.var.uri",
			GoEquivalent: `proxywasm.GetHttpRequestHeader(":path")`,
			Description:  "获取请求 URI 路径",
			ExampleCode: `uri, err := proxywasm.GetHttpRequestHeader(":path")
if err != nil {
    log.Warnf("failed to get URI: %v", err)
}`,
			RequiresImport: []string{"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"},
			Notes:          "WASM 中使用伪头部 :path 表示请求路径",
		},
		"ngx.var.host": {
			LuaAPI:       "ngx.var.host",
			GoEquivalent: `proxywasm.GetHttpRequestHeader(":authority")`,
			Description:  "获取请求主机名",
			ExampleCode: `host, err := proxywasm.GetHttpRequestHeader(":authority")
if err != nil {
    log.Warnf("failed to get host: %v", err)
}`,
			RequiresImport: []string{"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"},
			Notes:          "HTTP/2 中使用 :authority 伪头部",
		},
		"ngx.var.request_method": {
			LuaAPI:       "ngx.var.request_method",
			GoEquivalent: `proxywasm.GetHttpRequestHeader(":method")`,
			Description:  "获取请求方法（GET、POST 等）",
			ExampleCode: `method, err := proxywasm.GetHttpRequestHeader(":method")
if err != nil {
    log.Warnf("failed to get method: %v", err)
}`,
			RequiresImport: []string{"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"},
		},
		"ngx.var.remote_addr": {
			LuaAPI:       "ngx.var.remote_addr",
			GoEquivalent: `proxywasm.GetHttpRequestHeader("x-forwarded-for")`,
			Description:  "获取客户端 IP 地址",
			ExampleCode: `clientIP, err := proxywasm.GetHttpRequestHeader("x-forwarded-for")
if err != nil {
    log.Warnf("failed to get client IP: %v", err)
}`,
			RequiresImport: []string{"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"},
			Notes:          "通常通过 X-Forwarded-For 头获取",
		},
		"ngx.req.get_headers": {
			LuaAPI:       "ngx.req.get_headers()",
			GoEquivalent: `proxywasm.GetHttpRequestHeaders()`,
			Description:  "获取所有请求头",
			ExampleCode: `headers, err := proxywasm.GetHttpRequestHeaders()
if err != nil {
    log.Warnf("failed to get headers: %v", err)
}
for _, h := range headers {
    log.Infof("%s: %s", h[0], h[1])
}`,
			RequiresImport: []string{"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"},
		},
		"ngx.req.set_header": {
			LuaAPI:       "ngx.req.set_header(name, value)",
			GoEquivalent: `proxywasm.ReplaceHttpRequestHeader(name, value)`,
			Description:  "设置或替换请求头",
			ExampleCode: `err := proxywasm.ReplaceHttpRequestHeader("X-Custom-Header", "value")
if err != nil {
    log.Warnf("failed to set header: %v", err)
}`,
			RequiresImport: []string{"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"},
			Notes:          "如果头不存在会自动添加",
		},
		"ngx.exit": {
			LuaAPI:       "ngx.exit(status)",
			GoEquivalent: `proxywasm.SendHttpResponse(status, headers, body, -1)`,
			Description:  "立即返回响应并终止请求",
			ExampleCode: `proxywasm.SendHttpResponse(403, 
    [][2]string{{"content-type", "text/plain"}}, 
    []byte("Forbidden"), -1)
return types.ActionPause`,
			RequiresImport: []string{
				"github.com/higress-group/proxy-wasm-go-sdk/proxywasm",
				"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types",
			},
			Notes: "需要返回 types.ActionPause 来停止请求处理",
		},
		"ngx.say": {
			LuaAPI:       "ngx.say(text)",
			GoEquivalent: `proxywasm.SendHttpResponse(200, headers, []byte(text), -1)`,
			Description:  "输出响应内容",
			ExampleCode: `proxywasm.SendHttpResponse(200,
    [][2]string{{"content-type", "text/plain"}},
    []byte("Hello from WASM"), -1)`,
			RequiresImport: []string{"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"},
		},
		"ngx.shared.DICT": {
			LuaAPI:       "ngx.shared.DICT:get/set()",
			GoEquivalent: "使用 Redis 或外部缓存服务",
			Description:  "共享字典需要外部存储替代",
			ExampleCode: `// 需要集成 Redis 客户端
// import "github.com/higress-group/wasm-go/extensions/redis"
cluster := redis.NewRedisCluster()
value, err := cluster.Get("key")`,
			RequiresImport: []string{"github.com/higress-group/wasm-go/extensions/redis"},
			Notes:          "⚠️ WASM 沙箱不支持共享内存，必须使用外部存储",
		},
		"ngx.location.capture": {
			LuaAPI:       "ngx.location.capture(uri)",
			GoEquivalent: "使用 HTTP 客户端发起子请求",
			Description:  "内部子请求需要改为 HTTP 调用",
			ExampleCode: `// 使用 wrapper.DispatchHttpCall
cluster := wrapper.NewClusterWrapper()
cluster.DispatchHttpCall(
    "backend-service",
    [][2]string{{":method", "GET"}, {":path", "/api/data"}},
    nil,
    func(statusCode int, responseHeaders [][2]string, responseBody []byte) {
        log.Infof("Response: %s", string(responseBody))
    },
    10000, // timeout in ms
)`,
			RequiresImport: []string{"github.com/higress-group/wasm-go/pkg/wrapper"},
			Notes:          "⚠️ 异步调用，需要使用回调函数",
		},
	}
}

// generateCodeTemplate 生成代码模板提示
func generateCodeTemplate(analysis AnalysisResultForAI, pluginName string) string {
	return fmt.Sprintf(`请生成以下结构的 Go WASM 插件代码：

## 包和导入
`+"```go"+`
package main

import (
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/tidwall/gjson"
)

func main() {}
`+"```"+`

## 插件注册
`+"```go"+`
func init() {
	wrapper.SetCtx(
		"%s",
		wrapper.ParseConfigBy(parseConfig),
		%s
	)
}
`+"```"+`

## 配置结构体
`+"```go"+`
type %sConfig struct {
	// 根据 Lua 代码的逻辑定义配置字段
	// 建议至少包含：
	// - 启用/禁用开关
	// - 可配置的阈值/参数
	// - 调试模式开关
}

func parseConfig(json gjson.Result, config *%sConfig, log log.Log) error {
	// 解析配置
	return nil
}
`+"```"+`

## 回调函数实现
根据检测到的特性，实现以下回调：
%s

请基于提供的 Lua 代码逻辑生成等价的 Go WASM 插件代码。`,
		pluginName,
		generateCallbackRegistrations(analysis),
		strings.Title(pluginName),
		strings.Title(pluginName),
		generateCallbackTemplates(analysis),
	)
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

// generateCallbackTemplates 生成回调函数模板
func generateCallbackTemplates(analysis AnalysisResultForAI) string {
	templates := []string{}

	if analysis.Features["ngx.var"] || analysis.Features["request_headers"] || analysis.Features["header_manipulation"] {
		templates = append(templates, `
### onHttpRequestHeaders
`+"```go"+`
func onHttpRequestHeaders(ctx wrapper.HttpContext, config YourConfig, log log.Log) types.Action {
	// 实现请求头处理逻辑
	// 1. 获取需要的变量和头部
	// 2. 执行业务逻辑
	// 3. 修改请求头（如需要）
	
	return types.ActionContinue
}
`+"```")
	}

	if analysis.Features["request_body"] {
		templates = append(templates, `
### onHttpRequestBody
`+"```go"+`
func onHttpRequestBody(ctx wrapper.HttpContext, config YourConfig, body []byte, log log.Log) types.Action {
	// 实现请求体处理逻辑
	// 1. 解析请求体
	// 2. 执行业务逻辑
	// 3. 修改请求体（如需要）
	
	return types.ActionContinue
}
`+"```")
	}

	if analysis.Features["response_headers"] || analysis.Features["response_control"] {
		templates = append(templates, `
### onHttpResponseHeaders
`+"```go"+`
func onHttpResponseHeaders(ctx wrapper.HttpContext, config YourConfig, log log.Log) types.Action {
	// 实现响应头处理逻辑
	// 如需提前返回响应：
	// proxywasm.SendHttpResponse(status, headers, body, -1)
	// return types.ActionPause
	
	return types.ActionContinue
}
`+"```")
	}

	return strings.Join(templates, "\n")
}

// generateBestPractices 生成最佳实践建议
func generateBestPractices(analysis AnalysisResultForAI) []string {
	practices := []string{
		"始终检查错误返回值，使用 log.Warnf/Errorf 记录错误",
		"避免在回调函数中执行耗时操作，保持处理逻辑轻量",
		"使用有意义的变量名，遵循 Go 命名规范（驼峰命名）",
		"添加详细的注释说明业务逻辑",
		"配置结构体字段使用 JSON 标签，便于配置解析",
	}

	if analysis.Complexity == "complex" {
		practices = append(practices,
			"复杂逻辑建议拆分为多个辅助函数",
			"考虑添加调试日志开关，便于问题排查",
		)
	}

	if len(analysis.Warnings) > 0 {
		practices = append(practices,
			"⚠️ 代码中使用了需要特殊处理的 API，请仔细阅读警告信息",
		)
	}

	if analysis.Features["shared_dict"] {
		practices = append(practices,
			"共享字典功能需要配置 Redis 等外部缓存服务",
			"注意 Redis 调用的错误处理和超时控制",
		)
	}

	if analysis.Features["internal_request"] {
		practices = append(practices,
			"内部子请求改为异步 HTTP 调用，注意处理回调逻辑",
			"设置合理的超时时间，避免请求挂起",
		)
	}

	return practices
}

// generateExampleSnippets 生成示例代码片段
func generateExampleSnippets(analysis AnalysisResultForAI) map[string]string {
	snippets := make(map[string]string)

	// 错误处理示例
	snippets["error_handling"] = `// 正确的错误处理
value, err := proxywasm.GetHttpRequestHeader("x-custom-header")
if err != nil {
    log.Warnf("failed to get header: %v", err)
    return types.ActionContinue
}

// 使用 value...
`

	// 字符串操作示例（总是提供，作为常用参考）
	snippets["string_operations"] = `// 字符串匹配示例
import "strings"

uri, _ := proxywasm.GetHttpRequestHeader(":path")
if strings.HasPrefix(uri, "/api/") {
    // API 请求处理
}

if strings.Contains(uri, "/admin") {
    // 管理路径处理
}
`

	// 正则表达式示例（总是提供，作为常用参考）
	snippets["regex_operations"] = `// 正则表达式匹配
import "regexp"

pattern := regexp.MustCompile(` + "`^/api/v[0-9]+/`" + `)
uri, _ := proxywasm.GetHttpRequestHeader(":path")
if pattern.MatchString(uri) {
    // 匹配成功
}
`

	// 自定义响应示例
	if analysis.Features["response_control"] {
		snippets["custom_response"] = `// 返回自定义响应
proxywasm.SendHttpResponse(
    403, // 状态码
    [][2]string{
        {"content-type", "application/json"},
        {"x-custom-header", "value"},
    },
    []byte(` + "`" + `{"error": "Forbidden"}` + "`" + `),
    -1, // 使用 body 的长度
)
return types.ActionPause // 停止后续处理
`
	}

	return snippets
}

// ValidateWasmCode 验证生成的 Go WASM 代码
func ValidateWasmCode(goCode, pluginName string) ValidationReport {
	report := ValidationReport{
		IsValid:        true,
		Errors:         []ValidationError{},
		Warnings:       []string{},
		Suggestions:    []string{},
		Score:          100,
		MissingImports: []string{},
	}

	// 检查必要的包声明
	if !strings.Contains(goCode, "package main") {
		report.Errors = append(report.Errors, ValidationError{
			Type:       "syntax",
			Severity:   "error",
			Message:    "缺少 'package main' 声明",
			Suggestion: "添加 package main 在文件开头",
		})
		report.IsValid = false
		report.Score -= 20
	}

	// 检查 main 函数
	if !strings.Contains(goCode, "func main()") {
		report.Errors = append(report.Errors, ValidationError{
			Type:       "syntax",
			Severity:   "error",
			Message:    "缺少 main() 函数",
			Suggestion: "添加空的 func main() {} 函数",
		})
		report.IsValid = false
		report.Score -= 15
	}

	// 检查 init 函数
	if !strings.Contains(goCode, "func init()") {
		report.Errors = append(report.Errors, ValidationError{
			Type:       "api_usage",
			Severity:   "error",
			Message:    "缺少 init() 函数",
			Suggestion: "添加 init() 函数来注册插件",
		})
		report.IsValid = false
		report.Score -= 15
	}

	// 检查 wrapper.SetCtx 调用
	if !strings.Contains(goCode, "wrapper.SetCtx") {
		report.Errors = append(report.Errors, ValidationError{
			Type:       "api_usage",
			Severity:   "error",
			Message:    "缺少 wrapper.SetCtx 调用",
			Suggestion: "在 init() 函数中调用 wrapper.SetCtx 注册插件",
		})
		report.IsValid = false
		report.Score -= 20
	}

	// 检查必要的 import
	requiredImports := map[string]string{
		"github.com/higress-group/proxy-wasm-go-sdk/proxywasm":       "proxywasm",
		"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types": "types",
		"github.com/higress-group/wasm-go/pkg/wrapper":               "wrapper",
	}

	for importPath := range requiredImports {
		if !strings.Contains(goCode, importPath) {
			report.MissingImports = append(report.MissingImports, importPath)
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("可能缺少导入: %s", importPath))
			report.Score -= 5
		}
	}

	// 检查配置结构体
	configPattern := regexp.MustCompile(`type\s+\w+Config\s+struct`)
	if !configPattern.MatchString(goCode) {
		report.Warnings = append(report.Warnings, "未找到配置结构体定义")
		report.Suggestions = append(report.Suggestions, "定义配置结构体以支持动态配置")
		report.Score -= 5
	}

	// 检查 parseConfig 函数
	if !strings.Contains(goCode, "func parseConfig") {
		report.Warnings = append(report.Warnings, "未找到 parseConfig 函数")
		report.Suggestions = append(report.Suggestions, "实现 parseConfig 函数来解析配置")
		report.Score -= 5
	}

	// 检查回调函数
	callbacks := []string{
		"onHttpRequestHeaders",
		"onHttpRequestBody",
		"onHttpResponseHeaders",
	}

	hasCallback := false
	for _, cb := range callbacks {
		if strings.Contains(goCode, "func "+cb) {
			hasCallback = true
			break
		}
	}

	if !hasCallback {
		report.Warnings = append(report.Warnings, "未找到任何回调函数实现")
		report.Suggestions = append(report.Suggestions, "至少实现一个回调函数（如 onHttpRequestHeaders）")
		report.Score -= 10
	}

	// 检查错误处理
	if strings.Count(goCode, "if err != nil") < 2 {
		report.Suggestions = append(report.Suggestions, "增加错误处理逻辑，提高代码健壮性")
		report.Score -= 5
	}

	// 检查日志记录
	if !strings.Contains(goCode, "log.") {
		report.Suggestions = append(report.Suggestions, "添加日志记录，便于调试和问题排查")
		report.Score -= 3
	}

	// 检查潜在的常见错误
	if strings.Contains(goCode, "return nil") && strings.Contains(goCode, "types.Action") {
		report.Errors = append(report.Errors, ValidationError{
			Type:       "api_usage",
			Severity:   "error",
			Message:    "回调函数不应返回 nil，应返回 types.Action",
			Suggestion: "将 return nil 改为 return types.ActionContinue",
		})
		report.IsValid = false
		report.Score -= 15
	}

	// 检查是否有未使用的导入
	if strings.Contains(goCode, `import (`) {
		report.Suggestions = append(report.Suggestions, "确保所有导入的包都被使用，移除未使用的导入")
	}

	// 确保分数不为负
	if report.Score < 0 {
		report.Score = 0
	}

	return report
}

// GenerateDeploymentPackage 生成完整的部署配置包
func GenerateDeploymentPackage(pluginName, goCode, configSchema, namespace string) DeploymentPackage {
	pkg := DeploymentPackage{
		Dependencies: make(map[string]string),
	}

	// 生成 WasmPlugin YAML
	pkg.WasmPluginYAML = fmt.Sprintf(`apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: %s
  namespace: %s
  annotations:
    higress.io/description: "Generated from Nginx Lua plugin"
    higress.io/generated-at: "auto-generated"
spec:
  # 插件优先级（数字越小优先级越高）
  priority: 100
  
  # 匹配规则（可选，默认匹配所有路由）
  matchRules:
  - config:
      # 插件配置
      enable: true
    ingress:
    - default/*  # 匹配所有 namespace 为 default 的 ingress
  
  # 默认配置
  defaultConfig:
    enable: true
  
  # 插件镜像地址
  url: oci://your-registry.io/%s:v1.0.0
  
  # 镜像拉取策略
  imagePullPolicy: IfNotPresent
  
  # 镜像拉取凭证（如果需要）
  # imagePullSecret: your-registry-secret
  
  # 插件执行阶段
  phase: UNSPECIFIED_PHASE
  
  # 插件配置 Schema（可选）
  # configSchema:
  #   openAPIV3Schema: %s
`, pluginName, namespace, pluginName, "...")

	// 生成 Makefile
	pkg.Makefile = fmt.Sprintf(`# Makefile for %s WASM plugin

PLUGIN_NAME := %s
VERSION := v1.0.0
REGISTRY := your-registry.io
IMAGE := $(REGISTRY)/$(PLUGIN_NAME):$(VERSION)

# TinyGo 编译选项
TINYGO_FLAGS := -scheduler=none -target=wasi -gc=leaking -no-debug

.PHONY: build
build:
	@echo "Building WASM plugin..."
	tinygo build $(TINYGO_FLAGS) -o main.wasm main.go
	@echo "Build complete: main.wasm"

.PHONY: docker-build
docker-build: build
	@echo "Building Docker image..."
	docker build -t $(IMAGE) .
	@echo "Docker image built: $(IMAGE)"

.PHONY: docker-push
docker-push: docker-build
	@echo "Pushing Docker image..."
	docker push $(IMAGE)
	@echo "Docker image pushed: $(IMAGE)"

.PHONY: deploy
deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -f wasmplugin.yaml
	@echo "Deployment complete"

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f main.wasm
	@echo "Clean complete"

.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Format complete"

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build WASM plugin"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-push  - Push Docker image to registry"
	@echo "  deploy       - Deploy to Kubernetes"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format code"
`, pluginName, pluginName)

	// 生成 Dockerfile
	pkg.Dockerfile = fmt.Sprintf(`# Dockerfile for %s WASM plugin

FROM scratch

# 复制编译好的 WASM 文件
COPY main.wasm /plugin.wasm

# 设置入口点
ENTRYPOINT ["/plugin.wasm"]
`, pluginName)

	// 生成 ConfigMap（如果有复杂配置）
	if configSchema != "" {
		pkg.ConfigMap = fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
  namespace: %s
data:
  config.json: |
    {
      "enable": true
    }
  
  schema.json: |
    %s
`, pluginName, namespace, configSchema)
	}

	// 生成 README
	pkg.README = fmt.Sprintf(`# %s WASM Plugin

这是一个从 Nginx Lua 插件自动转换的 Higress WASM 插件。

## 📋 功能描述

[请根据原始 Lua 代码的功能填写描述]

## 🚀 快速开始

### 1. 构建插件

确保已安装 TinyGo（版本 >= 0.28.0）：

`+"```bash"+`
# 编译 WASM 插件
make build

# 构建 Docker 镜像
make docker-build

# 推送到镜像仓库
make docker-push
`+"```"+`

### 2. 部署到 Higress

`+"```bash"+`
# 应用配置
kubectl apply -f wasmplugin.yaml

# 查看部署状态
kubectl get wasmplugin -n %s
kubectl describe wasmplugin %s -n %s
`+"```"+`

### 3. 验证插件工作

`+"```bash"+`
# 发送测试请求
curl -v http://your-domain/test-path

# 查看插件日志
kubectl logs -n higress-system -l app=higress-gateway --tail=100 | grep %s
`+"```"+`

## ⚙️ 配置说明

插件配置示例：

`+"```yaml"+`
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: %s
spec:
  defaultConfig:
    enable: true
    # 在这里添加其他配置项
`+"```"+`

### 配置字段

| 字段 | 类型 | 说明 | 默认值 |
|-----|------|------|--------|
| enable | boolean | 是否启用插件 | true |

## 🔍 开发说明

### 本地开发

`+"```bash"+`
# 格式化代码
make fmt

# 运行测试
make test

# 清理构建产物
make clean
`+"```"+`

### 依赖项

- TinyGo >= 0.28.0
- Docker
- kubectl

### 目录结构

`+"```"+`
.
├── main.go           # 插件主代码
├── wasmplugin.yaml   # K8s 部署配置
├── Makefile          # 构建脚本
├── Dockerfile        # Docker 镜像定义
└── README.md         # 说明文档
`+"```"+`

## 📚 相关文档

- [Higress WASM 插件开发指南](https://higress.io/zh-cn/docs/user/wasm-go)
- [Proxy-Wasm Go SDK](https://github.com/higress-group/proxy-wasm-go-sdk)
- [TinyGo 文档](https://tinygo.org/docs/reference/usage/)

## 🐛 问题排查

### 插件未生效

1. 检查 WasmPlugin 资源状态：
   `+"```bash"+`
   kubectl get wasmplugin -A
   `+"```"+`

2. 查看 Higress Gateway 日志：
   `+"```bash"+`
   kubectl logs -n higress-system -l app=higress-gateway
   `+"```"+`

3. 验证插件配置：
   `+"```bash"+`
   kubectl describe wasmplugin %s -n %s
   `+"```"+`

### 编译错误

- 确保 TinyGo 版本正确
- 检查 Go 模块依赖是否完整
- 尝试清理并重新构建：`+"`make clean && make build`"+`

## 📝 版本历史

- v1.0.0 - 初始版本（从 Nginx Lua 转换）

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

[请根据实际情况填写]
`,
		pluginName,
		namespace, pluginName, namespace,
		pluginName,
		pluginName,
		pluginName, namespace,
	)

	// 生成测试脚本
	pkg.TestScript = fmt.Sprintf(`#!/bin/bash
# Test script for %s plugin

set -e

echo "🧪 Testing %s WASM plugin..."

# 配置
GATEWAY_URL="${GATEWAY_URL:-http://localhost}"
TEST_PATH="${TEST_PATH:-/}"

echo ""
echo "📍 Gateway URL: $GATEWAY_URL"
echo "📍 Test Path: $TEST_PATH"
echo ""

# 测试 1: 基本请求
echo "Test 1: Basic request"
response=$(curl -s -o /dev/null -w "%%{http_code}" "$GATEWAY_URL$TEST_PATH")
if [ "$response" -eq 200 ]; then
    echo "✅ Test 1 passed (HTTP $response)"
else
    echo "❌ Test 1 failed (HTTP $response)"
    exit 1
fi

# 测试 2: 带自定义头的请求
echo ""
echo "Test 2: Request with custom headers"
response=$(curl -s -o /dev/null -w "%%{http_code}" -H "X-Test-Header: test" "$GATEWAY_URL$TEST_PATH")
if [ "$response" -eq 200 ]; then
    echo "✅ Test 2 passed (HTTP $response)"
else
    echo "❌ Test 2 failed (HTTP $response)"
    exit 1
fi

# 测试 3: 检查插件是否注入了响应头
echo ""
echo "Test 3: Check response headers"
headers=$(curl -s -I "$GATEWAY_URL$TEST_PATH")
if echo "$headers" | grep -q "x-"; then
    echo "✅ Test 3 passed (found custom headers)"
else
    echo "⚠️  Test 3: no custom headers found (may be expected)"
fi

echo ""
echo "🎉 All tests completed!"
`, pluginName, pluginName)

	// 添加依赖列表
	pkg.Dependencies = map[string]string{
		"github.com/higress-group/proxy-wasm-go-sdk": "v0.0.0-latest",
		"github.com/higress-group/wasm-go":           "v0.0.0-latest",
		"github.com/tidwall/gjson":                   "v1.17.0",
	}

	return pkg
}

// FormatToolResultWithAIContext 格式化工具结果，包含 AI 上下文
func FormatToolResultWithAIContext(userMessage, aiInstructions string, structuredData interface{}) ToolResult {
	jsonData, _ := json.MarshalIndent(structuredData, "", "  ")

	output := fmt.Sprintf(`%s

---

<ai_context>
%s
</ai_context>

---

<structured_data>
%s
</structured_data>
`,
		userMessage,
		aiInstructions,
		string(jsonData),
	)

	return ToolResult{
		Content: []Content{{
			Type: "text",
			Text: output,
		}},
	}
}
