package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
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
	_ = util.OverwriteRequestHost(githubDomain)
	if apiName == ApiNameChatCompletion {
		_ = util.OverwriteRequestPath(githubCompletionPath)
	}
	if apiName == ApiNameEmbeddings {
		_ = util.OverwriteRequestPath(githubEmbeddingPath)
	}
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", m.config.GetRandomToken())
	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return types.HeaderStopIteration, nil
}

func (m *githubProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion && apiName != ApiNameEmbeddings {
		return types.ActionContinue, errUnsupportedApiName
	}
	if apiName == ApiNameChatCompletion {
		return m.onChatCompletionRequestBody(ctx, body, log)
	}
	if apiName == ApiNameEmbeddings {
		return m.onEmbeddingsRequestBody(ctx, body, log)
	}
	return types.ActionContinue, errUnsupportedApiName
}

func (m *githubProvider) onChatCompletionRequestBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}
	if request.Model == "" {
		return types.ActionContinue, errors.New("missing model in chat completion request")
	}
	// 映射模型
	mappedModel := getMappedModel(request.Model, m.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	ctx.SetContext(ctxKeyFinalRequestModel, mappedModel)
	request.Model = mappedModel
	return types.ActionContinue, replaceJsonRequestBody(request, log)
}

func (m *githubProvider) onEmbeddingsRequestBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
	request := &embeddingsRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
	}
	if request.Model == "" {
		return types.ActionContinue, errors.New("missing model in embeddings request")
	}
	// 映射模型
	mappedModel := getMappedModel(request.Model, m.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	ctx.SetContext(ctxKeyFinalRequestModel, mappedModel)
	request.Model = mappedModel
	return types.ActionContinue, replaceJsonRequestBody(request, log)
}
