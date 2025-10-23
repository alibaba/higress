// MCP Server implementation for Nginx Migration Tools - Standalone Mode
package standalone

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"nginx-migration-mcp/internal/rag"
	"nginx-migration-mcp/tools"
)

// NewMCPServer creates a new MCP server instance
func NewMCPServer(config *ServerConfig) *MCPServer {
	// 初始化 RAG 管理器
	ragConfig, err := rag.LoadRAGConfig("config/rag.json")
	if err != nil {
		log.Printf("⚠️  Failed to load RAG config: %v, RAG will be disabled", err)
		ragConfig = &rag.RAGConfig{Enabled: false}
	}

	ragManager := rag.NewRAGManager(ragConfig)

	return &MCPServer{
		config:     config,
		ragManager: ragManager,
	}
}

// HandleMessage processes an incoming MCP message
func (s *MCPServer) HandleMessage(msg MCPMessage) MCPMessage {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "tools/list":
		return s.handleToolsList(msg)
	case "tools/call":
		return s.handleToolsCall(msg)
	default:
		return s.errorResponse(msg.ID, -32601, "Method not found")
	}
}

func (s *MCPServer) handleInitialize(msg MCPMessage) MCPMessage {
	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": true,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    s.config.Server.Name,
				"version": s.config.Server.Version,
			},
		},
	}
}

func (s *MCPServer) handleToolsList(msg MCPMessage) MCPMessage {
	toolsList := tools.GetMCPTools()

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"tools": toolsList,
		},
	}
}

func (s *MCPServer) handleToolsCall(msg MCPMessage) MCPMessage {
	var params CallToolParams
	paramsBytes, _ := json.Marshal(msg.Params)
	json.Unmarshal(paramsBytes, &params)

	handlers := tools.GetToolHandlers(s)
	handler, exists := handlers[params.Name]

	if !exists {
		return s.errorResponse(msg.ID, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
	}

	result := handler(params.Arguments)

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  result,
	}
}

func (s *MCPServer) errorResponse(id interface{}, code int, message string) MCPMessage {
	return MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
}

// Tool implementations

func (s *MCPServer) parseNginxConfig(args map[string]interface{}) tools.ToolResult {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing config_content"}}}
	}

	serverCount := strings.Count(configContent, "server {")
	locationCount := strings.Count(configContent, "location")
	hasSSL := strings.Contains(configContent, "ssl")
	hasProxy := strings.Contains(configContent, "proxy_pass")
	hasRewrite := strings.Contains(configContent, "rewrite")

	complexity := "Simple"
	if serverCount > 1 || (hasRewrite && hasSSL) {
		complexity = "Complex"
	} else if hasRewrite || hasSSL {
		complexity = "Medium"
	}

	analysis := fmt.Sprintf(`Nginx配置分析结果

基础信息:
- Server块: %d个
- Location块: %d个  
- SSL配置: %t
- 反向代理: %t
- URL重写: %t

复杂度: %s

迁移建议:`, serverCount, locationCount, hasSSL, hasProxy, hasRewrite, complexity)

	if hasProxy {
		analysis += "\n- 反向代理将转换为HTTPRoute backendRefs"
	}
	if hasRewrite {
		analysis += "\n- URL重写将使用URLRewrite过滤器"
	}
	if hasSSL {
		analysis += "\n- SSL配置需要迁移到Gateway资源"
	}

	return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: analysis}}}
}

func (s *MCPServer) convertToHigress(args map[string]interface{}) tools.ToolResult {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing config_content"}}}
	}

	namespace := s.config.Defaults.Namespace
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	hostname := s.config.Defaults.Hostname
	lines := strings.Split(configContent, "\n")
	for _, line := range lines {
		if strings.Contains(line, "server_name") && !strings.Contains(line, "#") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				hostname = strings.TrimSuffix(parts[1], ";")
				break
			}
		}
	}

	yamlConfig := fmt.Sprintf(`转换后的Higress配置

apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s
  namespace: %s
  annotations:
    higress.io/migrated-from: "nginx"
spec:
  parentRefs:
  - name: %s
    namespace: %s
  hostnames:
  - %s
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: %s
    backendRefs:
    - name: %s
      port: %d

---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    app: backend
  ports:
  - port: %d
    targetPort: %d

转换完成

应用步骤:
1. 保存为 higress-config.yaml
2. 执行: kubectl apply -f higress-config.yaml
3. 验证: kubectl get httproute -n %s`,
		s.config.GenerateRouteName(hostname), namespace,
		s.config.Gateway.Name, s.config.Gateway.Namespace, hostname, s.config.Defaults.PathPrefix,
		s.config.GenerateServiceName(hostname), s.config.Service.DefaultPort,
		s.config.GenerateServiceName(hostname), namespace,
		s.config.Service.DefaultPort, s.config.Service.DefaultTarget, namespace)

	return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: yamlConfig}}}
}

