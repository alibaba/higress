package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	httpPostMethod = "POST"
	awsService     = "bedrock"
	// bedrock-runtime.{awsRegion}.amazonaws.com
	bedrockDefaultDomain = "bedrock-runtime.%s.amazonaws.com"
	// converse路径 /model/{modelId}/converse
	bedrockChatCompletionPath = "/model/%s/converse"
	// converseStream路径 /model/{modelId}/converse-stream
	bedrockStreamChatCompletionPath = "/model/%s/converse-stream"
)

type bedrockProviderInitializer struct {
}

func (b *bedrockProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if len(config.awsAccessKey) == 0 || len(config.awsSecretKey) == 0 {
		return errors.New("missing bedrock access authentication parameters")
	}
	if len(config.awsRegion) == 0 {
		return errors.New("missing bedrock region parameters")
	}
	return nil
}

func (b *bedrockProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): bedrockChatCompletionPath,
	}
}

func (b *bedrockProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(b.DefaultCapabilities())
	return &bedrockProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type bedrockProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (b *bedrockProvider) GetProviderType() string {
	return providerTypeBedrock
}

func (b *bedrockProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	b.config.handleRequestHeaders(b, ctx, apiName)
	return nil
}

func (b *bedrockProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestHostHeader(headers, fmt.Sprintf(bedrockDefaultDomain, b.config.awsRegion))
}

func (b *bedrockProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !b.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return b.config.handleRequestBody(b, b.contextCache, ctx, apiName, body)
}

func (b *bedrockProvider) insertHttpContextMessage(body []byte, content string, onlyOneSystemBeforeFile bool) ([]byte, error) {
	request := &bedrockTextGenRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		return nil, fmt.Errorf("unable to unmarshal request: %v", err)
	}

	if len(request.System) > 0 {
		request.System = append(request.System, systemContentBlock{Text: content})
	} else {
		request.System = []systemContentBlock{{Text: content}}
	}

	requestBytes, err := json.Marshal(request)
	b.setAuthHeaders(requestBytes, nil)
	return requestBytes, err
}

func (b *bedrockProvider) TransformRequestBodyHeaders(ctx wrapper.HttpContext, apiName ApiName, body []byte, headers http.Header) ([]byte, error) {
	switch apiName {
	case ApiNameChatCompletion:
		return b.onChatCompletionRequestBody(ctx, body, headers)
	default:
		return b.config.defaultTransformRequestBody(ctx, apiName, body)
	}
}

func (b *bedrockProvider) TransformResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	if apiName == ApiNameChatCompletion {
		return b.onChatCompletionResponseBody(ctx, body)
	}
	return nil, errUnsupportedApiName
}

type bedrockConverseStreamResponse struct {
	MessageStart      *MessageStartEvent           `json:"messageStart,omitempty"`
	ContentBlockStart *ContentBlockStartEvent      `json:"contentBlockStart,omitempty"`
	ContentBlockDelta *ContentBlockDeltaEvent      `json:"contentBlockDelta,omitempty"`
	ContentBlockStop  *ContentBlockStopEvent       `json:"contentBlockStop,omitempty"`
	MessageStop       *MessageStopEvent            `json:"messageStop,omitempty"`
	Metadata          *ConverseStreamMetadataEvent `json:"metadata,omitempty"`
}

type MessageStartEvent struct {
	Role string `json:"role"`
}

type ContentBlockStartEvent struct {
	ContentBlockIndex int `json:"contentBlockIndex"`
	Start             struct {
		ToolUse struct {
			ToolUseId string `json:"toolUseId"`
			Name      string `json:"name"`
		} `json:"toolUse,omitempty"`
	} `json:"start"`
}

type ContentBlockDeltaEvent struct {
	ContentBlockIndex int `json:"contentBlockIndex"`
	Delta             struct {
		Text             string `json:"text,omitempty"`
		ReasoningContent struct {
			Signature string `json:"signature"`
			Text      string `json:"text"`
		} `json:"reasoningContent,omitempty"`
		ToolUse struct {
			Input string `json:"input"`
		} `json:"toolUse,omitempty"`
	} `json:"delta"`
}

