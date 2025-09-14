package provider

import (
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// cerebrasProvider is the provider for Cerebras service.

const (
	defaultCerebrasDomain = "api.cerebras.ai"
)

type cerebrasProviderInitializer struct{}

func (m *cerebrasProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in Cerebras provider config")
	}
	return nil
}

func (m *cerebrasProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): PathOpenAIChatCompletions,
		string(ApiNameModels):         PathOpenAIModels,
	}
}

func (m *cerebrasProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	if config.openaiCustomUrl != "" {
		// Handle custom URL like OpenAI
		customUrl := strings.TrimPrefix(strings.TrimPrefix(config.openaiCustomUrl, "http://"), "https://")
		pairs := strings.SplitN(customUrl, "/", 2)
		customPath := "/"
		if len(pairs) == 2 {
			customPath += pairs[1]
		}
		capabilities := m.DefaultCapabilities()
		for key, mapPath := range capabilities {
			capabilities[key] = path.Join(customPath, strings.TrimPrefix(mapPath, "/v1"))
		}
		config.setDefaultCapabilities(capabilities)
		log.Debugf("ai-proxy: cerebras provider customDomain:%s, customPath:%s, capabilities:%v",
			pairs[0], customPath, capabilities)
		return &cerebrasProvider{
			config:       config,
			customDomain: pairs[0],
			customPath:   customPath,
		}, nil
	}

	// Set default capabilities
	config.setDefaultCapabilities(m.DefaultCapabilities())

	return &cerebrasProvider{
		config: config,
	}, nil
}

type cerebrasProvider struct {
	config       ProviderConfig
	customDomain string
	customPath   string
}

func (p *cerebrasProvider) GetProviderType() string {
	return providerTypeCerebras
}

func (p *cerebrasProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	p.config.handleRequestHeaders(p, ctx, apiName)
	return nil
}

func (p *cerebrasProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	if p.customPath != "" {
		util.OverwriteRequestPathHeader(headers, p.customPath)
	} else if apiName != "" {
		util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), p.config.capabilities)
	}

	if p.customDomain != "" {
		util.OverwriteRequestHostHeader(headers, p.customDomain)
	} else {
		util.OverwriteRequestHostHeader(headers, defaultCerebrasDomain)
	}
	if len(p.config.apiTokens) > 0 {
		util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+p.config.GetApiTokenInUse(ctx))
	}
	headers.Del("Content-Length")
}

func (p *cerebrasProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !p.config.needToProcessRequestBody(apiName) {
		// We don't need to process the request body for other APIs.
		return types.ActionContinue, nil
	}
	return p.config.handleRequestBody(p, nil, ctx, apiName, body)
}
