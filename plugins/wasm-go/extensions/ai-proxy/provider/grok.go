package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// grokProvider is the provider for Grok service.
const (
	grokDomain             = "api.x.ai"
	grokChatCompletionPath = "/v1/chat/completions"
)

type grokProviderInitializer struct{}

func (g *grokProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (g *grokProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): grokChatCompletionPath,
	}
}

func (g *grokProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(g.DefaultCapabilities())
	return &grokProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type grokProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (g *grokProvider) GetProviderType() string {
	return providerTypeGrok
}

func (g *grokProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	g.config.handleRequestHeaders(g, ctx, apiName)
	return nil
}

func (g *grokProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !g.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return g.config.handleRequestBody(g, g.contextCache, ctx, apiName, body)
}

func (g *grokProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), g.config.capabilities)
	util.OverwriteRequestHostHeader(headers, grokDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+g.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (g *grokProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, grokChatCompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}
