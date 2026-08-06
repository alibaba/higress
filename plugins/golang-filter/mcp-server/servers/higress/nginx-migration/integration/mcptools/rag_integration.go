// Package mcptools 提供 RAG 集成到 MCP 工具的示例实现
package mcptools

import (
	"fmt"
	"log"
	"strings"

	"nginx-migration-mcp/internal/rag"
)

// RAGToolContext MCP 工具的 RAG 上下文
type RAGToolContext struct {
	Manager *rag.RAGManager
}

// NewRAGToolContext 创建 RAG 工具上下文
func NewRAGToolContext(configPath string) (*RAGToolContext, error) {
	// 加载配置
	config, err := rag.LoadRAGConfig(configPath)
	if err != nil {
		log.Printf("⚠️  Failed to load RAG config: %v, RAG will be disabled", err)
		// 创建禁用状态的配置
		config = &rag.RAGConfig{Enabled: false}
	}

	// 创建 RAG 管理器
	manager := rag.NewRAGManager(config)

	return &RAGToolContext{
		Manager: manager,
	}, nil
}

// ==================== 工具示例：generate_conversion_hints ====================

// GenerateConversionHintsWithRAG 生成转换提示（带 RAG 增强）
func (ctx *RAGToolContext) GenerateConversionHintsWithRAG(analysisResult string, pluginName string) (string, error) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("# %s 插件转换指南\n\n", pluginName))

	// 提取 Nginx APIs（这里简化处理）
	nginxAPIs := extractNginxAPIs(analysisResult)

	// === 核心：使用工具级别的 RAG 查询 ===
	toolName := "generate_conversion_hints"
	ragContext, err := ctx.Manager.QueryForTool(
		toolName,
		fmt.Sprintf("Nginx Lua API %s 在 Higress WASM 中的实现和转换方法", strings.Join(nginxAPIs, ", ")),
		"lua_migration",
	)

	if err != nil {
		log.Printf("❌ RAG query failed for %s: %v", toolName, err)
		// 降级到规则生成
		ragContext = &rag.RAGContext{
			Enabled: false,
			Message: fmt.Sprintf("RAG query failed: %v", err),
		}
	}

	// 添加 RAG 上下文信息（如果有）
	if ragContext.Enabled && len(ragContext.Documents) > 0 {
		result.WriteString(ragContext.FormatContextForAI())
	} else {
		// RAG 未启用或查询失败
		result.WriteString(fmt.Sprintf("> ℹ️  %s\n\n", ragContext.Message))
		result.WriteString("> 使用基于规则的转换指南\n\n")
	}

	// 为每个 API 生成转换提示（基于规则）
	result.WriteString("## 🔄 API 转换详情\n\n")
	for _, api := range nginxAPIs {
		result.WriteString(fmt.Sprintf("### %s\n\n", api))
		result.WriteString(generateBasicMapping(api))
		result.WriteString("\n")
	}

	// 添加使用建议
	result.WriteString("\n---\n\n")
	result.WriteString("## 💡 使用建议\n\n")
	if ragContext.Enabled {
		result.WriteString("✅ 上述参考文档来自 Higress 官方知识库，请参考这些文档中的示例代码和最佳实践来生成 WASM 插件代码。\n\n")
		result.WriteString("建议按照知识库中的示例实现，确保代码符合 Higress 的最佳实践。\n")
	} else {
		result.WriteString("ℹ️  当前未启用 RAG 知识库或查询失败，使用基于规则的映射。\n\n")
		result.WriteString("建议参考 Higress 官方文档：https://higress.cn/docs/plugins/wasm-go-sdk/\n")
	}

	return result.String(), nil
}

// ==================== 工具示例：validate_wasm_code ====================

