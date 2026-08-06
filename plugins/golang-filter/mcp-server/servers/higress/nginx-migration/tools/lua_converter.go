// Lua to WASM conversion logic for Nginx migration
package tools

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// LuaAnalyzer analyzes Lua script features and generates conversion mappings
type LuaAnalyzer struct {
	Features   map[string]bool
	Variables  map[string]string
	Functions  []LuaFunction
	Warnings   []string
	Complexity string
}

type LuaFunction struct {
	Name  string
	Body  string
	Phase string // request_headers, request_body, response_headers, etc.
}

// ConversionResult holds the generated WASM plugin code
type ConversionResult struct {
	PluginName     string
	GoCode         string
	ConfigSchema   string
	Dependencies   []string
	WasmPluginYAML string
}

// AnalyzeLuaScript performs detailed analysis of Lua script
func AnalyzeLuaScript(luaCode string) *LuaAnalyzer {
	analyzer := &LuaAnalyzer{
		Features:   make(map[string]bool),
		Variables:  make(map[string]string),
		Functions:  []LuaFunction{},
		Warnings:   []string{},
		Complexity: "simple",
	}

	lines := strings.Split(luaCode, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		// 分析ngx变量使用
		analyzer.analyzeNginxVars(line)

		// 分析API调用
		analyzer.analyzeAPICalls(line)

		// 分析函数定义
		analyzer.analyzeFunctions(line, luaCode)
	}

	// 根据特性确定复杂度
	analyzer.determineComplexity()

	return analyzer
}

func (la *LuaAnalyzer) analyzeNginxVars(line string) {
	// 匹配 ngx.var.xxx 模式
	varPattern := regexp.MustCompile(`ngx\.var\.(\w+)`)
	matches := varPattern.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			la.Features["ngx.var"] = true

			// 映射常见变量到WASM等价物
			switch varName {
			case "uri":
				la.Variables[varName] = "proxywasm.GetHttpRequestHeader(\":path\")"
			case "request_method":
				la.Variables[varName] = "proxywasm.GetHttpRequestHeader(\":method\")"
			case "host":
				la.Variables[varName] = "proxywasm.GetHttpRequestHeader(\":authority\")"
			case "remote_addr":
				la.Variables[varName] = "proxywasm.GetHttpRequestHeader(\"x-forwarded-for\")"
			case "request_uri":
				la.Variables[varName] = "proxywasm.GetHttpRequestHeader(\":path\")"
			case "scheme":
				la.Variables[varName] = "proxywasm.GetHttpRequestHeader(\":scheme\")"
			default:
				la.Variables[varName] = fmt.Sprintf("proxywasm.GetHttpRequestHeader(\"%s\")", varName)
			}
		}
	}
}

func (la *LuaAnalyzer) analyzeAPICalls(line string) {
	apiCalls := map[string]string{
		"ngx.req.get_headers":   "request_headers",
		"ngx.req.get_body_data": "request_body",
		"ngx.req.read_body":     "request_body",
		"ngx.exit":              "response_control",
		"ngx.say":               "response_control",
		"ngx.print":             "response_control",
		"ngx.shared":            "shared_dict",
		"ngx.location.capture":  "internal_request",
		"ngx.req.set_header":    "header_manipulation",
		"ngx.header":            "response_headers",
	}

	for apiCall, feature := range apiCalls {
		if strings.Contains(line, apiCall) {
			la.Features[feature] = true

			// 添加特定警告
			switch feature {
			case "shared_dict":
				la.Warnings = append(la.Warnings, "共享字典需要使用Redis或其他外部缓存替代")
			case "internal_request":
				la.Warnings = append(la.Warnings, "内部请求需要改为HTTP客户端调用")
			}
		}
	}
}

func (la *LuaAnalyzer) analyzeFunctions(line string, fullCode string) {
	// 检测函数定义
	funcPattern := regexp.MustCompile(`function\s+(\w+)\s*\(`)
	matches := funcPattern.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		if len(match) > 1 {
			funcName := match[1]

			// 提取函数体 (简化实现)
			funcBody := la.extractFunctionBody(fullCode, funcName)

			// 根据函数名推断执行阶段
			phase := "request_headers"
			if strings.Contains(funcName, "body") {
				phase = "request_body"
			} else if strings.Contains(funcName, "response") {
				phase = "response_headers"
			}

			la.Functions = append(la.Functions, LuaFunction{
				Name:  funcName,
				Body:  funcBody,
				Phase: phase,
			})
		}
	}
}

