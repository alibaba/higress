package config

import (
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/cache"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/embedding"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/vector"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type BodyPathMapper struct {
	// @Title zh-CN 从请求 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	RequestPath string `required:"false" yaml:"requestBody" json:"requestBody"`
	// @Title zh-CN 从响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	ResponsePath string `required:"false" yaml:"responseBody" json:"responseBody"`
}

func (e *BodyPathMapper) SetRequestPathFromJson(json gjson.Result, key string, defaultValue string) {
	value := json.Get(key)
	if value.Exists() {
		e.RequestPath = value.String()
	} else {
		e.RequestPath = defaultValue
	}
}

func (e *BodyPathMapper) SetResponsePathFromJson(json gjson.Result, key string, defaultValue string) {
	value := json.Get(key)
	if value.Exists() {
		e.ResponsePath = value.String()
	} else {
		e.ResponsePath = defaultValue
	}
}

type PluginConfig struct {
	// @Title zh-CN 返回 HTTP 响应的模版
	// @Description zh-CN 用 %s 标记需要被 cache value 替换的部分
	ResponseTemplate string `required:"true" yaml:"responseTemplate" json:"responseTemplate"`
	// @Title zh-CN 返回流式 HTTP 响应的模版
	// @Description zh-CN 用 %s 标记需要被 cache value 替换的部分
	StreamResponseTemplate string `required:"true" yaml:"streamResponseTemplate" json:"streamResponseTemplate"`

	cacheProvider     cache.Provider     `yaml:"-"`
	embeddingProvider embedding.Provider `yaml:"-"`
	vectorProvider    vector.Provider    `yaml:"-"`

	embeddingProviderConfig embedding.ProviderConfig
	vectorProviderConfig    vector.ProviderConfig
	cacheProviderConfig     cache.ProviderConfig

	CacheKeyFrom         BodyPathMapper `required:"true" yaml:"cacheKeyFrom" json:"cacheKeyFrom"`
	CacheValueFrom       BodyPathMapper `required:"true" yaml:"cacheValueFrom" json:"cacheValueFrom"`
	CacheStreamValueFrom BodyPathMapper `required:"true" yaml:"cacheStreamValueFrom" json:"cacheStreamValueFrom"`
}

func (c *PluginConfig) FromJson(json gjson.Result) {
	c.embeddingProviderConfig.FromJson(json.Get("embedding"))
	c.vectorProviderConfig.FromJson(json.Get("vector"))
	c.cacheProviderConfig.FromJson(json.Get("cache"))

	c.CacheKeyFrom.SetRequestPathFromJson(json, "cacheKeyFrom.requestBody", "messages.@reverse.0.content")
	c.CacheValueFrom.SetResponsePathFromJson(json, "cacheValueFrom.responseBody", "choices.0.message.content")
	c.CacheStreamValueFrom.SetResponsePathFromJson(json, "cacheStreamValueFrom.responseBody", "choices.0.delta.content")

	c.StreamResponseTemplate = json.Get("streamResponseTemplate").String()
	if c.StreamResponseTemplate == "" {
		c.StreamResponseTemplate = `data:{"id":"from-cache","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}` + "\n\ndata:[DONE]\n\n"
	}
	c.ResponseTemplate = json.Get("responseTemplate").String()
	if c.ResponseTemplate == "" {
		c.ResponseTemplate = `{"id":"from-cache","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	}
}

func (c *PluginConfig) Validate() error {
	if err := c.cacheProviderConfig.Validate(); err != nil {
		return err
	}
	if err := c.embeddingProviderConfig.Validate(); err != nil {
		return err
	}
	if err := c.vectorProviderConfig.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *PluginConfig) Complete(log wrapper.Log) error {
	var err error
	c.embeddingProvider, err = embedding.CreateProvider(c.embeddingProviderConfig)
	if err != nil {
		return err
	}
	c.vectorProvider, err = vector.CreateProvider(c.vectorProviderConfig)
	if err != nil {
		return err
	}
	c.cacheProvider, err = cache.CreateProvider(c.cacheProviderConfig)
	if err != nil {
		return err
	}
	return nil
}

func (c *PluginConfig) GetEmbeddingProvider() embedding.Provider {
	return c.embeddingProvider
}

func (c *PluginConfig) GetVectorProvider() vector.Provider {
	return c.vectorProvider
}

func (c *PluginConfig) GetCacheProvider() cache.Provider {
	return c.cacheProvider
}
