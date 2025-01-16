package provider

import (
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// baichuanProvider is the provider for baichuan Ai service.

const (
	baichuanDomain             = "api.baichuan-ai.com"
	baichuanChatCompletionPath = "/v1/chat/completions"
)

type baichuanProviderInitializer struct {
}

func (m *baichuanProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *baichuanProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &baichuanProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type baichuanProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *baichuanProvider) GetProviderType() string {
	return providerTypeBaichuan
}

func (m *baichuanProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return nil
}

func (m *baichuanProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *baichuanProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, baichuanChatCompletionPath)
	util.OverwriteRequestHostHeader(headers, baichuanDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}
