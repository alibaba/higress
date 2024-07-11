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
	ollamaChatCompletionPath = "/v1/chat/completions"
)

type ollamaProviderInitializer struct {
}

func (m *ollamaProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.ollamaServerHost == "" {
		return errors.New("missing ollamaServerHost in provider config")
	}
	if config.ollamaServerPort == 0 {
		return errors.New("missing ollamaServerPort in provider config")
	}
	return nil
}

func (m *ollamaProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	serverPortStr := fmt.Sprintf("%d", config.ollamaServerPort)
	serviceDomain := config.ollamaServerHost + ":" + serverPortStr
	return &ollamaProvider{
		config:        config,
		serviceDomain: serviceDomain,
		contextCache:  createContextCache(&config),
	}, nil
}

type ollamaProvider struct {
	config        ProviderConfig
	serviceDomain string
	contextCache  *contextCache
}

func (m *ollamaProvider) GetProviderType() string {
	return providerTypeOllama
}

func (m *ollamaProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(ollamaChatCompletionPath)
	_ = util.OverwriteRequestHost(m.serviceDomain)
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	return types.ActionContinue, nil
}

func (m *ollamaProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}

	if m.config.modelMapping == nil && m.contextCache == nil {
		return types.ActionContinue, nil
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

	if m.contextCache != nil {
		err := m.contextCache.GetContent(func(content string, err error) {
			defer func() {
				_ = proxywasm.ResumeHttpRequest()
			}()
			if err != nil {
				log.Errorf("failed to load context file: %v", err)
				_ = util.SendResponse(500, "ai-proxy.ollama.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
			}
			insertContextMessage(request, content)
			if err := replaceJsonRequestBody(request, log); err != nil {
				_ = util.SendResponse(500, "ai-proxy.ollama.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
			}
		}, log)
		if err == nil {
			return types.ActionPause, nil
		} else {
			return types.ActionContinue, err
		}
	} else {
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.ollama.transform_body_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
			return types.ActionContinue, err
		}
		_ = proxywasm.ResumeHttpRequest()
		return types.ActionPause, nil
	}
}
