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
)

// minimaxProvider is the provider for minimax service.

const (
	minimaxDomain = "api.minimax.chat"
	// minimaxChatCompletionV2Path 接口请求响应格式与OpenAI相同
	// 接口文档: https://platform.minimaxi.com/document/guides/chat-model/V2?id=65e0736ab2845de20908e2dd
	minimaxChatCompletionV2Path = "/v1/text/chatcompletion_v2"
	// minimaxChatCompletionProPath 接口请求响应格式与OpenAI不同
	// 接口文档: https://platform.minimaxi.com/document/guides/chat-model/pro/api?id=6569c85948bc7b684b30377e
	minimaxChatCompletionProPath = "/v1/text/chatcompletion_pro"

	senderTypeUser string = "USER" // 用户发送的内容
	senderTypeBot  string = "BOT"  // 模型生成的内容

	// 默认机器人设置
	defaultBotName           string = "MM智能助理"
	defaultBotSettingContent string = "MM智能助理是一款由MiniMax自研的，没有调用其他产品的接口的大型语言模型。MiniMax是一家中国科技公司，一直致力于进行大模型相关的研究。"
	defaultSenderName        string = "小明"
)

// chatCompletionProModels 这些模型对应接口为ChatCompletion Pro
var chatCompletionProModels = map[string]struct{}{
	"abab6.5-chat":  {},
	"abab6.5s-chat": {},
	"abab5.5s-chat": {},
	"abab5.5-chat":  {},
}

type minimaxProviderInitializer struct {
}

func (m *minimaxProviderInitializer) ValidateConfig(config ProviderConfig) error {
	// 如果存在模型对应接口为ChatCompletion Pro必须配置minimaxGroupId
	if len(config.modelMapping) > 0 && config.minimaxGroupId == "" {
		for _, minimaxModel := range config.modelMapping {
			if _, exists := chatCompletionProModels[minimaxModel]; exists {
				return errors.New(fmt.Sprintf("missing minimaxGroupId in provider config when %s model is provided", minimaxModel))
			}
		}
	}
	return nil
}

func (m *minimaxProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &minimaxProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type minimaxProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *minimaxProvider) GetProviderType() string {
	return providerTypeMinimax
}

