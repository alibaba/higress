package provider

import (
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	stepfunDomain             = "api.stepfun.com"
	stepfunChatCompletionPath = "/v1/chat/completions"
)

type stepfunProviderInitializer struct {
}

func (m *stepfunProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *stepfunProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		// stepfun的chat接口path和OpenAI的chat接口一样
		string(ApiNameChatCompletion): stepfunChatCompletionPath,
	}
}

func (m *stepfunProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(m.DefaultCapabilities())
	return &stepfunProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type stepfunProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *stepfunProvider) GetProviderType() string {
	return providerTypeStepfun
}

func (m *stepfunProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	return nil
}

func (m *stepfunProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *stepfunProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestHostHeader(headers, stepfunDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}
