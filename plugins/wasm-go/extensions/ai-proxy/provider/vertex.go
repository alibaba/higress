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
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	vertexDomain = "{REGION}-aiplatform.googleapis.com"
	// /v1/projects/{PROJECT_ID}/locations/{REGION}/publishers/google/models/{MODEL_ID}:{ACTION}
	vertexPathTemplate             = "/v1/projects/%s/locations/%s/publishers/google/models/%s:%s"
	vertexChatCompletionPath       = "generateContent"
	vertexChatCompletionStreamPath = "streamGenerateContent?alt=sse"
	vertexEmbeddingPath            = "predict"
)

type vertexProviderInitializer struct {
}

func (v *vertexProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in vertex provider config")
	}
	if config.vertexRegion == "" || config.vertexProjectId == "" {
		return errors.New("missing vertexRegion or vertexProjectId in vertex provider config")
	}
	return nil
}

func (v *vertexProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): vertexPathTemplate,
		string(ApiNameEmbeddings):     vertexPathTemplate,
	}
}

func (v *vertexProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(v.DefaultCapabilities())
	return &vertexProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type vertexProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (v *vertexProvider) GetProviderType() string {
	return providerTypeVertex
}

func (v *vertexProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, vertexChatCompletionPath) || strings.Contains(path, vertexChatCompletionStreamPath) {
		return ApiNameChatCompletion
	}
	if strings.Contains(path, vertexEmbeddingPath) {
		return ApiNameEmbeddings
	}
	return ""
}

func (v *vertexProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	v.config.handleRequestHeaders(v, ctx, apiName)
	return nil
}

func (v *vertexProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	vertexRegionDomain := strings.Replace(vertexDomain, "{REGION}", v.config.vertexRegion, 1)
	util.OverwriteRequestHostHeader(headers, vertexRegionDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+v.config.GetApiTokenInUse(ctx))
}

func (v *vertexProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !v.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return v.config.handleRequestBody(v, v.contextCache, ctx, apiName, body)
}

func (v *vertexProvider) TransformRequestBodyHeaders(ctx wrapper.HttpContext, apiName ApiName, body []byte, headers http.Header) ([]byte, error) {
	if apiName == ApiNameChatCompletion {
		return v.onChatCompletionRequestBody(ctx, body, headers)
	} else {
		return v.onEmbeddingsRequestBody(ctx, body, headers)
	}
}

func (v *vertexProvider) onChatCompletionRequestBody(ctx wrapper.HttpContext, body []byte, headers http.Header) ([]byte, error) {
	request := &chatCompletionRequest{}
	err := v.config.parseRequestAndMapModel(ctx, request, body)
	if err != nil {
		return nil, err
	}
	path := v.getRequestPath(ApiNameChatCompletion, request.Model, request.Stream)
	util.OverwriteRequestPathHeader(headers, path)

	vertexRequest := v.buildVertexChatRequest(request)
	return json.Marshal(vertexRequest)
}

func (v *vertexProvider) onEmbeddingsRequestBody(ctx wrapper.HttpContext, body []byte, headers http.Header) ([]byte, error) {
	request := &embeddingsRequest{}
	if err := v.config.parseRequestAndMapModel(ctx, request, body); err != nil {
		return nil, err
	}
	path := v.getRequestPath(ApiNameEmbeddings, request.Model, false)
	util.OverwriteRequestPathHeader(headers, path)

	vertexRequest := v.buildEmbeddingRequest(request)
	return json.Marshal(vertexRequest)
}

func (v *vertexProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool) ([]byte, error) {
	log.Infof("[vertexProvider] receive chunk body:%s", string(chunk))
	if isLastChunk || len(chunk) == 0 {
		return nil, nil
	}
	if name != ApiNameChatCompletion {
		return chunk, nil
	}
	responseBuilder := &strings.Builder{}
	lines := strings.Split(string(chunk), "\n")
	for _, data := range lines {
		if len(data) < 6 {
			// ignore blank line or wrong format
			continue
		}
		data = data[6:]
		var vertexResp vertexChatResponse
		if err := json.Unmarshal([]byte(data), &vertexResp); err != nil {
			log.Errorf("unable to unmarshal vertex response: %v", err)
			continue
		}
		response := v.buildChatCompletionStreamResponse(ctx, &vertexResp)
		responseBody, err := json.Marshal(response)
		if err != nil {
			log.Errorf("unable to marshal response: %v", err)
			return nil, err
		}
		v.appendResponse(responseBuilder, string(responseBody))
	}
	modifiedResponseChunk := responseBuilder.String()
	log.Debugf("=== modified response chunk: %s", modifiedResponseChunk)
	return []byte(modifiedResponseChunk), nil
}

