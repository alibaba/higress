package provider

import (
	"errors"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"net/http"
	"strings"
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

func (m *githubProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *githubProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &githubProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

func (m *githubProvider) GetProviderType() string {
	return providerTypeGithub
}

func (m *githubProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion && apiName != ApiNameEmbeddings {
		return types.ActionContinue, errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return types.HeaderStopIteration, nil
}

func (m *githubProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion && apiName != ApiNameEmbeddings {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *githubProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestHostHeader(headers, githubDomain)
	if apiName == ApiNameChatCompletion {
		util.OverwriteRequestPathHeader(headers, githubCompletionPath)
	}
	if apiName == ApiNameEmbeddings {
		util.OverwriteRequestPathHeader(headers, githubEmbeddingPath)
	}
	util.OverwriteRequestAuthorizationHeader(headers, m.config.GetApiTokenInUse(ctx))
	headers.Del("Accept-Encoding")
	headers.Del("Content-Length")
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
