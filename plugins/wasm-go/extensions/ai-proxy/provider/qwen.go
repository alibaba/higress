package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// qwenProvider is the provider for Qwen service.

const (
	qwenResultFormatMessage = "message"

	qwenDomain             = "dashscope.aliyuncs.com"
	qwenChatCompletionPath = "/api/v1/services/aigc/text-generation/generation"
)

type qwenProviderInitializer struct {
}

func (m *qwenProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (m *qwenProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &qwenProvider{
		config: config,
	}, nil
}

type qwenProvider struct {
	config ProviderConfig
}

func (m *qwenProvider) GetPointcuts() map[Pointcut]interface{} {
	return map[Pointcut]interface{}{PointcutOnRequestHeaders: nil, PointcutOnRequestBody: nil, PointcutOnResponseHeaders: nil, PointcutOnResponseBody: nil}
}

func (m *qwenProvider) OnApiRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(qwenChatCompletionPath)
	_ = util.OverwriteRequestHost(qwenDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.apiToken)
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	return types.ActionContinue, nil
}

func (m *qwenProvider) OnApiRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}

	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}

	model := request.Model
	if model == "" {
		return types.ActionContinue, errors.New("missing model in chat completion request")
	}
	ctx.SetContext(ctxKeyOriginalRequestModel, model)
	mappedModel := getMappedModel(model, m.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel
	ctx.SetContext(ctxKeyFinalRequestModel, request.Model)

	qwenRequest := m.buildQwenTextGenerationRequest(request)
	return types.ActionContinue, replaceJsonRequestBody(qwenRequest, log)
}

func (m *qwenProvider) OnApiResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	contentType, err := proxywasm.GetHttpResponseHeader("Content-Type")
	if err != nil {
		return types.ActionContinue, fmt.Errorf("unable to load content-type from response header: %v", err)
	}
	streaming := strings.HasPrefix(contentType, contentTypeTextEventStream)
	ctx.SetContext(ctxKeyStreaming, streaming)
	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	return types.ActionContinue, nil
}

func (m *qwenProvider) OnApiResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	streaming := ctx.GetContext(ctxKeyStreaming).(bool)

	if !streaming {
		qwenResponse := &qwenTextGenResponse{}
		if err := json.Unmarshal(body, qwenResponse); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal Qwen response: %v", err)
		}
		response := m.buildChatCompletionResponse(ctx, qwenResponse)
		return types.ActionContinue, replaceJsonResponseBody(response, log)
	}

	lastNewLineIndex := len(body)
	var lastEventData []byte = nil
	keyword := []byte("data:")
	for i := len(body) - len(keyword); i >= 0; i-- {
		if body[i] != '\n' {
			continue
		}
		if bytes.Equal(body[i+1:i+1+len(keyword)], keyword) {
			lastEventData = body[i+1+len(keyword) : lastNewLineIndex]
			break
		} else {
			lastNewLineIndex = i
		}
	}
	if lastEventData == nil {
		return types.ActionContinue, fmt.Errorf("no event received")
	}
	qwenResponse := &qwenTextGenResponse{}
	if err := json.Unmarshal(lastEventData, qwenResponse); err != nil {
		log.Errorf("unable to unmarshal Qwen response: %v", err)
		return types.ActionContinue, fmt.Errorf("unable to unmarshal Qwen response: %v", err)
	}
	response := m.buildChatCompletionResponse(ctx, qwenResponse)
	response.Object = objectChatCompletionChunk

	body, err := json.Marshal(response)
	if err != nil {
		log.Errorf("unable to marshal response: %v", err)
		return types.ActionContinue, fmt.Errorf("unable to marshal response: %v", err)
	}
	body = append(append([]byte("id:1\nevent:result\n:HTTP_STATUS/200\ndata:"), body...), '\n', '\n')
	err = proxywasm.ReplaceHttpResponseBody(body)
	if err != nil {
		return types.ActionContinue, fmt.Errorf("unable to replace the original response body: %v", err)
	}
	return types.ActionContinue, nil
}

func (m *qwenProvider) buildQwenTextGenerationRequest(origRequest *chatCompletionRequest) *qwenTextGenRequest {
	return &qwenTextGenRequest{
		Model: origRequest.Model,
		Input: qwenTextGenInput{
			Messages: origRequest.Messages,
		},
		Parameters: qwenTextGenParameters{
			ResultFormat: qwenResultFormatMessage,
			MaxTokens:    origRequest.MaxTokens,
			N:            origRequest.N,
			Seed:         origRequest.Seed,
			Temperature:  origRequest.Temperature,
			TopP:         origRequest.TopP,
		},
	}
}

func (m *qwenProvider) buildChatCompletionResponse(ctx wrapper.HttpContext, qwenResponse *qwenTextGenResponse) *chatCompletionResponse {
	choices := make([]chatCompletionChoice, 0, len(qwenResponse.Output.Choices))
	for _, qwenChoice := range qwenResponse.Output.Choices {
		choices = append(choices, chatCompletionChoice{
			Message:      qwenChoice.Message,
			FinishReason: qwenChoice.FinishReason,
		})
	}
	return &chatCompletionResponse{
		Id:                qwenResponse.RequestId,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetContext(ctxKeyFinalRequestModel).(string),
		SystemFingerprint: "",
		Object:            objectChatCompletion,
		Choices:           choices,
		Usage: chatCompletionUsage{
			PromptTokens:     qwenResponse.Usage.InputTokens,
			CompletionTokens: qwenResponse.Usage.OutputTokens,
			TotalTokens:      qwenResponse.Usage.TotalTokens,
		},
	}
}

type qwenTextGenRequest struct {
	Model      string                `json:"model"`
	Input      qwenTextGenInput      `json:"input"`
	Parameters qwenTextGenParameters `json:"parameters,omitempty"`
}

type qwenTextGenInput struct {
	Messages []chatMessage `json:"messages"`
}

type qwenTextGenParameters struct {
	ResultFormat      string  `json:"result_format,omitempty"`
	MaxTokens         int     `json:"max_tokens,omitempty"`
	RepetitionPenalty float64 `json:"repetition_penalty,omitempty"`
	N                 int     `json:"n,omitempty"`
	Seed              int     `json:"seed,omitempty"`
	Temperature       float64 `json:"temperature,omitempty"`
	TopP              float64 `json:"top_p,omitempty"`
}

type qwenTextGenResponse struct {
	RequestId string            `json:"request_id"`
	Output    qwenTextGenOutput `json:"output"`
	Usage     qwenTextGenUsage  `json:"usage"`
}

type qwenTextGenOutput struct {
	FinishReason string              `json:"finish_reason"`
	Choices      []qwenTextGenChoice `json:"choices"`
}

type qwenTextGenChoice struct {
	FinishReason string      `json:"finish_reason"`
	Message      chatMessage `json:"message"`
}

type qwenTextGenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
