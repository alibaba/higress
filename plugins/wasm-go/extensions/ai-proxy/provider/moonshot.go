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

const (
	moonshotDomain             = "api.moonshot.cn"
	moonshotChatCompletionPath = "/v1/chat/completions"
)

type moonshotProviderInitializer struct {
}

func (m *moonshotProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.moonshotFileId != "" && config.context != nil {
		return errors.New("moonshotFileId and context cannot be configured at the same time")
	}
	return nil
}

func (m *moonshotProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &moonshotProvider{
		config: config,
		client: wrapper.NewClusterClient(wrapper.RouteCluster{
			Host: moonshotDomain,
		}),
		contextCache: createContextCache(&config),
	}, nil
}

type moonshotProvider struct {
	config ProviderConfig

	client       wrapper.HttpClient
	fileContent  string
	contextCache *contextCache
}

func (m *moonshotProvider) GetProviderType() string {
	return providerTypeMoonshot
}

func (m *moonshotProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(moonshotChatCompletionPath)
	_ = util.OverwriteRequestHost(moonshotDomain)
	_ = proxywasm.ReplaceHttpRequestHeader("Authorization", "Bearer "+m.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	return types.ActionContinue, nil
}

func (m *moonshotProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}

	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}

	model := request.Model
	if model == "" {
		return types.ActionContinue, errors.New("missing model in chat completion request")
	}
	mappedModel := getMappedModel(model, m.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel

	if m.config.moonshotFileId == "" && m.contextCache == nil {
		return types.ActionContinue, replaceJsonRequestBody(request, log)
	}

	err := m.getContextContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.moonshot.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
			return
		}
		err = m.performChatCompletion(ctx, content, request, log)
		if err != nil {
			_ = util.SendResponse(500, "ai-proxy.moonshot.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to perform chat completion: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (m *moonshotProvider) performChatCompletion(ctx wrapper.HttpContext, fileContent string, request *chatCompletionRequest, log wrapper.Log) error {
	insertContextMessage(request, fileContent)
	return replaceJsonRequestBody(request, log)
}

func (m *moonshotProvider) getContextContent(callback func(string, error), log wrapper.Log) error {
	if m.config.moonshotFileId != "" {
		if m.fileContent != "" {
			callback(m.fileContent, nil)
			return nil
		}
		return m.sendRequest(http.MethodGet, "/v1/files/"+m.config.moonshotFileId+"/content", "",
			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
				responseString := string(responseBody)
				if statusCode != http.StatusOK {
					log.Errorf("failed to load knowledge base file from AI service, status: %d body: %s", statusCode, responseString)
					callback("", fmt.Errorf("failed to load knowledge base file from moonshot service, status: %d", statusCode))
					return
				}
				responseJson := gjson.Parse(responseString)
				m.fileContent = responseJson.Get("content").String()
				callback(m.fileContent, nil)
			})
	}

	if m.contextCache != nil {
		return m.contextCache.GetContent(callback, log)
	}

	return errors.New("both moonshotFileId and context are not configured")
}

func (m *moonshotProvider) sendRequest(method, path string, body string, callback wrapper.ResponseCallback) error {
	switch method {
	case http.MethodGet:
		headers := util.CreateHeaders("Authorization", "Bearer "+m.config.GetRandomToken())
		return m.client.Get(path, headers, callback, m.config.timeout)
	case http.MethodPost:
		headers := util.CreateHeaders("Authorization", "Bearer "+m.config.GetRandomToken(), "Content-Type", "application/json")
		return m.client.Post(path, headers, []byte(body), callback, m.config.timeout)
	default:
		return errors.New("unsupported method: " + method)
	}
}
