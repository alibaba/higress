package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// openrouterProvider is the provider for OpenRouter service.
const (
	openrouterDomain             = "openrouter.ai"
	openrouterChatCompletionPath = "/api/v1/chat/completions"
	openrouterCompletionPath     = "/api/v1/completions"
)

type openrouterProviderInitializer struct{}

func (o *openrouterProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (o *openrouterProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): openrouterChatCompletionPath,
		string(ApiNameCompletion):     openrouterCompletionPath,
	}
}

func (o *openrouterProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(o.DefaultCapabilities())
	return &openrouterProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type openrouterProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (o *openrouterProvider) GetProviderType() string {
	return providerTypeOpenRouter
}

func (o *openrouterProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	o.config.handleRequestHeaders(o, ctx, apiName)
	return nil
}

func (o *openrouterProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !o.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return o.config.handleRequestBody(o, o.contextCache, ctx, apiName, body)
}

func (o *openrouterProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), o.config.capabilities)
	util.OverwriteRequestHostHeader(headers, openrouterDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+o.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (o *openrouterProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, openrouterChatCompletionPath) {
		return ApiNameChatCompletion
	}
	if strings.Contains(path, openrouterCompletionPath) {
		return ApiNameCompletion
	}
	return ""
}
