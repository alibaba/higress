package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	// 默认摘要提示词模板（与memory.go中的定义保持一致）
	// 改进版：包含通用的工具调用上下文模板，支持LLM与工具交互后动态传值
	defaultSummaryPromptTemplate = `你需要为以下工具调用的输出内容生成一个简洁的摘要。

**重要说明**：这些工具的输出内容可能会被压缩存储，当LLM在后续对话中发现某次工具调用的输出被压缩时，
可以根据本摘要中的信息快速定位和理解该工具调用，必要时可以重新执行该工具调用。

请按照以下格式为每个工具调用生成摘要：

### 工具调用信息
**工具名称**：{tool_name}
**调用ID**：{tool_call_id}
**调用参数**：
` + "`" + `json
{tool_arguments}
` + "`" + `

### 工具输出摘要
需要总结以下工具的输出内容，按照以下要求：
1. 清晰地列出工具的标识和调用时使用的参数
2. 突出主要内容要点、保留关键数据和结果
3. 使用简洁明了的语言，整体长度控制在500字符以内
4. 如果工具输出包含API返回码、错误信息，必须包含在摘要中
5. 如果工具输出包含重要的ID、路径、URL等信息，必须保留

工具输出内容：
` + "`" + `
%s
` + "`" + `

请生成摘要：`
)

// buildToolContextPrompt 构建包含工具上下文的完整提示词
// 该函数支持LLM与工具交互后，动态传入工具信息和输出内容
func buildToolContextPrompt(baseTemplate string, toolName string, toolCallId string, toolArgs string, toolOutput string) string {
	// Step 1: 替换工具信息占位符
	prompt := strings.ReplaceAll(baseTemplate, "{tool_name}", toolName)
	prompt = strings.ReplaceAll(prompt, "{tool_call_id}", toolCallId)
	prompt = strings.ReplaceAll(prompt, "{tool_arguments}", toolArgs)

	// Step 2: 将工具输出作为最终的格式化参数
	// 使用 %s 占位符，由调用方决定格式
	prompt = fmt.Sprintf(prompt, toolOutput)

	return prompt
}

// buildToolCallChainPrompt 构建基于工具调用链的提示词
// 支持包含前序工具的上下文信息，帮助LLM理解工具调用之间的关联性
func buildToolCallChainPrompt(baseTemplate string, currentTool ToolCallContextInfo, precedingTools []ToolCallContextInfo) string {
	// Step 1: 构建上下文链信息
	var chainBuilder strings.Builder

	// 添加前序工具信息
	if len(precedingTools) > 0 {
		chainBuilder.WriteString("\n### 前序工具调用\n")
		for i, tool := range precedingTools {
			chainBuilder.WriteString(fmt.Sprintf("\n**第%d个工具: %s** (ID: %s)\n", i+1, tool.ToolName, tool.ToolCallId))
			if tool.ToolArgs != "" {
				chainBuilder.WriteString(fmt.Sprintf("- 参数: %s\n", tool.ToolArgs))
			}
			// 提供结果摘要（不是完整内容）
			if len(tool.ToolOutput) > 100 {
				chainBuilder.WriteString(fmt.Sprintf("- 结果: %s...\n", tool.ToolOutput[:100]))
			} else {
				chainBuilder.WriteString(fmt.Sprintf("- 结果: %s\n", tool.ToolOutput))
			}
		}
		chainBuilder.WriteString(fmt.Sprintf("\n当前工具依赖于以上的处理结果。"))
	}

	// Step 2: 替换工具信息占位符
	prompt := strings.ReplaceAll(baseTemplate, "{tool_name}", currentTool.ToolName)
	prompt = strings.ReplaceAll(prompt, "{tool_call_id}", currentTool.ToolCallId)
	prompt = strings.ReplaceAll(prompt, "{tool_arguments}", currentTool.ToolArgs)

	// Step 3: 插入上下文链信息到%s占位符之前
	chainInfo := chainBuilder.String()
	if chainInfo != "" {
		// 将链信息插入到%s占位符之前
		prompt = strings.Replace(prompt, "%s", chainInfo+"\n\n### 当前工具输出\n%s", 1)
	}

	// Step 4: 最终格式化工具输出
	prompt = fmt.Sprintf(prompt, currentTool.ToolOutput)

	return prompt
}

// llmSummaryGenerator LLM摘要生成器实现
type llmSummaryGenerator struct {
	providerConfig *ProviderConfig
	config         *SummaryConfig
	httpClient     wrapper.HttpClient // HTTP客户端，用于调用LLM API
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
	if config.LLMTimeout == 0 {
		config.LLMTimeout = 5000 // 默认5秒超时
	}

	// 创建HTTP客户端
	httpClient := createLLMHttpClient(config)

	return &llmSummaryGenerator{
		providerConfig: providerConfig,
		config:         config,
		httpClient:     httpClient,
	}
}