func (v *vertexProvider) TransformResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	if apiName == ApiNameChatCompletion {
		return v.onChatCompletionResponseBody(ctx, body)
	} else {
		return v.onEmbeddingsResponseBody(ctx, body)
	}
}

func (v *vertexProvider) onChatCompletionResponseBody(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	vertexResponse := &vertexChatResponse{}
	if err := json.Unmarshal(body, vertexResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal vertex chat response: %v", err)
	}
	response := v.buildChatCompletionResponse(ctx, vertexResponse)
	return json.Marshal(response)
}

func (v *vertexProvider) buildChatCompletionResponse(ctx wrapper.HttpContext, response *vertexChatResponse) *chatCompletionResponse {
	fullTextResponse := chatCompletionResponse{
		Id:      response.ResponseId,
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
	for _, candidate := range response.Candidates {
		choice := chatCompletionChoice{
			Index: candidate.Index,
			Message: &chatMessage{
				Role: roleAssistant,
			},
			FinishReason: candidate.FinishReason,
		}
		if len(candidate.Content.Parts) > 0 {
			choice.Message.Content = candidate.Content.Parts[0].Text
		} else {
			choice.Message.Content = ""
		}
		fullTextResponse.Choices = append(fullTextResponse.Choices, choice)
	}
	return &fullTextResponse
}

func (v *vertexProvider) onEmbeddingsResponseBody(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	vertexResponse := &vertexEmbeddingResponse{}
	if err := json.Unmarshal(body, vertexResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal vertex embeddings response: %v", err)
	}
	response := v.buildEmbeddingsResponse(ctx, vertexResponse)
	return json.Marshal(response)
}

func (v *vertexProvider) buildEmbeddingsResponse(ctx wrapper.HttpContext, vertexResp *vertexEmbeddingResponse) *embeddingsResponse {
	response := embeddingsResponse{
		Object: "list",
		Data:   make([]embedding, 0, len(vertexResp.Predictions)),
		Model:  ctx.GetContext(ctxKeyFinalRequestModel).(string),
	}
	totalTokens := 0
	for _, item := range vertexResp.Predictions {
		response.Data = append(response.Data, embedding{
			Object:    `embedding`,
			Index:     0,
			Embedding: item.Embeddings.Values,
		})
		if item.Embeddings.Statistics != nil {
			totalTokens += item.Embeddings.Statistics.TokenCount
		}
	}
	response.Usage.TotalTokens = totalTokens
	return &response
}

func (v *vertexProvider) buildChatCompletionStreamResponse(ctx wrapper.HttpContext, vertexResp *vertexChatResponse) *chatCompletionResponse {
	var choice chatCompletionChoice
	if len(vertexResp.Candidates) > 0 && len(vertexResp.Candidates[0].Content.Parts) > 0 {
		choice.Delta = &chatMessage{Content: vertexResp.Candidates[0].Content.Parts[0].Text}
	}
	streamResponse := chatCompletionResponse{
		Id:      vertexResp.ResponseId,
		Object:  objectChatCompletionChunk,
		Created: time.Now().UnixMilli() / 1000,
		Model:   ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		Choices: []chatCompletionChoice{choice},
		Usage: usage{
			PromptTokens:     vertexResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: vertexResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      vertexResp.UsageMetadata.TotalTokenCount,
		},
	}
	return &streamResponse
}

func (v *vertexProvider) appendResponse(responseBuilder *strings.Builder, responseBody string) {
	responseBuilder.WriteString(fmt.Sprintf("%s %s\n\n", streamDataItemKey, responseBody))
}

func (v *vertexProvider) getRequestPath(apiName ApiName, modelId string, stream bool) string {
	action := ""
	if apiName == ApiNameEmbeddings {
		action = vertexEmbeddingPath
	} else if stream {
		action = vertexChatCompletionStreamPath
	} else {
		action = vertexChatCompletionPath
	}
	return fmt.Sprintf(vertexPathTemplate, v.config.vertexProjectId, v.config.vertexRegion, modelId, action)
}

func (v *vertexProvider) buildVertexChatRequest(request *chatCompletionRequest) *vertexChatRequest {
	safetySettings := make([]vertexChatSafetySetting, 0)
	for category, threshold := range v.config.geminiSafetySetting {
		safetySettings = append(safetySettings, vertexChatSafetySetting{
			Category:  category,
			Threshold: threshold,
		})
	}
	vertexRequest := vertexChatRequest{
		Contents:       make([]vertexChatContent, 0),
		SafetySettings: safetySettings,
		GenerationConfig: vertexChatGenerationConfig{
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
		vertexRequest.Tools = []vertexTool{
			{
				FunctionDeclarations: functions,
			},
		}
	}
	shouldAddDummyModelMessage := false
	for _, message := range request.Messages {
		content := vertexChatContent{
			Role: message.Role,
			Parts: []vertexPart{
				{
					Text: message.StringContent(),
				},
			},
		}

		// there's no assistant role in vertex and API shall vomit if role is not user or model
		if content.Role == roleAssistant {
			content.Role = "model"
		} else if content.Role == roleSystem { // converting system prompt to prompt from user for the same reason
			content.Role = roleUser
			shouldAddDummyModelMessage = true
		}
		vertexRequest.Contents = append(vertexRequest.Contents, content)

		// if a system message is the last message, we need to add a dummy model message to make vertex happy
		if shouldAddDummyModelMessage {
			vertexRequest.Contents = append(vertexRequest.Contents, vertexChatContent{
				Role: "model",
				Parts: []vertexPart{
					{
						Text: "Okay",
					},
				},
			})
			shouldAddDummyModelMessage = false
		}
	}

	return &vertexRequest
}

func (v *vertexProvider) buildEmbeddingRequest(request *embeddingsRequest) *vertexEmbeddingRequest {
	inputs := request.ParseInput()
	instances := make([]vertexEmbeddingInstance, len(inputs))
	for i, input := range inputs {
		instances[i] = vertexEmbeddingInstance{
			Content: input,
		}
	}
	return &vertexEmbeddingRequest{Instances: instances}
}

type vertexChatRequest struct {
	CachedContent     string                     `json:"cachedContent,omitempty"`
	Contents          []vertexChatContent        `json:"contents"`
	SystemInstruction *vertexSystemInstruction   `json:"systemInstruction,omitempty"`
	Tools             []vertexTool               `json:"tools,omitempty"`
	SafetySettings    []vertexChatSafetySetting  `json:"safetySettings,omitempty"`
	GenerationConfig  vertexChatGenerationConfig `json:"generationConfig,omitempty"`
	Labels            map[string]string          `json:"labels,omitempty"`
}

type vertexChatContent struct {
	// The producer of the content. Must be either 'user' or 'model'.
	Role  string       `json:"role,omitempty"`
	Parts []vertexPart `json:"parts"`
}

type vertexPart struct {
	Text       string    `json:"text,omitempty"`
	InlineData *blob     `json:"inlineData,omitempty"`
	FileData   *fileData `json:"fileData,omitempty"`
}

type blob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type fileData struct {
	MimeType string `json:"mimeType"`
	FileUri  string `json:"fileUri"`
}

type vertexSystemInstruction struct {
	Role  string       `json:"role"`
	Parts []vertexPart `json:"parts"`
}

type vertexTool struct {
	FunctionDeclarations any `json:"functionDeclarations"`
}

type vertexChatSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type vertexChatGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	CandidateCount  int     `json:"candidateCount,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type vertexEmbeddingRequest struct {
	Instances  []vertexEmbeddingInstance `json:"instances"`
	Parameters *vertexEmbeddingParams    `json:"parameters,omitempty"`
}

type vertexEmbeddingInstance struct {
	TaskType string `json:"task_type"`
	Title    string `json:"title,omitempty"`
	Content  string `json:"content"`
}

type vertexEmbeddingParams struct {
	AutoTruncate bool `json:"autoTruncate,omitempty"`
}

type vertexChatResponse struct {
	Candidates     []vertexChatCandidate    `json:"candidates"`
	ResponseId     string                   `json:"responseId,omitempty"`
	PromptFeedback vertexChatPromptFeedback `json:"promptFeedback"`
	UsageMetadata  vertexUsageMetadata      `json:"usageMetadata"`
}

type vertexChatCandidate struct {
	Content       vertexChatContent        `json:"content"`
	FinishReason  string                   `json:"finishReason"`
	Index         int                      `json:"index"`
	SafetyRatings []vertexChatSafetyRating `json:"safetyRatings"`
}

type vertexChatSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type vertexChatPromptFeedback struct {
	SafetyRatings []vertexChatSafetyRating `json:"safetyRatings"`
}

type vertexUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}

type vertexEmbeddingResponse struct {
	Predictions []vertexPredictions `json:"predictions"`
}

type vertexPredictions struct {
	Embeddings struct {
		Values     []float64         `json:"values"`
		Statistics *vertexStatistics `json:"statistics,omitempty"`
	} `json:"embeddings"`
}

type vertexStatistics struct {
	TokenCount int  `json:"token_count"`
	Truncated  bool `json:"truncated"`
}
