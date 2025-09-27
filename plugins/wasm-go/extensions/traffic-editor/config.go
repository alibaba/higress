package main

import (
	"github.com/tidwall/gjson"
)

type PluginConfig struct {
	DefaultConfig      *CommandSet              `json:"defaultConfig,omitempty"`
	ConditionalConfigs []*ConditionalCommandSet `json:"conditionalConfigs,omitempty"`
}

func (c *PluginConfig) FromJson(json gjson.Result) error {
	c.DefaultConfig = nil
	defaultConfigJson := json.Get("defaultConfig")
	if defaultConfigJson.Exists() && defaultConfigJson.IsObject() {
		c.DefaultConfig = &CommandSet{}
		if err := c.DefaultConfig.FromJson(defaultConfigJson); err != nil {
			return err
		}
	}

	c.ConditionalConfigs = nil
	conditionalConfigsJson := json.Get("conditionalConfigs")
	if conditionalConfigsJson.Exists() && conditionalConfigsJson.IsArray() {
		for _, item := range conditionalConfigsJson.Array() {
			config := &ConditionalCommandSet{}
			if err := config.FromJson(item); err != nil {
				return err
			}
			c.ConditionalConfigs = append(c.ConditionalConfigs, config)
		}
	}

	return nil
}
