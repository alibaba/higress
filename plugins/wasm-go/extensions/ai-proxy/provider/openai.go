package provider

import (
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
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

func (m *openaiProviderInitializer) ValidateConfig(config ProviderConfig) error {
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

func (m *openaiProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if m.customPath == "" {
		switch apiName {
		case ApiNameChatCompletion:
			_ = util.OverwriteRequestPath(defaultOpenaiChatCompletionPath)
		case ApiNameEmbeddings:
			ctx.DontReadRequestBody()
			_ = util.OverwriteRequestPath(defaultOpenaiEmbeddingsPath)
		}
	} else {
		_ = util.OverwriteRequestPath(m.customPath)
	}
	if m.customDomain == "" {
		_ = util.OverwriteRequestHost(defaultOpenaiDomain)
	} else {
		_ = util.OverwriteRequestHost(m.customDomain)
	}
	if len(m.config.apiTokens) > 0 {
		_ = util.OverwriteRequestAuthorization("Bearer " + m.config.GetRandomToken())
	}
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	return types.ActionContinue, nil
}

func (m *openaiProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		// We don't need to process the request body for other APIs.
		return types.ActionContinue, nil
	}
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
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
	if m.contextCache == nil {
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.openai.set_include_usage_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
		return types.ActionContinue, nil
	}
	err := m.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.openai.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.openai.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}
