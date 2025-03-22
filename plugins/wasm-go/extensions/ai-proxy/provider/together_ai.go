package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	togetherAIDomain         = "api.together.xyz"
	togetherAICompletionPath = "/v1/chat/completions"
)

type togetherAIProviderInitializer struct{}

func (m *togetherAIProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *togetherAIProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): togetherAICompletionPath,
	}
}

func (m *togetherAIProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(m.DefaultCapabilities())
	return &togetherAIProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type togetherAIProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *togetherAIProvider) GetProviderType() string {
	return providerTypeTogetherAI
}

func (m *togetherAIProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	return nil
}

func (m *togetherAIProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *togetherAIProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestHostHeader(headers, togetherAIDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (m *togetherAIProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, togetherAICompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}