type ContentBlockStopEvent struct {
	ContentBlockIndex int `json:"contentBlockIndex"`
}

type MessageStopEvent struct {
	StopReason string `json:"stopReason"`
}

type ConverseStreamMetadataEvent struct {
	Usage struct {
		InputTokens  int `json:"inputTokens"`
		OutputTokens int `json:"outputTokens"`
		TotalTokens  int `json:"totalTokens"`
	} `json:"usage"`
	Metrics struct {
		LatencyMs float64 `json:"latencyMs"`
	} `json:"metrics"`
}

func (b *bedrockProvider) OnStreamingEvent(ctx wrapper.HttpContext, name ApiName, event StreamEvent) ([]StreamEvent, error) {
	var outputEvents []StreamEvent
	bedrockResponse := &bedrockConverseStreamResponse{}
	if err := json.Unmarshal([]byte(event.Data), bedrockResponse); err != nil {
		log.Errorf("unable to unmarshal bedrock response: %v", err)
		return nil, fmt.Errorf("unable to unmarshal bedrock response: %v", err)
	}

	responses := b.buildChatCompletionStreamingResponse(ctx, bedrockResponse)
	for _, response := range responses {
		responseBody, err := json.Marshal(response)
		if err != nil {
			log.Errorf("unable to marshal response: %v", err)
			return nil, fmt.Errorf("unable to marshal response: %v", err)
		}
		modifiedEvent := event
		modifiedEvent.Data = string(responseBody)
		outputEvents = append(outputEvents, modifiedEvent)
	}

	return outputEvents, nil
}

func (b *bedrockProvider) buildChatCompletionStreamingResponse(ctx wrapper.HttpContext, bedrockResponse *bedrockConverseStreamResponse) []*chatCompletionResponse {
	requestId, _ := proxywasm.GetHttpResponseHeader("x-amzn-requestid")
	baseMessage := chatCompletionResponse{
		Id:                requestId,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		Choices:           make([]chatCompletionChoice, 0),
		SystemFingerprint: "",
		Object:            objectChatCompletionChunk,
	}
	responses := make([]*chatCompletionResponse, 0)
	if bedrockResponse.MessageStart != nil {
		response := *&baseMessage
		convertedMessage := &chatMessage{Role: bedrockResponse.MessageStart.Role}
		response.Choices = append(response.Choices, chatCompletionChoice{Delta: convertedMessage})
		responses = append(responses, &response)
	}
	if bedrockResponse.ContentBlockDelta != nil {
		response := *&baseMessage
		convertedMessage := &chatMessage{Content: bedrockResponse.ContentBlockDelta.Delta.Text, ReasoningContent: bedrockResponse.ContentBlockDelta.Delta.ReasoningContent.Text}
		response.Choices = append(response.Choices, chatCompletionChoice{Delta: convertedMessage})
		responses = append(responses, &response)
	}
	if bedrockResponse.MessageStop != nil {
		response := *&baseMessage
		response.Choices = append(response.Choices, chatCompletionChoice{Delta: &chatMessage{}, FinishReason: bedrockResponse.MessageStop.StopReason})
		responses = append(responses, &response)
	}
	if bedrockResponse.Metadata != nil {
		response := *&baseMessage
		response.Choices = []chatCompletionChoice{{Delta: &chatMessage{}}}
		response.Usage = usage{
			PromptTokens:     bedrockResponse.Metadata.Usage.InputTokens,
			CompletionTokens: bedrockResponse.Metadata.Usage.OutputTokens,
			TotalTokens:      bedrockResponse.Metadata.Usage.TotalTokens,
		}
		responses = append(responses, &response)
	}
	return responses
}

func (b *bedrockProvider) onChatCompletionResponseBody(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	bedrockResponse := &bedrockConverseResponse{}
	if err := json.Unmarshal(body, bedrockResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal bedrock response: %v", err)
	}
	response := b.buildChatCompletionResponse(ctx, bedrockResponse)
	return json.Marshal(response)
}

