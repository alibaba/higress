package provider

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"net/http"
	"strings"
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
	bedrockSignedHeaders            = "host;x-amz-date"
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

func (b *bedrockProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool) ([]byte, error) {
	events := extractAmazonEventStreamEvents(ctx, chunk)
	if len(events) == 0 {
		return chunk, fmt.Errorf("No events are extracted ")
	}
	var responseBuilder strings.Builder
	for _, event := range events {
		outputEvent, err := b.convertEventFromBedrockToOpenAI(ctx, event)
		if err != nil {
			log.Errorf("[onStreamingResponseBody] failed to process streaming event: %v\n%s", err, chunk)
			return chunk, err
		}
		responseBuilder.WriteString(string(outputEvent))
	}
	return []byte(responseBuilder.String()), nil
}

func (b *bedrockProvider) convertEventFromBedrockToOpenAI(ctx wrapper.HttpContext, bedrockEvent ConverseStreamEvent) ([]byte, error) {
	choices := make([]chatCompletionChoice, 0)
	chatChoice := chatCompletionChoice{
		Delta: &chatMessage{},
	}
	if bedrockEvent.Role != nil {
		chatChoice.Delta.Role = *bedrockEvent.Role
	}
	if bedrockEvent.Delta != nil {
		chatChoice.Delta = &chatMessage{Content: bedrockEvent.Delta.Text}
	}
	if bedrockEvent.StopReason != nil {
		chatChoice.FinishReason = stopReasonBedrock2OpenAI(*bedrockEvent.StopReason)
	}
	choices = append(choices, chatChoice)
	requestId := ctx.GetStringContext("X-Amzn-Requestid", "")
	openAIFormattedChunk := &chatCompletionResponse{
		Id:                requestId,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		SystemFingerprint: "",
		Object:            objectChatCompletion,
		Choices:           choices,
	}
	if bedrockEvent.Usage != nil {
		openAIFormattedChunk.Choices = choices[:0]
		openAIFormattedChunk.Usage = usage{
			CompletionTokens: bedrockEvent.Usage.OutputTokens,
			PromptTokens:     bedrockEvent.Usage.InputTokens,
			TotalTokens:      bedrockEvent.Usage.TotalTokens,
		}
	}

	openAIFormattedChunkBytes, _ := json.Marshal(openAIFormattedChunk)
	var openAIChunk strings.Builder
	openAIChunk.WriteString(ssePrefix)
	openAIChunk.WriteString(string(openAIFormattedChunkBytes))
	openAIChunk.WriteString("\n\n")
	return []byte(openAIChunk.String()), nil
}

type ConverseStreamEvent struct {
	ContentBlockIndex int                                   `json:"contentBlockIndex,omitempty"`
	Delta             *converseStreamEventContentBlockDelta `json:"delta,omitempty"`
	Role              *string                               `json:"role,omitempty"`
	StopReason        *string                               `json:"stopReason,omitempty"`
	Usage             *tokenUsage                           `json:"usage,omitempty"`
	Start             *contentBlockStart                    `json:"start,omitempty"`
}

type converseStreamEventContentBlockDelta struct {
	Text    *string            `json:"text,omitempty"`
	ToolUse *toolUseBlockDelta `json:"toolUse,omitempty"`
}

type toolUseBlockStart struct {
	Name      string `json:"name"`
	ToolUseID string `json:"toolUseId"`
}

type contentBlockStart struct {
	ToolUse *toolUseBlockStart `json:"toolUse,omitempty"`
}

type toolUseBlockDelta struct {
	Input string `json:"input"`
}

func extractAmazonEventStreamEvents(ctx wrapper.HttpContext, chunk []byte) []ConverseStreamEvent {
	body := chunk
	if bufferedStreamingBody, has := ctx.GetContext(ctxKeyStreamingBody).([]byte); has {
		body = append(bufferedStreamingBody, chunk...)
	}

	r := bytes.NewReader(body)
	var events []ConverseStreamEvent
	var lastRead int64 = -1
	messageBuffer := make([]byte, 1024)
	defer func() {
		log.Infof("extractAmazonEventStreamEvents: lastRead=%d, r.Size=%d", lastRead, r.Size())
		ctx.SetContext(ctxKeyStreamingBody, nil)
	}()

	for {
		msg, err := decodeMessage(r, messageBuffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Errorf("failed to decode message: %v", err)
			break
		}
		var event ConverseStreamEvent
		if err = json.Unmarshal(msg.Payload, &event); err == nil {
			events = append(events, event)
		}
		lastRead = r.Size() - int64(r.Len())
	}
	return events
}

type bedrockStreamMessage struct {
	Headers headers
	Payload []byte
}

type EventFrame struct {
	TotalLength   uint32
	HeadersLength uint32
	PreludeCRC    uint32
	Headers       map[string]interface{}
	Payload       []byte
	PayloadCRC    uint32
}

