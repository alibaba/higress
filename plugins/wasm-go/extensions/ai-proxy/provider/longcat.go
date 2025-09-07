package provider

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// longcatProvider is the provider for LongCat AI service.

const (
	longcatDomain = "api.longcat.chat"
)

type longcatProviderInitializer struct{}

func (m *longcatProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *longcatProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): PathOpenAIChatCompletions,
		string(ApiNameEmbeddings):     PathOpenAIEmbeddings,
		string(ApiNameModels):         PathOpenAIModels,
	}
}

func (m *longcatProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(m.DefaultCapabilities())
	return &longcatProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type longcatProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *longcatProvider) GetProviderType() string {
	return providerTypeLongcat
}

func (m *longcatProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	return nil
}

func (m *longcatProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestHostHeader(headers, longcatDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (m *longcatProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *longcatProvider) TransformRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	if m.config.responseJsonSchema != nil && apiName == ApiNameChatCompletion {
		request := &chatCompletionRequest{}
		if err := decodeChatCompletionRequest(body, request); err != nil {
			return nil, err
		}
		request.ResponseFormat = m.config.responseJsonSchema
		body, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		return body, nil
	}
	// For testing purposes, skip defaultTransformRequestBody if ctx is nil
	if ctx != nil {
		return m.config.defaultTransformRequestBody(ctx, apiName, body)
	}
	return body, nil
}
