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

func (m *qwenProvider) GetProviderType() string {
	return providerTypeQwen
}

const (
	forceStreaming = true
)

func (m *qwenProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(qwenChatCompletionPath)
	_ = util.OverwriteRequestHost(qwenDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())

	if m.config.protocol == protocolOriginal && m.config.context == nil {
		ctx.DontReadRequestBody()
		return types.ActionContinue, nil
	}

	if forceStreaming {
		_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
		_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

		_ = proxywasm.ReplaceHttpRequestHeader("Accept", "text/event-stream")
		_ = proxywasm.ReplaceHttpRequestHeader("X-DashScope-SSE", "enable")
		return types.ActionContinue, nil
	} else {
		// Delay the header processing to allow changing streaming mode in OnRequestBody
		return types.HeaderStopIteration, nil
	}
}

func (m *qwenProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}

	if m.config.protocol == protocolOriginal {
		if m.config.context == nil {
			return types.ActionContinue, nil
		}

		request := &qwenTextGenRequest{}
		if err := json.Unmarshal(body, request); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
		}

		err := m.contextCache.GetContent(func(content string, err error) {
			defer func() {
				_ = proxywasm.ResumeHttpRequest()
			}()

			if err != nil {
				log.Errorf("failed to load context file: %v", err)
				_ = util.SendResponse(500, util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
			}
			m.insertContextMessage(request, content)
			if err := replaceJsonRequestBody(request, log); err != nil {
				_ = util.SendResponse(500, util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
			}
		}, log)
		if err == nil {
			return types.ActionPause, nil
		}
		return types.ActionContinue, err
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

	if !forceStreaming {
		if request.Stream {
			_ = proxywasm.ReplaceHttpRequestHeader("Accept", "text/event-stream")
			_ = proxywasm.ReplaceHttpRequestHeader("X-DashScope-SSE", "enable")
		} else {
			_ = proxywasm.ReplaceHttpRequestHeader("Accept", "*/*")
			_ = proxywasm.RemoveHttpRequestHeader("X-DashScope-SSE")
		}
	}

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
	if m.config.protocol == protocolOriginal {
		ctx.DontReadResponseBody()
		return types.ActionContinue, nil
	}

	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	return types.ActionContinue, nil
}

func (m *qwenProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	receivedBody := chunk
	if bufferedStreamingBody, has := ctx.GetContext(ctxKeyStreamingBody).([]byte); has {
		receivedBody = append(bufferedStreamingBody, chunk...)
	}

	eventStartIndex, lineStartIndex, valueStartIndex := -1, -1, -1

	defer func() {
		if eventStartIndex >= 0 && eventStartIndex < len(receivedBody) {
			// Just in case the received chunk is not a complete event.
			ctx.SetContext(ctxKeyStreamingBody, receivedBody[eventStartIndex:])
		} else {
			ctx.SetContext(ctxKeyStreamingBody, nil)
		}
	}()

	// Sample event response:
	//
	// event:result
	// :HTTP_STATUS/200
	// data:{"output":{"choices":[{"message":{"content":"你好！","role":"assistant"},"finish_reason":"null"}]},"usage":{"total_tokens":116,"input_tokens":114,"output_tokens":2},"request_id":"71689cfc-1f42-9949-86e8-9563b7f832b1"}
	//
	// event:error
	// :HTTP_STATUS/400
	// data:{"code":"InvalidParameter","message":"Preprocessor error","request_id":"0cbe6006-faec-9854-bf8b-c906d75c3bd8"}
	//

	var responseBuilder strings.Builder
	currentKey := ""
	currentEvent := &streamEvent{}
	i, length := 0, len(receivedBody)
	for i = 0; i < length; i++ {
		ch := receivedBody[i]
		if ch != '\n' {
			if lineStartIndex == -1 {
				if eventStartIndex == -1 {
					eventStartIndex = i
				}
				lineStartIndex = i
				valueStartIndex = -1
			}
			if valueStartIndex == -1 {
				if ch == ':' {
					valueStartIndex = i + 1
					currentKey = string(receivedBody[lineStartIndex:valueStartIndex])
				}
			} else if valueStartIndex == i && ch == ' ' {
				// Skip leading spaces in data.
				valueStartIndex = i + 1
			}
			continue
		}

		if lineStartIndex != -1 {
			value := string(receivedBody[valueStartIndex:i])
			log.Debugf("key: %s value: %s", currentKey, value)
			currentEvent.setValue(currentKey, value)
		} else {
			// Extra new line. The current event is complete.
			log.Debugf("processing event: %v", currentEvent)
			if err := m.convertStreamEvent(ctx, &responseBuilder, currentEvent, log); err != nil {
				return nil, err
			}
			// Reset event parsing state.
			eventStartIndex = -1
			currentEvent = &streamEvent{}
		}

		// Reset line parsing state.
		lineStartIndex = -1
		valueStartIndex = -1
		currentKey = ""
	}

	modifiedResponseChunk := responseBuilder.String()
	log.Debugf("=== modified response chunk: %s", modifiedResponseChunk)
	return []byte(modifiedResponseChunk), nil
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

	responses := make([]*chatCompletionResponse, 0)

	qwenChoice := qwenResponse.Output.Choices[0]
	message := qwenChoice.Message

	content := message.Content
	if rawPushedContent := ctx.GetContext(ctxKeyPushedMessageContent); rawPushedContent != nil {
		if pushedContent := rawPushedContent.(string); pushedContent != "" && strings.HasPrefix(content, pushedContent) {
			content = content[len(pushedContent):]
		}
	}
	if content != "" {
		deltaResponse := *&baseMessage
		deltaResponse.Choices = append(deltaResponse.Choices, chatCompletionChoice{Delta: &chatMessage{Role: message.Role, Content: content}})
		responses = append(responses, &deltaResponse)
		ctx.SetContext(ctxKeyPushedMessageContent, message.Content)
	}

	// Yes, Qwen uses a string "null" as null.
	if qwenChoice.FinishReason != "" && qwenChoice.FinishReason != "null" {
		finishResponse := *&baseMessage
		finishResponse.Choices = append(finishResponse.Choices, chatCompletionChoice{FinishReason: qwenChoice.FinishReason})
		responses = append(responses, &finishResponse)
	}

	return responses
}

func (m *qwenProvider) convertStreamEvent(ctx wrapper.HttpContext, responseBuilder *strings.Builder, event *streamEvent, log wrapper.Log) error {
	if event.Data == streamEndDataValue {
		m.appendStreamEvent(responseBuilder, event)
		return nil
	}

	if event.Event != eventResult || event.HttpStatus != httpStatus200 {
		// Something goes wrong. Just pass through the event.
		m.appendStreamEvent(responseBuilder, event)
		return nil
	}

	qwenResponse := &qwenTextGenResponse{}
	if err := json.Unmarshal([]byte(event.Data), qwenResponse); err != nil {
		log.Errorf("unable to unmarshal Qwen response: %v", err)
		return fmt.Errorf("unable to unmarshal Qwen response: %v", err)
	}

	responses := m.buildChatCompletionStreamingResponse(ctx, qwenResponse)
	for _, response := range responses {
		responseBody, err := json.Marshal(response)
		if err != nil {
			log.Errorf("unable to marshal response: %v", err)
			return fmt.Errorf("unable to marshal response: %v", err)
		}
		modifiedEvent := &*event
		modifiedEvent.Data = string(responseBody)
		m.appendStreamEvent(responseBuilder, modifiedEvent)
	}

	return nil
}

func (m *qwenProvider) insertContextMessage(request *qwenTextGenRequest, content string) {
	fileMessage := chatMessage{
		Role:    roleSystem,
		Content: content,
	}
	firstNonSystemMessageIndex := -1
	messages := request.Input.Messages
	if messages != nil {
		for i, message := range request.Input.Messages {
			if message.Role != roleSystem {
				firstNonSystemMessageIndex = i
				break
			}
		}
	}
	if firstNonSystemMessageIndex == -1 {
		request.Input.Messages = append(request.Input.Messages, fileMessage)
	} else {
		request.Input.Messages = append(request.Input.Messages[:firstNonSystemMessageIndex], append([]chatMessage{fileMessage}, request.Input.Messages[firstNonSystemMessageIndex:]...)...)
	}
}

func (m *qwenProvider) appendStreamEvent(responseBuilder *strings.Builder, event *streamEvent) {
	responseBuilder.WriteString(streamEventIdItemKey)
	responseBuilder.WriteString(event.Id)
	responseBuilder.WriteString("\n")
	responseBuilder.WriteString(streamEventNameItemKey)
	responseBuilder.WriteString(event.Event)
	responseBuilder.WriteString("\n")
	responseBuilder.WriteString(streamBuiltInItemKey)
	responseBuilder.WriteString(streamHttpStatusValuePrefix)
	responseBuilder.WriteString(event.HttpStatus)
	responseBuilder.WriteString("\n")
	responseBuilder.WriteString(streamDataItemKey)
	responseBuilder.WriteString(event.Data)
	responseBuilder.WriteString("\n\n")
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
