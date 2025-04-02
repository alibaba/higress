package provider

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// moonshotProvider is the provider for Moonshot AI service.

const (
	moonshotDomain             = "api.moonshot.cn"
	moonshotChatCompletionPath = "/v1/chat/completions"
)

type moonshotProviderInitializer struct {
}

func (m *moonshotProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.moonshotFileId != "" && config.context != nil {
		return errors.New("moonshotFileId and context cannot be configured at the same time")
	}
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *moonshotProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): moonshotChatCompletionPath,
	}
}

func (m *moonshotProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(m.DefaultCapabilities())
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

func (m *moonshotProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	return nil
}

func (m *moonshotProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestHostHeader(headers, moonshotDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}

// moonshot 有自己获取 context 的配置（moonshotFileId），因此无法复用 handleRequestBody 方法
// moonshot 的 body 没有修改，无须实现TransformRequestBody，使用默认的 defaultTransformRequestBody 方法
func (m *moonshotProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	// 非chat类型的请求，不做处理
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, nil
	}

	request := &chatCompletionRequest{}
	if err := m.config.parseRequestAndMapModel(ctx, request, body); err != nil {
		return types.ActionContinue, err
	}

	if m.config.moonshotFileId == "" && m.contextCache == nil {
		return types.ActionContinue, replaceJsonRequestBody(request)
	}

	apiKey := m.config.GetOrSetTokenWithContext(ctx)
	err := m.getContextContent(apiKey, func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.ErrorHandler("ai-proxy.moonshot.load_ctx_failed", fmt.Errorf("failed to load context file: %v", err))
			return
		}
		err = m.performChatCompletion(ctx, content, request)
		if err != nil {
			_ = util.ErrorHandler("ai-proxy.moonshot.insert_ctx_failed", fmt.Errorf("failed to perform chat completion: %v", err))
		}
	})
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}

func (m *moonshotProvider) performChatCompletion(ctx wrapper.HttpContext, fileContent string, request *chatCompletionRequest) error {
	insertContextMessage(request, fileContent)
	return replaceJsonRequestBody(request)
}

func (m *moonshotProvider) getContextContent(apiKey string, callback func(string, error)) error {
	if m.config.moonshotFileId != "" {
		if m.fileContent != "" {
			callback(m.fileContent, nil)
			return nil
		}
		return m.sendRequest(http.MethodGet, "/v1/files/"+m.config.moonshotFileId+"/content", "", apiKey,
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
		return m.contextCache.GetContent(callback)
	}

	return errors.New("both moonshotFileId and context are not configured")
}

func (m *moonshotProvider) sendRequest(method, path, body, apiKey string, callback wrapper.ResponseCallback) error {
	switch method {
	case http.MethodGet:
		headers := util.CreateHeaders("Authorization", "Bearer "+apiKey)
		return m.client.Get(path, headers, callback, m.config.timeout)
	case http.MethodPost:
		headers := util.CreateHeaders("Authorization", "Bearer "+apiKey, "Content-Type", "application/json")
		return m.client.Post(path, headers, []byte(body), callback, m.config.timeout)
	default:
		return errors.New("unsupported method: " + method)
	}
}

func (m *moonshotProvider) OnStreamingEvent(ctx wrapper.HttpContext, name ApiName, event StreamEvent) ([]StreamEvent, error) {
	if name != ApiNameChatCompletion {
		return nil, nil
	}

	if gjson.Get(event.Data, "choices.0.usage").Exists() {
		usageStr := gjson.Get(event.Data, "choices.0.usage").Raw
		newData, err := sjson.Delete(event.Data, "choices.0.usage")
		if err != nil {
			log.Errorf("convert usage event error: %v", err)
			return nil, err
		}
		newData, err = sjson.SetRaw(newData, "usage", usageStr)
		if err != nil {
			log.Errorf("convert usage event error: %v", err)
			return nil, err
		}
		event.Data = newData
	}
	return []StreamEvent{event}, nil
}

func (m *moonshotProvider) appendStreamEvent(responseBuilder *strings.Builder, event *StreamEvent) {
	responseBuilder.WriteString(streamDataItemKey)
	responseBuilder.WriteString(event.Data)
	responseBuilder.WriteString("\n\n")
}
