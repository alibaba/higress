package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/google/uuid"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// geminiProvider is the provider for google gemini/gemini flash service.

const (
	geminiApiKeyHeader = "x-goog-api-key"
	geminiDomain       = "generativelanguage.googleapis.com"
)

type geminiProviderInitializer struct {
}

func (g *geminiProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (g *geminiProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &geminiProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type geminiProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (g *geminiProvider) GetProviderType() string {
	return providerTypeGemini
}

func (g *geminiProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion && apiName != ApiNameEmbeddings {
		return types.ActionContinue, errUnsupportedApiName
	}

	_ = proxywasm.ReplaceHttpRequestHeader(geminiApiKeyHeader, g.config.GetRandomToken())
	_ = util.OverwriteRequestHost(geminiDomain)

	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return types.HeaderStopIteration, nil
}

func (g *geminiProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName == ApiNameChatCompletion {
		return g.onChatCompletionRequestBody(ctx, body, log)
	} else if apiName == ApiNameEmbeddings {
		return g.onEmbeddingsRequestBody(ctx, body, log)
	}
	return types.ActionContinue, errUnsupportedApiName
}

func (g *geminiProvider) onChatCompletionRequestBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
	// 使用gemini接口协议
	if g.config.protocol == protocolOriginal {
		request := &geminiChatRequest{}
		if err := json.Unmarshal(body, request); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
		}
		if request.Model == "" {
			return types.ActionContinue, errors.New("request model is empty")
		}
		// 根据模型重写requestPath
		path := g.getRequestPath(ApiNameChatCompletion, request.Model, request.Stream)
		_ = util.OverwriteRequestPath(path)

		// 移除多余的model和stream字段
		request = &geminiChatRequest{
			Contents:         request.Contents,
			SafetySettings:   request.SafetySettings,
			GenerationConfig: request.GenerationConfig,
			Tools:            request.Tools,
		}
		if g.config.context == nil {
			return types.ActionContinue, replaceJsonRequestBody(request, log)
		}

		err := g.contextCache.GetContent(func(content string, err error) {
			defer func() {
				_ = proxywasm.ResumeHttpRequest()
			}()

			if err != nil {
				log.Errorf("failed to load context file: %v", err)
				_ = util.SendResponse(500, "ai-proxy.gemini.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
			}
			g.setSystemContent(request, content)
			if err := replaceJsonRequestBody(request, log); err != nil {
				_ = util.SendResponse(500, "ai-proxy.gemini.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
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

	// 映射模型重写requestPath
	model := request.Model
	if model == "" {
		return types.ActionContinue, errors.New("missing model in chat completion request")
	}
	ctx.SetContext(ctxKeyOriginalRequestModel, model)
	mappedModel := getMappedModel(model, g.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel
	ctx.SetContext(ctxKeyFinalRequestModel, request.Model)
	path := g.getRequestPath(ApiNameChatCompletion, mappedModel, request.Stream)
	_ = util.OverwriteRequestPath(path)

	if g.config.context == nil {
		geminiRequest := g.buildGeminiChatRequest(request)
		return types.ActionContinue, replaceJsonRequestBody(geminiRequest, log)
	}

	err := g.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.gemini.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		geminiRequest := g.buildGeminiChatRequest(request)
		if err := replaceJsonRequestBody(geminiRequest, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.gemini.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (g *geminiProvider) onEmbeddingsRequestBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
	// 使用gemini接口协议
	if g.config.protocol == protocolOriginal {
		request := &geminiBatchEmbeddingRequest{}
		if err := json.Unmarshal(body, request); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
		}
		if request.Model == "" {
			return types.ActionContinue, errors.New("request model is empty")
		}
		// 根据模型重写requestPath
		path := g.getRequestPath(ApiNameEmbeddings, request.Model, false)
		_ = util.OverwriteRequestPath(path)

		// 移除多余的model字段
		request = &geminiBatchEmbeddingRequest{
			Requests: request.Requests,
		}
		return types.ActionContinue, replaceJsonRequestBody(request, log)
	}
	request := &embeddingsRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
	}

	// 映射模型重写requestPath
	model := request.Model
	if model == "" {
		return types.ActionContinue, errors.New("missing model in embeddings request")
	}
	ctx.SetContext(ctxKeyOriginalRequestModel, model)
	mappedModel := getMappedModel(model, g.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel
	ctx.SetContext(ctxKeyFinalRequestModel, request.Model)
	path := g.getRequestPath(ApiNameEmbeddings, mappedModel, false)
	_ = util.OverwriteRequestPath(path)

	geminiRequest := g.buildBatchEmbeddingRequest(request)
	return types.ActionContinue, replaceJsonRequestBody(geminiRequest, log)
}

func (g *geminiProvider) OnResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if g.config.protocol == protocolOriginal {
		ctx.DontReadResponseBody()
		return types.ActionContinue, nil
	}

	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	return types.ActionContinue, nil
}

func (g *geminiProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	log.Infof("chunk body:%s", string(chunk))
	if isLastChunk || len(chunk) == 0 {
		return nil, nil
	}
	// sample end event response:
	// data: {"candidates": [{"content": {"parts": [{"text": "我是 Gemini，一个大型多模态模型，由 Google 训练。我的职责是尽我所能帮助您，并尽力提供全面且信息丰富的答复。"}],"role": "model"},"finishReason": "STOP","index": 0,"safetyRatings": [{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT","probability": "NEGLIGIBLE"},{"category": "HARM_CATEGORY_HATE_SPEECH","probability": "NEGLIGIBLE"},{"category": "HARM_CATEGORY_HARASSMENT","probability": "NEGLIGIBLE"},{"category": "HARM_CATEGORY_DANGEROUS_CONTENT","probability": "NEGLIGIBLE"}]}],"usageMetadata": {"promptTokenCount": 2,"candidatesTokenCount": 35,"totalTokenCount": 37}}
	responseBuilder := &strings.Builder{}
	lines := strings.Split(string(chunk), "\n")
	for _, data := range lines {
		if len(data) < 6 {
			// ignore blank line or wrong format
			continue
		}
		data = data[6:]
		var geminiResp geminiChatResponse
		if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
			log.Errorf("unable to unmarshal gemini response: %v", err)
			continue
		}
		response := g.buildChatCompletionStreamResponse(ctx, &geminiResp)
		responseBody, err := json.Marshal(response)
		if err != nil {
			log.Errorf("unable to marshal response: %v", err)
			return nil, err
		}
		g.appendResponse(responseBuilder, string(responseBody))
	}
	modifiedResponseChunk := responseBuilder.String()
	log.Debugf("=== modified response chunk: %s", modifiedResponseChunk)
	return []byte(modifiedResponseChunk), nil
}

func (g *geminiProvider) OnResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName == ApiNameChatCompletion {
		return g.onChatCompletionResponseBody(ctx, body, log)
	} else if apiName == ApiNameEmbeddings {
		return g.onEmbeddingsResponseBody(ctx, body, log)
	}
	return types.ActionContinue, errUnsupportedApiName
}

func (g *geminiProvider) onChatCompletionResponseBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
	geminiResponse := &geminiChatResponse{}
	if err := json.Unmarshal(body, geminiResponse); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal gemini chat response: %v", err)
	}
	if geminiResponse.Error != nil {
		return types.ActionContinue, fmt.Errorf("gemini chat completion response error, error_code: %d, error_status:%s, error_message: %s",
			geminiResponse.Error.Code, geminiResponse.Error.Status, geminiResponse.Error.Message)
	}
	response := g.buildChatCompletionResponse(ctx, geminiResponse)
	return types.ActionContinue, replaceJsonResponseBody(response, log)
}

func (g *geminiProvider) onEmbeddingsResponseBody(ctx wrapper.HttpContext, body []byte, log wrapper.Log) (types.Action, error) {
	geminiResponse := &geminiEmbeddingResponse{}
	if err := json.Unmarshal(body, geminiResponse); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal gemini embeddings response: %v", err)
	}
	if geminiResponse.Error != nil {
		return types.ActionContinue, fmt.Errorf("gemini embeddings response error, error_code: %d, error_status:%s, error_message: %s",
			geminiResponse.Error.Code, geminiResponse.Error.Status, geminiResponse.Error.Message)
	}
	response := g.buildEmbeddingsResponse(ctx, geminiResponse)
	return types.ActionContinue, replaceJsonResponseBody(response, log)
}

func (g *geminiProvider) getRequestPath(apiName ApiName, geminiModel string, stream bool) string {
	action := ""
	if apiName == ApiNameEmbeddings {
		action = "batchEmbedContents"
	} else if stream {
		action = "streamGenerateContent?alt=sse"
	} else {
		action = "generateContent"
	}
	return fmt.Sprintf("/v1/models/%s:%s", geminiModel, action)
}

type geminiChatRequest struct {
	// Model and Stream are only used when using the gemini original protocol
	Model            string                     `json:"model,omitempty"`
	Stream           bool                       `json:"stream,omitempty"`
	Contents         []geminiChatContent        `json:"contents"`
	SafetySettings   []geminiChatSafetySetting  `json:"safety_settings,omitempty"`
	GenerationConfig geminiChatGenerationConfig `json:"generation_config,omitempty"`
	Tools            []geminiChatTools          `json:"tools,omitempty"`
}

type geminiChatContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiChatSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type geminiChatGenerationConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	TopK            float64  `json:"topK,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	CandidateCount  int      `json:"candidateCount,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiChatTools struct {
	FunctionDeclarations any `json:"function_declarations,omitempty"`
}

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	InlineData   *geminiInlineData   `json:"inlineData,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiFunctionCall struct {
	FunctionName string `json:"name"`
	Arguments    any    `json:"args"`
}

func (g *geminiProvider) buildGeminiChatRequest(request *chatCompletionRequest) *geminiChatRequest {
	var safetySettings []geminiChatSafetySetting
	{
	}
	for category, threshold := range g.config.geminiSafetySetting {
		safetySettings = append(safetySettings, geminiChatSafetySetting{
			Category:  category,
			Threshold: threshold,
		})
	}
	geminiRequest := geminiChatRequest{
		Contents:       make([]geminiChatContent, 0, len(request.Messages)),
		SafetySettings: safetySettings,
		GenerationConfig: geminiChatGenerationConfig{
			Temperature:     request.Temperature,
			TopP:            request.TopP,
			MaxOutputTokens: request.MaxTokens,
		},
	}
	if request.Tools != nil {
		functions := make([]function, 0, len(request.Tools))
		for _, tool := range request.Tools {
			functions = append(functions, tool.Function)
		}
		geminiRequest.Tools = []geminiChatTools{
			{
				FunctionDeclarations: functions,
			},
		}
	}
	shouldAddDummyModelMessage := false
	for _, message := range request.Messages {
		content := geminiChatContent{
			Role: message.Role,
			Parts: []geminiPart{
				{
					Text: message.StringContent(),
				},
			},
		}

		// there's no assistant role in gemini and API shall vomit if role is not user or model
		if content.Role == roleAssistant {
			content.Role = "model"
		} else if content.Role == roleSystem { // converting system prompt to prompt from user for the same reason
			content.Role = roleUser
			shouldAddDummyModelMessage = true
		}
		geminiRequest.Contents = append(geminiRequest.Contents, content)

		// if a system message is the last message, we need to add a dummy model message to make gemini happy
		if shouldAddDummyModelMessage {
			geminiRequest.Contents = append(geminiRequest.Contents, geminiChatContent{
				Role: "model",
				Parts: []geminiPart{
					{
						Text: "Okay",
					},
				},
			})
			shouldAddDummyModelMessage = false
		}
	}

	return &geminiRequest
}

func (g *geminiProvider) setSystemContent(request *geminiChatRequest, content string) {
	systemContents := []geminiChatContent{{
		Role: roleUser,
		Parts: []geminiPart{
			{
				Text: content,
			},
		},
	}}
	request.Contents = append(systemContents, request.Contents...)
}

type geminiBatchEmbeddingRequest struct {
	// Model are only used when using the gemini original protocol
	Model    string                   `json:"model,omitempty"`
	Requests []geminiEmbeddingRequest `json:"requests"`
}

type geminiEmbeddingRequest struct {
	Model                string            `json:"model"`
	Content              geminiChatContent `json:"content"`
	TaskType             string            `json:"taskType,omitempty"`
	Title                string            `json:"title,omitempty"`
	OutputDimensionality int               `json:"outputDimensionality,omitempty"`
}

func (g *geminiProvider) buildBatchEmbeddingRequest(request *embeddingsRequest) *geminiBatchEmbeddingRequest {
	inputs := request.ParseInput()
	requests := make([]geminiEmbeddingRequest, len(inputs))
	model := fmt.Sprintf("models/%s", request.Model)

	for i, input := range inputs {
		requests[i] = geminiEmbeddingRequest{
			Model: model,
			Content: geminiChatContent{
				Parts: []geminiPart{
					{
						Text: input,
					},
				},
			},
		}
	}

	return &geminiBatchEmbeddingRequest{
		Requests: requests,
	}
}

type geminiChatResponse struct {
	Candidates     []geminiChatCandidate    `json:"candidates"`
	PromptFeedback geminiChatPromptFeedback `json:"promptFeedback"`
	UsageMetadata  geminiUsageMetadata      `json:"usageMetadata"`
	Error          *geminiResponseError     `json:"error,omitempty"`
}

type geminiChatCandidate struct {
	Content       geminiChatContent        `json:"content"`
	FinishReason  string                   `json:"finishReason"`
	Index         int64                    `json:"index"`
	SafetyRatings []geminiChatSafetyRating `json:"safetyRatings"`
}

type geminiChatPromptFeedback struct {
	SafetyRatings []geminiChatSafetyRating `json:"safetyRatings"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}

