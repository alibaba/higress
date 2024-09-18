package provider

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	bailianDomain = "dashscope.aliyuncs.com"
)

type baiLianProviderInitializer struct{}

func (m *baiLianProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *baiLianProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &baiLianProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type baiLianProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *baiLianProvider) GetProviderType() string {
	return providerTypeBailian
}

func (m *baiLianProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameAgent {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestHost(bailianDomain)
	_ = util.OverwriteRequestAuthorization("Bearer " + m.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	return types.ActionContinue, nil
}