func (s *MCPServer) analyzeLuaPlugin(args map[string]interface{}) tools.ToolResult {
	luaCode, ok := args["lua_code"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing lua_code"}}}
	}

	// 使用新的 AI 友好分析
	analysis := tools.AnalyzeLuaPluginForAI(luaCode)

	// 生成用户友好的消息
	features := []string{}
	for feature := range analysis.Features {
		features = append(features, fmt.Sprintf("- %s", feature))
	}

	userMessage := fmt.Sprintf(`✅ Lua 插件分析完成

📊 **检测到的特性**：
%s

⚠️ **兼容性警告**：
%s

📈 **复杂度**：%s
🔄 **兼容性级别**：%s

💡 **迁移建议**：`,
		strings.Join(features, "\n"),
		strings.Join(analysis.Warnings, "\n- "),
		analysis.Complexity,
		analysis.Compatibility,
	)

	switch analysis.Compatibility {
	case "full":
		userMessage += "\n- 可直接迁移到 WASM 插件\n- 建议使用工具链进行转换"
	case "partial":
		userMessage += "\n- 需要部分重构\n- 强烈建议使用工具链并让 AI 参与代码生成"
	case "manual":
		userMessage += "\n- 需要手动重写\n- 建议分步骤进行，使用工具链辅助"
	}

	userMessage += "\n\n🔗 **后续操作**：\n"
	userMessage += "1. 调用 `generate_conversion_hints` 工具获取详细的转换提示\n"
	userMessage += "2. 基于提示生成 Go WASM 代码\n"
	userMessage += "3. 调用 `validate_wasm_code` 工具验证生成的代码\n"
	userMessage += "4. 调用 `generate_deployment_config` 工具生成部署配置\n"
	userMessage += "\n或者直接使用 `convert_lua_to_wasm` 进行一键转换。"

	// 生成 AI 指令
	aiInstructions := fmt.Sprintf(`你现在已经获得了 Lua 插件的分析结果。基于这些信息，你可以：

### 选项 1：使用工具链进行精细控制

调用 generate_conversion_hints 工具，传入以下分析结果：
`+"```json"+`
{
  "analysis_result": %s,
  "plugin_name": "your-plugin-name"
}
`+"```"+`

这将为你提供代码生成模板，然后基于模板生成 Go WASM 代码。

### 选项 2：一键转换

如果用户希望快速转换，可以直接调用 convert_lua_to_wasm 工具。

### 建议的对话流程

1. **询问用户**：是否需要详细的转换提示，还是直接生成代码？
2. **如果需要提示**：调用 generate_conversion_hints
3. **生成代码后**：询问是否需要验证（调用 validate_wasm_code）
4. **验证通过后**：询问是否需要生成部署配置（调用 generate_deployment_config）

### 关键注意事项

%s

### 代码生成要点

- 检测到的 Nginx 变量需要映射到 HTTP 头部
- 复杂度为 %s，请相应调整代码结构
- 兼容性级别为 %s，注意处理警告中的问题
`,
		string(mustMarshalJSON(analysis)),
		formatWarningsForAI(analysis.Warnings),
		analysis.Complexity,
		analysis.Compatibility,
	)

	return tools.FormatToolResultWithAIContext(userMessage, aiInstructions, analysis)
}

func mustMarshalJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func formatWarningsForAI(warnings []string) string {
	if len(warnings) == 0 {
		return "- 无特殊警告，可以直接转换"
	}
	result := []string{}
	for _, w := range warnings {
		result = append(result, fmt.Sprintf("- ⚠️ %s", w))
	}
	return strings.Join(result, "\n")
}

