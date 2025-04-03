package provider

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// githubProvider is the provider for GitHub OpenAI service.
const (
	githubDomain         = "models.inference.ai.azure.com"
	githubCompletionPath = "/chat/completions"
	githubEmbeddingPath  = "/embeddings"
)

type githubProviderInitializer struct {
}

type githubProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *githubProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *githubProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): githubCompletionPath,
		string(ApiNameEmbeddings):     githubEmbeddingPath,
	}
}

func (m *githubProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(m.DefaultCapabilities())
	return &githubProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

func (m *githubProvider) GetProviderType() string {
	return providerTypeGithub
}

func (m *githubProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return nil
}

func (m *githubProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *githubProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestHostHeader(headers, githubDomain)
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestAuthorizationHeader(headers, m.config.GetApiTokenInUse(ctx))
}

func (m *githubProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, githubCompletionPath) {
		return ApiNameChatCompletion
	}
	if strings.Contains(path, githubEmbeddingPath) {
		return ApiNameEmbeddings
	}
	return ""
}
