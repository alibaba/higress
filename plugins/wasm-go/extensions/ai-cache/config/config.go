package config

import (
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/cache"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/embedding"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/vectorDatabase"
	"github.com/tidwall/gjson"
)

type KVExtractor struct {
	// @Title zh-CN 从请求 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	RequestBody string `required:"false" yaml:"requestBody" json:"requestBody"`
	// @Title zh-CN 从响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串
	ResponseBody string `required:"false" yaml:"responseBody" json:"responseBody"`
}

type PluginConfig struct {
	EmbeddingProviderConfig      embedding.ProviderConfig      `required:"true" yaml:"embeddingProvider" json:"embeddingProvider"`
	VectorDatabaseProviderConfig vectorDatabase.ProviderConfig `required:"true" yaml:"vectorBaseProvider" json:"vectorBaseProvider"`
	CacheKeyFrom                 KVExtractor                   `required:"true" yaml:"cacheKeyFrom" json:"cacheKeyFrom"`
	CacheValueFrom               KVExtractor                   `required:"true" yaml:"cacheValueFrom" json:"cacheValueFrom"`
	CacheStreamValueFrom         KVExtractor                   `required:"true" yaml:"cacheStreamValueFrom" json:"cacheStreamValueFrom"`

	CacheKeyPrefix string            `required:"false" yaml:"cacheKeyPrefix" json:"cacheKeyPrefix"`
	RedisConfig    cache.RedisConfig `required:"true" yaml:"redisConfig" json:"redisConfig"`
	// 现在只支持RedisClient作为cacheClient
	redisProvider          cache.Provider          `yaml:"-"`
	embeddingProvider      embedding.Provider      `yaml:"-"`
	vectorDatabaseProvider vectorDatabase.Provider `yaml:"-"`
}

func (c *PluginConfig) FromJson(json gjson.Result) {
	c.EmbeddingProviderConfig.FromJson(json.Get("embeddingProvider"))
	c.VectorDatabaseProviderConfig.FromJson(json.Get("vectorBaseProvider"))
	c.RedisConfig.FromJson(json.Get("redis"))
}

func (c *PluginConfig) Validate() error {
	if err := c.RedisConfig.Validate(); err != nil {
		return err
	}
	if err := c.EmbeddingProviderConfig.Validate(); err != nil {
		return err
	}
	if err := c.VectorDatabaseProviderConfig.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *PluginConfig) Complete() error {
	var err error
	c.embeddingProvider, err = embedding.CreateProvider(c.EmbeddingProviderConfig)
	c.vectorDatabaseProvider, err = vectorDatabase.CreateProvider(c.VectorDatabaseProviderConfig)
	c.redisProvider, err = cache.CreateProvider(c.RedisConfig)
	return err
}

func (c *PluginConfig) GetEmbeddingProvider() embedding.Provider {
	return c.embeddingProvider
}

func (c *PluginConfig) GetVectorDatabaseProvider() vectorDatabase.Provider {
	return c.vectorDatabaseProvider
}

func (c *PluginConfig) GetCacheProvider() cache.Provider {
	return c.redisProvider
}
