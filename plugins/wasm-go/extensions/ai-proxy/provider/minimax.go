package provider

import (
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// minimaxProvider is the provider for minimax service.

const (
	minimaxDomain             = "api.minimax.chat"
	minimaxChatCompletionPath = "/v1/text/chatcompletion_v2"
)

type minimaxProviderInitializer struct {
}

func (m *minimaxProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (m *minimaxProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &minimaxProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type minimaxProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *minimaxProvider) GetProviderType() string {
	return providerTypeMinimax
}

func (m *minimaxProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(minimaxChatCompletionPath)
	_ = util.OverwriteRequestHost(minimaxDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	return types.ActionContinue, nil
}

func (m *minimaxProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}

	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}

	// 使用OpenAI接口协议,映射模型
	if m.config.protocol == protocolOpenAI {
		model := request.Model
		if model == "" {
			return types.ActionContinue, errors.New("missing model in chat completion request")
		}
		mappedModel := getMappedModel(model, m.config.modelMapping, log)
		if mappedModel == "" {
			return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
		}
		request.Model = mappedModel
	}

	if m.contextCache == nil {
		return types.ActionContinue, replaceJsonRequestBody(request, log)
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
