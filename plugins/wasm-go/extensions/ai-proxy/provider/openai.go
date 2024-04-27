package provider

import (
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// azureProvider is the provider for Azure OpenAI service.

const (
	openaiDomain             = "api.openai.com"
	openaiChatCompletionPath = "/v1/chat/completions"
)

type openaiProviderInitializer struct {
}

func (m *openaiProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (m *openaiProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &openaiProvider{
		config: config,
	}, nil
}

type openaiProvider struct {
	config ProviderConfig
}

func (m *openaiProvider) GetPointcuts() map[Pointcut]interface{} {
	return map[Pointcut]interface{}{PointcutOnRequestHeaders: nil}
}

func (m *openaiProvider) OnApiRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(openaiChatCompletionPath)
	_ = util.OverwriteRequestHost(openaiDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.apiToken)
	return types.ActionContinue, nil
}

func (m *openaiProvider) OnApiRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}

func (m *openaiProvider) OnApiResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}

func (m *openaiProvider) OnApiResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}
