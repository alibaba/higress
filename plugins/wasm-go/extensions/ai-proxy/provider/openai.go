package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// openaiProvider is the provider for OpenAI service.

const (
	defaultOpenaiDomain             = "api.openai.com"
	defaultOpenaiChatCompletionPath = "/v1/chat/completions"
	defaultOpenaiEmbeddingsPath     = "/v1/chat/embeddings"
)

type openaiProviderInitializer struct {
}

func (m *openaiProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	return nil
}

func (m *openaiProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	if config.openaiCustomUrl == "" {
		return &openaiProvider{
			config:       config,
			contextCache: createContextCache(&config),
		}, nil
	}
	customUrl := strings.TrimPrefix(strings.TrimPrefix(config.openaiCustomUrl, "http://"), "https://")
	pairs := strings.SplitN(customUrl, "/", 2)
	if len(pairs) != 2 {
		return nil, fmt.Errorf("invalid openaiCustomUrl:%s", config.openaiCustomUrl)
	}
	config.setDefaultCapabilities(ApiNameChatCompletion, ApiNameEmbeddings)
	return &openaiProvider{
		config:       config,
		customDomain: pairs[0],
		customPath:   "/" + pairs[1],
		contextCache: createContextCache(&config),
	}, nil
}

type openaiProvider struct {
	config       ProviderConfig
	customDomain string
	customPath   string
	contextCache *contextCache
}

func (m *openaiProvider) GetProviderType() string {
	return providerTypeOpenAI
}

func (m *openaiProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return nil
}

func (m *openaiProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	if m.customPath == "" {
		switch apiName {
		case ApiNameChatCompletion:
			util.OverwriteRequestPathHeader(headers, defaultOpenaiChatCompletionPath)
		case ApiNameEmbeddings:
			ctx.DontReadRequestBody()
			util.OverwriteRequestPathHeader(headers, defaultOpenaiEmbeddingsPath)
		}
	} else {
		util.OverwriteRequestPathHeader(headers, m.customPath)
	}
	if m.customDomain == "" {
		util.OverwriteRequestHostHeader(headers, defaultOpenaiDomain)
	} else {
		util.OverwriteRequestHostHeader(headers, m.customDomain)
	}
	if len(m.config.apiTokens) > 0 {
		util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	}
	headers.Del("Content-Length")
}

func (m *openaiProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		// We don't need to process the request body for other APIs.
		return types.ActionContinue, nil
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *openaiProvider) TransformRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) ([]byte, error) {
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return nil, err
	}
	if m.config.responseJsonSchema != nil {
		log.Debugf("[ai-proxy] set response format to %s", m.config.responseJsonSchema)
		request.ResponseFormat = m.config.responseJsonSchema
	}
	if request.Stream {
		// For stream requests, we need to include usage in the response.
		if request.StreamOptions == nil {
			request.StreamOptions = &streamOptions{IncludeUsage: true}
		} else if !request.StreamOptions.IncludeUsage {
			request.StreamOptions.IncludeUsage = true
		}
	}
	return json.Marshal(request)
}
