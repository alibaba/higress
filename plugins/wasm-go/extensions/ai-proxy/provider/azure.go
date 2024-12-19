package provider

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
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

func (m *azureProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) error {
	if apiName != ApiNameChatCompletion {
		return errUnsupportedApiName
	}
	m.config.handleRequestHeaders(m, ctx, apiName, log)
	return nil
}

func (m *azureProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body, log)
}

func (m *azureProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header, log wrapper.Log) {
	u, e := url.Parse(ctx.Path())
	if e == nil {
		customApiVersion := u.Query().Get("api-version")
		if customApiVersion == "" {
			util.OverwriteRequestPathHeader(headers, m.serviceUrl.RequestURI())
		} else {
			q := m.serviceUrl.Query()
			q.Set("api-version", customApiVersion)
			newUrl := *m.serviceUrl
			newUrl.RawQuery = q.Encode()
			util.OverwriteRequestPathHeader(headers, newUrl.RequestURI())
		}
	} else {
		log.Errorf("failed to parse request path: %v", e)
		util.OverwriteRequestPathHeader(headers, m.serviceUrl.RequestURI())
	}
	util.OverwriteRequestHostHeader(headers, m.serviceUrl.Host)
	util.OverwriteRequestAuthorizationHeader(headers, "api-key "+m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")
}
