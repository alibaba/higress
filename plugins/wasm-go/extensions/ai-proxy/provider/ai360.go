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

// ai360Provider is the provider for 360 OpenAI service.
const (
	ai360Domain = "api.360.cn"
)

type ai360ProviderInitializer struct {
}

type ai360Provider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *ai360ProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *ai360ProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &ai360Provider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

func (m *ai360Provider) GetProviderType() string {
	return providerTypeAi360
}

func (m *ai360Provider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion && apiName != ApiNameEmbeddings {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestHost(ai360Domain)
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", m.config.GetRandomToken())
	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return types.HeaderStopIteration, nil
}

func (m *ai360Provider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
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

func (m *ai360Provider) onChatCompletionRequestBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
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

func (m *ai360Provider) onEmbeddingsRequestBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
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
