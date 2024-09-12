package config

import (
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/cache"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/embedding"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/vector"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type PluginConfig struct {
	// @Title zh-CN 返回 HTTP 响应的模版
	// @Description zh-CN 用 %s 标记需要被 cache value 替换的部分
	ResponseTemplate string
	// @Title zh-CN 返回流式 HTTP 响应的模版
	// @Description zh-CN 用 %s 标记需要被 cache value 替换的部分
	StreamResponseTemplate string

	cacheProvider     cache.Provider
	embeddingProvider embedding.Provider
	vectorProvider    vector.Provider

	embeddingProviderConfig embedding.ProviderConfig
	vectorProviderConfig    vector.ProviderConfig
	cacheProviderConfig     cache.ProviderConfig

	CacheKeyFrom         string
	CacheValueFrom       string
	CacheStreamValueFrom string
	CacheToolCallsFrom   string
}

func (c *PluginConfig) FromJson(json gjson.Result) {
	c.embeddingProviderConfig.FromJson(json.Get("embedding"))
	c.vectorProviderConfig.FromJson(json.Get("vector"))
	c.cacheProviderConfig.FromJson(json.Get("cache"))

	c.CacheKeyFrom = json.Get("cacheKeyFrom").String()
	if c.CacheKeyFrom == "" {
		c.CacheKeyFrom = "messages.@reverse.0.content"
	}
	c.CacheValueFrom = json.Get("cacheValueFrom").String()
	if c.CacheValueFrom == "" {
		c.CacheValueFrom = "choices.0.message.content"
	}
	c.CacheStreamValueFrom = json.Get("cacheStreamValueFrom").String()
	if c.CacheStreamValueFrom == "" {
		c.CacheStreamValueFrom = "choices.0.delta.content"
	}
	c.CacheToolCallsFrom = json.Get("cacheToolCallsFrom").String()
	if c.CacheToolCallsFrom == "" {
		c.CacheToolCallsFrom = "choices.0.delta.content.tool_calls"
	}

	c.StreamResponseTemplate = json.Get("streamResponseTemplate").String()
	if c.StreamResponseTemplate == "" {
		c.StreamResponseTemplate = `data:{"id":"ai-cache.hit","choices":[{"index":0,"delta":{"role":"assistant","content":%s},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}` + "\n\ndata:[DONE]\n\n"
	}
	c.ResponseTemplate = json.Get("responseTemplate").String()
	if c.ResponseTemplate == "" {
		c.ResponseTemplate = `{"id":"ai-cache.hit","choices":[{"index":0,"message":{"role":"assistant","content":%s},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
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