type geminiResponseError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}

type geminiChatSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

func (g *geminiProvider) buildChatCompletionResponse(ctx wrapper.HttpContext, response *geminiChatResponse) *chatCompletionResponse {
	fullTextResponse := chatCompletionResponse{
		Id:      fmt.Sprintf("chatcmpl-%s", uuid.New().String()),
		Object:  objectChatCompletion,
		Created: time.Now().UnixMilli() / 1000,
		Model:   ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		Choices: make([]chatCompletionChoice, 0, len(response.Candidates)),
		Usage: usage{
			PromptTokens:     response.UsageMetadata.PromptTokenCount,
			CompletionTokens: response.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      response.UsageMetadata.TotalTokenCount,
		},
	}
	for i, candidate := range response.Candidates {
		choice := chatCompletionChoice{
			Index: i,
			Message: &chatMessage{
				Role: roleAssistant,
			},
			FinishReason: finishReasonStop,
		}
		if len(candidate.Content.Parts) > 0 {
			if candidate.Content.Parts[0].FunctionCall != nil {
				choice.Message.ToolCalls = g.buildToolCalls(&candidate)
			} else {
				choice.Message.Content = candidate.Content.Parts[0].Text
			}
		} else {
			choice.Message.Content = ""
			choice.FinishReason = candidate.FinishReason
		}
		fullTextResponse.Choices = append(fullTextResponse.Choices, choice)
	}
	return &fullTextResponse
}

