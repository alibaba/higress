package provider

import (
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	doubaoDomain             = "ark.cn-beijing.volces.com"
	doubaoChatCompletionPath = "/api/v3/chat/completions"
)

type doubaoProviderInitializer struct{}

func (m *doubaoProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *doubaoProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &doubaoProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type doubaoProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *doubaoProvider) GetProviderType() string {
	return providerTypeDoubao
}

func (m *doubaoProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	_ = util.OverwriteRequestHost(doubaoDomain)
	_ = util.OverwriteRequestAuthorization("Bearer " + m.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	if m.config.protocol == protocolOriginal {
		ctx.DontReadRequestBody()
		return types.ActionContinue, nil
	}
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(doubaoChatCompletionPath)
	return types.ActionContinue, nil
}

func (m *doubaoProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
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
	if m.contextCache != nil {
		err := m.contextCache.GetContent(func(content string, err error) {
			defer func() {
				_ = proxywasm.ResumeHttpRequest()
			}()
			if err != nil {
				log.Errorf("failed to load context file: %v", err)
				_ = util.SendResponse(500, "ai-proxy.doubao.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
			}
			insertContextMessage(request, content)
			if err := replaceJsonRequestBody(request, log); err != nil {
				_ = util.SendResponse(500, "ai-proxy.doubao.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
			}
		}, log)
		if err == nil {
			return types.ActionPause, nil
		} else {
			return types.ActionContinue, err
		}
	} else {
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.doubao.transform_body_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
			return types.ActionContinue, err
		}
		_ = proxywasm.ResumeHttpRequest()
		return types.ActionPause, nil
	}
}
