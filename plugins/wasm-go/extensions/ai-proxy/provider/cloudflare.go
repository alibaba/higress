package provider

import (
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

	if c.contextCache == nil {
		ctx.DontReadRequestBody()
	} else {
		_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	}

	return types.ActionContinue, nil
}

func (c *cloudflareProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	if c.contextCache == nil {
		return types.ActionContinue, nil
	}
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}
	err := c.contextCache.GetContent(func(content string, err error) {
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
