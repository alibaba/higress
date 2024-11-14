package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// baiduProvider is the provider for baidu ernie bot service.

const (
	baiduDomain             = "aip.baidubce.com"
	baiduChatCompletionPath = "/chat"
)

var baiduModelToPathSuffixMap = map[string]string{
	"ERNIE-4.0-8K":     "completions_pro",
	"ERNIE-3.5-8K":     "completions",
	"ERNIE-3.5-128K":   "ernie-3.5-128k",
	"ERNIE-Speed-8K":   "ernie_speed",
	"ERNIE-Speed-128K": "ernie-speed-128k",
	"ERNIE-Tiny-8K":    "ernie-tiny-8k",
	"ERNIE-Bot-8K":     "ernie_bot_8k",
	"BLOOMZ-7B":        "bloomz_7b1",
}

type baiduProviderInitializer struct {
}

func (b *baiduProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (b *baiduProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &baiduProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type baiduProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (b *baiduProvider) GetProviderType() string {
	return providerTypeBaidu
}

func (b *baiduProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	b.config.handleRequestHeaders(b, ctx, apiName, log)
	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return types.HeaderStopIteration, nil
}

func (b *baiduProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestHostHeader(headers, baiduDomain)
	headers.Del("Accept-Encoding")
	headers.Del("Content-Length")
}

func (b *baiduProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return b.config.handleRequestBody(b, b.contextCache, ctx, apiName, body, log)
}

func (b *baiduProvider) TransformRequestBodyHeaders(ctx wrapper.HttpContext, apiName ApiName, body []byte, headers http.Header, log wrapper.Log) ([]byte, error) {
	request := &chatCompletionRequest{}
	err := b.config.parseRequestAndMapModel(ctx, request, body, log)
	if err != nil {
		return nil, err
	}
	path := b.getRequestPath(ctx, request.Model)
	util.OverwriteRequestPathHeader(headers, path)

	baiduRequest := b.baiduTextGenRequest(request)
	return json.Marshal(baiduRequest)
}

func (b *baiduProvider) OnResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	// 使用文心一言接口协议,跳过OnStreamingResponseBody()和OnResponseBody()
	if b.config.protocol == protocolOriginal {
		ctx.DontReadResponseBody()
		return types.ActionContinue, nil
	}

	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	return types.ActionContinue, nil
}

func (b *baiduProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	if isLastChunk || len(chunk) == 0 {
		return nil, nil
	}
	// sample event response:
	// data: {"id":"as-vb0m37ti8y","object":"chat.completion","created":1709089502,"sentence_id":0,"is_end":false,"is_truncated":false,"result":"当然可以，","need_clear_history":false,"finish_reason":"normal","usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}

	// sample end event response:
	// data: {"id":"as-vb0m37ti8y","object":"chat.completion","created":1709089531,"sentence_id":20,"is_end":true,"is_truncated":false,"result":"","need_clear_history":false,"finish_reason":"normal","usage":{"prompt_tokens":5,"completion_tokens":420,"total_tokens":425}}
	responseBuilder := &strings.Builder{}
	lines := strings.Split(string(chunk), "\n")
	for _, data := range lines {
		if len(data) < 6 {
			// ignore blank line or wrong format
			continue
		}
		data = data[6:]
		var baiduResponse baiduTextGenStreamResponse
		if err := json.Unmarshal([]byte(data), &baiduResponse); err != nil {
			log.Errorf("unable to unmarshal baidu response: %v", err)
			continue
		}
		response := b.streamResponseBaidu2OpenAI(ctx, &baiduResponse)
		responseBody, err := json.Marshal(response)
		if err != nil {
			log.Errorf("unable to marshal response: %v", err)
			return nil, err
		}
		b.appendResponse(responseBuilder, string(responseBody))
	}
	modifiedResponseChunk := responseBuilder.String()
	log.Debugf("=== modified response chunk: %s", modifiedResponseChunk)
	return []byte(modifiedResponseChunk), nil
}

func (b *baiduProvider) OnResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	baiduResponse := &baiduTextGenResponse{}
	if err := json.Unmarshal(body, baiduResponse); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal baidu response: %v", err)
	}
	if baiduResponse.ErrorMsg != "" {
		return types.ActionContinue, fmt.Errorf("baidu response error, error_code: %d, error_message: %s", baiduResponse.ErrorCode, baiduResponse.ErrorMsg)
	}
	response := b.responseBaidu2OpenAI(ctx, baiduResponse)
	return types.ActionContinue, replaceJsonResponseBody(response, log)
}

