package provider

import (
    "errors"
    "net/http"

    "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
    "github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
    "github.com/higress-group/wasm-go/pkg/wrapper"
)

// qiniuProvider is the provider for Qiniu AI service.
const (
    qiniuDomain = "api.qnaigc.com" 
)

type qiniuProviderInitializer struct{}

func (m *qiniuProviderInitializer) ValidateConfig(config *ProviderConfig) error {
    if len(config.apiTokens) == 0 {
        return errors.New("no apiToken found in provider config")
    }
    return nil
}

func (m *qiniuProviderInitializer) DefaultCapabilities() map[string]string {
    return map[string]string{
        string(ApiNameChatCompletion): PathOpenAIChatCompletions,
        string(ApiNameModels):         PathOpenAIModels,
    }
}

func (m *qiniuProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
    config.setDefaultCapabilities(m.DefaultCapabilities())
    return &qiniuProvider{
        config:       config,
        contextCache: createContextCache(&config),
    }, nil
}

type qiniuProvider struct {
    config       ProviderConfig
    contextCache *contextCache
}

func (m *qiniuProvider) GetProviderType() string {
    return providerTypeQiniu
}

func (m *qiniuProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
    m.config.handleRequestHeaders(m, ctx, apiName)
    return nil
}

func (m *qiniuProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
    if !m.config.isSupportedAPI(apiName) {
        return types.ActionContinue, errUnsupportedApiName
    }
    return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *qiniuProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
    util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
    util.OverwriteRequestHostHeader(headers, qiniuDomain)
    util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
    headers.Del("Content-Length")
}
