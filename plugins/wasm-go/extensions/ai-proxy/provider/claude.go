package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"strings"
	"time"
)

// claudeProvider is the provider for Claude service.
const (
	claudeDomain             = "api.anthropic.com"
	claudeChatCompletionPath = "/v1/messages"
	defaultVersion           = "2023-06-01"
	defaultMaxTokens         = 4096
)

type claudeProviderInitializer struct{}

type claudeTextGenRequest struct {
	Model         string        `json:"model"`
	Messages      []chatMessage `json:"messages"`
	System        string        `json:"system,omitempty"`
	MaxTokens     int           `json:"max_tokens,omitempty"`
	StopSequences []string      `json:"stop_sequences,omitempty"`
	Stream        bool          `json:"stream,omitempty"`
	Temperature   float64       `json:"temperature,omitempty"`
	TopP          float64       `json:"top_p,omitempty"`
	TopK          int           `json:"top_k,omitempty"`
}

type claudeTextGenResponse struct {
	Id           string                 `json:"id"`
	Type         string                 `json:"type"`
	Role         string                 `json:"role"`
	Content      []claudeTextGenContent `json:"content"`
	Model        string                 `json:"model"`
	StopReason   *string                `json:"stop_reason"`
	StopSequence *string                `json:"stop_sequence"`
	Usage        claudeTextGenUsage     `json:"usage"`
	Error        *claudeTextGenError    `json:"error"`
}

type claudeTextGenContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type claudeTextGenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeTextGenError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type claudeTextGenStreamResponse struct {
	Type         string                `json:"type"`
	Message      claudeTextGenResponse `json:"message"`
	Index        int                   `json:"index"`
	ContentBlock *claudeTextGenContent `json:"content_block"`
	Delta        *claudeTextGenDelta   `json:"delta"`
	Usage        claudeTextGenUsage    `json:"usage"`
}

type claudeTextGenDelta struct {
	Type         string  `json:"type"`
	Text         string  `json:"text"`
	StopReason   *string `json:"stop_reason"`
	StopSequence *string `json:"stop_sequence"`
}

