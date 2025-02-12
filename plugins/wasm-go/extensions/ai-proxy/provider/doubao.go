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
	doubaoDomain             = "ark.cn-beijing.volces.com"
	doubaoChatCompletionPath = "/api/v3/chat/completions"
	doubaoEmbeddingsPath     = "/api/v3/embeddings"
)

type doubaoProviderInitializer struct{}

func (m *doubaoProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *doubaoProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): doubaoChatCompletionPath,
		string(ApiNameEmbeddings):     doubaoEmbeddingsPath,
	}
}

func (m *doubaoProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(m.DefaultCapabilities())
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

func (m *doubaoProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if !m.config.isSupportedAPI(apiName) {
		return errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return nil
}

func (m *doubaoProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *doubaoProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestHostHeader(headers, doubaoDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (m *doubaoProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, doubaoChatCompletionPath) {
		return ApiNameChatCompletion
	}
	if strings.Contains(path, doubaoEmbeddingsPath) {
		return ApiNameEmbeddings
	}
	return ""
}
