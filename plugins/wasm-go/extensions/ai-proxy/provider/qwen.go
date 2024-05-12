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

	streamDataKeyword    = "data: "
	streamDataEndKeyword = "[DONE]"
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

func (m *qwenProvider) GetProviderType() string {
	return providerTypeQwen
}

func (m *qwenProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(qwenChatCompletionPath)
	_ = util.OverwriteRequestHost(qwenDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	_ = proxywasm.ReplaceHttpRequestHeader("Accept", "text/event-stream")
	_ = proxywasm.ReplaceHttpRequestHeader("X-DashScope-SSE", "enable")
	return types.ActionContinue, nil

	// Delay the header processing to allow changing streaming mode in OnRequestBody
	//return types.HeaderStopIteration, nil
}

func (m *qwenProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
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

	//if request.Stream {
	//	_ = proxywasm.ReplaceHttpRequestHeader("Accept", "text/event-stream")
	//	_ = proxywasm.ReplaceHttpRequestHeader("X-DashScope-SSE", "enable")
	//} else {
	//	_ = proxywasm.ReplaceHttpRequestHeader("Accept", "*/*")
	//	_ = proxywasm.RemoveHttpRequestHeader("X-DashScope-SSE")
	//}

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

func (m *qwenProvider) OnResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	return types.ActionContinue, nil
}

func (m *qwenProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	bufferedStreamingBody := ctx.GetContext(ctxKeyStreamingBody).([]byte)
	receivedBody := append(bufferedStreamingBody, chunk...)

	lineStartIndex := -1
	skipCurrentLine := false
	var responseBuilder strings.Builder
	for i := 0; i < len(receivedBody); i++ {
		ch := receivedBody[i]
		if ch != '\n' {
			if lineStartIndex == -1 {
				lineStartIndex = i
				skipCurrentLine = false
			}
			if ch == ':' {
				if lineStartIndex == -1 {
					// Leading colon. Skip the line.
					skipCurrentLine = true
				} else if string(receivedBody[lineStartIndex:i]) != streamDataKeyword {
					// Not a data line. Skip it.
					skipCurrentLine = true
				}
			}
			continue
		}

		if lineStartIndex == -1 {
			// Leading newline. Skip.
			continue
		}
		if skipCurrentLine {
			lineStartIndex = -1
			skipCurrentLine = false
			continue
		}

		data := receivedBody[lineStartIndex+len(streamDataKeyword) : i]
		log.Debugf("=== event data: %s", data)

		if string(data) == streamDataEndKeyword {
			responseBuilder.WriteString(streamDataKeyword)
			responseBuilder.WriteString(streamDataEndKeyword)
			responseBuilder.WriteString("\n\n")
			continue
		}

		qwenResponse := &qwenTextGenResponse{}
		if err := json.Unmarshal(data, qwenResponse); err != nil {
			log.Errorf("unable to unmarshal Qwen response: %v", err)
			return nil, fmt.Errorf("unable to unmarshal Qwen response: %v", err)
		}

		responses := m.buildChatCompletionStreamingResponse(ctx, qwenResponse)
		for _, response := range responses {
			responseBody, err := json.Marshal(response)
			if err != nil {
				log.Errorf("unable to marshal response: %v", err)
				return nil, fmt.Errorf("unable to marshal response: %v", err)
			}
			responseBuilder.WriteString(streamDataKeyword)
			responseBuilder.Write(responseBody)
			responseBuilder.WriteString("\n\n")
		}
	}
	modifiedResponseBody := responseBuilder.String()
	log.Debugf("=== response data: %s", modifiedResponseBody)
	return []byte(modifiedResponseBody), nil
}

func (m *qwenProvider) OnResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	qwenResponse := &qwenTextGenResponse{}
	if err := json.Unmarshal(body, qwenResponse); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal Qwen response: %v", err)
	}
	response := m.buildChatCompletionResponse(ctx, qwenResponse)
	return types.ActionContinue, replaceJsonResponseBody(response, log)
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
