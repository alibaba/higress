package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	//TODO: decide the tritonDomain
	tritonDomain                        = "localhost"
	tritonChatGenerationPath            = "v2/models/{MODEL_NAME}/generate"
	tritonChatGenerationWithVersionPath = "v2/models/{MODEL_NAME}/versions/{MODEL_VERSION}/generate"
	tritonChatGenerationStreamPath      = "v2/models/{MODEL_NAME}[/versions/${MODEL_VERSION}]/generate_stream"
)

type tritonProviderInitializer struct{}

func (t *tritonProviderInitializer) ValidateConfig(config *ProviderConfig) error {
	if config.apiTokens == nil || len(config.apiTokens) == 0 {
		return errors.New("no apiToken found in provider config")
	}
	return nil
}

func (t *tritonProviderInitializer) DefaultCapabilities() map[string]string {
	return map[string]string{
		string(ApiNameChatCompletion): tritonChatGenerationPath,
		// string(d): tritonChatCompletionPath,
	}
}

func (t *tritonProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
	config.setDefaultCapabilities(t.DefaultCapabilities())
	return &tritonProvider{
		config:       config,
		contextCache: createContextCache(&config),
	}, nil
}

type tritonProvider struct {
	config       ProviderConfig
	contextCache *contextCache
}

func (t *tritonProvider) GetProviderType() string {
	return providerTypeTriton
}

func (t *tritonProvider) OnRequestHeaders(ctx wrapper.HttpContext, apiName ApiName) error {
	t.config.handleRequestHeaders(t, ctx, apiName)
	return nil
}

func (t *tritonProvider) OnRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) (types.Action, error) {
	if !t.config.isSupportedAPI(apiName) {
		return types.ActionContinue, errUnsupportedApiName
	}
	return t.config.handleRequestBody(t, t.contextCache, ctx, apiName, body)
}

func (t *tritonProvider) TransformRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, headers http.Header) {
	finalPath := t.getFinalRequestPath(ctx)
	util.OverwriteRequestPathHeader(headers, finalPath)
	util.OverwriteRequestHostHeader(headers, t.config.tritonDomain)
	headers.Del("Content-Length")
}

func (t *tritonProvider) GetApiName(path string) ApiName {
	if strings.Contains(path, tritonChatGenerationPath) {
		return ApiNameChatCompletion
	}
	return ""
}

func (t *tritonProvider) getFinalRequestPath(ctx wrapper.HttpContext) string {
	res := tritonChatGenerationPath
	if t.config.tritonModelVersion == "" {
		res = tritonChatGenerationWithVersionPath
		res = strings.Replace(res, "{MODEL_VERSION}", t.config.tritonModelVersion, 1)
	}
	res = strings.Replace(res, "{MODEL_NAME}", t.config.tritonModelName, 1)

	log.Debugf("[Triton Server]: Get final RequestPath: %v", res)
	return res
}

type TritonGenerateRequest struct {
	Id         string                  `json:"id"`
	TextInput  string                  `json:"text_input"`
	Parameters TritonGenerateParameter `json:"parameters"`
}

type TritonGenerateParameter struct {
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature"`
}

type TritonGenerateResponse struct {
	Id           string `json:"id"`
	ModelName    string `json:"model_name"`
	ModelVersion string `json:"model_version"`
	TextOutput   string `json:"text_output"`
	Error        string `json:"error"`
}

func (t *tritonProvider) TransformRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	request := &chatCompletionRequest{}
	if err := t.config.parseRequestAndMapModel(ctx, request, body); err != nil {
		return nil, err
	}
	tritonRequest := t.BuildTritonTexGenRequest(request)
	return json.Marshal(tritonRequest)
}

func (t *tritonProvider) BuildTritonTexGenRequest(origRequest *chatCompletionRequest) *TritonGenerateRequest {
	res := &TritonGenerateRequest{
		Id:        "",
		TextInput: "",
		Parameters: TritonGenerateParameter{
			Stream:      origRequest.Stream,
			Temperature: origRequest.Temperature,
		},
	}

	for _, msg := range origRequest.Messages {
		res.Id = msg.Id
		res.TextInput = msg.StringContent()
	}
	return res
}
func (t *tritonProvider) TransformResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	if apiName != ApiNameChatCompletion {
		return body, nil
	}
	tritonRes := &TritonGenerateResponse{}
	if err := json.Unmarshal(body, tritonRes); err != nil {
		return nil, fmt.Errorf("unable to unmarshal claude response: %v", err)
	}
	if tritonRes.Error != "" {
		return nil, fmt.Errorf("triton response error, error_message: %s", tritonRes.Error)
	}
	response := t.ParseResponse2OpenAI(tritonRes)
	return json.Marshal(response)

}

func (t *tritonProvider) ParseResponse2OpenAI(tritonRes *TritonGenerateResponse) *chatCompletionResponse {
	res := &chatCompletionResponse{
		Id:      tritonRes.Id,
		Choices: []chatCompletionChoice{},
		Created: 0,
		Model:   tritonRes.ModelName,
		Usage:   &usage{},
	}
	return res
}
