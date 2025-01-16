package provider

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	cohereDomain             = "api.cohere.com"
	cohereChatCompletionPath = "/v1/chat"
)

type cohereProviderInitializer struct{}

func (m *cohereProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *cohereProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &cohereProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type cohereProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

type cohereTextGenRequest struct {
	Message          string   `json:"message,omitempty"`
	Model            string   `json:"model,omitempty"`
	Stream           bool     `json:"stream,omitempty"`
	MaxTokens        int      `json:"max_tokens,omitempty"`
	Temperature      float64  `json:"temperature,omitempty"`
	K                int      `json:"k,omitempty"`
	P                float64  `json:"p,omitempty"`
	Seed             int      `json:"seed,omitempty"`
	StopSequences    []string `json:"stop_sequences,omitempty"`
	FrequencyPenalty float64  `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64  `json:"presence_penalty,omitempty"`
}

func (m *cohereProvider) GetProviderType() string {
	return providerTypeCohere
}

func (m *cohereProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return nil
}

func (m *cohereProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *cohereProvider) buildCohereRequest(origin *chatCompletionRequest) *cohereTextGenRequest {
	if len(origin.Messages) == 0 {
		return nil
	}
	return &cohereTextGenRequest{
		Message:          origin.Messages[0].StringContent(),
		Model:            origin.Model,
		MaxTokens:        origin.MaxTokens,
		Stream:           origin.Stream,
		Temperature:      origin.Temperature,
		K:                origin.N,
		P:                origin.TopP,
		Seed:             origin.Seed,
		StopSequences:    origin.Stop,
		FrequencyPenalty: origin.FrequencyPenalty,
		PresencePenalty:  origin.PresencePenalty,
	}
}

func (m *cohereProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, cohereChatCompletionPath)
	util.OverwriteRequestHostHeader(headers, cohereDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

func (m *cohereProvider) TransformRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) ([]byte, error) {
	request := &chatCompletionRequest{}
	if err := m.config.parseRequestAndMapModel(ctx, request, body, log); err != nil {
		return nil, err
	}

	cohereRequest := m.buildCohereRequest(request)
	return json.Marshal(cohereRequest)
}

func (m *cohereProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, cohereChatCompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}
