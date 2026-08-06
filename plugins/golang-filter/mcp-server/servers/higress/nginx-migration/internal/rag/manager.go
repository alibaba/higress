// Package rag 提供 RAG（检索增强生成）功能
// 支持可选的知识库集成，通过配置开关控制
package rag

import (
	"fmt"
	"log"
	"strings"
)

// RAGManager 管理 RAG 功能的开关和查询
type RAGManager struct {
	enabled bool       // RAG 功能是否启用
	client  *RAGClient // RAG 客户端（仅在 enabled=true 时有效）
	config  *RAGConfig // 配置
}

// NewRAGManager 创建 RAG 管理器
// 如果配置中 enabled=false，则返回禁用状态的管理器
func NewRAGManager(config *RAGConfig) *RAGManager {
	if config == nil || !config.Enabled {
		log.Println("📖 RAG: Disabled (using rule-based generation)")
		return &RAGManager{
			enabled: false,
			config:  config,
		}
	}

	// 验证必要配置
	if config.KnowledgeBaseID == "" || config.WorkspaceID == "" {
		log.Println("⚠️  RAG: Missing workspace ID or knowledge base ID, disabling RAG")
		return &RAGManager{
			enabled: false,
			config:  config,
		}
	}

	// 检查 SDK 认证凭证
	if config.AccessKeyID == "" || config.AccessKeySecret == "" {
		log.Println("⚠️  RAG: Missing AccessKey credentials, disabling RAG")
		return &RAGManager{
			enabled: false,
			config:  config,
		}
	}

	// 初始化 RAG 客户端（使用 SDK）
	log.Println("🔧 RAG: Using Alibaba Cloud SDK authentication")
	client, err := NewRAGClient(config)
	if err != nil {
		log.Printf("❌ RAG: Failed to initialize SDK client: %v, disabling RAG\n", err)
		return &RAGManager{
			enabled: false,
			config:  config,
		}
	}

	log.Printf("✅ RAG: Enabled (Provider: %s, KB: %s)\n", config.Provider, config.KnowledgeBaseID)

	return &RAGManager{
		enabled: true,
		client:  client,
		config:  config,
	}
}

// IsEnabled 返回 RAG 是否启用
func (m *RAGManager) IsEnabled() bool {
	return m.enabled
}

// QueryWithContext 查询知识库并返回上下文
// 如果 RAG 未启用，返回空上下文（不报错）
func (m *RAGManager) QueryWithContext(query string, scenario string, opts ...QueryOption) (*RAGContext, error) {
	// RAG 未启用，返回空上下文
	if !m.enabled {
		return &RAGContext{
			Enabled: false,
			Message: "RAG is disabled, using rule-based generation",
		}, nil
	}

	// 构建查询
	ragQuery := &RAGQuery{
		Query:    query,
		Scenario: scenario,
		TopK:     m.config.DefaultTopK,
	}

	// 应用可选参数
	for _, opt := range opts {
		opt(ragQuery)
	}

	// 查询知识库
	resp, err := m.client.SearchWithCache(ragQuery)
	if err != nil {
		// 如果配置了降级策略，返回空上下文而不是报错
		if m.config.FallbackOnError {
			log.Printf("⚠️  RAG query failed, falling back to rules: %v\n", err)
			return &RAGContext{
				Enabled: false,
				Message: fmt.Sprintf("RAG query failed, using fallback: %v", err),
			}, nil
		}
		return nil, fmt.Errorf("RAG query failed: %w", err)
	}

	// 构建上下文
	return m.buildContext(resp), nil
}

// QueryForTool 为特定工具查询（支持工具级配置覆盖）
// 这是工具级别配置的核心实现
func (m *RAGManager) QueryForTool(toolName string, query string, scenario string) (*RAGContext, error) {
	// 全局 RAG 未启用
	if !m.enabled {
		return &RAGContext{
			Enabled: false,
			Message: "RAG is disabled globally",
		}, nil
	}

	// 检查工具级配置
	if toolConfig, ok := m.config.Tools[toolName]; ok {
		// 工具有专门的配置
		if !toolConfig.UseRAG {
			// 工具明确不使用 RAG
			return &RAGContext{
				Enabled: false,
				Message: fmt.Sprintf("RAG is disabled for tool: %s", toolName),
			}, nil
		}

		// 使用工具级配置覆盖全局配置
		log.Printf("🔧 Using tool-specific RAG config for: %s (context_mode=%s, top_k=%d)",
			toolName, toolConfig.ContextMode, toolConfig.TopK)

		return m.QueryWithContext(query, scenario,
			WithTopK(toolConfig.TopK),
			WithContextMode(toolConfig.ContextMode),
		)
	}

	// 没有工具级配置，使用默认全局配置
	log.Printf("🔧 Using global RAG config for: %s", toolName)
	return m.QueryWithContext(query, scenario)
}