func (b *bedrockProvider) onChatCompletionRequestBody(ctx wrapper.HttpContext, body []byte, headers http.Header) ([]byte, error) {
	request := &chatCompletionRequest{}
	err := b.config.parseRequestAndMapModel(ctx, request, body)
	if err != nil {
		return nil, err
	}

	streaming := request.Stream
	if streaming {
		headers.Set("Accept", "text/event-stream")
		util.OverwriteRequestPathHeader(headers, fmt.Sprintf(bedrockStreamChatCompletionPath, request.Model))
	} else {
		headers.Set("Accept", "*/*")
		util.OverwriteRequestPathHeader(headers, fmt.Sprintf(bedrockChatCompletionPath, request.Model))
	}
	return b.buildBedrockTextGenerationRequest(request, headers)
}

func (b *bedrockProvider) buildBedrockTextGenerationRequest(origRequest *chatCompletionRequest, headers http.Header) ([]byte, error) {
	messages := make([]bedrockMessage, 0, len(origRequest.Messages))
	for i := range origRequest.Messages {
		messages = append(messages, chatMessage2BedrockMessage(origRequest.Messages[i]))
	}
	request := &bedrockTextGenRequest{
		Messages: messages,
		InferenceConfig: bedrockInferenceConfig{
			MaxTokens:   origRequest.MaxTokens,
			Temperature: origRequest.Temperature,
			TopP:        origRequest.TopP,
		},
		AdditionalModelRequestFields: map[string]interface{}{},
		PerformanceConfig: PerformanceConfiguration{
			Latency: "standard",
		},
	}
	requestBytes, err := json.Marshal(request)
	b.setAuthHeaders(requestBytes, headers)
	return requestBytes, err
}

func (b *bedrockProvider) buildChatCompletionResponse(ctx wrapper.HttpContext, bedrockResponse *bedrockConverseResponse) *chatCompletionResponse {
	var outputContent string
	if len(bedrockResponse.Output.Message.Content) > 0 {
		outputContent = bedrockResponse.Output.Message.Content[0].Text
	}
	choices := []chatCompletionChoice{
		{
			Index: 0,
			Message: &chatMessage{
				Role:    bedrockResponse.Output.Message.Role,
				Content: outputContent,
			},
			FinishReason: bedrockResponse.StopReason,
		},
	}
	requestId, _ := proxywasm.GetHttpResponseHeader("x-amzn-requestid")
	return &chatCompletionResponse{
		Id:                requestId,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		SystemFingerprint: "",
		Object:            objectChatCompletion,
		Choices:           choices,
		Usage: usage{
			PromptTokens:     bedrockResponse.Usage.InputTokens,
			CompletionTokens: bedrockResponse.Usage.OutputTokens,
			TotalTokens:      bedrockResponse.Usage.TotalTokens,
		},
	}
}

type bedrockTextGenRequest struct {
	Messages                     []bedrockMessage         `json:"messages"`
	System                       []systemContentBlock     `json:"system,omitempty"`
	InferenceConfig              bedrockInferenceConfig   `json:"inferenceConfig,omitempty"`
	AdditionalModelRequestFields map[string]interface{}   `json:"additionalModelRequestFields,omitempty"`
	PerformanceConfig            PerformanceConfiguration `json:"performanceConfig,omitempty"`
}

type PerformanceConfiguration struct {
	Latency string `json:"latency,omitempty"`
}

type bedrockMessage struct {
	Role    string                  `json:"role"`
	Content []bedrockMessageContent `json:"content"`
}

type bedrockMessageContent struct {
	Text  string      `json:"text,omitempty"`
	Image *imageBlock `json:"image,omitempty"`
}

type systemContentBlock struct {
	Text string `json:"text,omitempty"`
}

type imageBlock struct {
	Format string      `json:"format,omitempty"`
	Source imageSource `json:"source,omitempty"`
}

