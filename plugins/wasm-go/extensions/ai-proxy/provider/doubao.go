package provider

import (
	"errors"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	doubaoDomain             = "ark.cn-beijing.volces.com"
	doubaoChatCompletionPath = "/api/v3/chat/completions"
)

type doubaoProviderInitializer struct{}

func (m *doubaoProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (m *doubaoProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &doubaoProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type doubaoProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (m *doubaoProvider) GetProviderType() string {
	return providerTypeDoubao
}

func (m *doubaoProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	_ = util.OverwriteRequestHost(doubaoDomain)
	_ = util.OverwriteRequestAuthorization("Bearer " + m.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	if m.config.protocol == protocolOriginal {
		ctx.DontReadRequestBody()
		return types.ActionContinue, nil
	}
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(doubaoChatCompletionPath)
	return types.ActionContinue, nil
}

func (m *doubaoProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	if m.contextCache == nil {
		return types.ActionContinue, nil
	}
	request := &chatCompletionRequest{}
	if err := decodeChatCompletionRequest(body, request); err != nil {
		return types.ActionContinue, err
	}

	model := request.Model
	if model == "" {
		return types.ActionContinue, errors.New("missing model in chat completion request")
	}
	ctx.SetContext(ctxKeyOriginalRequestModel, model)
	mappedModel := getMappedModel(model, m.config.modelMapping, log)
	if mappedModel == "" {
		return types.ActionContinue, errors.New("model becomes empty after applying the configured mapping")
	}
	request.Model = mappedModel
	ctx.SetContext(ctxKeyFinalRequestModel, request.Model)

	return types.ActionContinue, nil
}

func (m *doubaoProvider) OnResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if m.config.protocol == protocolOriginal {
		ctx.DontReadResponseBody()
		return types.ActionContinue, nil
	}

	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	return types.ActionContinue, nil
}

func (m *doubaoProvider) OnStreamingResponseBody(ctx wrapper.HttpContext, name ApiName, chunk []byte, isLastChunk bool, log wrapper.Log) ([]byte, error) {
	if m.config.qwenEnableCompatible || name != ApiNameChatCompletion {
		return chunk, nil
	}

	receivedBody := chunk
	if bufferedStreamingBody, has := ctx.GetContext(ctxKeyStreamingBody).([]byte); has {
		receivedBody = append(bufferedStreamingBody, chunk...)
	}

	incrementalStreaming := ctx.GetBoolContext(ctxKeyIncrementalStreaming, false)

	eventStartIndex, lineStartIndex, valueStartIndex := -1, -1, -1

	defer func() {
		if eventStartIndex >= 0 && eventStartIndex < len(receivedBody) {
			// Just in case the received chunk is not a complete event.
			ctx.SetContext(ctxKeyStreamingBody, receivedBody[eventStartIndex:])
		} else {
			ctx.SetContext(ctxKeyStreamingBody, nil)
		}
	}()

	var responseBuilder strings.Builder
	currentKey := ""
	currentEvent := &streamEvent{}
	i, length := 0, len(receivedBody)
	for i = 0; i < length; i++ {
		ch := receivedBody[i]
		if ch != '\n' {
			if lineStartIndex == -1 {
				if eventStartIndex == -1 {
					eventStartIndex = i
				}
				lineStartIndex = i
				valueStartIndex = -1
			}
			if valueStartIndex == -1 {
				if ch == ':' {
					valueStartIndex = i + 1
					currentKey = string(receivedBody[lineStartIndex:valueStartIndex])
				}
			} else if valueStartIndex == i && ch == ' ' {
				// Skip leading spaces in data.
				valueStartIndex = i + 1
			}
			continue
		}

		if lineStartIndex != -1 {
			value := string(receivedBody[valueStartIndex:i])
			currentEvent.setValue(currentKey, value)
		} else {
			// Extra new line. The current event is complete.
			log.Debugf("processing event: %v", currentEvent)
			if err := m.convertStreamEvent(ctx, &responseBuilder, currentEvent, incrementalStreaming, log); err != nil {
				return nil, err
			}
			// Reset event parsing state.
			eventStartIndex = -1
			currentEvent = &streamEvent{}
		}

		// Reset line parsing state.
		lineStartIndex = -1
		valueStartIndex = -1
		currentKey = ""
	}

	modifiedResponseChunk := responseBuilder.String()
	log.Debugf("=== modified response chunk: %s", modifiedResponseChunk)
	return []byte(modifiedResponseChunk), nil
}

func (m *doubaoProvider) OnResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	return types.ActionContinue, nil
}