func (c *claudeProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (c *claudeProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &claudeProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type claudeProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (c *claudeProvider) GetProviderType() string {
	return providerTypeClaude
}

func (c *claudeProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}

	_ = util.OverwriteRequestPath(claudeChatCompletionPath)
	_ = util.OverwriteRequestHost(claudeDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("x-api-key", c.config.GetRandomToken())

	if c.config.claudeVersion == "" {
		c.config.claudeVersion = defaultVersion
	}
	_ = proxywasm.AddHttpRequestHeader("anthropic-version", c.config.claudeVersion)
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	return types.ActionContinue, nil
}

func (c *claudeProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}

	// use original protocol
	if c.config.protocol == protocolOriginal {
		if c.config.context == nil {
			return types.ActionContinue, nil
		}

		request := &claudeTextGenRequest{}
		if err := json.Unmarshal(body, request); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
		}

		err := c.contextCache.GetContent(func(content string, err error) {
			defer func() {
				_ = proxywasm.ResumeHttpRequest()
			}()

			if err != nil {
				log.Errorf("failed to load context file: %v", err)
				_ = util.SendResponse(500, "ai-proxy.claude.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
			}
			if err := replaceJsonRequestBody(request, log); err != nil {
				_ = util.SendResponse(500, "ai-proxy.claude.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
			}
		}, log)
		if err == nil {
			return types.ActionPause, nil
		}
		return types.ActionContinue, err
	}

	// use openai protocol
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}

	model := request.Model
	if model == "" {
		return types.ActionContinue, errors.New("missing model in chat completion request")
	}
	ctx.SetContext(ctxKeyOriginalRequestModel, model)
	mappedModel := getMappedModel(model, c.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel
	ctx.SetContext(ctxKeyFinalRequestModel, request.Model)

	streaming := request.Stream
	if streaming {
		_ = proxywasm.ReplaceHttpRequestHeader("Accept", "text/event-stream")
	}

	if c.config.context == nil {
		claudeRequest := c.buildClaudeTextGenRequest(request)
		return types.ActionContinue, replaceJsonRequestBody(claudeRequest, log)
	}

	err := c.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.claude.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		claudeRequest := c.buildClaudeTextGenRequest(request)
		if err := replaceJsonRequestBody(claudeRequest, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.claude.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (c *claudeProvider) OnResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	claudeResponse := &claudeTextGenResponse{}
	if err := json.Unmarshal(body, claudeResponse); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal claude response: %v", err)
	}
	if claudeResponse.Error != nil {
		return types.ActionContinue, fmt.Errorf("claude response error, error_type: %s, error_message: %s", claudeResponse.Error.Type, claudeResponse.Error.Message)
	}
	response := c.responseClaude2OpenAI(ctx, claudeResponse)
	return types.ActionContinue, replaceJsonResponseBody(response, log)
}

func (c *claudeProvider) OnResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	// use original protocol, skip OnStreamingResponseBody() and OnResponseBody()
	if c.config.protocol == protocolOriginal {
		ctx.DontReadResponseBody()
		return types.ActionContinue, nil
	}

	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	return types.ActionContinue, nil
}

func (c *claudeProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	if isLastChunk || len(chunk) == 0 {
		return nil, nil
	}

	responseBuilder := &strings.Builder{}
	lines := strings.Split(string(chunk), "\n")
	for _, data := range lines {
		// only process the line starting with "data:"
		if strings.HasPrefix(data, "data:") {
			// extract json data from the line
			jsonData := strings.TrimPrefix(data, "data:")
			var claudeResponse claudeTextGenStreamResponse
			if err := json.Unmarshal([]byte(jsonData), &claudeResponse); err != nil {
				log.Errorf("unable to unmarshal claude response: %v", err)
				continue
			}
			response := c.streamResponseClaude2OpenAI(ctx, &claudeResponse, log)
			if response != nil {
				responseBody, err := json.Marshal(response)
				if err != nil {
					log.Errorf("unable to marshal response: %v", err)
					return nil, err
				}
				c.appendResponse(responseBuilder, string(responseBody))
			}
		}
	}
	modifiedResponseChunk := responseBuilder.String()
	log.Debugf("modified response chunk: %s", modifiedResponseChunk)
	return []byte(modifiedResponseChunk), nil
}

func (c *claudeProvider) buildClaudeTextGenRequest(origRequest *chatCompletionRequest) *claudeTextGenRequest {
	claudeRequest := claudeTextGenRequest{
		Model:         origRequest.Model,
		MaxTokens:     origRequest.MaxTokens,
		StopSequences: origRequest.Stop,
		Stream:        origRequest.Stream,
		Temperature:   origRequest.Temperature,
		TopP:          origRequest.TopP,
	}
	if claudeRequest.MaxTokens == 0 {
		claudeRequest.MaxTokens = defaultMaxTokens
	}

	for _, message := range origRequest.Messages {
		if message.Role == roleSystem {
			claudeRequest.System = message.Content
			continue
		}
		claudeMessage := chatMessage{
			Role:    message.Role,
			Content: message.Content,
		}
		claudeRequest.Messages = append(claudeRequest.Messages, claudeMessage)
	}
	return &claudeRequest
}

func (c *claudeProvider) responseClaude2OpenAI(ctx wrapper.HttpContext, origResponse *claudeTextGenResponse) *chatCompletionResponse {
	choice := chatCompletionChoice{
		Index:        0,
		Message:      &chatMessage{Role: roleAssistant, Content: origResponse.Content[0].Text},
		FinishReason: stopReasonClaude2OpenAI(origResponse.StopReason),
	}

	return &chatCompletionResponse{
		Id:                origResponse.Id,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetContext(ctxKeyFinalRequestModel).(string),
		SystemFingerprint: "",
		Object:            objectChatCompletion,
		Choices:           []chatCompletionChoice{choice},
		Usage: usage{
			PromptTokens:     origResponse.Usage.InputTokens,
			CompletionTokens: origResponse.Usage.OutputTokens,
			TotalTokens:      origResponse.Usage.InputTokens + origResponse.Usage.OutputTokens,
		},
	}
}

func stopReasonClaude2OpenAI(reason *string) string {
	if reason == nil {
		return ""
	}
	switch *reason {
	case "end_turn":
		return finishReasonStop
	case "stop_sequence":
		return finishReasonStop
	case "max_tokens":
		return finishReasonLength
	default:
		return *reason
	}
}

func (c *claudeProvider) streamResponseClaude2OpenAI(ctx wrapper.HttpContext, origResponse *claudeTextGenStreamResponse, log wrapper.Log) *chatCompletionResponse {
	switch origResponse.Type {
	case "message_start":
		choice := chatCompletionChoice{
			Index: 0,
			Delta: &chatMessage{Role: roleAssistant, Content: ""},
		}
		return createChatCompletionResponse(ctx, origResponse, choice)

	case "content_block_delta":
		choice := chatCompletionChoice{
			Index: 0,
			Delta: &chatMessage{Content: origResponse.Delta.Text},
		}
		return createChatCompletionResponse(ctx, origResponse, choice)

	case "message_delta":
		choice := chatCompletionChoice{
			Index:        0,
			Delta:        &chatMessage{},
			FinishReason: stopReasonClaude2OpenAI(origResponse.Delta.StopReason),
		}
		return createChatCompletionResponse(ctx, origResponse, choice)
	case "content_block_stop", "message_stop":
		log.Debugf("skip processing response type: %s", origResponse.Type)
		return nil
	default:
		log.Errorf("Unexpected response type: %s", origResponse.Type)
		return nil
	}
}

func createChatCompletionResponse(ctx wrapper.HttpContext, response *claudeTextGenStreamResponse, choice chatCompletionChoice) *chatCompletionResponse {
	return &chatCompletionResponse{
		Id:      response.Message.Id,
		Created: time.Now().UnixMilli() / 1000,
		Model:   ctx.GetContext(ctxKeyFinalRequestModel).(string),
		Object:  objectChatCompletionChunk,
		Choices: []chatCompletionChoice{choice},
	}
}

func (c *claudeProvider) appendResponse(responseBuilder *strings.Builder, responseBody string) {
	responseBuilder.WriteString(fmt.Sprintf("%s %s\n\n", streamDataItemKey, responseBody))
}
