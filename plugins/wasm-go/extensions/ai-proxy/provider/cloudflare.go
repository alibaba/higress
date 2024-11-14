package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	cloudflareDomain = "api.cloudflare.com"
	// https://developers.cloudflare.com/workers-ai/configuration/open-ai-compatibility/
	cloudflareChatCompletionPath     = "/v1/chat/completions"
	cloudflareChatCompletionFullPath = "/client/v4/accounts/{account_id}/ai/v1/chat/completions"
)

type cloudflareProviderInitializer struct {
}

func (c *cloudflareProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
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
	c.config.handleRequestHeaders(c, ctx, apiName, log)
	return types.ActionContinue, nil
}

func (c *cloudflareProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return c.config.handleRequestBody(c, c.contextCache, ctx, apiName, body, log)
}

func (c *cloudflareProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, strings.Replace(cloudflareChatCompletionFullPath, "{account_id}", c.config.cloudflareAccountId, 1))
	util.OverwriteRequestHostHeader(headers, cloudflareDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+c.config.GetApiTokenInUse(ctx))
	headers.Del("Accept-Encoding")
	headers.Del("Content-Length")
}

func (c *cloudflareProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, cloudflareChatCompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}
