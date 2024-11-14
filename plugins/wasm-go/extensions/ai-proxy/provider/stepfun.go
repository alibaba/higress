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
	stepfunDomain             = "api.stepfun.com"
	stepfunChatCompletionPath = "/v1/chat/completions"
)

type stepfunProviderInitializer struct {
}

func (m *stepfunProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *stepfunProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &stepfunProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type stepfunProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *stepfunProvider) GetProviderType() string {
	return providerTypeStepfun
}

func (m *stepfunProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return types.ActionContinue, nil
}

func (m *stepfunProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *stepfunProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, stepfunChatCompletionPath)
	util.OverwriteRequestHostHeader(headers, stepfunDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (m *stepfunProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, stepfunChatCompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}
