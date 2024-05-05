package provider

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
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
	providerTypeOpenAI   = "openai"

	roleSystem = "system"

	ctxKeyStreaming            = "streaming"
	ctxKeyOriginalRequestModel = "originalRequestModel"
	ctxKeyFinalRequestModel    = "finalRequestModel"

	contentTypeTextEventStream = "text/event-stream"

	objectChatCompletion      = "chat.completion"
	objectChatCompletionChunk = "chat.completion.chunk"

	finishReasonStop = "stop"

	wildcard = "*"

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
		providerTypeOpenAI:   &openaiProviderInitializer{},
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
	// @Title zh-CN API Token
	// @Description zh-CN 在请求AI服务时用于认证的API Token。不同的AI服务提供商可能有不同的名称。例Moonshot AI的API Token称为API Key
	apiToken string `required:"false" yaml:"apiToken" json:"apiToken"`
	// @Title zh-CN 请求超时
	// @Description zh-CN 请求AI服务的超时时间，单位为毫秒。默认值为120000，即2分钟
	timeout uint32 `required:"false" yaml:"timeout" json:"timeout"`
	// @Title zh-CN Moonshot File ID
	// @Description zh-CN 仅适用于Moonshot AI服务。Moonshot AI服务的文件 ID，其内容用于补充 AI 请求上下文
	moonshotFileId string `required:"false" yaml:"moonshotFileId" json:"moonshotFileId"`
	// @Title zh-CN Azure OpenAI Service URL
	// @Description zh-CN 仅适用于Azure OpenAI服务。要请求的OpenAI服务的完整URL，包含api-version等参数
	azureServiceUrl string `required:"false" yaml:"azureServiceUrl" json:"azureServiceUrl"`
	// @Title zh-CN 模型名称映射表
	// @Description zh-CN 用于将请求中的模型名称映射为目标AI服务商支持的模型名称。支持通过“*”来配置全局映射
	modelMapping map[string]string `required:"false" yaml:"modelMapping" json:"modelMapping"`
	// @Title zh-CN 模型对话上下文
	// @Description zh-CN 配置一个外部获取对话上下文的文件来源，用于在AI请求中补充对话上下文
	context *ContextConfig `required:"false" yaml:"context" json:"context"`
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.typ = json.Get("type").String()
	c.apiToken = json.Get("apiToken").String()
	c.timeout = uint32(json.Get("timeout").Uint())
	if c.timeout == 0 {
		c.timeout = defaultTimeout
	}
	c.moonshotFileId = json.Get("moonshotFileId").String()
	c.azureServiceUrl = json.Get("azureServiceUrl").String()
	c.modelMapping = make(map[string]string)
	for k, v := range json.Get("modelMapping").Map() {
		c.modelMapping[k] = v.String()
	}
	contextJson := json.Get("context")
	if contextJson.Exists() {
		c.context = &ContextConfig{}
		c.context.FromJson(contextJson)
	}
}

func (c *ProviderConfig) Validate() error {
	if c.apiToken == "" {
		return errors.New("missing apiToken in provider config")
	}
	if c.timeout < 0 {
		return errors.New("invalid timeout in config")
	}
	if c.context != nil {
		if err := c.context.Validate(); err != nil {
			return err
		}
	}

	if c.typ == "" {
		return errors.New("missing type in provider config")
	}
	initializer, has := providerInitializers[c.typ]
	if !has {
		return errors.New("unknown provider type: " + c.typ)
	}
	if err := initializer.ValidateConfig(*c); err != nil {
		return err
	}
	return nil
}

func CreateProvider(pc ProviderConfig) (Provider, error) {
	initializer, has := providerInitializers[pc.typ]
	if !has {
		return nil, errors.New("unknown provider type: " + pc.typ)
	}
	return initializer.CreateProvider(pc)
}

func getMappedModel(model string, modelMapping map[string]string, log wrapper.Log) string {
	if modelMapping == nil || len(modelMapping) == 0 {
		return model
	}
	if v, ok := modelMapping[model]; ok && len(v) != 0 {
		log.Debugf("model %s is mapped to %s explictly", model, v)
		return v
	}
	if v, ok := modelMapping[wildcard]; ok {
		log.Debugf("model %s is mapped to %s via wildcard", model, v)
		return v
	}
	return model
}
