package provider

import (
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// azureProvider is the provider for Azure OpenAI service.

type azureProviderInitializer struct {
}

func (m *azureProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.azureModelDeploymentName == "" {
		return errors.New("missing azureModelDeploymentName in provider config")
	}
	if config.azureApiVersion == "" {
		return errors.New("missing azureApiVersion in provider config")
	}
	return nil
}

func (m *azureProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &azureProvider{
		config: config,
	}, nil
}

type azureProvider struct {
	config ProviderConfig
}

func (m *azureProvider) GetPointcuts() map[Pointcut]interface{} {
	return map[Pointcut]interface{}{PointcutOnRequestHeaders: nil}
}

func (m *azureProvider) OnApiRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	path := fmt.Sprintf("/openai/deployments/%s/chat/completions?api-version=%s", m.config.azureModelDeploymentName, m.config.azureApiVersion)
	_ = util.OverwriteRequestPath(path)
	_ = util.OverwriteRequestHost(m.config.domain)
	_ = proxywasm.ReplaceHttpRequestHeader("api-key", m.config.apiToken)
	return types.ActionContinue, nil
}

func (m *azureProvider) OnApiRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}

func (m *azureProvider) OnApiResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}

func (m *azureProvider) OnApiResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}
