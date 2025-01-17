package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// sparkProvider is the provider for SparkLLM AI service.
const (
	sparkHost               = "spark-api-open.xf-yun.com"
	sparkChatCompletionPath = "/v1/chat/completions"
)

type sparkProviderInitializer struct {
}

type sparkProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

type sparkRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	TopK        int           `json:"top_k,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Tools       []tool        `json:"tools,omitempty"`
	ToolChoice  string        `json:"tool_choice,omitempty"`
}

type sparkResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Sid     string                 `json:"sid"`
	Choices []chatCompletionChoice `json:"choices"`
	Usage   usage                  `json:"usage,omitempty"`
}

type sparkStreamResponse struct {
	sparkResponse
	Id      string `json:"id"`
	Created int64  `json:"created"`
}

func (i *sparkProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	return nil
}

func (i *sparkProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &sparkProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

func (p *sparkProvider) GetProviderType() string {
	return providerTypeSpark
}

func (p *sparkProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	p.config.handleRequestHeaders(p, ctx, apiName, log)
	return nil
}

func (p *sparkProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return p.config.handleRequestBody(p, p.contextCache, ctx, apiName, body, log)
}

func (p *sparkProvider) TransformResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) ([]byte, error) {
	sparkResponse := &sparkResponse{}
	if err := json.Unmarshal(body, sparkResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal spark response: %v", err)
	}
	if sparkResponse.Code != 0 {
		return nil, fmt.Errorf("spark response error, error_code: %d, error_message: %s", sparkResponse.Code, sparkResponse.Message)
	}
	response := p.responseSpark2OpenAI(ctx, sparkResponse)
	return json.Marshal(response)
}

func (p *sparkProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	if isLastChunk || len(chunk) == 0 {
		return nil, nil
	}
	responseBuilder := &strings.Builder{}
	lines := strings.Split(string(chunk), "\n")
	for _, data := range lines {
		if len(data) < 6 {
			// ignore blank line or wrong format
			continue
		}
		data = data[6:]
		// The final response is `data: [DONE]`
		if data == "[DONE]" {
			continue
		}
		var sparkResponse sparkStreamResponse
		if err := json.Unmarshal([]byte(data), &sparkResponse); err != nil {
			log.Errorf("unable to unmarshal spark response: %v", err)
			continue
		}
		response := p.streamResponseSpark2OpenAI(ctx, &sparkResponse)
		responseBody, err := json.Marshal(response)
		if err != nil {
			log.Errorf("unable to marshal response: %v", err)
			return nil, err
		}
		p.appendResponse(responseBuilder, string(responseBody))
	}
	modifiedResponseChunk := responseBuilder.String()
	log.Debugf("=== modified response chunk: %s", modifiedResponseChunk)
	return []byte(modifiedResponseChunk), nil
}

func (p *sparkProvider) responseSpark2OpenAI(ctx wrapper.HttpContext, response *sparkResponse) *chatCompletionResponse {
	choices := make([]chatCompletionChoice, len(response.Choices))
	for idx, c := range response.Choices {
		choices[idx] = chatCompletionChoice{
			Index:   c.Index,
			Message: &chatMessage{Role: c.Message.Role, Content: c.Message.Content},
		}
	}
	return &chatCompletionResponse{
		Id:      response.Sid,
		Created: time.Now().UnixMilli() / 1000,
		Object:  objectChatCompletion,
		Model:   ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		Choices: choices,
		Usage:   response.Usage,
	}
}

func (p *sparkProvider) streamResponseSpark2OpenAI(ctx wrapper.HttpContext, response *sparkStreamResponse) *chatCompletionResponse {
	choices := make([]chatCompletionChoice, len(response.Choices))
	for idx, c := range response.Choices {
		choices[idx] = chatCompletionChoice{
			Index: c.Index,
			Delta: &chatMessage{Role: c.Delta.Role, Content: c.Delta.Content},
		}
	}
	return &chatCompletionResponse{
		Id:      response.Sid,
		Created: response.Created,
		Model:   ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		Object:  objectChatCompletion,
		Choices: choices,
		Usage:   response.Usage,
	}
}

func (p *sparkProvider) appendResponse(responseBuilder *strings.Builder, responseBody string) {
	responseBuilder.WriteString(fmt.Sprintf("%s %s\n\n", streamDataItemKey, responseBody))
}

func (p *sparkProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, sparkChatCompletionPath)
	util.OverwriteRequestHostHeader(headers, sparkHost)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+p.config.GetApiTokenInUse(ctx))
}
