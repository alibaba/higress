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
	zhipuAiDomain             = "open.bigmodel.cn"
	zhipuAiChatCompletionPath = "/api/paas/v4/chat/completions"
	zhipuAiEmbeddingsPath     = "/api/paas/v4/embeddings"
)

type zhipuAiProviderInitializer struct{}

func (m *zhipuAiProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *zhipuAiProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): zhipuAiChatCompletionPath,
		string(ApiNameEmbeddings):     zhipuAiEmbeddingsPath,
	}
}

func (m *zhipuAiProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(m.DefaultCapabilities())
	return &zhipuAiProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type zhipuAiProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *zhipuAiProvider) GetProviderType() string {
	return providerTypeZhipuAi
}

func (m *zhipuAiProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if !m.config.isSupportedAPI(apiName) {
		return m.config.handleUnsupportedAPI()
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return nil
}

func (m *zhipuAiProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, m.config.handleUnsupportedAPI()
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *zhipuAiProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestHostHeader(headers, zhipuAiDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (m *zhipuAiProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, zhipuAiChatCompletionPath) {
		return ApiNameChatCompletion
	}
	if strings.Contains(path, zhipuAiEmbeddingsPath) {
		return ApiNameEmbeddings
	}
	return ""
}