type baiduTextGenRequest struct {
	Model           string        `json:"model"`
	Messages        []chatMessage `json:"messages"`
	Temperature     float64       `json:"temperature,omitempty"`
	TopP            float64       `json:"top_p,omitempty"`
	PenaltyScore    float64       `json:"penalty_score,omitempty"`
	Stream          bool          `json:"stream,omitempty"`
	System          string        `json:"system,omitempty"`
	DisableSearch   bool          `json:"disable_search,omitempty"`
	EnableCitation  bool          `json:"enable_citation,omitempty"`
	MaxOutputTokens int           `json:"max_output_tokens,omitempty"`
	UserId          string        `json:"user_id,omitempty"`
}

func (b *baiduProvider) getRequestPath(ctx wrapper.HttpContext, baiduModel string) string {
	// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/clntwmv7t
	suffix, ok := baiduModelToPathSuffixMap[baiduModel]
	if !ok {
		suffix = baiduModel
	}
	return fmt.Sprintf("/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/%s?access_token=%s", suffix, b.config.GetApiTokenInUse(ctx))
}

func (b *baiduProvider) setSystemContent(request *baiduTextGenRequest, content string) {
	request.System = content
}

func (b *baiduProvider) baiduTextGenRequest(request *chatCompletionRequest) *baiduTextGenRequest {
	baiduRequest := baiduTextGenRequest{
		Messages:        make([]chatMessage, 0, len(request.Messages)),
		Temperature:     request.Temperature,
		TopP:            request.TopP,
		PenaltyScore:    request.FrequencyPenalty,
		Stream:          request.Stream,
		DisableSearch:   false,
		EnableCitation:  false,
		MaxOutputTokens: request.MaxTokens,
		UserId:          request.User,
	}
	for _, message := range request.Messages {
		if message.Role == roleSystem {
			baiduRequest.System = message.StringContent()
		} else {
			baiduRequest.Messages = append(baiduRequest.Messages, chatMessage{
				Role:    message.Role,
				Content: message.Content,
			})
		}
	}
	return &baiduRequest
}

type baiduTextGenResponse struct {
	Id               string                    `json:"id"`
	Object           string                    `json:"object"`
	Created          int64                     `json:"created"`
	Result           string                    `json:"result"`
	IsTruncated      bool                      `json:"is_truncated"`
	NeedClearHistory bool                      `json:"need_clear_history"`
	Usage            baiduTextGenResponseUsage `json:"usage"`
	baiduTextGenResponseError
}

type baiduTextGenResponseError struct {
	ErrorCode int    `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

type baiduTextGenStreamResponse struct {
	baiduTextGenResponse
	SentenceId int  `json:"sentence_id"`
	IsEnd      bool `json:"is_end"`
}

type baiduTextGenResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (b *baiduProvider) responseBaidu2OpenAI(ctx wrapper.HttpContext, response *baiduTextGenResponse) *chatCompletionResponse {
	choice := chatCompletionChoice{
		Index:        0,
		Message:      &chatMessage{Role: roleAssistant, Content: response.Result},
		FinishReason: finishReasonStop,
	}
	return &chatCompletionResponse{
		Id:                response.Id,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		SystemFingerprint: "",
		Object:            objectChatCompletion,
		Choices:           []chatCompletionChoice{choice},
		Usage: usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}
}

func (b *baiduProvider) streamResponseBaidu2OpenAI(ctx wrapper.HttpContext, response *baiduTextGenStreamResponse) *chatCompletionResponse {
	choice := chatCompletionChoice{
		Index:   0,
		Message: &chatMessage{Role: roleAssistant, Content: response.Result},
	}
	if response.IsEnd {
		choice.FinishReason = finishReasonStop
	}
	return &chatCompletionResponse{
		Id:                response.Id,
		Created:           time.Now().UnixMilli() / 1000,
		Model:             ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
		SystemFingerprint: "",
		Object:            objectChatCompletionChunk,
		Choices:           []chatCompletionChoice{choice},
		Usage: usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}
}

func (b *baiduProvider) appendResponse(responseBuilder *strings.Builder, responseBody string) {
	responseBuilder.WriteString(fmt.Sprintf("%s %s\n\n", streamDataItemKey, responseBody))
}

func (b *baiduProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, baiduChatCompletionPath) {
		return ApiNameChatCompletion
	}
	return ""
}
