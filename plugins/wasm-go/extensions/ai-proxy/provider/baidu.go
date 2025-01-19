package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// baiduProvider is the provider for baidu service.
const (
	baiduDomain             = "qianfan.baidubce.com"
	baiduChatCompletionPath = "/v2/chat/completions"
)

type baiduProviderInitializer struct{}

func (g *baiduProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (g *baiduProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &baiduProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type baiduProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (g *baiduProvider) GetProviderType() string {
	return providerTypeBaidu
}

func (g *baiduProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	g.config.handleRequestHeaders(g, ctx, apiName, log)
	return nil
}

func (g *baiduProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return g.config.handleRequestBody(g, g.contextCache, ctx, apiName, body, log)
}

func (g *baiduProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, baiduChatCompletionPath)
	util.OverwriteRequestHostHeader(headers, baiduDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+g.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (g *baiduProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, baiduChatCompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}
