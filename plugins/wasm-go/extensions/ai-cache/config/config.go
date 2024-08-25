package config

import (
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/cache"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/embedding"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/vector"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type KVExtractor struct {
	// @Title zh-CN 从请求 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	RequestBody string `required:"false" yaml:"requestBody" json:"requestBody"`
	// @Title zh-CN 从响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	ResponseBody string `required:"false" yaml:"responseBody" json:"responseBody"`
}

type PluginConfig struct {
	EmbeddingProviderConfig embedding.ProviderConfig `required:"true" yaml:"embeddingProvider" json:"embeddingProvider"`
	VectorProviderConfig    vector.ProviderConfig    `required:"true" yaml:"vectorBaseProvider" json:"vectorBaseProvider"`
	CacheKeyFrom            KVExtractor              `required:"true" yaml:"cacheKeyFrom" json:"cacheKeyFrom"`
	CacheValueFrom          KVExtractor              `required:"true" yaml:"cacheValueFrom" json:"cacheValueFrom"`
	CacheStreamValueFrom    KVExtractor              `required:"true" yaml:"cacheStreamValueFrom" json:"cacheStreamValueFrom"`
	// @Title zh-CN 返回 HTTP 响应的模版
	// @Description zh-CN 用 %s 标记需要被 cache value 替换的部分
	ReturnResponseTemplate string `required:"true" yaml:"returnResponseTemplate" json:"returnResponseTemplate"`
	// @Title zh-CN 返回流式 HTTP 响应的模版
	// @Description zh-CN 用 %s 标记需要被 cache value 替换的部分
	ReturnTestResponseTemplate string `required:"true" yaml:"returnTestResponseTemplate" json:"returnTestResponseTemplate"`

	CacheKeyPrefix string `required:"false" yaml:"cacheKeyPrefix" json:"cacheKeyPrefix"`

	ReturnStreamResponseTemplate string `required:"true" yaml:"returnStreamResponseTemplate" json:"returnStreamResponseTemplate"`
	// @Title zh-CN 缓存的过期时间
	// @Description zh-CN 单位是秒，默认值为0，即永不过期
	CacheTTL int `required:"false" yaml:"cacheTTL" json:"cacheTTL"`
	// @Title zh-CN Redis缓存Key的前缀
	// @Description zh-CN 默认值是"higress-ai-cache:"

	RedisConfig cache.RedisConfig `required:"true" yaml:"redisConfig" json:"redisConfig"`
	// 现在只支持RedisClient作为cacheClient
	redisProvider     cache.Provider     `yaml:"-"`
	embeddingProvider embedding.Provider `yaml:"-"`
	vectorProvider    vector.Provider    `yaml:"-"`
}

func (c *PluginConfig) FromJson(json gjson.Result) {
	c.EmbeddingProviderConfig.FromJson(json.Get("embeddingProvider"))
	c.VectorProviderConfig.FromJson(json.Get("vectorProvider"))
	c.RedisConfig.FromJson(json.Get("redis"))
	if c.CacheKeyFrom.RequestBody == "" {
		c.CacheKeyFrom.RequestBody = "messages.@reverse.0.content"
	}
	c.CacheKeyFrom.RequestBody = json.Get("cacheKeyFrom.requestBody").String()
	if c.CacheKeyFrom.RequestBody == "" {
		c.CacheKeyFrom.RequestBody = "messages.@reverse.0.content"
	}
	c.CacheValueFrom.ResponseBody = json.Get("cacheValueFrom.responseBody").String()
	if c.CacheValueFrom.ResponseBody == "" {
		c.CacheValueFrom.ResponseBody = "choices.0.message.content"
	}
	c.CacheStreamValueFrom.ResponseBody = json.Get("cacheStreamValueFrom.responseBody").String()
	if c.CacheStreamValueFrom.ResponseBody == "" {
		c.CacheStreamValueFrom.ResponseBody = "choices.0.delta.content"
	}
	c.ReturnResponseTemplate = json.Get("returnResponseTemplate").String()
	if c.ReturnResponseTemplate == "" {
		c.ReturnResponseTemplate = `{"id":"from-cache","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	}
	c.ReturnStreamResponseTemplate = json.Get("returnStreamResponseTemplate").String()
	if c.ReturnStreamResponseTemplate == "" {
		c.ReturnStreamResponseTemplate = `data:{"id":"from-cache","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}` + "\n\ndata:[DONE]\n\n"
	}
	c.ReturnTestResponseTemplate = json.Get("returnTestResponseTemplate").String()
	if c.ReturnTestResponseTemplate == "" {
		c.ReturnTestResponseTemplate = `{"id":"random-generate","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	}
}

func (c *PluginConfig) Validate() error {
	if err := c.RedisConfig.Validate(); err != nil {
		return err
	}
	if err := c.EmbeddingProviderConfig.Validate(); err != nil {
		return err
	}
	if err := c.VectorProviderConfig.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *PluginConfig) Complete(log wrapper.Log) error {
	var err error
	c.embeddingProvider, err = embedding.CreateProvider(c.EmbeddingProviderConfig)
	if err != nil {
		return err
	}
	c.vectorProvider, err = vector.CreateProvider(c.VectorProviderConfig)
	if err != nil {
		return err
	}
	c.redisProvider, err = cache.CreateProvider(c.RedisConfig, log)
	if err != nil {
		return err
	}
	return nil
}

func (c *PluginConfig) GetEmbeddingProvider() embedding.Provider {
	return c.embeddingProvider
}

func (c *PluginConfig) GetvectorProvider() vector.Provider {
	return c.vectorProvider
}

func (c *PluginConfig) GetCacheProvider() cache.Provider {
	return c.redisProvider
}
