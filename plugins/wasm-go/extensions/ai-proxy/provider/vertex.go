package provider

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	vertexAuthDomain = "oauth2.googleapis.com"
	vertexDomain     = "{REGION}-aiplatform.googleapis.com"
	// /v1/projects/{PROJECT_ID}/locations/{REGION}/publishers/google/models/{MODEL_ID}:{ACTION}
	vertexPathTemplate               = "/v1/projects/%s/locations/%s/publishers/google/models/%s:%s"
	vertexChatCompletionAction       = "generateContent"
	vertexChatCompletionStreamAction = "streamGenerateContent?alt=sse"
	vertexEmbeddingAction            = "predict"
	reasoningContextMarkerStart      = "<think>"
	reasoningContextMarkerEnd        = "</think>"
)

type vertexProviderInitializer struct{}

func (v *vertexProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.vertexAuthKey == "" {
		return errors.New("missing vertexAuthKey in vertex provider config")
	}
	if config.vertexRegion == "" || config.vertexProjectId == "" {
		return errors.New("missing vertexRegion or vertexProjectId in vertex provider config")
	}
	if config.vertexAuthServiceName == "" {
		return errors.New("missing vertexAuthServiceName in vertex provider config")
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
		config: config,
		client: wrapper.NewClusterClient(wrapper.DnsCluster{
			Domain:      vertexAuthDomain,
			ServiceName: config.vertexAuthServiceName,
			Port:        443,
		}),
		contextCache: createContextCache(&config),
	}, nil
}

type vertexProvider struct {
	client       wrapper.HttpClient
	config       ProviderConfig
	contextCache *contextCache
}

func (v *vertexProvider) GetProviderType() string {
	return providerTypeVertex
}

func (v *vertexProvider) GetApiName(path string) ApiName {
	if strings.HasSuffix(path, vertexChatCompletionAction) || strings.HasSuffix(path, vertexChatCompletionStreamAction) {
		return ApiNameChatCompletion
	}
	if strings.HasSuffix(path, vertexEmbeddingAction) {
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
}

func (v *vertexProvider) getToken() (cached bool, err error) {
	cacheKeyName := v.buildTokenKey()
	cachedAccessToken, err := v.getCachedAccessToken(cacheKeyName)
	if err == nil && cachedAccessToken != "" {
		_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+cachedAccessToken)
		return true, nil
	}

	var key ServiceAccountKey
	if err := json.Unmarshal([]byte(v.config.vertexAuthKey), &key); err != nil {
		return false, fmt.Errorf("[vertex]: unable to unmarshal auth key json: %v", err)
	}

	if key.ClientEmail == "" || key.PrivateKey == "" || key.TokenURI == "" {
		return false, fmt.Errorf("[vertex]: missing auth params")
	}

	jwtToken, err := createJWT(&key)
	if err != nil {
		log.Errorf("[vertex]: unable to create JWT token: %v", err)
		return false, err
	}

	err = v.getAccessToken(jwtToken)
	if err != nil {
		log.Errorf("[vertex]: unable to get access token: %v", err)
		return false, err
	}

	return false, err
}

