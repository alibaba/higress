package provider

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

// moonshotProvider is the provider for Moonshot AI service.

type moonshotProviderInitializer struct {
}

func (m *moonshotProviderInitializer) ValidateConfig(config ProviderConfig) error {
	return nil
}

func (m *moonshotProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	client, err := createClient(config)
	if err != nil {
		return nil, err
	}
	return &moonshotProvider{
		config: config,
		client: client,
	}, nil
}

type moonshotProvider struct {
	config ProviderConfig

	client      wrapper.HttpClient
	fileContent string
}

func (m *moonshotProvider) GetPointcuts() map[Pointcut]interface{} {
	if m.config.moonshotFileId != "" {
		return map[Pointcut]interface{}{PointcutOnRequestHeaders: nil, PointcutOnRequestBody: nil}
	}
	return map[Pointcut]interface{}{PointcutOnRequestHeaders: nil}
}

func (m *moonshotProvider) OnApiRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath("/v1/chat/completions")
	_ = util.OverwriteRequestHost(m.config.domain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.apiToken)
	if m.config.moonshotFileId != "" {
		_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	}
	return types.ActionContinue, nil
}

func (m *moonshotProvider) OnApiRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}

	if m.config.moonshotFileId == "" {
		return types.ActionContinue, nil
	}

	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}

	if m.fileContent != "" {
		err := m.performChatCompletion(ctx, m.fileContent, request, log)
		if err == nil {
			return types.ActionPause, nil
		}
		return types.ActionContinue, err
	}

	err := m.sendRequest(http.MethodGet, "/v1/files/"+m.config.moonshotFileId+"/content", "",
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			responseString := string(responseBody)
			if statusCode != http.StatusOK {
				log.Errorf("failed to load knowledge base file from AI service, status: %d body: %s", statusCode, responseString)
				_ = util.SendResponse(500, util.MimeTypeApplicationJson, fmt.Sprintf("failed to load knowledge base file from moonshot service, status: %d", statusCode))
				_ = proxywasm.ResumeHttpRequest()
				return
			}
			responseJson := gjson.Parse(responseString)
			base := responseJson.Get("content").String()
			err := m.performChatCompletion(ctx, base, request, log)
			if err != nil {
				_ = util.SendResponse(500, util.MimeTypeApplicationJson, fmt.Sprintf("failed to perform chat completion: %v", err))
			}
			_ = proxywasm.ResumeHttpRequest()
		})
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (m *moonshotProvider) performChatCompletion(ctx wrapper.HttpContext, fileContent string, request *chatCompletionRequest, log wrapper.Log) error {
	fileMessage := chatMessage{
		Role:    roleSystem,
		Content: fileContent,
	}
	firstNonSystemMessageIndex := -1
	for i, message := range request.Messages {
		if message.Role != roleSystem {
			firstNonSystemMessageIndex = i
			break
		}
	}
	if firstNonSystemMessageIndex == -1 {
		request.Messages = append(request.Messages, fileMessage)
	} else {
		request.Messages = append(request.Messages[:firstNonSystemMessageIndex], append([]chatMessage{fileMessage}, request.Messages[firstNonSystemMessageIndex:]...)...)
	}
	return replaceJsonRequestBody(request)
}

func (m *moonshotProvider) OnApiResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}

func (m *moonshotProvider) OnApiResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}

func (m *moonshotProvider) sendRequest(method, path string, body string, callback wrapper.ResponseCallback) error {
	timeout := m.config.timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	switch method {
	case http.MethodGet:
		headers := util.CreateHeaders("Authorization", "Bearer "+m.config.apiToken)
		return m.client.Get(path, headers, callback, timeout)
	case http.MethodPost:
		headers := util.CreateHeaders("Authorization", "Bearer "+m.config.apiToken, "Content-Type", "application/json")
		return m.client.Post(path, headers, []byte(body), callback, timeout)
	default:
		return errors.New("unsupported method: " + method)
	}
}