func (la *LuaAnalyzer) extractFunctionBody(fullCode, funcName string) string {
	// 简化的函数体提取 - 实际实现应该更复杂
	pattern := fmt.Sprintf(`function\s+%s\s*\([^)]*\)(.*?)end`, funcName)
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(fullCode)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func (la *LuaAnalyzer) determineComplexity() {
	warningCount := len(la.Warnings)
	featureCount := len(la.Features)

	if warningCount > 3 || featureCount > 6 {
		la.Complexity = "complex"
	} else if warningCount > 1 || featureCount > 3 {
		la.Complexity = "medium"
	}
}

// ConvertLuaToWasm converts analyzed Lua script to WASM plugin
func ConvertLuaToWasm(analyzer *LuaAnalyzer, pluginName string) (*ConversionResult, error) {
	result := &ConversionResult{
		PluginName: pluginName,
		Dependencies: []string{
			"github.com/higress-group/proxy-wasm-go-sdk/proxywasm",
			"github.com/higress-group/wasm-go/pkg/wrapper",
			"github.com/higress-group/wasm-go/pkg/log",
		},
	}

	// 生成Go代码
	goCode, err := generateGoCode(analyzer, pluginName)
	if err != nil {
		return nil, err
	}
	result.GoCode = goCode

	// 生成配置模式
	result.ConfigSchema = generateConfigSchema(analyzer)

	// 生成WasmPlugin YAML
	result.WasmPluginYAML = generateWasmPluginYAML(pluginName)

	return result, nil
}

func generateGoCode(analyzer *LuaAnalyzer, pluginName string) (string, error) {
	tmpl := `// Generated WASM plugin from Lua script
// Plugin: {{.PluginName}}
package main

import (
	"net/http"
	"strings"
	
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"{{.PluginName}}",
		wrapper.ParseConfigBy(parseConfig),
		{{- if .HasRequestHeaders}}
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		{{- end}}
		{{- if .HasRequestBody}}
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		{{- end}}
		{{- if .HasResponseHeaders}}
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		{{- end}}
	)
}

type {{.ConfigTypeName}} struct {
	// Generated from Lua analysis
	{{- range .ConfigFields}}
	{{.Name}} {{.Type}} ` + "`json:\"{{.JSONName}}\"`" + `
	{{- end}}
}

func parseConfig(json gjson.Result, config *{{.ConfigTypeName}}, log log.Log) error {
	{{- range .ConfigFields}}
	config.{{.Name}} = json.Get("{{.JSONName}}").{{.ParseMethod}}()
	{{- end}}
	return nil
}

{{- if .HasRequestHeaders}}
func onHttpRequestHeaders(ctx wrapper.HttpContext, config {{.ConfigTypeName}}, log log.Log) types.Action {
	{{.RequestHeadersLogic}}
	return types.ActionContinue
}
{{- end}}

{{- if .HasRequestBody}}
func onHttpRequestBody(ctx wrapper.HttpContext, config {{.ConfigTypeName}}, body []byte, log log.Log) types.Action {
	{{.RequestBodyLogic}}
	return types.ActionContinue
}
{{- end}}

{{- if .HasResponseHeaders}}
func onHttpResponseHeaders(ctx wrapper.HttpContext, config {{.ConfigTypeName}}, log log.Log) types.Action {
	{{.ResponseHeadersLogic}}
	return types.ActionContinue
}
{{- end}}
`

	// 准备模板数据
	data := map[string]interface{}{
		"PluginName":           pluginName,
		"ConfigTypeName":       strings.Title(pluginName) + "Config",
		"HasRequestHeaders":    analyzer.Features["request_headers"] || analyzer.Features["ngx.var"],
		"HasRequestBody":       analyzer.Features["request_body"],
		"HasResponseHeaders":   analyzer.Features["response_headers"] || analyzer.Features["response_control"],
		"ConfigFields":         generateConfigFields(analyzer),
		"RequestHeadersLogic":  generateRequestHeadersLogic(analyzer),
		"RequestBodyLogic":     generateRequestBodyLogic(analyzer),
		"ResponseHeadersLogic": generateResponseHeadersLogic(analyzer),
	}

	t, err := template.New("wasm").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = t.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func generateConfigFields(analyzer *LuaAnalyzer) []map[string]string {
	fields := []map[string]string{}

	// 基于分析的特性生成配置字段
	if analyzer.Features["response_control"] {
		fields = append(fields, map[string]string{
			"Name":        "EnableCustomResponse",
			"Type":        "bool",
			"JSONName":    "enable_custom_response",
			"ParseMethod": "Bool",
		})
	}

	return fields
}

func generateRequestHeadersLogic(analyzer *LuaAnalyzer) string {
	logic := []string{}

	// 基于变量使用生成逻辑
	for varName, wasmCall := range analyzer.Variables {
		logic = append(logic, fmt.Sprintf(`
	// Access to ngx.var.%s
	%s, err := %s
	if err != nil {
		log.Warnf("Failed to get %s: %%v", err)
	}`, varName, varName, wasmCall, varName))
	}

	if analyzer.Features["header_manipulation"] {
		logic = append(logic, `
	// Header manipulation logic
	err := proxywasm.AddHttpRequestHeader("x-converted-from", "nginx-lua")
	if err != nil {
		log.Warnf("Failed to add header: %v", err)
	}`)
	}

	return strings.Join(logic, "\n")
}

func generateRequestBodyLogic(analyzer *LuaAnalyzer) string {
	if analyzer.Features["request_body"] {
		return `
	// Process request body
	bodyStr := string(body)
	log.Infof("Processing request body: %s", bodyStr)
	
	// Add your body processing logic here
	`
	}
	return "// No request body processing needed"
}

func generateResponseHeadersLogic(analyzer *LuaAnalyzer) string {
	if analyzer.Features["response_control"] {
		return `
	// Response control logic
	if config.EnableCustomResponse {
		proxywasm.SendHttpResponseWithDetail(200, "lua-converted", nil, []byte("Response from converted Lua plugin"), -1)
		return types.ActionContinue
	}
	`
	}
	return "// No response processing needed"
}

func generateConfigSchema(analyzer *LuaAnalyzer) string {
	schema := `{
	"type": "object",
	"properties": {`

	properties := []string{}

	if analyzer.Features["response_control"] {
		properties = append(properties, `
		"enable_custom_response": {
			"type": "boolean",
			"description": "Enable custom response handling"
		}`)
	}

	schema += strings.Join(properties, ",")
	schema += `
	}
}`

	return schema
}

func generateWasmPluginYAML(pluginName string) string {
	return fmt.Sprintf(`apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: %s
  namespace: higress-system
spec:
  defaultConfig:
    enable_custom_response: true
  url: oci://your-registry/%s:latest
`, pluginName, pluginName)
}
