// MCP Server implementation for Nginx Migration Tools - Standalone Mode
package standalone

import (
	"encoding/json"
	"fmt"
	"strings"

	"nginx-migration-mcp/tools"
)

// NewMCPServer creates a new MCP server instance
func NewMCPServer(config *ServerConfig) *MCPServer {
	return &MCPServer{config: config}
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

这将为你提供：
- 详细的 API 映射表
- 代码生成模板
- 最佳实践建议
- 示例代码片段

然后基于这些提示生成 Go WASM 代码。

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

	// 格式化输出
	userMessage := fmt.Sprintf(`🎯 代码转换提示已生成

## 📚 API 映射表

为你准备了 %d 个 Lua API 到 Go WASM 的映射规则。

## 📝 代码生成模板

已生成针对插件 **%s** 的完整代码模板，包含：
- 包结构和导入
- 配置结构体定义
- 回调函数框架
- 原始 Lua 逻辑注释

## ✨ 最佳实践

提供了 %d 条最佳实践建议。

## 💡 示例代码片段

准备了 %d 个常用场景的示例代码。

---

**现在你可以**：
1. 基于这些提示开始编写 Go WASM 代码
2. 参考 API 映射表进行精确转换
3. 遵循最佳实践建议
4. 使用示例代码片段作为参考

⚠️ **警告**: %s
`,
		len(hints.APIMappings),
		pluginName,
		len(hints.BestPractices),
		len(hints.ExampleSnippets),
		formatWarningsListForUser(hints.Warnings),
	)

	// 生成详细的 AI 指令
	aiInstructions := fmt.Sprintf(`现在你需要基于以下信息生成 Go WASM 插件代码。

## API 映射表

以下是完整的 Lua API 到 Go WASM API 的映射：

%s

## 代码模板

%s

## 最佳实践

%s

## 示例代码片段

%s

## 生成代码的要求

1. **严格遵循模板结构**
2. **使用映射表中的 Go API**
3. **保持 Lua 代码的业务逻辑等价**
4. **添加详细注释**
5. **实现完整的错误处理**
6. **包含配置解析逻辑**

## 输出格式

请按以下格式输出代码：

### main.go
`+"```go"+`
[完整的 Go 代码]
`+"```"+`

生成代码后，建议调用 validate_wasm_code 工具进行验证。
`,
		formatAPIMappingsForAI(hints.APIMappings),
		hints.CodeTemplate,
		formatBestPracticesForAI(hints.BestPractices),
		formatExampleSnippetsForAI(hints.ExampleSnippets),
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

	// 格式化输出
	statusEmoji := "✅"
	statusText := "通过"
	if !report.IsValid {
		statusEmoji = "❌"
		statusText = "未通过"
	}

	userMessage := fmt.Sprintf(`%s 代码验证结果：%s

## 📊 验证评分：%d/100

### 错误 (%d 个)
%s

### 警告 (%d 个)
%s

### 改进建议 (%d 个)
%s

### 缺失的导入包 (%d 个)
%s

---

`,
		statusEmoji,
		statusText,
		report.Score,
		len(report.Errors),
		formatValidationErrors(report.Errors),
		len(report.Warnings),
		formatList(report.Warnings),
		len(report.Suggestions),
		formatList(report.Suggestions),
		len(report.MissingImports),
		formatList(report.MissingImports),
	)

	if report.IsValid {
		userMessage += "🎉 **代码验证通过！**\n\n"
		userMessage += "**下一步**：调用 `generate_deployment_config` 工具生成部署配置。"
	} else {
		userMessage += "⚠️ **请修复上述错误后重新验证。**"
	}

	// AI 指令
	aiInstructions := ""
	if !report.IsValid {
		aiInstructions = `代码验证发现错误，需要修复。

## 修复建议

基于验证报告中的错误和建议，修改代码：

` + formatValidationErrorsForAI(report.Errors) + `

修复后，再次调用 validate_wasm_code 工具进行验证。
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

这将生成完整的部署配置包，包括：
- WasmPlugin YAML
- Makefile
- Dockerfile
- README
- 测试脚本
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

## 📦 生成的文件

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

## 🚀 快速部署

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

func formatAPIMappingsForAI(mappings map[string]tools.APIMappingDetail) string {
	result := []string{}
	for _, mapping := range mappings {
		result = append(result, fmt.Sprintf(`
### %s

**Lua**:
`+"```lua"+`
%s
`+"```"+`

**Go WASM**:
`+"```go"+`
%s
`+"```"+`

**说明**: %s

**示例**:
`+"```go"+`
%s
`+"```"+`

%s
`,
			mapping.LuaAPI,
			mapping.LuaAPI,
			mapping.GoEquivalent,
			mapping.Description,
			mapping.ExampleCode,
			func() string {
				if mapping.Notes != "" {
					return "**注意**: " + mapping.Notes
				}
				return ""
			}(),
		))
	}
	return strings.Join(result, "\n---\n")
}

func formatBestPracticesForAI(practices []string) string {
	result := []string{}
	for i, p := range practices {
		result = append(result, fmt.Sprintf("%d. %s", i+1, p))
	}
	return strings.Join(result, "\n")
}

func formatExampleSnippetsForAI(snippets map[string]string) string {
	result := []string{}
	for name, code := range snippets {
		result = append(result, fmt.Sprintf(`
### %s
`+"```go"+`
%s
`+"```",
			name,
			code,
		))
	}
	return strings.Join(result, "\n")
}

func formatValidationErrors(errors []tools.ValidationError) string {
	if len(errors) == 0 {
		return "无"
	}
	result := []string{}
	for _, e := range errors {
		result = append(result, fmt.Sprintf("- [%s] %s\n  建议：%s", e.Severity, e.Message, e.Suggestion))
	}
	return strings.Join(result, "\n")
}

func formatValidationErrorsForAI(errors []tools.ValidationError) string {
	if len(errors) == 0 {
		return "无错误"
	}
	result := []string{}
	for i, e := range errors {
		result = append(result, fmt.Sprintf(`
### 错误 %d: %s

**类型**: %s
**严重程度**: %s
**建议**: %s

修复此问题的方法：
%s
`,
			i+1,
			e.Message,
			e.Type,
			e.Severity,
			e.Suggestion,
			e.Suggestion, // 可以扩展更详细的修复说明
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
