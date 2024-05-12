package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

	qwenTopPMin = 0.000001
	qwenTopPMax = 0.999999
)

type qwenProviderInitializer struct {
}

func (m *qwenProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (m *qwenProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &qwenProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type qwenProvider struct {
	config ProviderConfig

	contextCache *contextCache
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
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	// Always use non-streaming mode for Qwen
	// TODO: Support Qwen streaming
	_ = proxywasm.ReplaceHttpRequestHeader("Accept", "*/*")
	_ = proxywasm.RemoveHttpRequestHeader("X-DashScope-SSE")

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

	ctx.SetContext(ctxKeyStreaming, request.Stream)

	if m.config.context == nil {
		qwenRequest := m.buildQwenTextGenerationRequest(request)
		return types.ActionContinue, replaceJsonRequestBody(qwenRequest, log)
	}

	err := m.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		qwenRequest := m.buildQwenTextGenerationRequest(request)
		if err := replaceJsonRequestBody(qwenRequest, log); err != nil {
			_ = util.SendResponse(500, util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (m *qwenProvider) OnApiResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	streaming := ctx.GetContext(ctxKeyStreaming).(bool)
	log.Debugf("=== response header streaming: %v", streaming)
	if streaming {
		_ = proxywasm.ReplaceHttpResponseHeader("Content-Type", "text/event-stream")
	}
	return types.ActionContinue, nil
}

func (m *qwenProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	return nil, nil
}

func (m *qwenProvider) OnApiResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	streaming := ctx.GetContext(ctxKeyStreaming).(bool)

	log.Debugf("=== response body streaming: %v", streaming)

	if !streaming {
		qwenResponse := &qwenTextGenResponse{}
		if err := json.Unmarshal(body, qwenResponse); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal Qwen response: %v", err)
		}
		response := m.buildChatCompletionResponse(ctx, qwenResponse)
		return types.ActionContinue, replaceJsonResponseBody(response, log)
	}

	// TODO: Support Qwen streaming
	//lastNewLineIndex := len(body)
	//var lastEventData []byte = nil
	//keyword := []byte("data:")
	//for i := len(body) - len(keyword); i >= 0; i-- {
	//	if body[i] != '\n' {
	//		continue
	//	}
	//	if bytes.Equal(body[i+1:i+1+len(keyword)], keyword) {
	//		lastEventData = body[i+1+len(keyword) : lastNewLineIndex]
	//		break
	//	} else {
	//		lastNewLineIndex = i
	//	}
	//}
	//if lastEventData == nil {
	//	log.Debugf("=== no event received")
	//	return types.ActionContinue, fmt.Errorf("no event received")
	//}

	log.Debugf("=== last event data: %s", body)
	lastEventData := body
	qwenResponse := &qwenTextGenResponse{}
	if err := json.Unmarshal(lastEventData, qwenResponse); err != nil {
		log.Errorf("unable to unmarshal Qwen response: %v", err)
		return types.ActionContinue, fmt.Errorf("unable to unmarshal Qwen response: %v", err)
	}

	var responseBuilder strings.Builder
	responses := m.buildChatCompletionStreamingResponse(ctx, qwenResponse)
	for _, response := range responses {
		responseBody, err := json.Marshal(response)
		if err != nil {
			log.Errorf("unable to marshal response: %v", err)
			return types.ActionContinue, fmt.Errorf("unable to marshal response: %v", err)
		}
		responseBuilder.WriteString("data: ")
		responseBuilder.Write(responseBody)
		responseBuilder.WriteString("\n\n")
	}
	responseBuilder.WriteString("data: [DONE]\n\n")

	finalResponseBody := responseBuilder.String()
	log.Debugf("=== response data: %s", finalResponseBody)
	err := proxywasm.ReplaceHttpResponseBody([]byte(finalResponseBody))
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
			TopP:         math.Max(qwenTopPMin, math.Min(origRequest.TopP, qwenTopPMax)),
		},
	}
}

func (m *qwenProvider) buildChatCompletionResponse(ctx wrapper.HttpContext, qwenResponse *qwenTextGenResponse) *chatCompletionResponse {
	choices := make([]chatCompletionChoice, 0, len(qwenResponse.Output.Choices))
	for _, qwenChoice := range qwenResponse.Output.Choices {
		choices = append(choices, chatCompletionChoice{
			Message:      &qwenChoice.Message,
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

func (m *qwenProvider) buildChatCompletionStreamingResponse(ctx wrapper.HttpContext, qwenResponse *qwenTextGenResponse) []*chatCompletionResponse {
	baseMessage := chatCompletionResponse{
		Id:                qwenResponse.RequestId,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetContext(ctxKeyFinalRequestModel).(string),
		SystemFingerprint: "",
		Object:            objectChatCompletionChunk,
	}
	roleResponse := *&baseMessage
	deltaResponse := *&baseMessage
	finishResponse := *&baseMessage
	for _, qwenChoice := range qwenResponse.Output.Choices {
		message := qwenChoice.Message
		roleResponse.Choices = append(roleResponse.Choices, chatCompletionChoice{Delta: &chatMessage{Role: message.Role}})
		deltaResponse.Choices = append(deltaResponse.Choices, chatCompletionChoice{Delta: &chatMessage{Content: message.Content}})
		finishResponse.Choices = append(finishResponse.Choices, chatCompletionChoice{FinishReason: finishReasonStop})
	}
	return []*chatCompletionResponse{
		&roleResponse,
		&deltaResponse,
		&finishResponse,
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