// buildContext 构建上下文
func (m *RAGManager) buildContext(resp *RAGResponse) *RAGContext {
	ctx := &RAGContext{
		Enabled:   true,
		Documents: make([]ContextDocument, 0, len(resp.Documents)),
	}

	for _, doc := range resp.Documents {
		ctxDoc := ContextDocument{
			Title:      doc.Title,
			Source:     doc.Source,
			URL:        doc.URL,
			Score:      doc.Score,
			Highlights: doc.Highlights,
		}

		// 根据 context_mode 决定返回的内容
		switch m.config.ContextMode {
		case "full":
			ctxDoc.Content = doc.Content
		case "summary":
			ctxDoc.Content = m.summarize(doc.Content)
		case "highlights":
			if len(doc.Highlights) > 0 {
				ctxDoc.Content = strings.Join(doc.Highlights, "\n\n")
			} else {
				ctxDoc.Content = m.summarize(doc.Content)
			}
		default:
			ctxDoc.Content = doc.Content
		}

		// 控制长度
		if len(ctxDoc.Content) > m.config.MaxContextLength {
			ctxDoc.Content = ctxDoc.Content[:m.config.MaxContextLength] + "\n\n[内容已截断...]"
		}

		ctx.Documents = append(ctx.Documents, ctxDoc)
	}

	ctx.Message = fmt.Sprintf("Retrieved %d relevant documents from knowledge base (latency: %dms)",
		len(ctx.Documents), resp.Latency)

	return ctx
}

// summarize 简单的内容摘要（截取前N个字符）
func (m *RAGManager) summarize(content string, maxLen ...int) string {
	length := 500 // 默认500字符
	if len(maxLen) > 0 {
		length = maxLen[0]
	}

	if len(content) <= length {
		return content
	}

	// 尝试在句号或换行处截断
	truncated := content[:length]
	if idx := strings.LastIndexAny(truncated, "。\n."); idx > length/2 {
		return content[:idx+1]
	}

	return truncated + "..."
}

// FormatContextForAI 格式化上下文，供 AI 使用
// 返回 Markdown 格式的文档上下文
func (ctx *RAGContext) FormatContextForAI() string {
	if !ctx.Enabled || len(ctx.Documents) == 0 {
		return fmt.Sprintf("> ℹ️  %s\n", ctx.Message)
	}

	var result strings.Builder

	result.WriteString("## 📚 知识库参考文档\n\n")
	result.WriteString(fmt.Sprintf("> %s\n\n", ctx.Message))

	for i, doc := range ctx.Documents {
		result.WriteString(fmt.Sprintf("### 参考文档 %d: %s\n\n", i+1, doc.Title))

		// 元信息
		result.WriteString(fmt.Sprintf("**来源**: %s  \n", doc.Source))
		if doc.URL != "" {
			result.WriteString(fmt.Sprintf("**链接**: %s  \n", doc.URL))
		}
		result.WriteString(fmt.Sprintf("**相关度**: %.2f  \n\n", doc.Score))

		// 文档内容（重点）
		result.WriteString("**相关内容**:\n\n")
		result.WriteString("```\n")
		result.WriteString(doc.Content)
		result.WriteString("\n```\n\n")

		// 高亮片段
		if len(doc.Highlights) > 0 {
			result.WriteString("**关键片段**:\n\n")
			for _, h := range doc.Highlights {
				result.WriteString(fmt.Sprintf("- %s\n", h))
			}
			result.WriteString("\n")
		}

		result.WriteString("---\n\n")
	}

	return result.String()
}

// ==================== 类型定义 ====================

// RAGContext 表示 RAG 查询返回的上下文
type RAGContext struct {
	Enabled   bool              `json:"enabled"`   // RAG 是否启用
	Documents []ContextDocument `json:"documents"` // 检索到的文档
	Message   string            `json:"message"`   // 提示信息
}

// ContextDocument 表示上下文中的一个文档
type ContextDocument struct {
	Title      string   `json:"title"`      // 文档标题
	Content    string   `json:"content"`    // 文档内容（根据 context_mode 调整）
	Source     string   `json:"source"`     // 来源路径
	URL        string   `json:"url"`        // 在线链接
	Score      float64  `json:"score"`      // 相关度分数
	Highlights []string `json:"highlights"` // 高亮片段
}

// ==================== 查询选项 ====================

// QueryOption 查询选项函数
type QueryOption func(*RAGQuery)

// WithTopK 设置返回文档数量
func WithTopK(k int) QueryOption {
	return func(q *RAGQuery) {
		q.TopK = k
	}
}

// WithContextMode 设置上下文模式
func WithContextMode(mode string) QueryOption {
	return func(q *RAGQuery) {
		q.ContextMode = mode
	}
}

// WithFilters 设置过滤条件
func WithFilters(filters map[string]interface{}) QueryOption {
	return func(q *RAGQuery) {
		q.Filters = filters
	}
}
