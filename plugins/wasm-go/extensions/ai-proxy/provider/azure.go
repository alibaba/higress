package provider

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	pathAzureFiles   = "/openai/files"
	pathAzureBatches = "/openai/batches"
)

// azureProvider is the provider for Azure OpenAI service.
type azureProviderInitializer struct {
}

func (m *azureProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		// TODO: azure's pattern is the same as openai, just need to handle the prefix, can be done in TransformRequestHeaders to support general capabilities
		string(ApiNameChatCompletion): PathOpenAIChatCompletions,
		string(ApiNameEmbeddings):     PathOpenAIEmbeddings,
		string(ApiNameFiles):          PathOpenAIFiles,
		string(ApiNameBatches):        PathOpenAIBatches,
	}
}

func (m *azureProviderInitializer) ValidateConfig(config *ProviderConfig) error {
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
	config.setDefaultCapabilities(m.DefaultCapabilities())
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

func (m *azureProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	return nil
}

func (m *azureProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *azureProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	finalRequestUrl := *m.serviceUrl
	if u, e := url.Parse(ctx.Path()); e == nil {
		if len(u.Query()) != 0 {
			q := m.serviceUrl.Query()
			for k, v := range u.Query() {
				switch len(v) {
				case 0:
					break
				case 1:
					q.Set(k, v[0])
					break
				default:
					delete(q, k)
					for _, vv := range v {
						q.Add(k, vv)
					}
				}
			}
			finalRequestUrl.RawQuery = q.Encode()
		}

		if filesIndex := strings.Index(u.Path, "/files"); filesIndex != -1 {
			finalRequestUrl.Path = pathAzureFiles + u.Path[filesIndex+len("/files"):]
		} else if batchesIndex := strings.Index(u.Path, "/batches"); batchesIndex != -1 {
			finalRequestUrl.Path = pathAzureBatches + u.Path[batchesIndex+len("/batches"):]
		}
	} else {
		log.Errorf("failed to parse request path: %v", e)
	}
	util.OverwriteRequestPathHeader(headers, finalRequestUrl.RequestURI())

	util.OverwriteRequestHostHeader(headers, m.serviceUrl.Host)
	headers.Set("api-key", m.config.GetApiTokenInUse(ctx))
	headers.Del("Content-Length")

	if !m.config.isSupportedAPI(apiName) {
		// If the API is not supported, we should not read the request body and keep it as it is.
		ctx.DontReadRequestBody()
	}
}
