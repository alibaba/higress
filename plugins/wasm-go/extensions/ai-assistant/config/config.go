package config

import (
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-assistant/provider"
)

// @Name ai-assistant
// @Category custom
// @Phase UNSPECIFIED_PHASE
// @Priority 0
// @Title zh-CN Hello World
// @Description zh-CN This is a demo plugin
// @IconUrl
// @Version 0.1.0
//
// @Contact.name
// @Contact.url
// @Contact.email
//
// @Example
// firstField: hello
// secondField: world
// @End
type PluginConfig struct {
	// @Title 第一个字段，注解格式为 @Title [语言] [标题]，语言缺省值为 en-US
	// @Description 字符串的前半部分，注解格式为 @Description [语言] [描述]，语言缺省值为 en-US
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