func (s *MCPServer) convertLuaToWasm(args map[string]interface{}) tools.ToolResult {
	luaCode, ok := args["lua_code"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing lua_code"}}}
	}

	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing plugin_name"}}}
	}

	analyzer := tools.AnalyzeLuaScript(luaCode)
	result, err := tools.ConvertLuaToWasm(analyzer, pluginName)
	if err != nil {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}}}
	}

	response := fmt.Sprintf(`Lua脚本转换完成

转换分析:
- 复杂度: %s
- 检测特性: %d个
- 兼容性警告: %d个

注意事项:
%s

生成的文件:

==== main.go ====
%s

==== WasmPlugin配置 ====
%s

部署步骤:
1. 创建插件目录: mkdir -p extensions/%s
2. 保存Go代码到: extensions/%s/main.go  
3. 构建插件: PLUGIN_NAME=%s make build
4. 应用配置: kubectl apply -f wasmplugin.yaml

提示:
- 请根据实际需求调整配置
- 测试插件功能后再部署到生产环境
- 如有共享状态需求，请配置Redis等外部存储
`,
		analyzer.Complexity,
		len(analyzer.Features),
		len(analyzer.Warnings),
		strings.Join(analyzer.Warnings, "\n- "),
		result.GoCode,
		result.WasmPluginYAML,
		pluginName, pluginName, pluginName)

	return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: response}}}
}

// GenerateConversionHints 生成详细的代码转换提示
func (s *MCPServer) GenerateConversionHints(args map[string]interface{}) tools.ToolResult {
	analysisResultStr, ok := args["analysis_result"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing analysis_result"}}}
	}

	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing plugin_name"}}}
	}

	// 解析分析结果
	var analysis tools.AnalysisResultForAI
	if err := json.Unmarshal([]byte(analysisResultStr), &analysis); err != nil {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: fmt.Sprintf("Error parsing analysis_result: %v", err)}}}
	}

	// 生成转换提示
	hints := tools.GenerateConversionHints(analysis, pluginName)

	// === RAG 增强：查询 Nginx API 转换文档 ===
	var ragDocs string

	// 构建更精确的查询语句
	queryBuilder := []string{}
	if len(analysis.APICalls) > 0 {
		queryBuilder = append(queryBuilder, "Nginx Lua API 转换到 Higress WASM")

		// 针对不同的 API 类型使用不同的查询关键词
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

		// 添加具体的 API 调用
		if len(analysis.APICalls) > 0 && len(analysis.APICalls) <= 5 {
			queryBuilder = append(queryBuilder, fmt.Sprintf("涉及 API: %s", strings.Join(analysis.APICalls, ", ")))
		}
	} else {
		queryBuilder = append(queryBuilder, "Higress WASM 插件开发 基础示例 Go SDK 使用")
	}

	// 添加复杂度相关的查询
	if analysis.Complexity == "high" {
		queryBuilder = append(queryBuilder, "复杂插件实现 高级功能")
	}

	queryString := strings.Join(queryBuilder, " ")

	ragContext, err := s.ragManager.QueryForTool(
		"generate_conversion_hints",
		queryString,
		"lua_migration",
	)

	if err == nil && ragContext.Enabled && len(ragContext.Documents) > 0 {
		log.Printf("✅ RAG: Found %d documents for conversion hints", len(ragContext.Documents))
		ragDocs = "\n\n## 📚 参考文档（来自知识库）\n\n" + ragContext.FormatContextForAI()
	} else {
		if err != nil {
			log.Printf("⚠️  RAG query failed: %v", err)
		}
		ragDocs = ""
	}

	// 格式化输出
	userMessage := fmt.Sprintf(`🎯 代码转换提示

**插件名称**: %s
**代码模板**: %s
**RAG 状态**: %s

%s
`,
		pluginName,
		hints.CodeTemplate,
		func() string {
			if ragContext != nil && ragContext.Enabled {
				return fmt.Sprintf("✅ 已加载 %d 个参考文档", len(ragContext.Documents))
			}
			return "⚡ 使用规则库（RAG 未启用）"
		}(),
		func() string {
			if len(hints.Warnings) > 0 {
				return "⚠️ **警告**: " + formatWarningsListForUser(hints.Warnings)
			}
			return ""
		}(),
	)

	// 生成详细的 AI 指令
	aiInstructions := fmt.Sprintf(`现在你需要基于以下信息生成 Go WASM 插件代码。

## 📋 任务概述

**插件名称**: %s
**原始 Lua 特性**: %s
**复杂度**: %s
**兼容性**: %s

## 🎯 代码模板

%s
%s

## ✅ 生成代码的要求

### 必须实现
1. **实现所需的回调函数**: %s
2. **保持 Lua 代码的业务逻辑完全等价**
3. **包含完整的错误处理逻辑**
4. **实现配置解析函数（如果需要动态配置）**

### 代码质量
5. **添加清晰的注释**：标注每段代码对应的原始 Lua 逻辑
6. **遵循 Go 代码规范**：使用驼峰命名，适当的包结构
7. **添加日志记录**：关键步骤使用 log.Info/Warn/Error
8. **错误返回规范**：失败时返回 types.ActionPause，成功返回 types.ActionContinue
%s

### 性能优化
9. **避免不必要的内存分配**
10. **合理使用缓存**（如果涉及重复查询）

## 📚 参考资源

- Higress WASM Go SDK 文档: https://higress.io/zh-cn/docs/user/wasm-go
%s

## 📤 输出格式

请按以下格式输出代码：

### main.go
`+"```go"+`
[完整的 Go 代码，包含所有必要的导入、配置结构体、init函数和回调函数]
`+"```"+`

### 代码说明
简要说明：
- 实现了哪些回调函数
- 如何处理错误情况
- 与原 Lua 代码的对应关系

生成代码后，**强烈建议**调用 validate_wasm_code 工具进行验证。
`,
		pluginName,
		formatFeaturesList(analysis.Features),
		analysis.Complexity,
		analysis.Compatibility,
		hints.CodeTemplate,
		ragDocs,
		hints.CodeTemplate, // 再次显示模板作为提醒
		func() string {
			if ragContext != nil && ragContext.Enabled && len(ragContext.Documents) > 0 {
				return "\n\n### 知识库参考\n11. **优先参考上述知识库文档中的示例代码和最佳实践**\n12. **使用文档中推荐的 API 调用方式**"
			}
			return ""
		}(),
		func() string {
			if ragContext != nil && ragContext.Enabled && len(ragContext.Documents) > 0 {
				return fmt.Sprintf("- 已从知识库检索到 %d 个相关文档（见上方）", len(ragContext.Documents))
			}
			return "- 使用基于规则的代码生成"
		}(),
	)

	return tools.FormatToolResultWithAIContext(userMessage, aiInstructions, hints)
}

