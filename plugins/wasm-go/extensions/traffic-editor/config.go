package main

import (
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/traffic-editor/pkg"
)

type PluginConfig struct {
	DefaultConfig      *pkg.CommandSet              `json:"defaultConfig,omitempty"`
	ConditionalConfigs []*pkg.ConditionalCommandSet `json:"conditionalConfigs,omitempty"`
}

func (c *PluginConfig) FromJson(json gjson.Result) error {
	c.DefaultConfig = nil
	defaultConfigJson := json.Get("defaultConfig")
	if defaultConfigJson.Exists() && defaultConfigJson.IsObject() {
		c.DefaultConfig = &pkg.CommandSet{}
		if err := c.DefaultConfig.FromJson(defaultConfigJson); err != nil {
			return err
		}
	}

	c.ConditionalConfigs = nil
	conditionalConfigsJson := json.Get("conditionalConfigs")
	if conditionalConfigsJson.Exists() && conditionalConfigsJson.IsArray() {
		for _, item := range conditionalConfigsJson.Array() {
			config := &pkg.ConditionalCommandSet{}
			if err := config.FromJson(item); err != nil {
				return err
			}
			c.ConditionalConfigs = append(c.ConditionalConfigs, config)
		}
	}

	return nil
}
