package embedding

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	PROVIDER_TYPE_DASHSCOPE   = "dashscope"
	PROVIDER_TYPE_TEXTIN      = "textin"
	PROVIDER_TYPE_COHERE      = "cohere"
	PROVIDER_TYPE_OPENAI      = "openai"
	PROVIDER_TYPE_OLLAMA      = "ollama"
	PROVIDER_TYPE_HUGGINGFACE = "huggingface"
	PROVIDER_TYPE_XFYUN       = "xfyun"
	PROVIDER_TYPE_AZURE       = "azure"
)

type providerInitializer interface {
	InitConfig(json gjson.Result)
	ValidateConfig() error
	CreateProvider(ProviderConfig) (Provider, error)
}

var (
	providerInitializers = map[string]providerInitializer{
		PROVIDER_TYPE_DASHSCOPE:   &dashScopeProviderInitializer{},
		PROVIDER_TYPE_TEXTIN:      &textInProviderInitializer{},
		PROVIDER_TYPE_COHERE:      &cohereProviderInitializer{},
		PROVIDER_TYPE_OPENAI:      &openAIProviderInitializer{},
		PROVIDER_TYPE_OLLAMA:      &ollamaProviderInitializer{},
		PROVIDER_TYPE_HUGGINGFACE: &huggingfaceProviderInitializer{},
		PROVIDER_TYPE_XFYUN:       &xfyunProviderInitializer{},
		PROVIDER_TYPE_AZURE:       &azureProviderInitializer{},
	}
)

type ProviderConfig struct {
	// @Title zh-CN 文本特征提取服务提供者类型
	// @Description zh-CN 文本特征提取服务提供者类型，例如 DashScope
	typ string
	// @Title zh-CN DashScope 文本特征提取服务名称
	// @Description zh-CN 文本特征提取服务名称
	serviceName string
	// @Title zh-CN 文本特征提取服务域名
	// @Description zh-CN 文本特征提取服务域名
	serviceHost string
	// @Title zh-CN 文本特征提取服务端口
	// @Description zh-CN 文本特征提取服务端口
	servicePort int64
	// @Title zh-CN 文本特征提取服务超时时间
	// @Description zh-CN 文本特征提取服务超时时间
	timeout uint32
	// @Title zh-CN 文本特征提取服务使用的模型
	// @Description zh-CN 用于文本特征提取的模型名称, 在 DashScope 中默认为 "text-embedding-v1"
	model string

	initializer providerInitializer
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.typ = json.Get("type").String()
	i, has := providerInitializers[c.typ]
	if has {
		i.InitConfig(json)
		c.initializer = i
	}
	c.serviceName = json.Get("serviceName").String()
	c.serviceHost = json.Get("serviceHost").String()
	c.servicePort = json.Get("servicePort").Int()
	c.timeout = uint32(json.Get("timeout").Int())
	c.model = json.Get("model").String()
	if c.timeout == 0 {
		c.timeout = 10000
	}
}

func (c *ProviderConfig) Validate() error {
	if c.serviceName == "" {
		return errors.New("embedding service name is required")
	}
	if c.typ == "" {
		return errors.New("embedding service type is required")
	}
	if c.initializer == nil {
		return errors.New("unknown embedding service provider type: " + c.typ)
	}
	if err := c.initializer.ValidateConfig(); err != nil {
		return err
	}
	return nil
}

func (c *ProviderConfig) GetProviderType() string {
	return c.typ
}

func CreateProvider(pc ProviderConfig) (Provider, error) {
	initializer, has := providerInitializers[pc.typ]
	if !has {
		return nil, errors.New("unknown provider type: " + pc.typ)
	}
	return initializer.CreateProvider(pc)
}

type Provider interface {
	GetProviderType() string
	GetEmbedding(
		queryString string,
		ctx wrapper.HttpContext,
		callback func(emb []float64, err error)) error
}
