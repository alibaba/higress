// Copyright (c) 2023 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"fmt"

	"github.com/go-jose/go-jose/v3"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/tidwall/gjson"
)

// RuleSet 插件是否至少在一个 domain 或 route 上生效
var RuleSet bool

// ParseGlobalConfig 从wrapper提供的配置中解析并转换到插件运行时需要使用的配置。
// 此处解析的是全局配置，域名和路由级配置由 ParseRuleConfig 负责。
func ParseGlobalConfig(json gjson.Result, config *JWTAuthConfig, log log.Log) error {
	RuleSet = false
	consumers := json.Get("consumers")
	if !consumers.IsArray() {
		return fmt.Errorf("failed to parse configuration for consumers: consumers is not a array")
	}

	consumerNames := map[string]struct{}{}
	for _, v := range consumers.Array() {
		c, err := ParseConsumer(v, consumerNames)
		if err != nil {
			log.Warn(err.Error())
			continue
		}
		config.Consumers = append(config.Consumers, c)
	}
	if len(config.Consumers) == 0 {
		return fmt.Errorf("at least one consumer should be configured for a rule")
	}

	return nil
}

// ParseRuleConfig 从wrapper提供的配置中解析并转换到插件运行时需要使用的配置。
// 此处解析的是域名和路由级配置，全局配置由 ParseConfig 负责。
func ParseRuleConfig(json gjson.Result, global JWTAuthConfig, config *JWTAuthConfig, log log.Log) error {
	// override config via global
	*config = global

	allow := json.Get("allow")
	if !allow.Exists() {
		return fmt.Errorf("allow is required")
	}

	if len(allow.Array()) == 0 {
		return fmt.Errorf("allow cannot be empty")
	}

	for _, item := range allow.Array() {
		config.Allow = append(config.Allow, item.String())
	}

	RuleSet = true
	return nil
}

func ParseConsumer(consumer gjson.Result, names map[string]struct{}) (c *Consumer, err error) {
	c = &Consumer{}

	// 从gjson中取得原始JSON字符串，并使用标准库反序列化，以降低代码复杂度。
	err = json.Unmarshal([]byte(consumer.Raw), c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse consumer: %s", err.Error())
	}

	// 检查consumer是否重复
	if _, ok := names[c.Name]; ok {
		return nil, fmt.Errorf("consumer already exists: %s", c.Name)
	}

	// 检查JWKs是否合法
	jwks := &jose.JSONWebKeySet{}
	err = json.Unmarshal([]byte(c.JWKs), jwks)
	if err != nil {
		return nil, fmt.Errorf("jwks is invalid, consumer:%s, status:%s, jwks:%s", c.Name, err.Error(), c.JWKs)
	}

	// 检查是否需要使用默认jwt抽取来源
	if c.FromHeaders == nil && c.FromParams == nil && c.FromCookies == nil {
		c.FromHeaders = &DefaultFromHeader
		c.FromParams = &DefaultFromParams
		c.FromCookies = &DefaultFromCookies
	}

	// 检查ClaimsToHeaders
	if c.ClaimsToHeaders != nil {
		// header去重
		c2h := map[string]struct{}{}

		// 此处需要先把指针解引用到临时变量
		tmp := *c.ClaimsToHeaders
		for i := range tmp {
			if _, ok := c2h[tmp[i].Header]; ok {
				return nil, fmt.Errorf("claim to header already exists: %s", c2h[tmp[i].Header])
			}
			c2h[tmp[i].Header] = struct{}{}

			// 为Override填充默认值
			if tmp[i].Override == nil {
				tmp[i].Override = &DefaultClaimToHeaderOverride
			}
		}
	}

	// 为ClockSkewSeconds填充默认值
	if c.ClockSkewSeconds == nil {
		c.ClockSkewSeconds = &DefaultClockSkewSeconds
	}

	// 为KeepToken填充默认值
	if c.KeepToken == nil {
		c.KeepToken = &DefaultKeepToken
	}

	// consumer合法，记录consumer名称
	names[c.Name] = struct{}{}
	return c, nil
}
