package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
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
	qwenTextEmbeddingPath  = "/api/v1/services/embeddings/text-embedding/text-embedding"

	qwenTopPMin = 0.000001
	qwenTopPMax = 0.999999

	qwenDummySystemMessageContent = "You are a helpful assistant."

	qwenLongModelName = "qwen-long"
)

type qwenProviderInitializer struct {
}

func (m *qwenProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if len(config.qwenFileIds) != 0 && config.context != nil {
		return errors.New("qwenFileIds and context cannot be configured at the same time")
	}
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
	needRequestBody := false
	if apiName == ApiNameChatCompletion {
		_ = util.OverwriteRequestPath(qwenChatCompletionPath)
		needRequestBody = m.config.context != nil
	} else if apiName == ApiNameEmbeddings {
		_ = util.OverwriteRequestPath(qwenTextEmbeddingPath)
	} else {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestHost(qwenDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())

	if m.config.protocol == protocolOriginal && !needRequestBody {
		ctx.DontReadRequestBody()
		return types.ActionContinue, nil
	}

	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return types.HeaderStopIteration, nil
}

func (m *qwenProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName == ApiNameChatCompletion {
		return m.onChatCompletionRequestBody(ctx, body, log)
	}
	if apiName == ApiNameEmbeddings {
		return m.onEmbeddingsRequestBody(ctx, body, log)
	}
	return types.ActionContinue, errUnsupportedApiName
}

func (m *qwenProvider) onChatCompletionRequestBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
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
				_ = util.SendResponse(500, "ai-proxy.qwen.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
			}
			m.insertContextMessage(request, content, false)
			if err := replaceJsonRequestBody(request, log); err != nil {
				_ = util.SendResponse(500, "ai-proxy.qwen.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
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

	streaming := request.Stream
	if streaming {
		_ = proxywasm.ReplaceHttpRequestHeader("Accept", "text/event-stream")
		_ = proxywasm.ReplaceHttpRequestHeader("X-DashScope-SSE", "enable")
	} else {
		_ = proxywasm.ReplaceHttpRequestHeader("Accept", "*/*")
		_ = proxywasm.RemoveHttpRequestHeader("X-DashScope-SSE")
	}

	if m.config.context == nil {
		qwenRequest := m.buildQwenTextGenerationRequest(request, streaming)
		if streaming {
			ctx.SetContext(ctxKeyIncrementalStreaming, qwenRequest.Parameters.IncrementalOutput)
		}
		return types.ActionContinue, replaceJsonRequestBody(qwenRequest, log)
	}

	err := m.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.qwen.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		qwenRequest := m.buildQwenTextGenerationRequest(request, streaming)
		if streaming {
			ctx.SetContext(ctxKeyIncrementalStreaming, qwenRequest.Parameters.IncrementalOutput)
		}
		if err := replaceJsonRequestBody(qwenRequest, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.qwen.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (m *qwenProvider) onEmbeddingsRequestBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
	request := &embeddingsRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
	}

	log.Debugf("=== embeddings request: %v", request)

	model := request.Model
	if model == "" {
		return types.ActionContinue, errors.New("missing model in the request")
	}
	ctx.SetContext(ctxKeyOriginalRequestModel, model)
	mappedModel := getMappedModel(model, m.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel
	ctx.SetContext(ctxKeyFinalRequestModel, request.Model)

	if qwenRequest, err := m.buildQwenTextEmbeddingRequest(request); err == nil {
		return types.ActionContinue, replaceJsonRequestBody(qwenRequest, log)
	} else {
		return types.ActionContinue, err
	}
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
	if name != ApiNameChatCompletion {
		return chunk, nil
	}

	receivedBody := chunk
	if bufferedStreamingBody, has := ctx.GetContext(ctxKeyStreamingBody).([]byte); has {
		receivedBody = append(bufferedStreamingBody, chunk...)
	}

	incrementalStreaming, err := ctx.GetContext(ctxKeyIncrementalStreaming).(bool)
	if !err {
		incrementalStreaming = false
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

	// Sample Qwen event response:
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
			currentEvent.setValue(currentKey, value)
		} else {
			// Extra new line. The current event is complete.
			log.Debugf("processing event: %v", currentEvent)
			if err := m.convertStreamEvent(ctx, &responseBuilder, currentEvent, incrementalStreaming, log); err != nil {
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
	if apiName == ApiNameChatCompletion {
		return m.onChatCompletionResponseBody(ctx, body, log)
	}
	if apiName == ApiNameEmbeddings {
		return m.onEmbeddingsResponseBody(ctx, body, log)
	}
	return types.ActionContinue, errUnsupportedApiName
}

func (m *qwenProvider) onChatCompletionResponseBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
	qwenResponse := &qwenTextGenResponse{}
	if err := json.Unmarshal(body, qwenResponse); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal Qwen response: %v", err)
	}
	response := m.buildChatCompletionResponse(ctx, qwenResponse)
	return types.ActionContinue, replaceJsonResponseBody(response, log)
}

func (m *qwenProvider) onEmbeddingsResponseBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
	qwenResponse := &qwenTextEmbeddingResponse{}
	if err := json.Unmarshal(body, qwenResponse); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal Qwen response: %v", err)
	}
	response := m.buildEmbeddingsResponse(ctx, qwenResponse)
	return types.ActionContinue, replaceJsonResponseBody(response, log)
}

func (m *qwenProvider) buildQwenTextGenerationRequest(origRequest *chatCompletionRequest, streaming bool) *qwenTextGenRequest {
	messages := make([]qwenMessage, 0, len(origRequest.Messages))
	for i := range origRequest.Messages {
		messages = append(messages, chatMessage2QwenMessage(origRequest.Messages[i]))
	}
	request := &qwenTextGenRequest{
		Model: origRequest.Model,
		Input: qwenTextGenInput{
			Messages: messages,
		},
		Parameters: qwenTextGenParameters{
			ResultFormat:      qwenResultFormatMessage,
			MaxTokens:         origRequest.MaxTokens,
			N:                 origRequest.N,
			Seed:              origRequest.Seed,
			Temperature:       origRequest.Temperature,
			TopP:              math.Max(qwenTopPMin, math.Min(origRequest.TopP, qwenTopPMax)),
			IncrementalOutput: streaming && (origRequest.Tools == nil || len(origRequest.Tools) == 0),
			EnableSearch:      m.config.qwenEnableSearch,
			Tools:             origRequest.Tools,
		},
	}
	if len(m.config.qwenFileIds) != 0 && origRequest.Model == qwenLongModelName {
		builder := strings.Builder{}
		for _, fileId := range m.config.qwenFileIds {
			if builder.Len() != 0 {
				builder.WriteRune(',')
			}
			builder.WriteString("fileid://")
			builder.WriteString(fileId)
		}
		contextMessageId := m.insertContextMessage(request, builder.String(), true)
		if contextMessageId == 0 {
			// The context message cannot come first. We need to add another dummy system message before it.
			request.Input.Messages = append([]qwenMessage{{Role: roleSystem, Content: qwenDummySystemMessageContent}}, request.Input.Messages...)
		}
	}
	return request
}

func (m *qwenProvider) buildChatCompletionResponse(ctx wrapper.HttpContext, qwenResponse *qwenTextGenResponse) *chatCompletionResponse {
	choices := make([]chatCompletionChoice, 0, len(qwenResponse.Output.Choices))
	for _, qwenChoice := range qwenResponse.Output.Choices {
		message := qwenMessageToChatMessage(qwenChoice.Message)
		choices = append(choices, chatCompletionChoice{
			Message:      &message,
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
		Usage: usage{
			PromptTokens:     qwenResponse.Usage.InputTokens,
			CompletionTokens: qwenResponse.Usage.OutputTokens,
			TotalTokens:      qwenResponse.Usage.TotalTokens,
		},
	}
}

func (m *qwenProvider) buildChatCompletionStreamingResponse(ctx wrapper.HttpContext, qwenResponse *qwenTextGenResponse, incrementalStreaming bool, log wrapper.Log) []*chatCompletionResponse {
	baseMessage := chatCompletionResponse{
		Id:                qwenResponse.RequestId,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetContext(ctxKeyFinalRequestModel).(string),
		Choices:           make([]chatCompletionChoice, 0),
		SystemFingerprint: "",
		Object:            objectChatCompletionChunk,
	}

	responses := make([]*chatCompletionResponse, 0)

	qwenChoice := qwenResponse.Output.Choices[0]
	// Yes, Qwen uses a string "null" as null.
	finished := qwenChoice.FinishReason != "" && qwenChoice.FinishReason != "null"
	message := qwenChoice.Message

	deltaContentMessage := &chatMessage{Role: message.Role, Content: message.Content}
	deltaToolCallsMessage := &chatMessage{Role: message.Role, ToolCalls: append([]toolCall{}, message.ToolCalls...)}
	if !incrementalStreaming {
		for _, tc := range message.ToolCalls {
			if tc.Function.Arguments == "" && !finished {
				// We don't push any tool call until its arguments are available.
				return nil
			}
		}
		if pushedMessage, ok := ctx.GetContext(ctxKeyPushedMessage).(qwenMessage); ok {
			if message.Content == "" {
				message.Content = pushedMessage.Content
			}
			deltaContentMessage.Content = util.StripPrefix(deltaContentMessage.Content, pushedMessage.Content)
			if len(deltaToolCallsMessage.ToolCalls) > 0 && pushedMessage.ToolCalls != nil {
				for i, tc := range deltaToolCallsMessage.ToolCalls {
					if i >= len(pushedMessage.ToolCalls) {
						break
					}
					pushedFunction := pushedMessage.ToolCalls[i].Function
					tc.Function.Id = util.StripPrefix(tc.Function.Id, pushedFunction.Id)
					tc.Function.Name = util.StripPrefix(tc.Function.Name, pushedFunction.Name)
					tc.Function.Arguments = util.StripPrefix(tc.Function.Arguments, pushedFunction.Arguments)
					deltaToolCallsMessage.ToolCalls[i] = tc
				}
			}
		}
		ctx.SetContext(ctxKeyPushedMessage, message)
	}

	if !deltaContentMessage.IsEmpty() {
		response := *&baseMessage
		response.Choices = append(response.Choices, chatCompletionChoice{Delta: deltaContentMessage})
		responses = append(responses, &response)
	}
	if !deltaToolCallsMessage.IsEmpty() {
		response := *&baseMessage
		response.Choices = append(response.Choices, chatCompletionChoice{Delta: deltaToolCallsMessage})
		responses = append(responses, &response)
	}

	if finished {
		finishResponse := *&baseMessage
		finishResponse.Choices = append(finishResponse.Choices, chatCompletionChoice{Delta: &chatMessage{}, FinishReason: qwenChoice.FinishReason})

		usageResponse := *&baseMessage
		usageResponse.Choices = []chatCompletionChoice{{Delta: &chatMessage{}}}
		usageResponse.Usage = usage{
			PromptTokens:     qwenResponse.Usage.InputTokens,
			CompletionTokens: qwenResponse.Usage.OutputTokens,
			TotalTokens:      qwenResponse.Usage.TotalTokens,
		}

		responses = append(responses, &finishResponse, &usageResponse)
	}

	return responses
}

func (m *qwenProvider) convertStreamEvent(ctx wrapper.HttpContext, responseBuilder *strings.Builder, event *streamEvent, incrementalStreaming bool, log wrapper.Log) error {
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

	responses := m.buildChatCompletionStreamingResponse(ctx, qwenResponse, incrementalStreaming, log)
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

func (m *qwenProvider) insertContextMessage(request *qwenTextGenRequest, content string, onlyOneSystemBeforeFile bool) int {
	fileMessage := qwenMessage{
		Role:    roleSystem,
		Content: content,
	}
	var firstNonSystemMessageIndex int
	messages := request.Input.Messages
	if messages != nil {
		for i, message := range request.Input.Messages {
			if message.Role != roleSystem {
				firstNonSystemMessageIndex = i
				break
			}
		}
	}
	if firstNonSystemMessageIndex == 0 {
		request.Input.Messages = append([]qwenMessage{fileMessage}, request.Input.Messages...)
		return 0
	} else if !onlyOneSystemBeforeFile {
		request.Input.Messages = append(request.Input.Messages[:firstNonSystemMessageIndex], append([]qwenMessage{fileMessage}, request.Input.Messages[firstNonSystemMessageIndex:]...)...)
		return firstNonSystemMessageIndex
	} else {
		builder := strings.Builder{}
		for _, message := range request.Input.Messages[:firstNonSystemMessageIndex] {
			if builder.Len() != 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(message.Content)
		}
		request.Input.Messages = append([]qwenMessage{{Role: roleSystem, Content: builder.String()}, fileMessage}, request.Input.Messages[firstNonSystemMessageIndex:]...)
		return 1
	}
}

func (m *qwenProvider) appendStreamEvent(responseBuilder *strings.Builder, event *streamEvent) {
	responseBuilder.WriteString(streamDataItemKey)
	responseBuilder.WriteString(event.Data)
	responseBuilder.WriteString("\n\n")
}

func (m *qwenProvider) buildQwenTextEmbeddingRequest(request *embeddingsRequest) (*qwenTextEmbeddingRequest, error) {
	var texts []string
	if str, isString := request.Input.(string); isString {
		texts = []string{str}
	} else if strs, isArray := request.Input.([]interface{}); isArray {
		texts = make([]string, 0, len(strs))
		for _, item := range strs {
			if str, isString := item.(string); isString {
				texts = append(texts, str)
			} else {
				return nil, errors.New("unsupported input type in array: " + reflect.TypeOf(item).String())
			}
		}
	} else {
		return nil, errors.New("unsupported input type: " + reflect.TypeOf(request.Input).String())
	}
	return &qwenTextEmbeddingRequest{
		Model: request.Model,
		Input: qwenTextEmbeddingInput{
			Texts: texts,
		},
	}, nil
}

func (m *qwenProvider) buildEmbeddingsResponse(ctx wrapper.HttpContext, qwenResponse *qwenTextEmbeddingResponse) *embeddingsResponse {
	data := make([]embedding, 0, len(qwenResponse.Output.Embeddings))
	for _, qwenEmbedding := range qwenResponse.Output.Embeddings {
		data = append(data, embedding{
			Object:    "embedding",
			Index:     qwenEmbedding.TextIndex,
			Embedding: qwenEmbedding.Embedding,
		})
	}
	return &embeddingsResponse{
		Object: "list",
		Data:   data,
		Model:  ctx.GetContext(ctxKeyFinalRequestModel).(string),
		Usage: usage{
			PromptTokens: qwenResponse.Usage.TotalTokens,
			TotalTokens:  qwenResponse.Usage.TotalTokens,
		},
	}
}

type qwenTextGenRequest struct {
	Model      string                `json:"model"`
	Input      qwenTextGenInput      `json:"input"`
	Parameters qwenTextGenParameters `json:"parameters,omitempty"`
}

type qwenTextGenInput struct {
	Messages []qwenMessage `json:"messages"`
}

type qwenTextGenParameters struct {
	ResultFormat      string  `json:"result_format,omitempty"`
	MaxTokens         int     `json:"max_tokens,omitempty"`
	RepetitionPenalty float64 `json:"repetition_penalty,omitempty"`
	N                 int     `json:"n,omitempty"`
	Seed              int     `json:"seed,omitempty"`
	Temperature       float64 `json:"temperature,omitempty"`
	TopP              float64 `json:"top_p,omitempty"`
	IncrementalOutput bool    `json:"incremental_output,omitempty"`
	EnableSearch      bool    `json:"enable_search,omitempty"`
	Tools             []tool  `json:"tools,omitempty"`
}

type qwenTextGenResponse struct {
	RequestId string            `json:"request_id"`
	Output    qwenTextGenOutput `json:"output"`
	Usage     qwenUsage         `json:"usage"`
}

type qwenTextGenOutput struct {
	FinishReason string              `json:"finish_reason"`
	Choices      []qwenTextGenChoice `json:"choices"`
}

type qwenTextGenChoice struct {
	FinishReason string      `json:"finish_reason"`
	Message      qwenMessage `json:"message"`
}

type qwenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type qwenMessage struct {
	Name      string     `json:"name,omitempty"`
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

type qwenTextEmbeddingRequest struct {
	Model      string                      `json:"model"`
	Input      qwenTextEmbeddingInput      `json:"input"`
	Parameters qwenTextEmbeddingParameters `json:"parameters,omitempty"`
}

type qwenTextEmbeddingInput struct {
	Texts []string `json:"texts"`
}

type qwenTextEmbeddingParameters struct {
	TextType string `json:"text_type,omitempty"`
}

type qwenTextEmbeddingResponse struct {
	RequestId string                  `json:"request_id"`
	Output    qwenTextEmbeddingOutput `json:"output"`
	Usage     qwenUsage               `json:"usage"`
}

type qwenTextEmbeddingOutput struct {
	RequestId  string               `json:"request_id"`
	Embeddings []qwenTextEmbeddings `json:"embeddings"`
}

type qwenTextEmbeddings struct {
	TextIndex int       `json:"text_index"`
	Embedding []float64 `json:"embedding"`
}

func qwenMessageToChatMessage(qwenMessage qwenMessage) chatMessage {
	return chatMessage{
		Name:      qwenMessage.Name,
		Role:      qwenMessage.Role,
		Content:   qwenMessage.Content,
		ToolCalls: qwenMessage.ToolCalls,
	}
}

func chatMessage2QwenMessage(chatMessage chatMessage) qwenMessage {
	return qwenMessage{
		Name:      chatMessage.Name,
		Role:      chatMessage.Role,
		Content:   chatMessage.Content,
		ToolCalls: chatMessage.ToolCalls,
	}
}
