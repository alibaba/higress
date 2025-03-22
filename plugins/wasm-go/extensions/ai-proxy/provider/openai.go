package provider

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// openaiProvider is the provider for OpenAI service.

const (
	defaultOpenaiDomain             = "api.openai.com"
	defaultOpenaiChatCompletionPath = "/v1/chat/completions"
	defaultOpenaiCompletionPath     = "/v1/completions"
	defaultOpenaiEmbeddingsPath     = "/v1/embeddings"
	defaultOpenaiAudioSpeech        = "/v1/audio/speech"
	defaultOpenaiImageGeneration    = "/v1/images/generations"
)

type openaiProviderInitializer struct {
}

func (m *openaiProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	return nil
}

func (m *openaiProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameCompletion):      defaultOpenaiCompletionPath,
		string(ApiNameChatCompletion):  defaultOpenaiChatCompletionPath,
		string(ApiNameEmbeddings):      defaultOpenaiEmbeddingsPath,
		string(ApiNameImageGeneration): defaultOpenaiImageGeneration,
		string(ApiNameAudioSpeech):     defaultOpenaiAudioSpeech,
	}
}

func isDirectPath(path string) bool {
	return strings.HasSuffix(path, "/completions") ||
		strings.HasSuffix(path, "/embeddings") ||
		strings.HasSuffix(path, "/audio/speech") ||
		strings.HasSuffix(path, "/images/generations")
}

func (m *openaiProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	if config.openaiCustomUrl == "" {
		config.setDefaultCapabilities(m.DefaultCapabilities())
		return &openaiProvider{
			config:       config,
			contextCache: createContextCache(&config),
		}, nil
	}
	customUrl := strings.TrimPrefix(strings.TrimPrefix(config.openaiCustomUrl, "http://"), "https://")
	pairs := strings.SplitN(customUrl, "/", 2)
	customPath := "/"
	if len(pairs) == 2 {
		customPath += pairs[1]
	}
	isDirectCustomPath := isDirectPath(customPath)
	capabilities := m.DefaultCapabilities()
	if !isDirectCustomPath {
		for key, mapPath := range capabilities {
			capabilities[key] = path.Join(customPath, strings.TrimPrefix(mapPath, "/v1"))
		}
	}
	config.setDefaultCapabilities(capabilities)
	log.Debugf("ai-proxy: openai provider customDomain:%s, customPath:%s, isDirectCustomPath:%v, capabilities:%v",
		pairs[0], customPath, isDirectCustomPath, capabilities)
	return &openaiProvider{
		config:             config,
		customDomain:       pairs[0],
		customPath:         customPath,
		isDirectCustomPath: isDirectCustomPath,
		contextCache:       createContextCache(&config),
	}, nil
}

type openaiProvider struct {
	config             ProviderConfig
	customDomain       string
	customPath         string
	isDirectCustomPath bool
	contextCache       *contextCache
}

func (m *openaiProvider) GetProviderType() string {
	return providerTypeOpenAI
}

func (m *openaiProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	m.config.handleRequestHeaders(m, ctx, apiName)
	return nil
}

func (m *openaiProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	if m.customPath != "" {
		if m.isDirectCustomPath || apiName == "" {
			util.OverwriteRequestPathHeader(headers, m.customPath)
		} else {
			util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
		}
	} else {
		util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	}
	if m.customDomain != "" {
		util.OverwriteRequestHostHeader(headers, m.customDomain)
	} else {
		util.OverwriteRequestHostHeader(headers, defaultOpenaiDomain)
	}
	if len(m.config.apiTokens) > 0 {
		util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+m.config.GetApiTokenInUse(ctx))
	}
	headers.Del("Content-Length")
}

func (m *openaiProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		// We don't need to process the request body for other APIs.
		return types.ActionContinue, nil
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *openaiProvider) TransformRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	if m.config.responseJsonSchema != nil {
		request := &chatCompletionRequest{}
		if err := decodeChatCompletionRequest(body, request); err != nil {
			return nil, err
		}
		log.Debugf("[ai-proxy] set response format to %s", m.config.responseJsonSchema)
		request.ResponseFormat = m.config.responseJsonSchema
		body, _ = json.Marshal(request)
	}
	return m.config.defaultTransformRequestBody(ctx, apiName, body)
}
