package config

import (
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/provider"
)

// @Name ai-proxy
// @Category custom
// @Phase UNSPECIFIED_PHASE
// @Priority 0
// @Title zh-CN AI代理
// @Description zh-CN 通过AI助手提供智能对话服务
// @IconUrl https://img.alicdn.com/imgextra/i1/O1CN018iKKih1iVx287RltL_!!6000000004419-2-tps-42-42.png
// @Version 0.1.0
//
// @Contact.name CH3CHO
// @Contact.url https://github.com/CH3CHO
// @Contact.email ch3cho@qq.com
//
// @Example
// { "provider": { "type": "qwen", "apiToken": "YOUR_DASHSCOPE_API_TOKEN", "modelMapping": { "*": "qwen-turbo" } } }
// @End
type PluginConfig struct {
	// @Title zh-CN AI服务提供商配置
	// @Description zh-CN AI服务提供商配置，包含API接口、模型和知识库文件等信息
	providerConfig provider.ProviderConfig `required:"true" yaml:"provider"`

	provider provider.Provider `yaml:"-"`
}

func (c *PluginConfig) FromJson(json gjson.Result) {
	c.providerConfig.FromJson(json.Get("provider"))
}

func (c *PluginConfig) Validate() error {
	if err := c.providerConfig.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *PluginConfig) Complete() error {
	var err error
	c.provider, err = provider.CreateProvider(c.providerConfig)
	return err
}

func (c *PluginConfig) GetProvider() provider.Provider {
	return c.provider
}
