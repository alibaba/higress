package provider

import (
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"strings"
)

const (
	cloudflareDomain = "api.cloudflare.com"
	// https://developers.cloudflare.com/workers-ai/configuration/open-ai-compatibility/
	cloudflareChatCompletionPath = "/client/v4/accounts/{account_id}/ai/v1/chat/completions"
)

type cloudflareProviderInitializer struct {
}

func (c *cloudflareProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (c *cloudflareProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &cloudflareProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type cloudflareProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (c *cloudflareProvider) GetProviderType() string {
	return providerTypeCloudflare
}

func (c *cloudflareProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(strings.Replace(cloudflareChatCompletionPath, "{account_id}", c.config.cloudflareAccountId, 1))
	_ = util.OverwriteRequestHost(cloudflareDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+c.config.GetRandomToken())

	if c.config.context == nil && c.config.protocol == protocolOriginal {
		ctx.DontReadRequestBody()
	}
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	return types.ActionContinue, nil
}

func (c *cloudflareProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
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
	ctx.SetContext(ctxKeyOriginalRequestModel, model)
	mappedModel := getMappedModel(model, c.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel
	ctx.SetContext(ctxKeyFinalRequestModel, request.Model)

	streaming := request.Stream
	if streaming {
		_ = proxywasm.ReplaceHttpRequestHeader("Accept", "text/event-stream")
	}

	if c.contextCache == nil {
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.cloudflare.transform_body_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
		return types.ActionContinue, nil
	}
	err := c.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.cloudflare.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.cloudflare.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}
