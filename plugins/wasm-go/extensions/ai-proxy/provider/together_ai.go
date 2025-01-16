package provider

import (
	"errors"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"net/http"
	"strings"
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

func (m *togetherAIProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
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

func (m *togetherAIProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return nil
}

func (m *togetherAIProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *togetherAIProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, togetherAICompletionPath)
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
