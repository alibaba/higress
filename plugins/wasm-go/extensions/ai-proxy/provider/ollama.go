package provider

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// ollamaProvider is the provider for Ollama service.

type ollamaProviderInitializer struct {
}

func (m *ollamaProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.ollamaServerHost == "" {
		return errors.New("missing ollamaServerHost in provider config")
	}
	if config.ollamaServerPort == 0 {
		return errors.New("missing ollamaServerPort in provider config")
	}
	return nil
}

func (m *ollamaProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		// ollama的chat接口path和OpenAI的chat接口一样
		string(ApiNameChatCompletion): PathOpenAIChatCompletions,
		string(ApiNameEmbeddings):     PathOpenAIEmbeddings,
	}
}

func (m *ollamaProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	serverPortStr := fmt.Sprintf("%d", config.ollamaServerPort)
	serviceDomain := config.ollamaServerHost + ":" + serverPortStr
	config.setDefaultCapabilities(m.DefaultCapabilities())
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

func (m *ollamaProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	return nil
}

func (m *ollamaProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !m.config.isSupportedAPI(apiName) {
		return types.ActionContinue, nil
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *ollamaProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	util.OverwriteRequestHostHeader(headers, m.serviceDomain)
	headers.Del("Content-Length")
}
