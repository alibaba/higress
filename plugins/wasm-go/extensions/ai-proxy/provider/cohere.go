package provider

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	cohereDomain       = "api.cohere.com"
	chatCompletionPath = "/v1/chat"
)

type cohereProviderInitializer struct{}

func (m *cohereProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *cohereProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &cohereProvider{
		config: config,
	}, nil
}

type cohereProvider struct {
	config ProviderConfig
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

func (m *cohereProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestHost(cohereDomain)
	_ = util.OverwriteRequestPath(chatCompletionPath)
	_ = util.OverwriteRequestAuthorization("Bearer " + m.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	return types.ActionContinue, nil
}

func (m *cohereProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	if m.config.protocol == protocolOriginal {
		request := &cohereTextGenRequest{}
		if err := json.Unmarshal(body, request); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
		}
		return m.handleRequestBody(log, request)
	}
	origin := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, origin); err != nil {
		return types.ActionContinue, err
	}
	request := m.buildCohereRequest(origin)
	return m.handleRequestBody(log, request)
}

func (m *cohereProvider) handleRequestBody(log wrapper.Log, request interface{}) (types.Action, error) {
	defer func() {
		_ = proxywasm.ResumeHttpRequest()
	}()
	err := replaceJsonRequestBody(request, log)
	if err != nil {
		_ = util.SendResponse(500, "ai-proxy.cohere.proxy_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
	}
	return types.ActionContinue, err
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