func (g *geminiProvider) buildToolCalls(candidate *geminiChatCandidate) []toolCall {
	var toolCalls []toolCall

	item := candidate.Content.Parts[0]
	if item.FunctionCall != nil {
		return toolCalls
	}
	argsBytes, err := json.Marshal(item.FunctionCall.Arguments)
	if err != nil {
		proxywasm.LogErrorf("get toolCalls from gemini response failed: " + err.Error())
		return toolCalls
	}
	toolCall := toolCall{
		Id:   fmt.Sprintf("call_%s", uuid.New().String()),
		Type: "function",
		Function: functionCall{
			Arguments: string(argsBytes),
			Name:      item.FunctionCall.FunctionName,
		},
	}
	toolCalls = append(toolCalls, toolCall)
	return toolCalls
}

func (g *geminiProvider) buildChatCompletionStreamResponse(ctx wrapper.HttpContext, geminiResp *geminiChatResponse) *chatCompletionResponse {
	var choice chatCompletionChoice
	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		choice.Delta = &chatMessage{Content: geminiResp.Candidates[0].Content.Parts[0].Text}
	}
	streamResponse := chatCompletionResponse{
		Id:      fmt.Sprintf("chatcmpl-%s", uuid.New().String()),
		Object:  objectChatCompletionChunk,
		Created: time.Now().UnixMilli() / 1000,
		Model:   ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		Choices: []chatCompletionChoice{choice},
		Usage: usage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
	}
	return &streamResponse
}

type geminiEmbeddingResponse struct {
	Embeddings []geminiEmbeddingData `json:"embeddings"`
	Error      *geminiResponseError  `json:"error,omitempty"`
}

type geminiEmbeddingData struct {
	Values []float64 `json:"values"`
}

func (g *geminiProvider) buildEmbeddingsResponse(ctx wrapper.HttpContext, geminiResp *geminiEmbeddingResponse) *embeddingsResponse {
	response := embeddingsResponse{
		Object: "list",
		Data:   make([]embedding, 0, len(geminiResp.Embeddings)),
		Model:  ctx.GetContext(ctxKeyFinalRequestModel).(string),
		Usage: usage{
			TotalTokens: 0,
		},
	}
	for _, item := range geminiResp.Embeddings {
		response.Data = append(response.Data, embedding{
			Object:    `embedding`,
			Index:     0,
			Embedding: item.Values,
		})
	}
	return &response
}

func (g *geminiProvider) appendResponse(responseBuilder *strings.Builder, responseBody string) {
	responseBuilder.WriteString(fmt.Sprintf("%s %s\n\n", streamDataItemKey, responseBody))
}
