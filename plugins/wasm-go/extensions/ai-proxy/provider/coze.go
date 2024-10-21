package provider

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	cozeDomain = "api.coze.cn"
)

type cozeProviderInitializer struct{}

func (m *cozeProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *cozeProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
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

func (m *cozeProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	_ = util.OverwriteRequestHost(cozeDomain)
	_ = util.OverwriteRequestAuthorization("Bearer " + m.config.GetRandomToken())
	return types.ActionContinue, nil
}