type headers []header

type header struct {
	Name  string
	Value Value
}

func (hs *headers) Set(name string, value Value) {
	var i int
	for ; i < len(*hs); i++ {
		if (*hs)[i].Name == name {
			(*hs)[i].Value = value
			return
		}
	}

	*hs = append(*hs, header{
		Name: name, Value: value,
	})
}

func decodeMessage(reader io.Reader, payloadBuf []byte) (m bedrockStreamMessage, err error) {
	crc := crc32.New(crc32.MakeTable(crc32.IEEE))
	hashReader := io.TeeReader(reader, crc)

	prelude, err := decodePrelude(hashReader, crc)
	if err != nil {
		return bedrockStreamMessage{}, err
	}

	if prelude.HeadersLen > 0 {
		lr := io.LimitReader(hashReader, int64(prelude.HeadersLen))
		m.Headers, err = decodeHeaders(lr)
		if err != nil {
			return bedrockStreamMessage{}, err
		}
	}

	if payloadLen := prelude.PayloadLen(); payloadLen > 0 {
		buf, err := decodePayload(payloadBuf, io.LimitReader(hashReader, int64(payloadLen)))
		if err != nil {
			return bedrockStreamMessage{}, err
		}
		m.Payload = buf
	}

	msgCRC := crc.Sum32()
	if err := validateCRC(reader, msgCRC); err != nil {
		return bedrockStreamMessage{}, err
	}

	return m, nil
}

func decodeHeaders(r io.Reader) (headers, error) {
	hs := headers{}

	for {
		name, err := decodeHeaderName(r)
		if err != nil {
			if err == io.EOF {
				// EOF while getting header name means no more headers
				break
			}
			return nil, err
		}

		value, err := decodeHeaderValue(r)
		if err != nil {
			return nil, err
		}

		hs.Set(name, value)
	}

	return hs, nil
}

func decodeHeaderValue(r io.Reader) (Value, error) {
	var raw rawValue

	typ, err := decodeUint8(r)
	if err != nil {
		return nil, err
	}
	raw.Type = valueType(typ)

	var v Value

	switch raw.Type {
	case stringValueType:
		var tv StringValue
		err = tv.decode(r)
		v = tv
	default:
		log.Errorf("unknown value type %d", raw.Type)
	}

	// Error could be EOF, let caller deal with it
	return v, err
}

type Value interface {
	Get() interface{}
}

type StringValue string

func (v StringValue) Get() interface{} {
	return string(v)
}

func (v *StringValue) decode(r io.Reader) error {
	s, err := decodeStringValue(r)
	if err != nil {
		return err
	}

	*v = StringValue(s)
	return nil
}

func decodeBytesValue(r io.Reader) ([]byte, error) {
	var raw rawValue
	var err error
	raw.Len, err = decodeUint16(r)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, raw.Len)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func decodeUint16(r io.Reader) (uint16, error) {
	var b [2]byte
	bs := b[:]
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(bs), nil
}

func decodeStringValue(r io.Reader) (string, error) {
	v, err := decodeBytesValue(r)
	return string(v), err
}

type rawValue struct {
	Type  valueType
	Len   uint16 // Only set for variable length slices
	Value []byte // byte representation of value, BigEndian encoding.
}

type valueType uint8

const (
	trueValueType valueType = iota
	falseValueType
	int8ValueType  // Byte
	int16ValueType // Short
	int32ValueType // Integer
	int64ValueType // Long
	bytesValueType
	stringValueType
	timestampValueType
	uuidValueType
)

func decodeHeaderName(r io.Reader) (string, error) {
	var n headerName

	var err error
	n.Len, err = decodeUint8(r)
	if err != nil {
		return "", err
	}

	name := n.Name[:n.Len]
	if _, err := io.ReadFull(r, name); err != nil {
		return "", err
	}

	return string(name), nil
}

func decodeUint8(r io.Reader) (uint8, error) {
	type byteReader interface {
		ReadByte() (byte, error)
	}

	if br, ok := r.(byteReader); ok {
		v, err := br.ReadByte()
		return v, err
	}

	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	return b[0], err
}

const maxHeaderNameLen = 255

type headerName struct {
	Len  uint8
	Name [maxHeaderNameLen]byte
}

func decodePayload(buf []byte, r io.Reader) ([]byte, error) {
	w := bytes.NewBuffer(buf[0:0])

	_, err := io.Copy(w, r)
	return w.Bytes(), err
}

type messagePrelude struct {
	Length     uint32
	HeadersLen uint32
	PreludeCRC uint32
}

func (p messagePrelude) ValidateLens() error {
	if p.Length == 0 {
		return fmt.Errorf("message prelude want: 16, have: %v", int(p.Length))
	}
	return nil
}

