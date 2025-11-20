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
