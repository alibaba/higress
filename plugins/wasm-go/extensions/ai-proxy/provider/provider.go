package provider

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

type ApiName string
type Pointcut string

const (
	ApiNameChatCompletion ApiName = "chatCompletion"

	PointcutOnRequestHeaders  Pointcut = "onRequestHeaders"
	PointcutOnRequestBody     Pointcut = "onRequestBody"
	PointcutOnResponseHeaders Pointcut = "onResponseHeaders"
	PointcutOnResponseBody    Pointcut = "onResponseBody"

	providerTypeMoonshot = "moonshot"
	providerTypeAzure    = "azure"
	providerTypeQwen     = "qwen"

	roleSystem = "system"

	ctxKeyStreaming = "streaming"

	contentTypeTextEventStream = "text/event-stream"

	objectChatCompletion      = "chat.completion"
	objectChatCompletionChunk = "chat.completion.chunk"

	finishReasonStop = "stop"

	defaultTimeout = 2 * 60 * 1000 // ms
)

type providerInitializer interface {
	ValidateConfig(ProviderConfig) error
	CreateProvider(ProviderConfig) (Provider, error)
}

var (
	errUnsupportedApiName = errors.New("unsupported API name")

	providerInitializers = map[string]providerInitializer{
		providerTypeMoonshot: &moonshotProviderInitializer{},
		providerTypeAzure:    &azureProviderInitializer{},
		providerTypeQwen:     &qwenProviderInitializer{},
	}
)

type Provider interface {
	GetPointcuts() map[Pointcut]interface{}
	OnApiRequestHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error)
	OnApiRequestBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error)
	OnApiResponseHeaders(ctx wrapper.HttpContext, apiName ApiName, log wrapper.Log) (types.Action, error)
	OnApiResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte, log wrapper.Log) (types.Action, error)
}

type ProviderConfig struct {
	// @Title zh-CN AI服务提供商
	// @Description zh-CN AI服务提供商类型，目前支持的取值为："moonshot"
	typ string `required:"true" yaml:"type" json:"type"`
	// @Title zh-CN AI服务域名
	// @Description zh-CN AI服务提供商接口所使用的域名
	domain string `required:"false" yaml:"domain" json:"serviceDomain"`
	// @Title zh-CN API Token
	// @Description zh-CN 在请求AI服务时用于认证的API Token。不同的AI服务提供商可能有不同的名称。例Moonshot AI的API Token称为API Key
	apiToken string `required:"false" yaml:"apiToken" json:"apiToken"`
	// @Title zh-CN 请求超时
	// @Description zh-CN 请求AI服务的超时时间，单位为毫秒。默认值为120000，即2分钟
	timeout uint32 `required:"false" yaml:"timeout" json:"timeout"`
	// @Title zh-CN Moonshot File ID
	// @Description zh-CN 仅适用于Moonshot AI服务。Moonshot AI服务的文件 ID，其内容用于补充 AI 请求上下文
	moonshotFileId string `required:"false" yaml:"moonshotFileId" json:"moonshotFileId"`
	// @Title zh-CN Azure OpenAI Deployment ID
	// @Description zh-CN 仅适用于Azure OpenAI服务。要请求的AI模型的部署ID
	azureModelDeploymentName string `required:"false" yaml:"azureModelDeploymentName" json:"azureModelDeploymentName"`
	// @Title zh-CN Azure OpenAI API Version
	// @Description zh-CN 仅适用于Azure OpenAI服务。要请求的Azure OpenAI服务的API版本
	azureApiVersion string `required:"false" yaml:"azureApiVersion" json:"azureApiVersion"`
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.typ = json.Get("type").String()
	c.domain = json.Get("domain").String()
	c.apiToken = json.Get("apiToken").String()
	c.timeout = uint32(json.Get("timeout").Uint())
	c.moonshotFileId = json.Get("moonshotFileId").String()
	c.azureModelDeploymentName = json.Get("azureModelDeploymentName").String()
	c.azureApiVersion = json.Get("azureApiVersion").String()
}

func (c *ProviderConfig) Validate() error {
	if c.apiToken == "" {
		return errors.New("missing apiToken in provider config")
	}
	if c.domain == "" {
		return errors.New("missing domain in provider config")
	}
	if c.timeout < 0 {
		return errors.New("invalid timeout in config")
	}

	if c.typ == "" {
		return errors.New("missing type in provider config")
	}
	initializer, has := providerInitializers[c.typ]
	if !has {
		return errors.New("unknown provider type: " + c.typ)
	}
	return initializer.ValidateConfig(*c)
}

func CreateProvider(pc ProviderConfig) (Provider, error) {
	initializer, has := providerInitializers[pc.typ]
	if !has {
		return nil, errors.New("unknown provider type: " + pc.typ)
	}
	return initializer.CreateProvider(pc)
}

func createClient(config ProviderConfig) (wrapper.HttpClient, error) {
	return wrapper.NewClusterClient(wrapper.RouteCluster{
		Host: config.domain,
	}), nil
}

func decodeChatCompletionRequest(body []byte, request *chatCompletionRequest) error {
	if err := json.Unmarshal(body, request); err != nil {
		return fmt.Errorf("unable to unmarshal request: %v", err)
	}
	if request.Messages == nil || len(request.Messages) == 0 {
		return errors.New("no message found in the request body")
	}
	return nil
}

func replaceJsonRequestBody(request interface{}) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("unable to marshal request: %v", err)
	}
	err = proxywasm.ReplaceHttpRequestBody(body)
	if err != nil {
		return fmt.Errorf("unable to replace the original request body: %v", err)
	}
	return err
}

func replaceJsonResponseBody(response interface{}) error {
	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("unable to marshal response: %v", err)
	}
	err = proxywasm.ReplaceHttpResponseBody(body)
	if err != nil {
		return fmt.Errorf("unable to replace the original response body: %v", err)
	}
	return err
}