func (m *minimaxProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestHost(minimaxDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")

	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return types.HeaderStopIteration, nil
}

func (m *minimaxProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	// 解析并映射模型,设置上下文
	model, err := m.parseModel(body)
	if err != nil {
		return types.ActionContinue, err
	}
	ctx.SetContext(ctxKeyOriginalRequestModel, model)
	mappedModel := getMappedModel(model, m.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	ctx.SetContext(ctxKeyFinalRequestModel, mappedModel)
	_, ok := chatCompletionProModels[mappedModel]
	if ok {
		// 使用ChatCompletion Pro接口
		return m.handleRequestBodyByChatCompletionPro(body, log)
	} else {
		// 使用ChatCompletion v2接口
		return m.handleRequestBodyByChatCompletionV2(body, log)
	}
}

// handleRequestBodyByChatCompletionPro 使用ChatCompletion Pro接口处理请求体
func (m *minimaxProvider) handleRequestBodyByChatCompletionPro(body []byte, log wrapper.Log) (types.Action, error) {
	// 使用minimax接口协议
	if m.config.protocol == protocolOriginal {
		request := &minimaxChatCompletionV2Request{}
		if err := json.Unmarshal(body, request); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
		}
		if request.Model == "" {
			return types.ActionContinue, errors.New("request model is empty")
		}
		// 根据模型重写requestPath
		if m.config.minimaxGroupId == "" {
			return types.ActionContinue, errors.New(fmt.Sprintf("missing minimaxGroupId in provider config when use %s model ", request.Model))
		}
		_ = util.OverwriteRequestPath(fmt.Sprintf("%s?GroupId=%s", minimaxChatCompletionProPath, m.config.minimaxGroupId))

		if m.config.context == nil {
			return types.ActionContinue, nil
		}

		err := m.contextCache.GetContent(func(content string, err error) {
			defer func() {
				_ = proxywasm.ResumeHttpRequest()
			}()

			if err != nil {
				log.Errorf("failed to load context file: %v", err)
				_ = util.SendResponse(500, "ai-proxy.minimax.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
			}
			m.setBotSettings(request, content)
			if err := replaceJsonRequestBody(request, log); err != nil {
				_ = util.SendResponse(500, "ai-proxy.minimax.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
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
	request.Model = getMappedModel(request.Model, m.config.modelMapping, log)
	_ = util.OverwriteRequestPath(fmt.Sprintf("%s?GroupId=%s", minimaxChatCompletionProPath, m.config.minimaxGroupId))

	if m.config.context == nil {
		minimaxRequest := m.buildMinimaxChatCompletionV2Request(request, "")
		return types.ActionContinue, replaceJsonRequestBody(minimaxRequest, log)
	}

	err := m.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.minimax.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		minimaxRequest := m.buildMinimaxChatCompletionV2Request(request, content)
		if err := replaceJsonRequestBody(minimaxRequest, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.minimax.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace Request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

// handleRequestBodyByChatCompletionV2 使用ChatCompletion v2接口处理请求体
func (m *minimaxProvider) handleRequestBodyByChatCompletionV2(body []byte, log wrapper.Log) (types.Action, error) {
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}

	// 映射模型重写requestPath
	request.Model = getMappedModel(request.Model, m.config.modelMapping, log)
	_ = util.OverwriteRequestPath(minimaxChatCompletionV2Path)

	if m.contextCache == nil {
		return types.ActionContinue, replaceJsonRequestBody(request, log)
	}

	err := m.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.minimax.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.minimax.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (m *minimaxProvider) OnResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	// 使用minimax接口协议,跳过OnStreamingResponseBody()和OnResponseBody()
	if m.config.protocol == protocolOriginal {
		ctx.DontReadResponseBody()
		return types.ActionContinue, nil
	}
	// 模型对应接口为ChatCompletion v2,跳过OnStreamingResponseBody()和OnResponseBody()
	model := ctx.GetContext(ctxKeyFinalRequestModel)
	if model != nil {
		_, ok := chatCompletionProModels[model.(string)]
		if !ok {
			ctx.DontReadResponseBody()
			return types.ActionContinue, nil
		}
	}
	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	return types.ActionContinue, nil
}

// OnStreamingResponseBody 只处理使用OpenAI协议 且 模型对应接口为ChatCompletion Pro的流式响应
func (m *minimaxProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	if isLastChunk || len(chunk) == 0 {
		return nil, nil
	}
	// sample event response:
	// data: {"created":1689747645,"model":"abab6.5s-chat","reply":"","choices":[{"messages":[{"sender_type":"BOT","sender_name":"MM智能助理","text":"am from China."}]}],"output_sensitive":false}

	// sample end event response:
	// data: {"created":1689747645,"model":"abab6.5s-chat","reply":"I am from China.","choices":[{"finish_reason":"stop","messages":[{"sender_type":"BOT","sender_name":"MM智能助理","text":"I am from China."}]}],"usage":{"total_tokens":187},"input_sensitive":false,"output_sensitive":false,"id":"0106b3bc9fd844a9f3de1aa06004e2ab","base_resp":{"status_code":0,"status_msg":""}}
	responseBuilder := &strings.Builder{}
	lines := strings.Split(string(chunk), "\n")
	for _, data := range lines {
		if len(data) < 6 {
			// ignore blank line or wrong format
			continue
		}
		data = data[6:]
		var minimaxResp minimaxChatCompletionV2Resp
		if err := json.Unmarshal([]byte(data), &minimaxResp); err != nil {
			log.Errorf("unable to unmarshal minimax response: %v", err)
			continue
		}
		response := m.responseV2ToOpenAI(&minimaxResp)
		responseBody, err := json.Marshal(response)
		if err != nil {
			log.Errorf("unable to marshal response: %v", err)
			return nil, err
		}
		m.appendResponse(responseBuilder, string(responseBody))
	}
	modifiedResponseChunk := responseBuilder.String()
	log.Debugf("=== modified response chunk: %s", modifiedResponseChunk)
	return []byte(modifiedResponseChunk), nil
}

// OnResponseBody 只处理使用OpenAI协议 且 模型对应接口为ChatCompletion Pro的流式响应
func (m *minimaxProvider) OnResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	minimaxResp := &minimaxChatCompletionV2Resp{}
	if err := json.Unmarshal(body, minimaxResp); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal minimax response: %v", err)
	}
	if minimaxResp.BaseResp.StatusCode != 0 {
		return types.ActionContinue, fmt.Errorf("minimax response error, error_code: %d, error_message: %s", minimaxResp.BaseResp.StatusCode, minimaxResp.BaseResp.StatusMsg)
	}
	response := m.responseV2ToOpenAI(minimaxResp)
	return types.ActionContinue, replaceJsonResponseBody(response, log)
}

// minimaxChatCompletionV2Request 表示ChatCompletion V2请求的结构体
type minimaxChatCompletionV2Request struct {
	Model             string                  `json:"model"`
	Stream            bool                    `json:"stream,omitempty"`
	TokensToGenerate  int64                   `json:"tokens_to_generate,omitempty"`
	Temperature       float64                 `json:"temperature,omitempty"`
	TopP              float64                 `json:"top_p,omitempty"`
	MaskSensitiveInfo bool                    `json:"mask_sensitive_info"` // 是否开启隐私信息打码,默认true
	Messages          []minimaxMessage        `json:"messages"`
	BotSettings       []minimaxBotSetting     `json:"bot_setting"`
	ReplyConstraints  minimaxReplyConstraints `json:"reply_constraints"`
}

// minimaxMessage 表示对话中的消息
type minimaxMessage struct {
	SenderType string `json:"sender_type"`
	SenderName string `json:"sender_name"`
	Text       string `json:"text"`
}

// minimaxBotSetting 表示机器人的设置
type minimaxBotSetting struct {
	BotName string `json:"bot_name"`
	Content string `json:"content"`
}

// minimaxReplyConstraints 表示模型回复要求
type minimaxReplyConstraints struct {
	SenderType string `json:"sender_type"`
	SenderName string `json:"sender_name"`
}

// minimaxChatCompletionV2Resp Minimax Chat Completion V2响应结构体
type minimaxChatCompletionV2Resp struct {
	Created             int64           `json:"created"`
	Model               string          `json:"model"`
	Reply               string          `json:"reply"`
	InputSensitive      bool            `json:"input_sensitive,omitempty"`
	InputSensitiveType  int64           `json:"input_sensitive_type,omitempty"`
	OutputSensitive     bool            `json:"output_sensitive,omitempty"`
	OutputSensitiveType int64           `json:"output_sensitive_type,omitempty"`
	Choices             []minimaxChoice `json:"choices,omitempty"`
	Usage               minimaxUsage    `json:"usage,omitempty"`
	Id                  string          `json:"id"`
	BaseResp            minimaxBaseResp `json:"base_resp"`
}

// minimaxBaseResp 包含错误状态码和详情
type minimaxBaseResp struct {
	StatusCode int64  `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

// minimaxChoice 结果选项
type minimaxChoice struct {
	Messages     []minimaxMessage `json:"messages"`
	Index        int64            `json:"index"`
	FinishReason string           `json:"finish_reason"`
}

// minimaxUsage 令牌使用情况
type minimaxUsage struct {
	TotalTokens int64 `json:"total_tokens"`
}

func (m *minimaxProvider) parseModel(body []byte) (string, error) {
	var tempMap map[string]interface{}
	if err := json.Unmarshal(body, &tempMap); err != nil {
		return "", err
	}
	model, ok := tempMap["model"].(string)
	if !ok {
		return "", errors.New("missing model in chat completion request")
	}
	return model, nil
}

func (m *minimaxProvider) setBotSettings(request *minimaxChatCompletionV2Request, botSettingContent string) {
	if len(request.BotSettings) == 0 {
		request.BotSettings = []minimaxBotSetting{
			{
				BotName: defaultBotName,
				Content: func() string {
					if botSettingContent != "" {
						return botSettingContent
					}
					return defaultBotSettingContent
				}(),
			},
		}
	} else if botSettingContent != "" {
		newSetting := minimaxBotSetting{
			BotName: request.BotSettings[0].BotName,
			Content: botSettingContent,
		}
		request.BotSettings = append([]minimaxBotSetting{newSetting}, request.BotSettings...)
	}
}

func (m *minimaxProvider) buildMinimaxChatCompletionV2Request(request *chatCompletionRequest, botSettingContent string) *minimaxChatCompletionV2Request {
	var messages []minimaxMessage
	var botSetting []minimaxBotSetting
	var botName string

	determineName := func(name string, defaultName string) string {
		if name != "" {
			return name
		}
		return defaultName
	}

	for _, message := range request.Messages {
		switch message.Role {
		case roleSystem:
			botName = determineName(message.Name, defaultBotName)
			botSetting = append(botSetting, minimaxBotSetting{
				BotName: botName,
				Content: message.Content,
			})
		case roleAssistant:
			messages = append(messages, minimaxMessage{
				SenderType: senderTypeBot,
				SenderName: determineName(message.Name, defaultBotName),
				Text:       message.Content,
			})
		case roleUser:
			messages = append(messages, minimaxMessage{
				SenderType: senderTypeUser,
				SenderName: determineName(message.Name, defaultSenderName),
				Text:       message.Content,
			})
		}
	}

	replyConstraints := minimaxReplyConstraints{
		SenderType: senderTypeBot,
		SenderName: determineName(botName, defaultBotName),
	}
	result := &minimaxChatCompletionV2Request{
		Model:             request.Model,
		Stream:            request.Stream,
		TokensToGenerate:  int64(request.MaxTokens),
		Temperature:       request.Temperature,
		TopP:              request.TopP,
		MaskSensitiveInfo: true,
		Messages:          messages,
		BotSettings:       botSetting,
		ReplyConstraints:  replyConstraints,
	}

	m.setBotSettings(result, botSettingContent)
	return result
}

func (m *minimaxProvider) responseV2ToOpenAI(response *minimaxChatCompletionV2Resp) *chatCompletionResponse {
	var choices []chatCompletionChoice
	messageIndex := 0
	for _, choice := range response.Choices {
		for _, message := range choice.Messages {
			message := &chatMessage{
				Name:    message.SenderName,
				Role:    roleAssistant,
				Content: message.Text,
			}
			choices = append(choices, chatCompletionChoice{
				FinishReason: choice.FinishReason,
				Index:        messageIndex,
				Message:      message,
			})
			messageIndex++
		}
	}
	return &chatCompletionResponse{
		Id:      response.Id,
		Object:  objectChatCompletion,
		Created: response.Created,
		Model:   response.Model,
		Choices: choices,
		Usage: usage{
			TotalTokens: int(response.Usage.TotalTokens),
		},
	}
}

func (m *minimaxProvider) appendResponse(responseBuilder *strings.Builder, responseBody string) {
	responseBuilder.WriteString(fmt.Sprintf("%s %s\n\n", streamDataItemKey, responseBody))
}
