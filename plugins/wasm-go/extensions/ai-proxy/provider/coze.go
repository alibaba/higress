package provider

import (
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	cozeDomain = "api.coze.cn"
)

type cozeProviderInitializer struct{}

func (m *cozeProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *cozeProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{}
}

func (m *cozeProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(m.DefaultCapabilities())
	return &cozeProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type cozeProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *cozeProvider) GetProviderType() string {
	return providerTypeCoze
}

func (m *cozeProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	return nil
}

func (m *cozeProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	util.OverwriteRequestHostHeader(headers, cozeDomain)
	util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}