type imageSource struct {
	Bytes string `json:"bytes,omitempty"`
}

type bedrockInferenceConfig struct {
	StopSequences []string `json:"stopSequences,omitempty"`
	MaxTokens     int      `json:"max_tokens,omitempty"`
	Temperature   float64  `json:"temperature,omitempty"`
	TopP          float64  `json:"top_p,omitempty"`
}

type bedrockConverseResponse struct {
	Metrics    converseMetrics             `json:"metrics"`
	Output     converseOutputMemberMessage `json:"output"`
	StopReason string                      `json:"stopReason"`
	Usage      tokenUsage                  `json:"usage"`
}

type converseMetrics struct {
	LatencyMs int `json:"latencyMs"`
}

type converseOutputMemberMessage struct {
	Message message `json:"message"`
}

type message struct {
	Content []contentBlockMemberText `json:"content"`

	Role string `json:"role"`
}

type contentBlockMemberText struct {
	Text string `json:"text"`
}

type tokenUsage struct {
	InputTokens int `json:"inputTokens,omitempty"`

	OutputTokens int `json:"outputTokens,omitempty"`

	TotalTokens int `json:"totalTokens"`
}

func chatMessage2BedrockMessage(chatMessage chatMessage) bedrockMessage {
	if chatMessage.IsStringContent() {
		return bedrockMessage{
			Role:    chatMessage.Role,
			Content: []bedrockMessageContent{{Text: chatMessage.StringContent()}},
		}
	} else {
		var contents []bedrockMessageContent
		openaiContent := chatMessage.ParseContent()
		for _, part := range openaiContent {
			var content bedrockMessageContent
			if part.Type == contentTypeText {
				content.Text = part.Text
			} else {
				log.Warnf("imageUrl is not supported: %s", part.Type)
				continue
			}
			contents = append(contents, content)
		}
		return bedrockMessage{
			Role:    chatMessage.Role,
			Content: contents,
		}
	}
}

func (b *bedrockProvider) setAuthHeaders(body []byte, headers http.Header) {
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102")
	path, _ := proxywasm.GetHttpRequestHeader(":path")
	signature := b.generateSignature(path, amzDate, dateStamp, body)
	if headers != nil {
		headers.Set("X-Amz-Date", amzDate)
		headers.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/%s/aws4_request, SignedHeaders=host;x-amz-date, Signature=%s", b.config.awsAccessKey, dateStamp, b.config.awsRegion, awsService, signature))
	} else {
		_ = proxywasm.ReplaceHttpRequestHeader("X-Amz-Date", amzDate)
		_ = proxywasm.ReplaceHttpRequestHeader("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/%s/aws4_request, SignedHeaders=host;x-amz-date, Signature=%s", b.config.awsAccessKey, dateStamp, b.config.awsRegion, awsService, signature))
	}
}

func (b *bedrockProvider) generateSignature(path, amzDate, dateStamp string, body []byte) string {
	hashedPayload := sha256Hex(body)

	endpoint := fmt.Sprintf(bedrockDefaultDomain, b.config.awsRegion)
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-date:%s\n", endpoint, amzDate)
	signedHeaders := "host;x-amz-date"
	canonicalRequest := fmt.Sprintf("%s\n%s\n\n%s\n%s\n%s",
		httpPostMethod, path, canonicalHeaders, signedHeaders, hashedPayload)

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, b.config.awsRegion, awsService)
	hashedCanonReq := sha256Hex([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, hashedCanonReq)

	signingKey := getSignatureKey(b.config.awsSecretKey, dateStamp, b.config.awsRegion, awsService)
	signature := hmacHex(signingKey, stringToSign)
	return signature
}

func getSignatureKey(key, dateStamp, region, service string) []byte {
	kDate := hmacSha256([]byte("AWS4"+key), dateStamp)
	kRegion := hmacSha256(kDate, region)
	kService := hmacSha256(kRegion, service)
	kSigning := hmacSha256(kService, "aws4_request")
	return kSigning
}

func hmacSha256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func hmacHex(key []byte, data string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
