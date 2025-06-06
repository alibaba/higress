package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/google/uuid"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// geminiProvider is the provider for google gemini/gemini flash service.

const (
	geminiApiKeyHeader             = "x-goog-api-key"
	geminiApiVersion               = "v1beta" // 可选: v1 ,v1beta
	geminiDomain                   = "generativelanguage.googleapis.com"
	geminiChatCompletionPath       = "generateContent"
	geminiChatCompletionStreamPath = "streamGenerateContent?alt=sse"
	geminiEmbeddingPath            = "batchEmbedContents"
	geminiModelsPath               = "models"
	geminiImageGenerationPath      = "predict"
)

type geminiProviderInitializer struct{}

func (g *geminiProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (g *geminiProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion):  "",
		string(ApiNameEmbeddings):      "",
		string(ApiNameModels):          "",
		string(ApiNameImageGeneration): "",
	}
}

func (g *geminiProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(g.DefaultCapabilities())
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

func (g *geminiProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	g.config.handleRequestHeaders(g, ctx, apiName)
	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return nil
}

func (g *geminiProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestHostHeader(headers, geminiDomain)
	headers.Set(geminiApiKeyHeader, g.config.GetApiTokenInUse(ctx))
	util.OverwriteRequestAuthorizationHeader(headers, "")
}

func (g *geminiProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !g.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return g.config.handleRequestBody(g, g.contextCache, ctx, apiName, body)
}

func (g *geminiProvider) TransformRequestBodyHeaders(ctx wrapper.HttpContext, apiName ApiName, body []byte, headers http.Header) ([]byte, error) {
	switch apiName {
	case ApiNameChatCompletion:
		return g.onChatCompletionRequestBody(ctx, body, headers)
	case ApiNameEmbeddings:
		return g.onEmbeddingsRequestBody(ctx, body, headers)
	case ApiNameImageGeneration:
		return g.onImageGenerationRequestBody(ctx, body, headers)
	}
	return body, nil
}

func (g *geminiProvider) onImageGenerationRequestBody(ctx wrapper.HttpContext, body []byte, headers http.Header) ([]byte, error) {
	request := &imageGenerationRequest{}
	if err := g.config.parseRequestAndMapModel(ctx, request, body); err != nil {
		return nil, err
	}
	path := g.getRequestPath(ApiNameImageGeneration, request.Model, false)
	log.Debugf("request path:%s", path)
	util.OverwriteRequestPathHeader(headers, path)
	geminiRequest := g.buildGeminiImageGenerationRequest(request)
	return json.Marshal(geminiRequest)
}

func (g *geminiProvider) buildGeminiImageGenerationRequest(request *imageGenerationRequest) *geminiImageGenerationRequest {
	geminiRequest := &geminiImageGenerationRequest{
		Instances: []geminiImageGenerationInstance{{Prompt: request.Prompt}},
		Parameters: &geminiImageGenerationParameters{
			SampleCount: request.N,
		},
	}

	return geminiRequest
}

func (g *geminiProvider) onChatCompletionRequestBody(ctx wrapper.HttpContext, body []byte, headers http.Header) ([]byte, error) {
	request := &chatCompletionRequest{}
	err := g.config.parseRequestAndMapModel(ctx, request, body)
	if err != nil {
		return nil, err
	}
	path := g.getRequestPath(ApiNameChatCompletion, request.Model, request.Stream)
	util.OverwriteRequestPathHeader(headers, path)

	geminiRequest := g.buildGeminiChatRequest(request)
	return json.Marshal(geminiRequest)
}

func (g *geminiProvider) onEmbeddingsRequestBody(ctx wrapper.HttpContext, body []byte, headers http.Header) ([]byte, error) {
	request := &embeddingsRequest{}
	if err := g.config.parseRequestAndMapModel(ctx, request, body); err != nil {
		return nil, err
	}
	path := g.getRequestPath(ApiNameEmbeddings, request.Model, false)
	util.OverwriteRequestPathHeader(headers, path)

	geminiRequest := g.buildBatchEmbeddingRequest(request)
	return json.Marshal(geminiRequest)
}

func (g *geminiProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool) ([]byte, error) {
	log.Debugf("chunk body:%s", string(chunk))
	if isLastChunk || len(chunk) == 0 {
		return nil, nil
	}
	if name != ApiNameChatCompletion {
		return chunk, nil
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

func (g *geminiProvider) TransformResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	switch apiName {
	case ApiNameChatCompletion:
		return g.onChatCompletionResponseBody(ctx, body)
	case ApiNameEmbeddings:
		return g.onEmbeddingsResponseBody(ctx, body)
	case ApiNameImageGeneration:
		return g.onImageGenerationResponseBody(ctx, body)
	default:
		return body, nil
	}
}

func (g *geminiProvider) onImageGenerationResponseBody(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	geminiResponse := &geminiImageGenerationResponse{}
	if err := json.Unmarshal(body, geminiResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal gemini image generation response: %v", err)
	}
	response := g.buildImageGenerationResponse(ctx, geminiResponse)
	return json.Marshal(response)
}

func (g *geminiProvider) buildImageGenerationResponse(ctx wrapper.HttpContext, geminiResponse *geminiImageGenerationResponse) *imageGenerationResponse {
	data := make([]imageGenerationData, len(geminiResponse.Predictions))
	for i, prediction := range geminiResponse.Predictions {
		data[i] = imageGenerationData{
			B64: prediction.BytesBase64Encoded,
		}
	}
	response := &imageGenerationResponse{
		Created: time.Now().UnixMilli() / 1000,
		Data:    data,
	}
	return response
}

func (g *geminiProvider) onChatCompletionResponseBody(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	log.Debugf("chat completion response body:%s", string(body))
	geminiResponse := &geminiChatResponse{}
	if err := json.Unmarshal(body, geminiResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal gemini chat response: %v", err)
	}
	if geminiResponse.Error != nil {
		return nil, fmt.Errorf("gemini chat completion response error, error_code: %d, error_status:%s, error_message: %s",
			geminiResponse.Error.Code, geminiResponse.Error.Status, geminiResponse.Error.Message)
	}
	response := g.buildChatCompletionResponse(ctx, geminiResponse)
	return json.Marshal(response)
}

func (g *geminiProvider) onEmbeddingsResponseBody(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	geminiResponse := &geminiEmbeddingResponse{}
	if err := json.Unmarshal(body, geminiResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal gemini embeddings response: %v", err)
	}
	if geminiResponse.Error != nil {
		return nil, fmt.Errorf("gemini embeddings response error, error_code: %d, error_status:%s, error_message: %s",
			geminiResponse.Error.Code, geminiResponse.Error.Status, geminiResponse.Error.Message)
	}
	response := g.buildEmbeddingsResponse(ctx, geminiResponse)
	return json.Marshal(response)
}

func (g *geminiProvider) getRequestPath(apiName ApiName, model string, stream bool) string {
	action := ""
	switch apiName {
	case ApiNameModels:
		return fmt.Sprintf("/%s/%s", geminiApiVersion, geminiModelsPath)
	case ApiNameEmbeddings:
		action = geminiEmbeddingPath
	case ApiNameChatCompletion:
		if stream {
			action = geminiChatCompletionStreamPath
		} else {
			action = geminiChatCompletionPath
		}
	case ApiNameImageGeneration:
		action = geminiImageGenerationPath
	}
	return fmt.Sprintf("/%s/models/%s:%s", geminiApiVersion, model, action)
}

type geminiGenerationContentRequest struct {
	// Model and Stream are only used when using the gemini original protocol
	Model            string                     `json:"model,omitempty"`
	Stream           bool                       `json:"stream,omitempty"`
	Contents         []geminiChatContent        `json:"contents"`
	SafetySettings   []geminiChatSafetySetting  `json:"safetySettings,omitempty"`
	GenerationConfig geminiChatGenerationConfig `json:"generationConfig,omitempty"`
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

type geminiThinkingConfig struct {
	IncludeThoughts bool  `json:"includeThoughts,omitempty"`
	ThinkingBudget  int64 `json:"thinkingBudget,omitempty"`
}

type geminiChatGenerationConfig struct {
	Temperature        float64               `json:"temperature,omitempty"`
	TopP               float64               `json:"topP,omitempty"`
	TopK               int64                 `json:"topK,omitempty"`
	Seed               int64                 `json:"seed,omitempty"`
	Logprobs           bool                  `json:"logprobs,omitempty"`
	MaxOutputTokens    int                   `json:"maxOutputTokens,omitempty"`
	CandidateCount     int                   `json:"candidateCount,omitempty"`
	StopSequences      []string              `json:"stopSequences,omitempty"`
	PresencePenalty    int64                 `json:"presencePenalty,omitempty"`
	FrequencyPenalty   int64                 `json:"frequencyPenalty,omitempty"`
	ResponseModalities []string              `json:"responseModalities,omitempty"`
	NegativePrompt     string                `json:"negativePrompt,omitempty"`
	ThinkingConfig     *geminiThinkingConfig `json:"thinkingConfig,omitempty"`
	MediaResolution    string                `json:"mediaResolution,omitempty"`
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

// geminiImageGenerationRequest is the request body for generate image using Imagen 3
type geminiImageGenerationRequest struct {
	Instances  []geminiImageGenerationInstance  `json:"instances"`
	Parameters *geminiImageGenerationParameters `json:"parameters,omitempty"`
}

type geminiImageGenerationInstance struct {
	Prompt string `json:"prompt"`
}

type geminiImageGenerationParameters struct {
	SampleCount int    `json:"sampleCount,omitempty"`
	AspectRatio string `json:"aspectRatio,omitempty"`
}

type geminiImageGenerationPrediction struct {
	BytesBase64Encoded string `json:"bytesBase64Encoded"`
	MimeType           string `json:"mimeType"`
}

type geminiImageGenerationResponse struct {
	Predictions []geminiImageGenerationPrediction `json:"predictions"`
}

func (g *geminiProvider) buildGeminiChatRequest(request *chatCompletionRequest) *geminiGenerationContentRequest {
	var safetySettings []geminiChatSafetySetting
	for category, threshold := range g.config.geminiSafetySetting {
		safetySettings = append(safetySettings, geminiChatSafetySetting{
			Category:  category,
			Threshold: threshold,
		})
	}
	geminiRequest := geminiGenerationContentRequest{
		Contents:       make([]geminiChatContent, 0, len(request.Messages)),
		SafetySettings: safetySettings,
		GenerationConfig: geminiChatGenerationConfig{
			Temperature:        request.Temperature,
			TopP:               request.TopP,
			MaxOutputTokens:    request.MaxTokens,
			PresencePenalty:    int64(request.PresencePenalty),
			FrequencyPenalty:   int64(request.FrequencyPenalty),
			Logprobs:           request.Logprobs,
			ResponseModalities: request.Modalities,
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

func (g *geminiProvider) setSystemContent(request *geminiGenerationContentRequest, content string) {
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
		Usage: usage{
			PromptTokens:     response.UsageMetadata.PromptTokenCount,
			CompletionTokens: response.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      response.UsageMetadata.TotalTokenCount,
		},
	}
	choiceIndex := 0
	for _, candidate := range response.Candidates {
		for _, part := range candidate.Content.Parts {
			choice := chatCompletionChoice{
				Index: choiceIndex,
				Message: &chatMessage{
					Role: roleAssistant,
				},
				FinishReason: finishReasonStop,
			}
			if part.FunctionCall != nil {
				choice.Message.ToolCalls = g.buildToolCalls(&candidate)
			} else if part.InlineData != nil {
				choice.Message.Content = part.InlineData.Data
			} else {
				choice.Message.Content = part.Text
			}

			choice.FinishReason = candidate.FinishReason
			fullTextResponse.Choices = append(fullTextResponse.Choices, choice)
			choiceIndex += 1
		}
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
		log.Errorf("get toolCalls from gemini response failed: " + err.Error())
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

func (g *geminiProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, geminiChatCompletionPath) || strings.Contains(path, geminiChatCompletionStreamPath) {
		return ApiNameChatCompletion
	}
	if strings.Contains(path, geminiEmbeddingPath) {
		return ApiNameEmbeddings
	}
	if strings.Contains(path, geminiImageGenerationPath) {
		return ApiNameImageGeneration
	}
	return ""
}
