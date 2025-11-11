package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	// 默认摘要提示词模板（与memory.go中的定义保持一致）
	defaultSummaryPromptTemplate = `请为以下工具输出内容生成一个简洁的摘要，保留关键信息和重要细节。摘要应该：
1. 突出主要内容要点
2. 保留关键数据和结果
3. 使用简洁明了的语言
4. 长度控制在500字符以内

工具输出内容：
%s

摘要：`
)

// llmSummaryGenerator LLM摘要生成器实现
type llmSummaryGenerator struct {
	providerConfig *ProviderConfig
	config         *SummaryConfig
}

// NewLLMSummaryGenerator 创建LLM摘要生成器
func NewLLMSummaryGenerator(providerConfig *ProviderConfig, config *SummaryConfig) SummaryGenerator {
	if config == nil {
		config = &SummaryConfig{
			Method:         "simple",
			MaxLength:      DefaultSummaryMaxLength,
			PromptTemplate: defaultSummaryPromptTemplate,
		}
	}

	// 设置默认值
	if config.MaxLength == 0 {
		config.MaxLength = DefaultSummaryMaxLength
	}
	if config.PromptTemplate == "" {
		config.PromptTemplate = defaultSummaryPromptTemplate
	}
	if config.LLMModel == "" {
		// 如果没有指定模型，尝试从provider配置中获取
		// 这里可以根据实际情况调整
		config.LLMModel = "gpt-3.5-turbo" // 默认模型
	}

	return &llmSummaryGenerator{
		providerConfig: providerConfig,
		config:         config,
	}
}

// GenerateSummary 使用LLM生成智能摘要
func (g *llmSummaryGenerator) GenerateSummary(ctx wrapper.HttpContext, content string) (string, error) {
	if len(content) == 0 {
		return "", nil
	}

	// 如果内容较短，直接返回（不需要LLM摘要）
	if len(content) <= g.config.MaxLength {
		return content, nil
	}

	// 构建提示词
	prompt := fmt.Sprintf(g.config.PromptTemplate, content)

	// 构建LLM请求
	request := &chatCompletionRequest{
		Model: g.config.LLMModel,
		Messages: []chatMessage{
			{
				Role:    roleUser,
				Content: prompt,
			},
		},
		MaxTokens:   200, // 限制摘要长度
		Temperature: 0.3, // 较低温度，确保摘要稳定
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary request: %v", err)
	}

	log.Debugf("[LLMSummaryGenerator] calling LLM to generate summary, content length: %d", len(content))

	// 调用LLM API生成摘要
	// 注意：这里需要调用实际的LLM provider
	// 由于在WASM环境中，我们需要通过HTTP请求调用
	summary, err := g.callLLMForSummary(ctx, requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to call LLM for summary: %v", err)
	}

	// 确保摘要不超过最大长度
	if len(summary) > g.config.MaxLength {
		summary = summary[:g.config.MaxLength] + "..."
	}

	log.Infof("[LLMSummaryGenerator] generated summary, original length: %d, summary length: %d", len(content), len(summary))
	return summary, nil
}

// callLLMForSummary 调用LLM API生成摘要
// 这是一个简化实现，实际应该通过provider调用LLM服务
func (g *llmSummaryGenerator) callLLMForSummary(ctx wrapper.HttpContext, requestBody []byte) (string, error) {
	// 注意：在WASM环境中，直接调用LLM API比较复杂
	// 这里提供一个框架，实际实现需要：
	// 1. 通过HTTP客户端调用LLM API
	// 2. 或者通过provider的内部方法调用

	// 方案1：如果provider支持内部调用，可以直接使用
	// 方案2：通过HTTP请求调用LLM服务（需要HTTP客户端支持）

	// 这里先返回一个占位实现，实际使用时需要根据环境实现
	log.Warnf("[LLMSummaryGenerator] LLM summary generation not fully implemented, using fallback")

	// 降级：使用简单摘要作为占位
	// 实际实现时，应该调用真实的LLM API
	return generateSummary(string(requestBody)), fmt.Errorf("LLM summary generation not implemented yet")
}

// simpleSummaryGenerator 简单摘要生成器实现（默认）
type simpleSummaryGenerator struct{}

// NewSimpleSummaryGenerator 创建简单摘要生成器
func NewSimpleSummaryGenerator() SummaryGenerator {
	return &simpleSummaryGenerator{}
}

// GenerateSummary 使用简单方法生成摘要
func (g *simpleSummaryGenerator) GenerateSummary(ctx wrapper.HttpContext, content string) (string, error) {
	return generateSummary(content), nil
}

// asyncLLMSummaryGenerator 异步LLM摘要生成器
// 用于在后台异步生成摘要，不阻塞主流程
type asyncLLMSummaryGenerator struct {
	llmGenerator *llmSummaryGenerator
}

// NewAsyncLLMSummaryGenerator 创建异步LLM摘要生成器
func NewAsyncLLMSummaryGenerator(providerConfig *ProviderConfig, config *SummaryConfig) SummaryGenerator {
	return &asyncLLMSummaryGenerator{
		llmGenerator: NewLLMSummaryGenerator(providerConfig, config).(*llmSummaryGenerator),
	}
}

// GenerateSummary 异步生成摘要
// 先返回简单摘要，然后在后台生成LLM摘要并更新
func (g *asyncLLMSummaryGenerator) GenerateSummary(ctx wrapper.HttpContext, content string) (string, error) {
	// 先返回简单摘要，不阻塞
	simpleSummary := generateSummary(content)

	// 在后台异步生成LLM摘要
	go func() {
		// 注意：在WASM环境中，goroutine可能有限制
		// 这里提供一个框架，实际实现需要根据环境调整
		llmSummary, err := g.llmGenerator.GenerateSummary(ctx, content)
		if err != nil {
			log.Warnf("[AsyncLLMSummaryGenerator] failed to generate LLM summary asynchronously: %v", err)
			return
		}

		// 更新Redis中的摘要（需要contextId，这里需要从context中获取）
		// 实际实现时需要保存contextId以便后续更新
		log.Infof("[AsyncLLMSummaryGenerator] generated LLM summary asynchronously, length: %d", len(llmSummary))
	}()

	return simpleSummary, nil
}

// 辅助函数：从请求体中提取模型名称
func extractModelFromRequest(body []byte) string {
	var request chatCompletionRequest
	if err := json.Unmarshal(body, &request); err == nil && request.Model != "" {
		return request.Model
	}
	return ""
}

// 辅助函数：构建摘要请求
func buildSummaryRequest(content string, model string, maxTokens int) *chatCompletionRequest {
	if maxTokens == 0 {
		maxTokens = 200
	}

	prompt := fmt.Sprintf(defaultSummaryPromptTemplate, content)

	return &chatCompletionRequest{
		Model: model,
		Messages: []chatMessage{
			{
				Role:    roleUser,
				Content: prompt,
			},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.3,
	}
}

// 辅助函数：解析LLM响应获取摘要
func extractSummaryFromResponse(responseBody []byte) (string, error) {
	var response chatCompletionResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal LLM response: %v", err)
	}

	if len(response.Choices) == 0 || response.Choices[0].Message == nil {
		return "", fmt.Errorf("empty response from LLM")
	}

	content := response.Choices[0].Message.Content
	if contentStr, ok := content.(string); ok {
		return strings.TrimSpace(contentStr), nil
	}

	return "", fmt.Errorf("invalid content type in LLM response")
}
