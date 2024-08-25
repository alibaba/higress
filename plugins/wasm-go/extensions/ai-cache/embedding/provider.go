package embedding

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	providerTypeDashScope    = "dashscope"
	CacheKeyContextKey       = "cacheKey"
	CacheContentContextKey   = "cacheContent"
	PartialMessageContextKey = "partialMessage"
	ToolCallsContextKey      = "toolCalls"
	StreamContextKey         = "stream"
	CacheKeyPrefix           = "higressAiCache"
	DefaultCacheKeyPrefix    = "higressAiCache"
	queryEmbeddingKey        = "queryEmbedding"
)

type providerInitializer interface {
	ValidateConfig(ProviderConfig) error
	CreateProvider(ProviderConfig) (Provider, error)
}

var (
	providerInitializers = map[string]providerInitializer{
		providerTypeDashScope: &dashScopeProviderInitializer{},
	}
)

type ProviderConfig struct {
	// @Title zh-CN 文本特征提取服务提供者类型
	// @Description zh-CN 文本特征提取服务提供者类型，例如 DashScope
	typ string `json:"type"`
	// @Title zh-CN DashScope 阿里云大模型服务名
	// @Description zh-CN 调用阿里云的大模型服务
	serviceName string             `require:"true" yaml:"serviceName" json:"serviceName"`
	serviceHost string             `require:"false" yaml:"serviceHost" json:"serviceHost"`
	servicePort int64              `require:"false" yaml:"servicePort" json:"servicePort"`
	apiKey      string             `require:"false" yaml:"apiKey" json:"apiKey"`
	timeout     uint32             `require:"false" yaml:"timeout" json:"timeout"`
	client      wrapper.HttpClient `yaml:"-"`
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.typ = json.Get("type").String()
	c.serviceName = json.Get("serviceName").String()
	c.serviceHost = json.Get("serviceHost").String()
	c.servicePort = json.Get("servicePort").Int()
	c.apiKey = json.Get("apiKey").String()
	c.timeout = uint32(json.Get("timeout").Int())
	if c.timeout == 0 {
		c.timeout = 1000
	}
}

func (c *ProviderConfig) Validate() error {
	if len(c.serviceName) == 0 {
		return errors.New("serviceName is required")
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
		log wrapper.Log,
		callback func(emb []float64)) error
}
