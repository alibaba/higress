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
	mistralDomain             = "api.mistral.ai"
	mistralChatCompletionPath = "/v1/chat/completions"
)

type mistralProviderInitializer struct{}

func (m *mistralProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *mistralProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &mistralProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type mistralProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *mistralProvider) GetProviderType() string {
	return providerTypeMistral
}

func (m *mistralProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return types.ActionContinue, nil
}

func (m *mistralProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *mistralProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestHostHeader(headers, mistralDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (m *mistralProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, mistralChatCompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}
