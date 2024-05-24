package provider

import (
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// ollamaProvider is the provider for Ollama service.

const (
	ollamaDomain             = "localhost:11434"
	ollamaChatCompletionPath = "/v1/chat/completions"
)

type ollamaProviderInitializer struct {
}

func (m *ollamaProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (m *ollamaProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &ollamaProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type ollamaProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *ollamaProvider) GetProviderType() string {
	return providerTypeOllama
}

func (m *ollamaProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	log.Debugf("[OnRequestHeaders] Called by ollama")
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(ollamaChatCompletionPath)
	_ = util.OverwriteRequestHost(ollamaDomain)
	log.Debugf("Request host overwritten to: %s", ollamaDomain)
	// _ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())

	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	return types.ActionContinue, nil
}

func (m *ollamaProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}

	model := request.Model
	if model == "" {
		return types.ActionContinue, errors.New("missing model in chat completion request")
	}
	mappedModel := getMappedModel(model, m.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel
	log.Debugf("Replace model by: %s", request.Model)

	// if m.contextCache == nil {
	// 	return types.ActionContinue, nil
	// }
	
	var getcontenterr error
	if m.contextCache != nil {
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
				log.Debugf("failed to replace json request body")
				_ = util.SendResponse(500, util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
			}
		}, log)
		if err == nil {
			return types.ActionPause, nil
		}
		getcontenterr = err
	} else {
		if err := replaceJsonRequestBody(request, log); err != nil {
			log.Debugf("failed to replace json request body")
			_ = util.SendResponse(500, util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
		return types.ActionPause, nil
	}
	

	return types.ActionContinue, getcontenterr
}