func (p messagePrelude) PayloadLen() uint32 {
	return p.Length - p.HeadersLen - 16
}

func decodePrelude(r io.Reader, crc hash.Hash32) (messagePrelude, error) {
	var p messagePrelude

	var err error
	p.Length, err = decodeUint32(r)
	if err != nil {
		return messagePrelude{}, err
	}

	p.HeadersLen, err = decodeUint32(r)
	if err != nil {
		return messagePrelude{}, err
	}

	if err := p.ValidateLens(); err != nil {
		return messagePrelude{}, err
	}

	preludeCRC := crc.Sum32()
	if err := validateCRC(r, preludeCRC); err != nil {
		return messagePrelude{}, err
	}

	p.PreludeCRC = preludeCRC

	return p, nil
}

func decodeUint32(r io.Reader) (uint32, error) {
	var b [4]byte
	bs := b[:]
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(bs), nil
}

func validateCRC(r io.Reader, expect uint32) error {
	msgCRC, err := decodeUint32(r)
	if err != nil {
		return err
	}

	if msgCRC != expect {
		return fmt.Errorf("message checksum mismatch")
	}

	return nil
}

func (b *bedrockProvider) TransformResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	ctx.SetContext("X-Amzn-Requestid", headers.Get("X-Amzn-Requestid"))
	if headers.Get("Content-Type") == "application/vnd.amazon.eventstream" {
		headers.Set("Content-Type", "text/event-stream; charset=utf-8")
	}
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

func (b *bedrockProvider) onChatCompletionResponseBody(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	bedrockResponse := &bedrockConverseResponse{}
	if err := json.Unmarshal(body, bedrockResponse); err != nil {
		log.Errorf("unable to unmarshal bedrock response: %v", err)
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
	headers.Set("Accept", "*/*")
	if streaming {
		util.OverwriteRequestPathHeader(headers, fmt.Sprintf(bedrockStreamChatCompletionPath, request.Model))
	} else {
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
			FinishReason: stopReasonBedrock2OpenAI(bedrockResponse.StopReason),
		},
	}
	requestId := ctx.GetStringContext("X-Amzn-Requestid", "")
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

func stopReasonBedrock2OpenAI(reason string) string {
	switch reason {
	case "end_turn":
		return finishReasonStop
	case "stop_sequence":
		return finishReasonStop
	case "max_tokens":
		return finishReasonLength
	default:
		return reason
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
	MaxTokens     int      `json:"maxTokens,omitempty"`
	Temperature   float64  `json:"temperature,omitempty"`
	TopP          float64  `json:"topP,omitempty"`
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
	if headers != nil {
		path = headers.Get(":path")
	}
	signature := b.generateSignature(path, amzDate, dateStamp, body)
	if headers != nil {
		headers.Set("X-Amz-Date", amzDate)
		headers.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/%s/aws4_request, SignedHeaders=%s, Signature=%s", b.config.awsAccessKey, dateStamp, b.config.awsRegion, awsService, bedrockSignedHeaders, signature))
	} else {
		_ = proxywasm.ReplaceHttpRequestHeader("X-Amz-Date", amzDate)
		_ = proxywasm.ReplaceHttpRequestHeader("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/%s/aws4_request, SignedHeaders=%s, Signature=%s", b.config.awsAccessKey, dateStamp, b.config.awsRegion, awsService, bedrockSignedHeaders, signature))
	}
}

func (b *bedrockProvider) generateSignature(path, amzDate, dateStamp string, body []byte) string {
	hashedPayload := sha256Hex(body)
	path = urlEncoding(path)

	endpoint := fmt.Sprintf(bedrockDefaultDomain, b.config.awsRegion)
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-date:%s\n", endpoint, amzDate)
	canonicalRequest := fmt.Sprintf("%s\n%s\n\n%s\n%s\n%s",
		httpPostMethod, path, canonicalHeaders, bedrockSignedHeaders, hashedPayload)

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, b.config.awsRegion, awsService)
	hashedCanonReq := sha256Hex([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, hashedCanonReq)

	signingKey := getSignatureKey(b.config.awsSecretKey, dateStamp, b.config.awsRegion, awsService)
	signature := hmacHex(signingKey, stringToSign)
	return signature
}

func urlEncoding(rawStr string) string {
	encodedStr := strings.ReplaceAll(rawStr, ":", "%3A")
	encodedStr = strings.ReplaceAll(encodedStr, "+", "%2B")
	encodedStr = strings.ReplaceAll(encodedStr, "=", "%3D")
	encodedStr = strings.ReplaceAll(encodedStr, "&", "%26")
	encodedStr = strings.ReplaceAll(encodedStr, "$", "%24")
	encodedStr = strings.ReplaceAll(encodedStr, "@", "%40")
	return encodedStr
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
