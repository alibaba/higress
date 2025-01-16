package provider

import (
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// ai360Provider is the provider for 360 OpenAI service.
const (
	ai360Domain = "api.360.cn"
)

type ai360ProviderInitializer struct {
}

type ai360Provider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *ai360ProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *ai360ProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &ai360Provider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

func (m *ai360Provider) GetProviderType() string {
	return providerTypeAi360
}

func (m *ai360Provider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion && apiName != ApiNameEmbeddings {
		return errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	// Delay the header processing to allow changing streaming mode in OnRequestBody
	return nil
}

func (m *ai360Provider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion && apiName != ApiNameEmbeddings {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *ai360Provider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	util.OverwriteRequestHostHeader(headers, ai360Domain)
	util.OverwriteRequestAuthorizationHeader(headers, m.config.GetApiTokenInUse(ctx))
}