// ValidateWasmCodeWithRAG 验证 WASM 代码（带 RAG 增强）
func (ctx *RAGToolContext) ValidateWasmCodeWithRAG(goCode string, pluginName string) (string, error) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("## 🔍 %s 插件代码验证报告\n\n", pluginName))

	// 基本验证（始终执行）
	basicIssues := validateBasicSyntax(goCode)
	apiIssues := validateAPIUsage(goCode)

	if len(basicIssues) > 0 {
		result.WriteString("### ⚠️  语法问题\n\n")
		for _, issue := range basicIssues {
			result.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		result.WriteString("\n")
	}

	if len(apiIssues) > 0 {
		result.WriteString("### ⚠️  API 使用问题\n\n")
		for _, issue := range apiIssues {
			result.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		result.WriteString("\n")
	}

	// === RAG 增强：查询最佳实践 ===
	toolName := "validate_wasm_code"
	ragContext, err := ctx.Manager.QueryForTool(
		toolName,
		"Higress WASM 插件开发最佳实践 错误处理 性能优化 代码规范",
		"best_practice",
	)

	if err != nil {
		log.Printf("❌ RAG query failed for %s: %v", toolName, err)
		ragContext = &rag.RAGContext{
			Enabled: false,
			Message: fmt.Sprintf("RAG query failed: %v", err),
		}
	}

	// 添加最佳实践建议
	if ragContext.Enabled && len(ragContext.Documents) > 0 {
		result.WriteString("### 💡 最佳实践建议（基于知识库）\n\n")

		for i, doc := range ragContext.Documents {
			result.WriteString(fmt.Sprintf("#### 建议 %d：%s\n\n", i+1, doc.Title))
			result.WriteString(fmt.Sprintf("**来源**: %s  \n", doc.Source))
			if doc.URL != "" {
				result.WriteString(fmt.Sprintf("**链接**: %s  \n", doc.URL))
			}
			result.WriteString("\n")

			// 只展示关键片段（validate 工具通常配置为 highlights 模式）
			if len(doc.Highlights) > 0 {
				result.WriteString("**关键要点**:\n\n")
				for _, h := range doc.Highlights {
					result.WriteString(fmt.Sprintf("- %s\n", h))
				}
			} else {
				result.WriteString("**参考内容**:\n\n")
				result.WriteString("```\n")
				result.WriteString(doc.Content)
				result.WriteString("\n```\n")
			}
			result.WriteString("\n")
		}

		// 基于知识库内容检查当前代码
		suggestions := checkCodeAgainstBestPractices(goCode, ragContext.Documents)
		if len(suggestions) > 0 {
			result.WriteString("### 📝 针对当前代码的改进建议\n\n")
			for _, s := range suggestions {
				result.WriteString(fmt.Sprintf("- %s\n", s))
			}
			result.WriteString("\n")
		}
	} else {
		result.WriteString("### 💡 基本建议\n\n")
		result.WriteString(fmt.Sprintf("> %s\n\n", ragContext.Message))
		result.WriteString(generateBasicValidationSuggestions(goCode))
	}

	// 验证总结
	if len(basicIssues) == 0 && len(apiIssues) == 0 {
		result.WriteString("\n---\n\n")
		result.WriteString("### ✅ 验证通过\n\n")
		result.WriteString("代码基本验证通过，没有发现明显问题。\n")
	}

	return result.String(), nil
}

// ==================== 工具示例：convert_lua_to_wasm ====================

// ConvertLuaToWasmWithRAG 快速转换（通常不使用 RAG）
func (ctx *RAGToolContext) ConvertLuaToWasmWithRAG(luaCode string, pluginName string) (string, error) {
	// 这个工具在配置中通常设置为 use_rag: false，保持快速响应

	toolName := "convert_lua_to_wasm"

	// 仍然可以查询，但如果配置禁用则会快速返回
	ragContext, _ := ctx.Manager.QueryForTool(
		toolName,
		"Lua to WASM conversion examples",
		"quick_convert",
	)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("# %s 插件转换结果\n\n", pluginName))

	if ragContext.Enabled {
		result.WriteString("> 🚀 使用 RAG 增强转换\n\n")
		result.WriteString(ragContext.FormatContextForAI())
	} else {
		result.WriteString("> ⚡ 快速转换模式（未启用 RAG）\n\n")
	}

	// 执行基于规则的转换
	wasmCode := performRuleBasedConversion(luaCode, pluginName)
	result.WriteString("## 生成的 Go WASM 代码\n\n")
	result.WriteString("```go\n")
	result.WriteString(wasmCode)
	result.WriteString("\n```\n")

	return result.String(), nil
}

// ==================== 辅助函数 ====================

func extractNginxAPIs(analysisResult string) []string {
	// 简化实现：从分析结果中提取 API
	apis := []string{"ngx.req.get_headers", "ngx.say", "ngx.var"}
	return apis
}

func generateBasicMapping(api string) string {
	mappings := map[string]string{
		"ngx.req.get_headers": "**Higress WASM**: `proxywasm.GetHttpRequestHeaders()`\n\n示例：\n```go\nheaders, err := proxywasm.GetHttpRequestHeaders()\nif err != nil {\n    proxywasm.LogError(\"failed to get headers\")\n    return types.ActionContinue\n}\n```",
		"ngx.say":             "**Higress WASM**: `proxywasm.SendHttpResponse()`",
		"ngx.var":             "**Higress WASM**: `proxywasm.GetProperty()`",
	}

	if mapping, ok := mappings[api]; ok {
		return mapping
	}
	return "映射信息暂未提供，请参考官方文档。"
}

func validateBasicSyntax(goCode string) []string {
	// 简化实现
	issues := []string{}
	if !strings.Contains(goCode, "package main") {
		issues = append(issues, "缺少 package main 声明")
	}
	return issues
}

func validateAPIUsage(goCode string) []string {
	// 简化实现
	issues := []string{}
	if strings.Contains(goCode, "proxywasm.") && !strings.Contains(goCode, "import") {
		issues = append(issues, "使用了 proxywasm API 但未导入相关包")
	}
	return issues
}

func checkCodeAgainstBestPractices(goCode string, docs []rag.ContextDocument) []string {
	// 简化实现：基于文档内容检查代码
	suggestions := []string{}

	// 检查错误处理
	if !strings.Contains(goCode, "if err != nil") {
		for _, doc := range docs {
			if strings.Contains(doc.Content, "错误处理") || strings.Contains(doc.Content, "error handling") {
				suggestions = append(suggestions, "建议添加完善的错误处理逻辑（参考知识库文档）")
				break
			}
		}
	}

	// 检查日志记录
	if !strings.Contains(goCode, "proxywasm.Log") {
		suggestions = append(suggestions, "建议添加适当的日志记录以便调试")
	}

	return suggestions
}

func generateBasicValidationSuggestions(goCode string) string {
	return "- 确保所有 API 调用都有错误处理\n" +
		"- 添加必要的日志记录\n" +
		"- 遵循 Higress WASM 插件开发规范\n"
}

func performRuleBasedConversion(luaCode string, pluginName string) string {
	// 简化实现：基于规则的转换
	return fmt.Sprintf(`package main

import (
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func main() {
	wrapper.SetCtx(
		"%s",
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	// TODO: 实现转换逻辑
	// 原始 Lua 代码：
	// %s
	
	return types.ActionContinue
}

type PluginConfig struct {
	// TODO: 添加配置字段
}
`, pluginName, luaCode)
}
