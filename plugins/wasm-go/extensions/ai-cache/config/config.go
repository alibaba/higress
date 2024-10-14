package config

import (
	"fmt"

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

	// CacheKeyFrom         string
	CacheValueFrom       string
	CacheStreamValueFrom string
	CacheToolCallsFrom   string

	// @Title zh-CN 启用语义化缓存
	// @Description zh-CN 控制是否启用语义化缓存功能。true 表示启用，false 表示禁用。
	EnableSemanticCache bool

	// @Title zh-CN 缓存键策略
	// @Description zh-CN 决定如何生成缓存键的策略。可选值: "lastQuestion" (使用最后一个问题), "allQuestions" (拼接所有问题) 或 "disable" (禁用缓存)
	CacheKeyStrategy string
}

func (c *PluginConfig) FromJson(json gjson.Result) {
	c.vectorProviderConfig.FromJson(json.Get("vector"))
	c.embeddingProviderConfig.FromJson(json.Get("embedding"))
	c.cacheProviderConfig.FromJson(json.Get("cache"))

	c.CacheKeyStrategy = json.Get("cacheKeyStrategy").String()
	if c.CacheKeyStrategy == "" {
		c.CacheKeyStrategy = "lastQuestion" // 设置默认值
	}
	// c.CacheKeyFrom = json.Get("cacheKeyFrom").String()
	// if c.CacheKeyFrom == "" {
	// 	c.CacheKeyFrom = "messages.@reverse.0.content"
	// }
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
		c.StreamResponseTemplate = `data:{"id":"from-cache","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}` + "\n\ndata:[DONE]\n\n"
	}
	c.ResponseTemplate = json.Get("responseTemplate").String()
	if c.ResponseTemplate == "" {
		c.ResponseTemplate = `{"id":"from-cache","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	}

	// 默认值为 true
	if json.Get("enableSemanticCache").Exists() {
		c.EnableSemanticCache = json.Get("enableSemanticCache").Bool()
	} else {
		c.EnableSemanticCache = true // 设置默认值为 true
	}
}

func (c *PluginConfig) Validate() error {
	// if cache provider is configured, validate it
	if c.cacheProviderConfig.GetProviderType() != "" {
		if err := c.cacheProviderConfig.Validate(); err != nil {
			return err
		}
	}
	if c.embeddingProviderConfig.GetProviderType() != "" {
		if err := c.embeddingProviderConfig.Validate(); err != nil {
			return err
		}
	}
	if err := c.vectorProviderConfig.Validate(); err != nil {
		return err
	}
	// 验证 CacheKeyStrategy 的值
	if c.CacheKeyStrategy != "lastQuestion" && c.CacheKeyStrategy != "allQuestions" && c.CacheKeyStrategy != "disable" {
		return fmt.Errorf("invalid CacheKeyStrategy: %s", c.CacheKeyStrategy)
	}
	// 如果启用了语义化缓存，确保必要的组件已配置
	if c.EnableSemanticCache {
		if c.embeddingProviderConfig.GetProviderType() == "" {
			return fmt.Errorf("semantic cache is enabled but embedding provider is not configured")
		}
	}
	return nil
}

func (c *PluginConfig) Complete(log wrapper.Log) error {
	var err error
	if c.embeddingProviderConfig.GetProviderType() != "" {
		c.embeddingProvider, err = embedding.CreateProvider(c.embeddingProviderConfig)
		if err != nil {
			return err
		}
	} else {
		log.Info("embedding provider is not configured")
		c.embeddingProvider = nil
	}
	if c.cacheProviderConfig.GetProviderType() != "" {
		c.cacheProvider, err = cache.CreateProvider(c.cacheProviderConfig)
		if err != nil {
			return err
		}
	} else {
		log.Info("cache provider is not configured")
		c.cacheProvider = nil
	}
	c.vectorProvider, err = vector.CreateProvider(c.vectorProviderConfig)
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
