package provider

import (
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// openaiProvider is the provider for OpenAI service.

const (
	openaiDomain             = "api.openai.com"
	openaiChatCompletionPath = "/v1/chat/completions"
	openaiEmbeddingsPath     = "/v1/chat/embeddings"
)

type openaiProviderInitializer struct {
}

func (m *openaiProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (m *openaiProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &openaiProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type openaiProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *openaiProvider) GetProviderType() string {
	return providerTypeOpenAI
}

func (m *openaiProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	switch apiName {
	case ApiNameChatCompletion:
		_ = util.OverwriteRequestPath(openaiChatCompletionPath)
		break
	case ApiNameEmbeddings:
		_ = util.OverwriteRequestPath(openaiEmbeddingsPath)
		break
	}
	_ = util.OverwriteRequestAuthorization("Bearer " + m.config.GetRandomToken())
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
	bodyAltered := false
	if request.Stream {
		// For stream requests, we need to include usage in the response.
		if request.StreamOptions == nil {
			request.StreamOptions = &streamOptions{IncludeUsage: true}
			bodyAltered = true
		} else if !request.StreamOptions.IncludeUsage {
			request.StreamOptions.IncludeUsage = true
			bodyAltered = true
		}
	}
	if m.contextCache == nil {
		if bodyAltered {
			if err := replaceJsonRequestBody(request, log); err != nil {
				_ = util.SendResponse(500, "ai-proxy.openai.set_include_usage_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
			}
		}
		return types.ActionContinue, nil
	} else {
		// If context cache is configured and body has been altered, the new body will be replaced when inserting the context data.
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