// ValidateWasmCode 验证生成的 Go WASM 代码
func (s *MCPServer) ValidateWasmCode(args map[string]interface{}) tools.ToolResult {
	goCode, ok := args["go_code"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing go_code"}}}
	}

	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing plugin_name"}}}
	}

	// 执行验证
	report := tools.ValidateWasmCode(goCode, pluginName)

	// 统计各类问题数量
	requiredCount := 0
	recommendedCount := 0
	optionalCount := 0
	bestPracticeCount := 0

	for _, issue := range report.Issues {
		switch issue.Category {
		case "required":
			requiredCount++
		case "recommended":
			recommendedCount++
		case "optional":
			optionalCount++
		case "best_practice":
			bestPracticeCount++
		}
	}

	// 构建用户消息
	userMessage := fmt.Sprintf(`##  代码验证报告

%s

### 发现的回调函数 (%d 个)
%s

### 配置结构
%s

### 问题分类

####  必须修复 (%d 个)
%s

####  建议修复 (%d 个)
%s

####  可选优化 (%d 个)
%s

####  最佳实践 (%d 个)
%s

### 缺失的导入包 (%d 个)
%s

---

`,
		report.Summary,
		len(report.FoundCallbacks),
		formatCallbacksList(report.FoundCallbacks),
		formatConfigStatus(report.HasConfig),
		requiredCount,
		formatIssuesByCategory(report.Issues, "required"),
		recommendedCount,
		formatIssuesByCategory(report.Issues, "recommended"),
		optionalCount,
		formatIssuesByCategory(report.Issues, "optional"),
		bestPracticeCount,
		formatIssuesByCategory(report.Issues, "best_practice"),
		len(report.MissingImports),
		formatList(report.MissingImports),
	)

	// === RAG 增强：查询最佳实践和代码规范 ===
	var ragBestPractices string

	// 根据验证结果构建更针对性的查询
	queryBuilder := []string{"Higress WASM 插件"}

	// 根据发现的问题类型添加关键词
	if requiredCount > 0 || recommendedCount > 0 {
		queryBuilder = append(queryBuilder, "常见错误")

		// 检查具体问题类型
		for _, issue := range report.Issues {
			switch issue.Type {
			case "error_handling":
				queryBuilder = append(queryBuilder, "错误处理")
			case "api_usage":
				queryBuilder = append(queryBuilder, "API 使用规范")
			case "config":
				queryBuilder = append(queryBuilder, "配置解析")
			case "logging":
				queryBuilder = append(queryBuilder, "日志记录")
			}
		}
	} else {
		// 代码已通过基础验证，查询优化建议
		queryBuilder = append(queryBuilder, "性能优化 最佳实践")
	}

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

	// 如果有缺失的导入，查询包管理相关信息
	if len(report.MissingImports) > 0 {
		queryBuilder = append(queryBuilder, "依赖包导入")
	}

	queryString := strings.Join(queryBuilder, " ")

	ragContext, err := s.ragManager.QueryForTool(
		"validate_wasm_code",
		queryString,
		"best_practice",
	)

	if err == nil && ragContext.Enabled && len(ragContext.Documents) > 0 {
		log.Printf("✅ RAG: Found %d best practice documents", len(ragContext.Documents))
		ragBestPractices = "\n\n### 📚 最佳实践建议（来自知识库）\n\n" + ragContext.FormatContextForAI()
		userMessage += ragBestPractices
	} else {
		if err != nil {
			log.Printf("⚠️  RAG query failed for validation: %v", err)
		}
	}

	// 根据问题级别给出建议
	hasRequired := requiredCount > 0
	if hasRequired {
		userMessage += "\n **请优先修复 \"必须修复\" 的问题，否则代码可能无法编译或运行。**\n\n"
	} else if recommendedCount > 0 {
		userMessage += "\n **代码基本结构正确。** 建议修复 \"建议修复\" 的问题以提高代码质量。\n\n"
	} else {
		userMessage += "\n **代码验证通过！** 可以继续生成部署配置。\n\n"
		userMessage += "**下一步**：调用 `generate_deployment_config` 工具生成部署配置。\n"
	}

	// AI 指令
	aiInstructions := ""
	if hasRequired {
		aiInstructions = `代码验证发现必须修复的问题。

## 修复指南

` + formatIssuesForAI(report.Issues, "required") + `

请修复上述问题后，再次调用 validate_wasm_code 工具进行验证。
`
	} else if recommendedCount > 0 {
		aiInstructions = `代码基本结构正确，建议修复以下问题：

` + formatIssuesForAI(report.Issues, "recommended") + `

可以选择修复这些问题，或直接调用 generate_deployment_config 工具生成部署配置。
`
	} else {
		aiInstructions = `代码验证通过！

## 下一步

调用 generate_deployment_config 工具，参数：
` + "```json" + `
{
  "plugin_name": "` + pluginName + `",
  "go_code": "[验证通过的代码]",
  "namespace": "higress-system"
}
` + "```" + `

这将生成完整的部署配置包。
`
	}

	return tools.FormatToolResultWithAIContext(userMessage, aiInstructions, report)
}

