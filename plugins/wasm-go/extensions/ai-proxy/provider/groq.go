package provider

import (
	"encoding/json"
	"errors"
	"net/http"

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

func (g *groqProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	originalHeaders := util.GetOriginaHttplHeaders()
	g.TransformRequestHeaders(originalHeaders, ctx, log)
	util.ReplaceOriginalHttpHeaders(originalHeaders)
	return types.ActionContinue, nil
}

func (g *groqProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	modifiedBody, err := g.TransformRequestBody(body, ctx, log)
	if err != nil {
		return types.ActionContinue, err
	}
	err = replaceHttpJsonRequestBody(modifiedBody, log)
	if err != nil {
		return types.ActionContinue, err
	}
	return types.ActionContinue, nil
}

func (g *groqProvider) TransformRequestHeaders(headers http.Header, ctx wrapper.HttpContext, log wrapper.Log) {
	util.OverwriteHttpRequestPath(headers, groqChatCompletionPath)
	util.OverwriteHttpRequestHost(headers, groqDomain)
	util.OverwriteHttpRequestAuthorization(headers, "Bearer "+g.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (g *groqProvider) TransformRequestBody(body []byte, ctx wrapper.HttpContext, log wrapper.Log) ([]byte, error) {
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return nil, err
	}

	err := g.config.setRequestModel(ctx, request, log)
	if err != nil {
		return nil, err
	}

	return json.Marshal(request)
}
