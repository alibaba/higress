package provider

import (
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// deepseekProvider is the provider for deepseek Ai service.

const (
	deepseekDomain             = "api.deepseek.com"
	deepseekChatCompletionPath = "/v1/chat/completions"
)

type deepseekProviderInitializer struct {
}

func (m *deepseekProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *deepseekProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
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

func (m *deepseekProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return nil
}

func (m *deepseekProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *deepseekProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, deepseekChatCompletionPath)
	util.OverwriteRequestHostHeader(headers, deepseekDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}