// GenerateDeploymentConfig 生成部署配置
func (s *MCPServer) GenerateDeploymentConfig(args map[string]interface{}) tools.ToolResult {
	pluginName, ok := args["plugin_name"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing plugin_name"}}}
	}

	goCode, ok := args["go_code"].(string)
	if !ok {
		return tools.ToolResult{Content: []tools.Content{{Type: "text", Text: "Error: Missing go_code"}}}
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
	userMessage := fmt.Sprintf(`🎉 部署配置生成完成！

已为插件 **%s** 生成完整的部署配置包。

##  生成的文件

### 1. WasmPlugin 配置
- 文件名：wasmplugin.yaml
- 命名空间：%s
- 包含默认配置和匹配规则

### 2. 构建脚本
- Makefile：自动化构建和部署
- Dockerfile：容器化打包

### 3. 文档
- README.md：完整的使用说明
- 包含快速开始、配置说明、问题排查

### 4. 测试脚本
- test.sh：自动化测试脚本

### 5. 依赖清单
- 列出了所有必需的 Go 模块

---

##  快速部署

`+"```bash"+`
# 1. 保存文件
# 保存 main.go
# 保存 wasmplugin.yaml
# 保存 Makefile
# 保存 Dockerfile

# 2. 构建插件
make build

# 3. 构建并推送镜像
make docker-build docker-push

# 4. 部署到 Kubernetes
make deploy

# 5. 验证部署
kubectl get wasmplugin -n %s
`+"```"+`

---

**文件内容请见下方结构化数据部分。**
`,
		pluginName,
		namespace,
		namespace,
	)

	aiInstructions := fmt.Sprintf(`部署配置已生成完毕。

## 向用户展示文件

请将以下文件内容清晰地展示给用户：

### 1. main.go
用户已经有这个文件。

### 2. wasmplugin.yaml
`+"```yaml"+`
%s
`+"```"+`

### 3. Makefile
`+"```makefile"+`
%s
`+"```"+`

### 4. Dockerfile
`+"```dockerfile"+`
%s
`+"```"+`

### 5. README.md
`+"```markdown"+`
%s
`+"```"+`

### 6. test.sh
`+"```bash"+`
%s
`+"```"+`

## 后续支持

询问用户是否需要：
1. 解释任何配置项的含义
2. 自定义某些配置
3. 帮助解决部署问题
`,
		pkg.WasmPluginYAML,
		pkg.Makefile,
		pkg.Dockerfile,
		pkg.README,
		pkg.TestScript,
	)

	return tools.FormatToolResultWithAIContext(userMessage, aiInstructions, pkg)
}

// 辅助格式化函数

func formatWarningsListForUser(warnings []string) string {
	if len(warnings) == 0 {
		return "无"
	}
	return strings.Join(warnings, "\n- ")
}

func formatFeaturesList(features map[string]bool) string {
	featureNames := map[string]string{
		"request_headers":     "请求头处理",
		"response_headers":    "响应头处理",
		"header_manipulation": "请求头修改",
		"request_body":        "请求体处理",
		"response_body":       "响应体处理",
		"response_control":    "响应控制",
		"upstream":            "上游服务",
		"redirect":            "重定向",
		"rewrite":             "URL重写",
	}

	var result []string
	for key, enabled := range features {
		if enabled {
			if name, ok := featureNames[key]; ok {
				result = append(result, name)
			} else {
				result = append(result, key)
			}
		}
	}

	if len(result) == 0 {
		return "基础功能"
	}
	return strings.Join(result, ", ")
}

func formatCallbacksList(callbacks []string) string {
	if len(callbacks) == 0 {
		return "无"
	}
	return "- " + strings.Join(callbacks, "\n- ")
}

func formatConfigStatus(hasConfig bool) string {
	if hasConfig {
		return " 已定义配置结构体"
	}
	return "- 未定义配置结构体（如不需要配置可忽略）"
}

func formatIssuesByCategory(issues []tools.ValidationIssue, category string) string {
	var filtered []string
	for _, issue := range issues {
		if issue.Category == category {
			filtered = append(filtered, fmt.Sprintf("- **[%s]** %s\n  💡 建议: %s\n  📌 影响: %s",
				issue.Type, issue.Message, issue.Suggestion, issue.Impact))
		}
	}
	if len(filtered) == 0 {
		return "无"
	}
	return strings.Join(filtered, "\n\n")
}

func formatIssuesForAI(issues []tools.ValidationIssue, category string) string {
	var filtered []tools.ValidationIssue
	for _, issue := range issues {
		if issue.Category == category {
			filtered = append(filtered, issue)
		}
	}

	if len(filtered) == 0 {
		return "无问题"
	}

	result := []string{}
	for i, issue := range filtered {
		result = append(result, fmt.Sprintf(`
### 问题 %d: %s

**类型**: %s
**建议**: %s
**影响**: %s

请根据建议修复此问题。
`,
			i+1,
			issue.Message,
			issue.Type,
			issue.Suggestion,
			issue.Impact,
		))
	}
	return strings.Join(result, "\n")
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "无"
	}
	return "- " + strings.Join(items, "\n- ")
}
