package provider

import (
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	yiDomain             = "api.lingyiwanwu.com"
	yiChatCompletionPath = "/v1/chat/completions"
)

type yiProviderInitializer struct {
}

func (m *yiProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (m *yiProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &yiProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type yiProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *yiProvider) GetProviderType() string {
	return providerTypeYi
}

func (m *yiProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(yiChatCompletionPath)
	_ = util.OverwriteRequestHost(yiDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())

	if m.contextCache == nil {
		ctx.DontReadRequestBody()
	} else {
		_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	}

	return types.ActionContinue, nil
}

func (m *yiProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
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
			_ = util.SendResponse(500, "ai-proxy.yi.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.yi.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}
