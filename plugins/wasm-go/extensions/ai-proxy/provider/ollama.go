package provider

import (
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"net/http"
)

// ollamaProvider is the provider for Ollama service.

const (
	ollamaChatCompletionPath = "/v1/chat/completions"
)

type ollamaProviderInitializer struct {
}

func (m *ollamaProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.ollamaServerHost == "" {
		return errors.New("missing ollamaServerHost in provider config")
	}
	if config.ollamaServerPort == 0 {
		return errors.New("missing ollamaServerPort in provider config")
	}
	return nil
}

func (m *ollamaProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	serverPortStr := fmt.Sprintf("%d", config.ollamaServerPort)
	serviceDomain := config.ollamaServerHost + ":" + serverPortStr
	return &ollamaProvider{
		config:        config,
		serviceDomain: serviceDomain,
		contextCache:  createContextCache(&config),
	}, nil
}

type ollamaProvider struct {
	config        ProviderConfig
	serviceDomain string
	contextCache  *contextCache
}

func (m *ollamaProvider) GetProviderType() string {
	return providerTypeOllama
}

func (m *ollamaProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return types.ActionContinue, nil
}

func (m *ollamaProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *ollamaProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestPathHeader(headers, ollamaChatCompletionPath)
	util.OverwriteRequestHostHeader(headers, m.serviceDomain)
	headers.Del("Content-Length")
}