// createLLMHttpClient 创建用于调用LLM API的HTTP客户端
func createLLMHttpClient(config *SummaryConfig) wrapper.HttpClient {
	// 如果配置了Cluster名称，使用RouteCluster
	if config.LLMServiceCluster != "" {
		return wrapper.NewClusterClient(wrapper.RouteCluster{
			Host: config.LLMServiceCluster,
		})
	}

	// 如果配置了URL，从URL解析host和port
	if config.LLMServiceUrl != "" {
		parsedURL, err := url.Parse(config.LLMServiceUrl)
		if err == nil && parsedURL.Host != "" {
			// 解析host和port
			host := parsedURL.Hostname()
			port := int64(443) // 默认HTTPS端口

			// 解析端口号
			if parsedURL.Port() != "" {
				if p, err := strconv.ParseInt(parsedURL.Port(), 10, 64); err == nil {
					port = p
				}
			} else {
				// 根据scheme设置默认端口
				if parsedURL.Scheme == "http" {
					port = 80
				}
			}

			// 使用FQDN方式（推荐，支持服务发现）
			return wrapper.NewClusterClient(wrapper.FQDNCluster{
				FQDN: host,
				Port: port,
			})
		}
	}

	// 默认使用RouteCluster（通过Gateway路由）
	return wrapper.NewClusterClient(wrapper.RouteCluster{})
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

	// 构建 LLM 请求
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
// 使用HTTP客户端通过wrapper.HttpClient调用LLM服务
func (g *llmSummaryGenerator) callLLMForSummary(ctx wrapper.HttpContext, requestBody []byte) (string, error) {
	if g.httpClient == nil {
		return "", fmt.Errorf("HTTP client not initialized")
	}

	// 确定请求URL路径
	// 如果配置了完整URL，提取路径部分；否则使用默认路径
	requestPath := "/v1/chat/completions"
	if g.config.LLMServiceUrl != "" {
		parsedURL, err := url.Parse(g.config.LLMServiceUrl)
		if err == nil && parsedURL.Path != "" {
			requestPath = parsedURL.Path
			if parsedURL.RawQuery != "" {
				requestPath += "?" + parsedURL.RawQuery
			}
		}
	}

	// 构建请求头部
	headers := [][2]string{
		{"Content-Type", "application/json"},
	}

	// 添加认证头部（使用当前provider的API Token）
	apiToken := g.providerConfig.GetApiTokenInUse(ctx)
	if apiToken != "" {
		headers = append(headers, [2]string{"Authorization", "Bearer " + apiToken})
	}

	// 设置超时时间
	timeout := uint32(g.config.LLMTimeout)
	if timeout == 0 {
		timeout = 5000 // 默认5秒
	}

	log.Debugf("[callLLMForSummary] calling LLM API: %s, timeout: %dms", requestPath, timeout)

	// 使用channel等待异步响应
	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// 发起异步HTTP调用
	err := g.httpClient.Post(requestPath, headers, requestBody, func(statusCode int, respHeaders http.Header, respBody []byte) {
		if statusCode == http.StatusOK {
			// 解析响应获取摘要
			summary, parseErr := extractSummaryFromResponse(respBody)
			if parseErr != nil {
				errChan <- fmt.Errorf("failed to parse LLM response: %v", parseErr)
				return
			}
			resultChan <- summary
		} else {
			// LLM API返回错误
			errMsg := string(respBody)
			if len(errMsg) > 200 {
				errMsg = errMsg[:200] + "..."
			}
			errChan <- fmt.Errorf("LLM API returned status %d: %s", statusCode, errMsg)
		}
	}, timeout)

	if err != nil {
		return "", fmt.Errorf("failed to dispatch HTTP call to LLM: %v", err)
	}

	// 等待响应或超时
	select {
	case summary := <-resultChan:
		log.Infof("[callLLMForSummary] successfully generated LLM summary, length: %d", len(summary))
		return summary, nil
	case err := <-errChan:
		log.Errorf("[callLLMForSummary] LLM API error: %v", err)
		return "", err
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		log.Errorf("[callLLMForSummary] timeout waiting for LLM response after %dms", timeout)
		return "", fmt.Errorf("timeout waiting for LLM response")
	}
}

// GenerateSummaryWithToolContext 使用LLM生成带工具上下文的智能摘要
func (g *llmSummaryGenerator) GenerateSummaryWithToolContext(ctx wrapper.HttpContext, content string, toolName string, toolCallId string, toolArgs string) (string, error) {
	if len(content) == 0 {
		return "", nil
	}

	// 如果内容较短，直接返回
	if len(content) <= g.config.MaxLength {
		return content, nil
	}

	// 使用通用的定位符模板构建提示词
	// buildToolContextPrompt 会动态上下文工具信息，然后将工具输出作为最终的格式化参数
	prompt := buildToolContextPrompt(g.config.PromptTemplate, toolName, toolCallId, toolArgs, content)

	// 构建 LLM 请求
	request := &chatCompletionRequest{
		Model: g.config.LLMModel,
		Messages: []chatMessage{
			{
				Role:    roleUser,
				Content: prompt,
			},
		},
		MaxTokens:   250, // 比基本摘要大一点，以容纳入工具上下文信息
		Temperature: 0.3,
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary request: %v", err)
	}

	log.Debugf("[LLMSummaryGenerator] calling LLM to generate summary with tool context, tool: %s, content length: %d", toolName, len(content))

	// 调用LLM生成摘要
	return g.callLLMForSummary(ctx, requestBody)
}

// GenerateSummaryWithCallChain 生成基于完整工具调用链的摘要
// 包含前序工具调用的上下文，帮助LLM理解依赖关系
func (g *llmSummaryGenerator) GenerateSummaryWithCallChain(ctx wrapper.HttpContext, content string, currentTool ToolCallContextInfo, precedingTools []ToolCallContextInfo) (string, error) {
	if len(content) == 0 {
		return "", nil
	}

	// 如果内容较短，直接返回
	if len(content) <= g.config.MaxLength {
		return content, nil
	}

	// 使用工具调用链提示词构建函数
	// buildToolCallChainPrompt 会包含前序工具信息，然后将工具输出作为最终的格式化参数
	prompt := buildToolCallChainPrompt(g.config.PromptTemplate, currentTool, precedingTools)

	// 构建 LLM 请求
	request := &chatCompletionRequest{
		Model: g.config.LLMModel,
		Messages: []chatMessage{
			{
				Role:    roleUser,
				Content: prompt,
			},
		},
		MaxTokens:   300, // 比单工具摘要更大，以容纳完整的调用链信息
		Temperature: 0.3,
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary request with call chain: %v", err)
	}

	log.Debugf("[LLMSummaryGenerator] calling LLM to generate summary with call chain, current tool: %s, preceding tools: %d, content length: %d", currentTool.ToolName, len(precedingTools), len(content))

	// 调用LLM生成摘要
	return g.callLLMForSummary(ctx, requestBody)
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

// GenerateSummaryWithToolContext 使用简单方法生成摘要无法包含工具上下文，所以只返回海扣的简单摘要
func (g *simpleSummaryGenerator) GenerateSummaryWithToolContext(ctx wrapper.HttpContext, content string, toolName string, toolCallId string, toolArgs string) (string, error) {
	// 简单摘要器不支持工具上下文，直接用简单摘要求解
	return generateSummary(content), nil
}

// GenerateSummaryWithCallChain 使用简单方法不支持工具调用链上下文
func (g *simpleSummaryGenerator) GenerateSummaryWithCallChain(ctx wrapper.HttpContext, content string, currentTool ToolCallContextInfo, precedingTools []ToolCallContextInfo) (string, error) {
	// 简单摘要器不支持需要复杂上下文的摘要生成，直接用简单摘要
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

// GenerateSummaryWithToolContext 异步生成带工具上下文的摘要
// 先返回简单摘要，然后在后台生成LLM摘要并更新
func (g *asyncLLMSummaryGenerator) GenerateSummaryWithToolContext(ctx wrapper.HttpContext, content string, toolName string, toolCallId string, toolArgs string) (string, error) {
	// 先返回简单摘要，不阻塞
	simpleSummary := generateSummary(content)

	// 在后台异步生成LLM摘要
	go func() {
		llmSummary, err := g.llmGenerator.GenerateSummaryWithToolContext(ctx, content, toolName, toolCallId, toolArgs)
		if err != nil {
			log.Warnf("[AsyncLLMSummaryGenerator] failed to generate LLM summary with tool context asynchronously: %v", err)
			return
		}

		log.Infof("[AsyncLLMSummaryGenerator] generated LLM summary with tool context asynchronously, tool: %s, length: %d", toolName, len(llmSummary))
	}()

	return simpleSummary, nil
}

// GenerateSummaryWithCallChain 异步生成基于完整工具调用链的摘要
// 先返回简单摘要，然后在后台生成LLM摘要
func (g *asyncLLMSummaryGenerator) GenerateSummaryWithCallChain(ctx wrapper.HttpContext, content string, currentTool ToolCallContextInfo, precedingTools []ToolCallContextInfo) (string, error) {
	// 先返回简单摘要，不阻塞
	simpleSummary := generateSummary(content)

	// 在后台异步生成LLM摘要
	go func() {
		llmSummary, err := g.llmGenerator.GenerateSummaryWithCallChain(ctx, content, currentTool, precedingTools)
		if err != nil {
			log.Warnf("[AsyncLLMSummaryGenerator] failed to generate LLM summary with call chain asynchronously: %v", err)
			return
		}

		log.Infof("[AsyncLLMSummaryGenerator] generated LLM summary with call chain asynchronously, current tool: %s, preceding tools: %d, length: %d", currentTool.ToolName, len(precedingTools), len(llmSummary))
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
