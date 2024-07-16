package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// baiduProvider is the provider for baidu ernie bot service.

const (
	baiduDomain = "aip.baidubce.com"
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
	_ = util.OverwriteRequestHost(baiduDomain)

	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return types.HeaderStopIteration, nil
}

func (b *baiduProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	// 使用文心一言接口协议
	if b.config.protocol == protocolOriginal {
		request := &baiduTextGenRequest{}
		if err := json.Unmarshal(body, request); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
		}
		if request.Model == "" {
			return types.ActionContinue, errors.New("request model is empty")
		}
		// 根据模型重写requestPath
		path := b.GetRequestPath(request.Model)
		_ = util.OverwriteRequestPath(path)

		if b.config.context == nil {
			return types.ActionContinue, nil
		}

		err := b.contextCache.GetContent(func(content string, err error) {
			defer func() {
				_ = proxywasm.ResumeHttpRequest()
			}()

			if err != nil {
				log.Errorf("failed to load context file: %v", err)
				_ = util.SendResponse(500, "ai-proxy.baidu.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
			}
			b.setSystemContent(request, content)
			if err := replaceJsonRequestBody(request, log); err != nil {
				_ = util.SendResponse(500, "ai-proxy.baidu.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
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
	mappedModel := getMappedModel(model, b.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel
	ctx.SetContext(ctxKeyFinalRequestModel, request.Model)
	path := b.GetRequestPath(mappedModel)
	_ = util.OverwriteRequestPath(path)

	if b.config.context == nil {
		baiduRequest := b.baiduTextGenRequest(request)
		return types.ActionContinue, replaceJsonRequestBody(baiduRequest, log)
	}

	err := b.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.baidu.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		baiduRequest := b.baiduTextGenRequest(request)
		if err := replaceJsonRequestBody(baiduRequest, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.baidu.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace Request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
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

func (b *baiduProvider) GetRequestPath(baiduModel string) string {
	// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/clntwmv7t
	suffix, ok := baiduModelToPathSuffixMap[baiduModel]
	if !ok {
		suffix = baiduModel
	}
	return fmt.Sprintf("/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/%s?access_token=%s", suffix, b.config.GetRandomToken())
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
			baiduRequest.System = message.Content
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
		Model:             ctx.GetContext(ctxKeyFinalRequestModel).(string),
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
		Model:             ctx.GetContext(ctxKeyFinalRequestModel).(string),
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

func (b *baiduProvider) appendResponse(responseBuilder *strings.Builder, responseBody string) {
	responseBuilder.WriteString(fmt.Sprintf("%s %s\n\n", streamDataItemKey, responseBody))
}
