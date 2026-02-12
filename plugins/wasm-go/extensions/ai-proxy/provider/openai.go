package provider

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// openaiProvider is the provider for OpenAI service.

const (
	defaultOpenaiDomain = "api.openai.com"
	defaultCodexDomain  = "chatgpt.com"
	defaultCodexPath    = "/backend-api"
)

// Context keys for Codex mode
const (
	ctxKeyCodexMode        = "codexMode"
	ctxKeyCodexAccountId   = "codexAccountId"
	ctxKeyCodexOriginator  = "codexOriginator"
	ctxKeyCodexInstructions = "codexInstructions"
)

type openaiProviderInitializer struct{}

func (m *openaiProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	return nil
}

func (m *openaiProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameCompletion):                           PathOpenAICompletions,
		string(ApiNameChatCompletion):                       PathOpenAIChatCompletions,
		string(ApiNameEmbeddings):                           PathOpenAIEmbeddings,
		string(ApiNameImageGeneration):                      PathOpenAIImageGeneration,
		string(ApiNameImageEdit):                            PathOpenAIImageEdit,
		string(ApiNameImageVariation):                       PathOpenAIImageVariation,
		string(ApiNameAudioSpeech):                          PathOpenAIAudioSpeech,
		string(ApiNameModels):                               PathOpenAIModels,
		string(ApiNameFiles):                                PathOpenAIFiles,
		string(ApiNameRetrieveFile):                         PathOpenAIRetrieveFile,
		string(ApiNameRetrieveFileContent):                  PathOpenAIRetrieveFileContent,
		string(ApiNameBatches):                              PathOpenAIBatches,
		string(ApiNameRetrieveBatch):                        PathOpenAIRetrieveBatch,
		string(ApiNameCancelBatch):                          PathOpenAICancelBatch,
		string(ApiNameResponses):                            PathOpenAIResponses,
		string(ApiNameFineTuningJobs):                       PathOpenAIFineTuningJobs,
		string(ApiNameRetrieveFineTuningJob):                PathOpenAIRetrieveFineTuningJob,
		string(ApiNameFineTuningJobEvents):                  PathOpenAIFineTuningJobEvents,
		string(ApiNameFineTuningJobCheckpoints):             PathOpenAIFineTuningJobCheckpoints,
		string(ApiNameCancelFineTuningJob):                  PathOpenAICancelFineTuningJob,
		string(ApiNameResumeFineTuningJob):                  PathOpenAIResumeFineTuningJob,
		string(ApiNamePauseFineTuningJob):                   PathOpenAIPauseFineTuningJob,
		string(ApiNameFineTuningCheckpointPermissions):      PathOpenAIFineTuningCheckpointPermissions,
		string(ApiNameDeleteFineTuningCheckpointPermission): PathOpenAIFineDeleteTuningCheckpointPermission,
		string(ApiNameVideos):                               PathOpenAIVideos,
		string(ApiNameRetrieveVideo):                        PathOpenAIRetrieveVideo,
		string(ApiNameVideoRemix):                           PathOpenAIVideoRemix,
		string(ApiNameRetrieveVideoContent):                 PathOpenAIRetrieveVideoContent,
	}
}

