package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// groqProvider is the provider for Groq service.
const (
	groqDomain             = "api.groq.com"
	groqChatCompletionPath = "/openai/v1/chat/completions"
)

type groqProviderInitializer struct{}

func (g *groqProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (g *groqProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &groqProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type groqProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (g *groqProvider) GetProviderType() string {
	return providerTypeGroq
}

func (g *groqProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	g.config.handleRequestHeaders(g, ctx, apiName, log)
	return nil
}

func (g *groqProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return g.config.handleRequestBody(g, g.contextCache, ctx, apiName, body, log)
}

func (g *groqProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, groqChatCompletionPath)
	util.OverwriteRequestHostHeader(headers, groqDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+g.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (g *groqProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, groqChatCompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}