func (v *vertexProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !v.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	if v.config.IsOriginal() {
		return types.ActionContinue, nil
	}
	headers := util.GetRequestHeaders()
	body, err := v.TransformRequestBodyHeaders(ctx, apiName, body, headers)
	util.ReplaceRequestHeaders(headers)
	_ = proxywasm.ReplaceHttpRequestBody(body)
	if err != nil {
		return types.ActionContinue, err
	}
	cached, err := v.getToken()
	if cached {
		return types.ActionContinue, nil
	}
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
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
	log.Infof("[vertexProvider] receive chunk body: %s", string(chunk))
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
		Usage: &usage{
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
			FinishReason: util.Ptr(candidate.FinishReason),
		}
		if len(candidate.Content.Parts) > 0 {
			part := candidate.Content.Parts[0]
			if part.FunctionCall != nil {
				args, _ := json.Marshal(part.FunctionCall.Args)
				choice.Message.ToolCalls = []toolCall{
					{
						Type: "function",
						Function: functionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(args),
						},
					},
				}
			} else if part.Thounght != nil && len(candidate.Content.Parts) > 1 {
				choice.Message.Content = reasoningContextMarkerStart + part.Text + reasoningContextMarkerEnd + candidate.Content.Parts[1].Text
			} else if part.Text != "" {
				choice.Message.Content = part.Text
			}
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
		part := vertexResp.Candidates[0].Content.Parts[0]
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			choice.Delta = &chatMessage{
				ToolCalls: []toolCall{
					{
						Type: "function",
						Function: functionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(args),
						},
					},
				},
			}
		} else if part.Thounght != nil {
			if ctx.GetContext("thinking_start") == nil {
				choice.Delta = &chatMessage{Content: reasoningContextMarkerStart + part.Text}
				ctx.SetContext("thinking_start", true)
			} else {
				choice.Delta = &chatMessage{Content: part.Text}
			}
		} else if part.Text != "" {
			if ctx.GetContext("thinking_start") != nil && ctx.GetContext("thinking_end") == nil {
				choice.Delta = &chatMessage{Content: reasoningContextMarkerEnd + part.Text}
				ctx.SetContext("thinking_end", true)
			} else {
				choice.Delta = &chatMessage{Content: part.Text}
			}
		}
	}
	streamResponse := chatCompletionResponse{
		Id:      vertexResp.ResponseId,
		Object:  objectChatCompletionChunk,
		Created: time.Now().UnixMilli() / 1000,
		Model:   ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		Choices: []chatCompletionChoice{choice},
		Usage: &usage{
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
		action = vertexEmbeddingAction
	} else if stream {
		action = vertexChatCompletionStreamAction
	} else {
		action = vertexChatCompletionAction
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
	if request.ReasoningEffort != "" {
		vertexRequest.GenerationConfig.ThinkingConfig = vertexThinkingConfig{
			IncludeThoughts: true,
			ThinkingBudget:  1024,
		}
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
	var lastFunctionName string
	for _, message := range request.Messages {
		content := vertexChatContent{
			Role:  message.Role,
			Parts: []vertexPart{},
		}
		if len(message.ToolCalls) > 0 {
			lastFunctionName = message.ToolCalls[0].Function.Name
			args := make(map[string]interface{})
			if err := json.Unmarshal([]byte(message.ToolCalls[0].Function.Arguments), &args); err != nil {
				log.Errorf("unable to unmarshal function arguments: %v", err)
			}
			content.Parts = append(content.Parts, vertexPart{
				FunctionCall: &vertexFunctionCall{
					Name: lastFunctionName,
					Args: args,
				},
			})
		} else {
			for _, part := range message.ParseContent() {
				switch part.Type {
				case contentTypeText:
					if message.Role == roleTool {
						content.Parts = append(content.Parts, vertexPart{
							FunctionResponse: &vertexFunctionResponse{
								Name: lastFunctionName,
								Response: vertexFunctionResponseDetail{
									Output: part.Text,
								},
							},
						})
					} else {
						content.Parts = append(content.Parts, vertexPart{
							Text: part.Text,
						})
					}
				case contentTypeImageUrl:
					vpart, err := convertImageContent(part.ImageUrl.Url)
					if err != nil {
						log.Errorf("unable to convert image content: %v", err)
					} else {
						content.Parts = append(content.Parts, vpart)
					}
				}
			}
		}

		// there's no assistant role in vertex and API shall vomit if role is not user or model
		switch content.Role {
		case roleAssistant:
			content.Role = "model"
		case roleTool:
			content.Role = roleUser
		case roleSystem: // converting system prompt to prompt from user for the same reason
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
	Text             string                  `json:"text,omitempty"`
	InlineData       *blob                   `json:"inlineData,omitempty"`
	FileData         *fileData               `json:"fileData,omitempty"`
	FunctionCall     *vertexFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *vertexFunctionResponse `json:"functionResponse,omitempty"`
	Thounght         *bool                   `json:"thought,omitempty"`
}

type blob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type fileData struct {
	MimeType string `json:"mimeType"`
	FileUri  string `json:"fileUri"`
}

type vertexFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

type vertexFunctionResponse struct {
	Name     string                       `json:"name"`
	Response vertexFunctionResponseDetail `json:"response"`
}

type vertexFunctionResponseDetail struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
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
	Temperature     float64              `json:"temperature,omitempty"`
	TopP            float64              `json:"topP,omitempty"`
	TopK            int                  `json:"topK,omitempty"`
	CandidateCount  int                  `json:"candidateCount,omitempty"`
	MaxOutputTokens int                  `json:"maxOutputTokens,omitempty"`
	ThinkingConfig  vertexThinkingConfig `json:"thinkingConfig,omitempty"`
}

type vertexThinkingConfig struct {
	IncludeThoughts bool `json:"includeThoughts,omitempty"`
	ThinkingBudget  int  `json:"thinkingBudget,omitempty"`
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

type ServiceAccountKey struct {
	ClientEmail  string `json:"client_email"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	TokenURI     string `json:"token_uri"`
}

func createJWT(key *ServiceAccountKey) (string, error) {
	// 解析 PEM 格式的 RSA 私钥
	block, _ := pem.Decode([]byte(key.PrivateKey))
	if block == nil {
		return "", fmt.Errorf("invalid PEM block")
	}
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	rsaKey := parsedKey.(*rsa.PrivateKey)

	// 构造 JWT Header
	jwtHeader := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": key.PrivateKeyID,
	}
	headerJSON, _ := json.Marshal(jwtHeader)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// 构造 JWT Claims
	now := time.Now().Unix()
	claims := map[string]interface{}{
		"iss":   key.ClientEmail,
		"scope": "https://www.googleapis.com/auth/cloud-platform",
		"aud":   key.TokenURI,
		"iat":   now,
		"exp":   now + 3600, // 1 小时有效期
	}
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := fmt.Sprintf("%s.%s", headerB64, claimsB64)
	hashed := sha256.Sum256([]byte(signingInput))
	signature, err := rsaKey.Sign(nil, hashed[:], crypto.SHA256)
	if err != nil {
		return "", err
	}
	sigB64 := base64.RawURLEncoding.EncodeToString(signature)

	return fmt.Sprintf("%s.%s.%s", headerB64, claimsB64, sigB64), nil
}

func (v *vertexProvider) getAccessToken(jwtToken string) error {
	headers := [][2]string{
		{"Content-Type", "application/x-www-form-urlencoded"},
	}
	reqBody := "grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=" + jwtToken
	err := v.client.Post("/token", headers, []byte(reqBody), func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		responseString := string(responseBody)
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if statusCode != http.StatusOK {
			log.Errorf("failed to create vertex access key, status: %d body: %s", statusCode, responseString)
			_ = util.ErrorHandler("ai-proxy.vertex.load_ak_failed", fmt.Errorf("failed to load vertex ak"))
			return
		}
		responseJson := gjson.Parse(responseString)
		accessToken := responseJson.Get("access_token").String()
		_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+accessToken)

		expiresIn := int64(3600)
		if expiresInVal := responseJson.Get("expires_in"); expiresInVal.Exists() {
			expiresIn = expiresInVal.Int()
		}
		expireTime := time.Now().Add(time.Duration(expiresIn) * time.Second).Unix()
		keyName := v.buildTokenKey()
		err := setCachedAccessToken(keyName, accessToken, expireTime)
		if err != nil {
			log.Errorf("[vertex]: unable to cache access token: %v", err)
		}
	}, v.config.timeout)
	return err
}

func (v *vertexProvider) buildTokenKey() string {
	region := v.config.vertexRegion
	projectID := v.config.vertexProjectId

	return fmt.Sprintf("vertex-%s-%s-access-token", region, projectID)
}

type cachedAccessToken struct {
	Token    string `json:"token"`
	ExpireAt int64  `json:"expireAt"`
}

func (v *vertexProvider) getCachedAccessToken(key string) (string, error) {
	data, _, err := proxywasm.GetSharedData(key)
	if err != nil {
		if errors.Is(err, types.ErrorStatusNotFound) {
			return "", nil
		}
		return "", err
	}
	if data == nil {
		return "", nil
	}

	var tokenInfo cachedAccessToken
	if err = json.Unmarshal(data, &tokenInfo); err != nil {
		return "", err
	}

	now := time.Now().Unix()
	refreshAhead := v.config.vertexTokenRefreshAhead

	if tokenInfo.ExpireAt > now+refreshAhead {
		return tokenInfo.Token, nil
	}

	return "", nil
}

func setCachedAccessToken(key string, accessToken string, expireTime int64) error {
	tokenInfo := cachedAccessToken{
		Token:    accessToken,
		ExpireAt: expireTime,
	}

	_, cas, err := proxywasm.GetSharedData(key)
	if err != nil && !errors.Is(err, types.ErrorStatusNotFound) {
		return err
	}

	data, err := json.Marshal(tokenInfo)
	if err != nil {
		return err
	}

	return proxywasm.SetSharedData(key, data, cas)
}

func convertImageContent(imageUrl string) (vertexPart, error) {
	part := vertexPart{}
	if strings.HasPrefix(imageUrl, "http") {
		arr := strings.Split(imageUrl, ".")
		mimeType := "image/" + arr[len(arr)-1]
		part.FileData = &fileData{
			MimeType: mimeType,
			FileUri:  imageUrl,
		}
		return part, nil
	} else {
		re := regexp.MustCompile(`^data:([^;]+);base64,`)
		matches := re.FindStringSubmatch(imageUrl)
		if len(matches) < 2 {
			return part, fmt.Errorf("invalid base64 format")
		}

		mimeType := matches[1] // e.g. image/png
		parts := strings.Split(mimeType, "/")
		if len(parts) < 2 {
			return part, fmt.Errorf("invalid mimeType")
		}
		part.InlineData = &blob{
			MimeType: mimeType,
			Data:     strings.TrimPrefix(imageUrl, matches[0]),
		}
		return part, nil
	}
}
