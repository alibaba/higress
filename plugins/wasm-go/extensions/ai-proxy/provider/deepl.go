package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// deeplProvider is the provider for DeepL service.
const (
	deeplHostPro            = "api.deepl.com"
	deeplHostFree           = "api-free.deepl.com"
	deeplChatCompletionPath = "/v2/translate"
)

type deeplProviderInitializer struct {
}

type deeplProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

// spec reference: https://developers.deepl.com/docs/v/zh/api-reference/translate/openapi-spec-for-text-translation
type deeplRequest struct {
	// "Model" parameter is used to distinguish which service to use
	Model              string   `json:"model,omitempty"`
	Text               []string `json:"text"`
	SourceLang         string   `json:"source_lang,omitempty"`
	TargetLang         string   `json:"target_lang"`
	Context            string   `json:"context,omitempty"`
	SplitSentences     string   `json:"split_sentences,omitempty"`
	PreserveFormatting bool     `json:"preserve_formatting,omitempty"`
	Formality          string   `json:"formality,omitempty"`
	GlossaryId         string   `json:"glossary_id,omitempty"`
	TagHandling        string   `json:"tag_handling,omitempty"`
	OutlineDetection   bool     `json:"outline_detection,omitempty"`
	NonSplittingTags   []string `json:"non_splitting_tags,omitempty"`
	SplittingTags      []string `json:"splitting_tags,omitempty"`
	IgnoreTags         []string `json:"ignore_tags,omitempty"`
}

type deeplResponse struct {
	Translations []deeplResponseTranslation `json:"translations,omitempty"`
	Message      string                     `json:"message,omitempty"`
}

type deeplResponseTranslation struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
}

func (d *deeplProviderInitializer) ValidateConfig(config ProviderConfig) error {
	if config.targetLang == "" {
		return errors.New("missing targetLang in deepl provider config")
	}
	return nil
}

func (d *deeplProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	return &deeplProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

func (d *deeplProvider) GetProviderType() string {
	return providerTypeDeepl
}

func (d *deeplProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	_ = util.OverwriteRequestPath(deeplChatCompletionPath)
	_ = util.OverwriteRequestAuthorization("DeepL-Auth-Key " + d.config.GetRandomToken())
	_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	return types.HeaderStopIteration, nil
}

func (d *deeplProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	if apiName != ApiNameChatCompletion {
		return types.ActionContinue, errUnsupportedApiName
	}
	if d.config.protocol == protocolOriginal {
		request := &deeplRequest{}
		if err := json.Unmarshal(body, request); err != nil {
			return types.ActionContinue, fmt.Errorf("unable to unmarshal request: %v", err)
		}
		if err := d.overwriteRequestHost(request.Model); err != nil {
			return types.ActionContinue, err
		}
		ctx.SetContext(ctxKeyFinalRequestModel, request.Model)
		return types.ActionContinue, replaceJsonRequestBody(request, log)
	} else {
		originRequest := &chatCompletionRequest{}
		if err := decodeChatCompletionRequest(body, originRequest); err != nil {
			return types.ActionContinue, err
		}
		if err := d.overwriteRequestHost(originRequest.Model); err != nil {
			return types.ActionContinue, err
		}
		ctx.SetContext(ctxKeyFinalRequestModel, originRequest.Model)
		deeplRequest := &deeplRequest{
			Text:       make([]string, 0),
			TargetLang: d.config.targetLang,
		}
		for _, msg := range originRequest.Messages {
			if msg.Role == roleSystem {
				deeplRequest.Context = msg.StringContent()
			} else {
				deeplRequest.Text = append(deeplRequest.Text, msg.StringContent())
			}
		}
		return types.ActionContinue, replaceJsonRequestBody(deeplRequest, log)
	}
}

func (d *deeplProvider) OnResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error) {
	_ = proxywasm.RemoveHttpResponseHeader("Content-Length")
	return types.ActionContinue, nil
}

func (d *deeplProvider) OnResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error) {
	deeplResponse := &deeplResponse{}
	if err := json.Unmarshal(body, deeplResponse); err != nil {
		return types.ActionContinue, fmt.Errorf("unable to unmarshal deepl response: %v", err)
	}
	response := d.responseDeepl2OpenAI(ctx, deeplResponse)
	return types.ActionContinue, replaceJsonResponseBody(response, log)
}

func (d *deeplProvider) responseDeepl2OpenAI(ctx wrapper.HttpContext, deeplResponse *deeplResponse) *chatCompletionResponse {
	var choices []chatCompletionChoice
	// Fail
	if deeplResponse.Message != "" {
		choices = make([]chatCompletionChoice, 1)
		choices[0] = chatCompletionChoice{
			Message: &chatMessage{Role: roleAssistant, Content: deeplResponse.Message},
			Index:   0,
		}
	} else {
		// Success
		choices = make([]chatCompletionChoice, len(deeplResponse.Translations))
		for idx, t := range deeplResponse.Translations {
			choices[idx] = chatCompletionChoice{
				Index:   idx,
				Message: &chatMessage{Role: roleAssistant, Content: t.Text, Name: t.DetectedSourceLanguage},
			}
		}
	}
	return &chatCompletionResponse{
		Created: time.Now().UnixMilli() / 1000,
		Object:  objectChatCompletion,
		Choices: choices,
		Model:   ctx.GetStringContext(ctxKeyFinalRequestModel, ""),
	}
}

func (d *deeplProvider) overwriteRequestHost(model string) error {
	if model == "Pro" {
		_ = util.OverwriteRequestHost(deeplHostPro)
	} else if model == "Free" {
		_ = util.OverwriteRequestHost(deeplHostFree)
	} else {
		return errors.New(`deepl model should be "Free" or "Pro"`)
	}
	return nil
}
