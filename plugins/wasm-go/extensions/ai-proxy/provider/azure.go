package provider

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// azureProvider is the provider for Azure OpenAI service.

type azureProviderInitializer struct {
}

func (m *azureProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.azureServiceUrl == "" {
		return errors.New("missing azureServiceUrl in provider config")
	}
	if _, err := url.Parse(config.azureServiceUrl); err != nil {
		return fmt.Errorf("invalid azureServiceUrl: %w", err)
	}
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *azureProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	var serviceUrl *url.URL
	if u, err := url.Parse(config.azureServiceUrl); err != nil {
		return nil, fmt.Errorf("invalid azureServiceUrl: %w", err)
	} else {
		serviceUrl = u
	}
	return &azureProvider{
		config:       config,
		serviceUrl:   serviceUrl,
		contextCache: createContextCache(&config),
	}, nil
}

type azureProvider struct {
	config ProviderConfig

	contextCache *contextCache
	serviceUrl   *url.URL
}

func (m *azureProvider) GetProviderType() string {
	return providerTypeAzure
}

func (m *azureProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	_ = util.OverwriteRequestPath(m.serviceUrl.RequestURI())
	_ = util.OverwriteRequestHost(m.serviceUrl.Host)
	_ = proxywasm.ReplaceHttpRequestHeader("api-key", m.config.GetApiTokenInUse(ctx))
	if apiName == ApiNameChatCompletion {
		_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	} else {
		ctx.DontReadRequestBody()
	}
	return types.ActionContinue, nil
}

func (m *azureProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		// We don't need to process the request body for other APIs.
		return types.ActionContinue, nil
	}
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}
	if m.contextCache == nil {
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.openai.set_include_usage_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
		return types.ActionContinue, nil
	}
	err := m.contextCache.GetContent(func(content string, err error) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		if err != nil {
			log.Errorf("failed to load context file: %v", err)
			_ = util.SendResponse(500, "ai-proxy.azure.load_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to load context file: %v", err))
		}
		insertContextMessage(request, content)
		if err := replaceJsonRequestBody(request, log); err != nil {
			_ = util.SendResponse(500, "ai-proxy.azure.insert_ctx_failed", util.MimeTypeTextPlain, fmt.Sprintf("failed to replace request body: %v", err))
		}
	}, log)
	if err == nil {
		return types.ActionPause, nil
	}
	return types.ActionContinue, err
}
