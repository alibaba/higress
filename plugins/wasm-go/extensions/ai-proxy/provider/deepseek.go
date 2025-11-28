package provider

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// deepseekProvider is the provider for deepseek Ai service.

const (
	deepseekDomain                = "api.deepseek.com"
	deepseekAnthropicMessagesPath = "/anthropic/v1/messages"
)

type deepseekProviderInitializer struct{}

func (m *deepseekProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *deepseekProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion):    PathOpenAIChatCompletions,
		string(ApiNameModels):            PathOpenAIModels,
		string(ApiNameAnthropicMessages): deepseekAnthropicMessagesPath,
	}
}

func (m *deepseekProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(m.DefaultCapabilities())
	return &deepseekProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type deepseekProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *deepseekProvider) GetProviderType() string {
	return providerTypeDeepSeek
}

func (m *deepseekProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	return nil
}

func (m *deepseekProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *deepseekProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestHostHeader(headers, deepseekDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

// TransformResponseBody 处理DeepSeek API响应，确保token计算符合DeepSeek标准
// DeepSeek使用OpenAI兼容的API格式，但需要特别处理reasoning tokens
func (m *deepseekProvider) TransformResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	if apiName != ApiNameChatCompletion {
		return body, nil
	}

	// 解析响应
	var response chatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Warnf("[DeepSeek] failed to unmarshal response: %v", err)
		return body, nil
	}

	// 处理usage信息，确保token计算正确
	if response.Usage != nil {
		// DeepSeek的token计算标准：
		// 1. 实际token数量以API返回为准
		// 2. 如果包含reasoning tokens，需要正确计算TotalTokens
		// 3. TotalTokens = PromptTokens + CompletionTokens（包括reasoning tokens）

		// 如果TotalTokens未设置或为0，重新计算
		if response.Usage.TotalTokens == 0 {
			response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
		}

		// 处理reasoning tokens（如果存在）
		if response.Usage.CompletionTokensDetails != nil && response.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
			// reasoning tokens已经包含在CompletionTokens中
			// 但需要确保TotalTokens计算正确
			log.Debugf("[DeepSeek] detected reasoning tokens: %d, completion tokens: %d, total tokens: %d",
				response.Usage.CompletionTokensDetails.ReasoningTokens,
				response.Usage.CompletionTokens,
				response.Usage.TotalTokens)
		}

		// 验证token计算
		calculatedTotal := response.Usage.PromptTokens + response.Usage.CompletionTokens
		if response.Usage.TotalTokens != calculatedTotal {
			log.Debugf("[DeepSeek] token count mismatch: API reported %d, calculated %d, using API value",
				response.Usage.TotalTokens, calculatedTotal)
			// 使用API返回的值（以API为准）
		}

		log.Debugf("[DeepSeek] token usage - prompt: %d, completion: %d, total: %d",
			response.Usage.PromptTokens,
			response.Usage.CompletionTokens,
			response.Usage.TotalTokens)
	}

	// 重新序列化响应
	modifiedBody, err := json.Marshal(response)
	if err != nil {
		log.Warnf("[DeepSeek] failed to marshal modified response: %v", err)
		return body, nil
	}

	return modifiedBody, nil
}