// isDirectPath checks if the path is a known standard OpenAI interface path.
func isDirectPath(path string) bool {
	return strings.HasSuffix(path, "/completions") ||
		strings.HasSuffix(path, "/embeddings") ||
		strings.HasSuffix(path, "/audio/speech") ||
		strings.HasSuffix(path, "/images/generations") ||
		strings.HasSuffix(path, "/images/variations") ||
		strings.HasSuffix(path, "/images/edits") ||
		strings.HasSuffix(path, "/models") ||
		strings.HasSuffix(path, "/responses") ||
		strings.HasSuffix(path, "/fine_tuning/jobs") ||
		strings.HasSuffix(path, "/fine_tuning/checkpoints") ||
		strings.HasSuffix(path, "/videos")
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
	// Handle Codex mode path rewriting
	if m.config.codexMode && apiName == ApiNameChatCompletion {
		// Codex API uses /codex/responses endpoint instead of /v1/chat/completions
		if m.customDomain != "" {
			util.OverwriteRequestPathHeader(headers, PathCodexResponses)
		} else {
			// Use default Codex path under backend-api
			util.OverwriteRequestPathHeader(headers, defaultCodexPath+PathCodexResponses)
		}
		ctx.SetContext(ctxKeyCodexMode, true)
		ctx.SetContext(ctxKeyCodexOriginator, m.config.codexOriginator)
	} else if m.isDirectCustomPath {
		util.OverwriteRequestPathHeader(headers, m.customPath)
	} else if apiName != "" {
		util.OverwriteRequestPathHeaderByCapability(headers, string(apiName), m.config.capabilities)
	}

	// Handle Codex mode domain
	if m.config.codexMode {
		if m.customDomain != "" {
			util.OverwriteRequestHostHeader(headers, m.customDomain)
		} else {
			util.OverwriteRequestHostHeader(headers, defaultCodexDomain)
		}
	} else {
		if m.customDomain != "" {
			util.OverwriteRequestHostHeader(headers, m.customDomain)
		} else {
			util.OverwriteRequestHostHeader(headers, defaultOpenaiDomain)
		}
	}

	var token string

	// 1. If apiTokens is configured, use it first
	if len(m.config.apiTokens) > 0 {
		token = m.config.GetApiTokenInUse(ctx)
		if token == "" {
			log.Warnf("[openaiProvider.TransformRequestHeaders] apiTokens count > 0 but GetApiTokenInUse returned empty")
		}
	} else {
		// If no apiToken is configured, try to extract from original request headers

		// 2. If authHeaderKey is configured, use the specified header
		if m.config.authHeaderKey != "" {
			if apiKey, err := proxywasm.GetHttpRequestHeader(m.config.authHeaderKey); err == nil && apiKey != "" {
				token = apiKey
				log.Debugf("[openaiProvider.TransformRequestHeaders] Using token from configured header: %s", m.config.authHeaderKey)
			}
		}

		// 3. If authHeaderKey is not configured, check default headers in priority order
		if token == "" {
			defaultHeaders := []string{"x-api-key", "x-authorization"}
			for _, headerName := range defaultHeaders {
				if apiKey, err := proxywasm.GetHttpRequestHeader(headerName); err == nil && apiKey != "" {
					token = apiKey
					log.Debugf("[openaiProvider.TransformRequestHeaders] Using token from %s header", headerName)
					break
				}
			}
		}

		// 4. Finally check Authorization header
		if token == "" {
			if auth, err := proxywasm.GetHttpRequestHeader("Authorization"); err == nil && auth != "" {
				// Extract token from "Bearer <token>" format
				if strings.HasPrefix(auth, "Bearer ") {
					token = strings.TrimPrefix(auth, "Bearer ")
					log.Debugf("[openaiProvider.TransformRequestHeaders] Using token from Authorization header (Bearer format)")
				} else {
					token = auth
					log.Debugf("[openaiProvider.TransformRequestHeaders] Using token from Authorization header (no Bearer prefix)")
				}
			}
		}
	}

	// 5. Set Authorization header (avoid duplicate Bearer prefix)
	if token != "" {
		// Remove Bearer prefix for Codex mode (uses raw JWT token)
		if m.config.codexMode {
			token = strings.TrimPrefix(token, "Bearer ")
			util.OverwriteRequestAuthorizationHeader(headers, "Bearer "+token)

			// Extract and store account ID from JWT token
			accountId, err := extractAccountIdFromJWT(token)
			if err != nil {
				log.Warnf("[openaiProvider.TransformRequestHeaders] Failed to extract account ID from JWT: %v", err)
			} else {
				ctx.SetContext(ctxKeyCodexAccountId, accountId)
				log.Debugf("[openaiProvider.TransformRequestHeaders] Extracted account ID: %s", accountId)
			}
		} else {
			// Check if token already contains Bearer prefix
			if !strings.HasPrefix(token, "Bearer ") {
				token = "Bearer " + token
			}
			util.OverwriteRequestAuthorizationHeader(headers, token)
		}
		log.Debugf("[openaiProvider.TransformRequestHeaders] Set Authorization header successfully")
	} else {
		log.Warnf("[openaiProvider.TransformRequestHeaders] No auth token available - neither configured in apiTokens nor in request headers")
	}

	// 6. Add Codex-specific headers
	if m.config.codexMode {
		originator := m.config.codexOriginator
		if originator == "" {
			originator = "pi"
		}
		headers.Set("originator", originator)
		headers.Set("OpenAI-Beta", "responses=experimental")
		headers.Set("accept", "text/event-stream")

		// Add chatgpt-account-id header if available
		if accountId, ok := ctx.GetContext(ctxKeyCodexAccountId).(string); ok && accountId != "" {
			headers.Set("chatgpt-account-id", accountId)
		}

		// Set User-Agent to simulate Codex client
		headers.Set("User-Agent", originator+" (higress-ai-proxy)")

		log.Debugf("[openaiProvider.TransformRequestHeaders] Codex mode enabled with originator: %s", originator)
	}

	headers.Del("Content-Length")
}

func (m *openaiProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !m.config.needToProcessRequestBody(apiName) {
		// We don't need to process the request body for other APIs.
		return types.ActionContinue, nil
	}
	return m.config.handleRequestBody(m, m.contextCache, ctx, apiName, body)
}

func (m *openaiProvider) TransformRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	// Handle Codex mode request body transformation
	if m.config.codexMode && apiName == ApiNameChatCompletion {
		codexBody, instructions, err := convertOpenAIToCodexRequest(body)
		if err != nil {
			log.Warnf("[openaiProvider.TransformRequestBody] Failed to convert to Codex format: %v", err)
			return body, err
		}
		ctx.SetContext(ctxKeyCodexInstructions, instructions)
		log.Debugf("[openaiProvider.TransformRequestBody] Converted request to Codex format, instructions length: %d", len(instructions))
		return codexBody, nil
	}

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
