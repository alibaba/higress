package provider

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-assistant/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	moonshotDefaultModel       = "moonshot-v1-8k"
	moonshotDefaultTemperature = 0.5

	moonshotChatCompletionRequestTemplate = `
{
    "model": "%s",
    "messages": [
        {
            "role": "system",
            "content": "你是 Kimi，由 Moonshot AI 提供的人工智能助手，你更擅长中文和英文的对话。你会为用户提供安全，有帮助，准确的回答。同时，你会拒绝一切涉及恐怖主义，种族歧视，黄色暴力等问题的回答。Moonshot AI 为专有名词，不可翻译成其他语言。"
        },
        {
            "role": "system",
            "content": "%s"
        },
        {
            "role": "user",
            "content": "%s"
        }
    ],
    "temperature": %.2f
}
`
)

// MoonshotProvider is the provider for Moonshot AI service.

type moonshotProvider struct {
	config ProviderConfig

	client wrapper.HttpClient

	fileContent string
}

func (m *moonshotProvider) ProcessChatRequest(ctx wrapper.HttpContext, prompt string, log wrapper.Log) (types.Action, error) {
	if m.fileContent != "" {
		err := m.performChatCompletion(ctx, m.fileContent, prompt, log)
		if err == nil {
			return types.ActionPause, nil
		}
		return types.ActionContinue, err
	}

	err := m.sendRequest(http.MethodGet, "/v1/files/"+m.config.fileId+"/content", "",
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			responseString := string(responseBody)
			if statusCode != http.StatusOK {
				log.Errorf("failed to load knowledge base file from AI service, status: %d body: %s", statusCode, responseString)
				_ = util.SendResponse(500, util.MimeTypeApplicationJson, fmt.Sprintf("failed to load knowledge base file from AI service, status: %d", statusCode))
				_ = proxywasm.ResumeHttpRequest()
				return
			}
			responseJson := gjson.Parse(responseString)
			base := responseJson.Get("content").String()
			err := m.performChatCompletion(ctx, base, prompt, log)
			if err != nil {
				_ = util.SendResponse(500, util.MimeTypeApplicationJson, fmt.Sprintf("failed to perform chat completion: %v", err))
				_ = proxywasm.ResumeHttpRequest()
				return
			}
		})
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (m *moonshotProvider) performChatCompletion(ctx wrapper.HttpContext, base string, prompt string, log wrapper.Log) error {
	model := m.config.model
	if model == "" {
		model = moonshotDefaultModel
	}
	requestBody := fmt.Sprintf(moonshotChatCompletionRequestTemplate, util.EscapeStringForJson(model),
		util.EscapeStringForJson(base), util.EscapeStringForJson(prompt), moonshotDefaultTemperature)
	return m.sendRequest(http.MethodPost, "/v1/chat/completions", requestBody,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			responseString := string(responseBody)
			if statusCode != http.StatusOK {
				log.Errorf("failed to get chat response from AI service, status: %d body: %s", statusCode, responseString)
				_ = util.SendResponse(500, util.MimeTypeApplicationJson, fmt.Sprintf("failed to get chat response from AI service, status: %d", statusCode))
				_ = proxywasm.ResumeHttpRequest()
				return
			}
			responseJson := gjson.Parse(responseString)
			choices := responseJson.Get("choices").Array()
			var aiResponse string
			if choices == nil || len(choices) == 0 {
				aiResponse = "AI service returned empty response"
			} else {
				aiResponse = choices[0].Get("message").Get("content").String()
			}
			_ = util.SendResponse(200, util.MimeTypeApplicationJson,
				fmt.Sprintf(chatResponseTemplate, util.EscapeStringForJson(aiResponse)))
			_ = proxywasm.ResumeHttpRequest()
		})
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
