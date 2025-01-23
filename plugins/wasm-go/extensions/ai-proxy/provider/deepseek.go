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
	deepseekDomain = "api.deepseek.com"
	// TODO: docs: https://api-docs.deepseek.com/api/create-chat-completion
	// accourding to the docs, the path should be /chat/completions, need to be verified
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

func (m *deepseekProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): deepseekChatCompletionPath,
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

func (m *deepseekProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if !m.config.isSupportedAPI(apiName) {
		return m.config.handleUnsupportedAPI()
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return nil
}

func (m *deepseekProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, m.config.handleUnsupportedAPI()
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *deepseekProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestHostHeader(headers, deepseekDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}
