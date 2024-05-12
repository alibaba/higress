package provider

import (
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// azureProvider is the provider for Azure OpenAI service.

const (
	openaiDomain             = "api.openai.com"
	openaiChatCompletionPath = "/v1/chat/completions"
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

func (m *openaiProvider) GetPointcuts() map[Pointcut]interface{} {
	return map[Pointcut]interface{}{PointcutOnRequestHeaders: nil}
}

func (m *openaiProvider) OnApiRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(openaiChatCompletionPath)
	_ = util.OverwriteRequestHost(openaiDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())

	if m.contextCache == nil {
		ctx.DontReadRequestBody()
	} else {
		_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	}

	return types.ActionContinue, nil
}

func (m *openaiProvider) OnApiRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	if m.contextCache == nil {
		return types.ActionContinue, nil
	}
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}
	err := m.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (m *openaiProvider) OnApiResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}

func (m *openaiProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	return nil, nil
}

func (m *openaiProvider) OnApiResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}
