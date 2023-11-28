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

package main

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

var (
	ruleSet         bool            // 插件是否至少在一个 domain 或 route 上生效
	protectionSpace = "MSE Gateway" // 认证失败时，返回响应头 WWW-Authenticate: Key realm=MSE Gateway

)

func main() {
	wrapper.SetCtx(
		"key-auth", // middleware name
		wrapper.ParseOverrideConfigBy(parseGlobalConfig, parseOverrideRuleConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type Consumer struct {
	Name       string `yaml:"name"`
	Credential string `yaml:"credential"`
}

type KeyAuthConfig struct {
	globalAuth *bool      `yaml:"global_auth"` //是否开启全局认证. 若不开启全局认证，则全局配置只提供凭证信息。只有在域名或路由上进行了配置才会启用认证。
	Keys       []string   `yaml:"keys"`        // key auth names
	InQuery    bool       `yaml:"in_query,omitempty"`
	InHeader   bool       `yaml:"in_header,omitempty"`
	consumers  []Consumer `yaml:"consumers"`
	allow      []string   `yaml:"allow"`

	credential2Name map[string]string `yaml:"-"`
}

func parseGlobalConfig(json gjson.Result, global *KeyAuthConfig, log wrapper.Log) error {
	log.Debug("global config")

	// init
	ruleSet = false
	global.credential2Name = make(map[string]string)

	// global_auth
	globalAuth := json.Get("global_auth")
	if globalAuth.Exists() {
		ga := globalAuth.Bool()
		global.globalAuth = &ga
	}

	// keys
	names := json.Get("keys")
	if !names.Exists() {
		return errors.New("keys is required")
	}
	if len(names.Array()) == 0 {
		return errors.New("keys cannot be empty")
	}

	for _, name := range names.Array() {
		global.Keys = append(global.Keys, name.String())
	}

	// in_query and in_header
	in_query := json.Get("in_query")
	in_header := json.Get("in_header")
	if !in_query.Exists() && !in_header.Exists() {
		return errors.New("must one of in_query/in_header required")
	}

	if in_query.Exists() {
		global.InQuery = in_query.Bool()
	}
	if in_header.Exists() {
		global.InHeader = in_header.Bool()
	}

	// consumers
	consumers := json.Get("consumers")
	if !consumers.Exists() {
		return errors.New("consumers is required")
	}
	if len(consumers.Array()) == 0 {
		return errors.New("consumers cannot be empty")
	}

	for _, item := range consumers.Array() {
		name := item.Get("name")
		if !name.Exists() || name.String() == "" {
			return errors.New("consumer name is required")
		}
		credential := item.Get("credential")
		if !credential.Exists() || credential.String() == "" {
			return errors.New("consumer credential is required")
		}
		if _, ok := global.credential2Name[credential.String()]; ok {
			return errors.New("duplicate consumer credential: " + credential.String())
		}

		consumer := Consumer{
			Name:       name.String(),
			Credential: credential.String(),
		}
		global.consumers = append(global.consumers, consumer)
		global.credential2Name[credential.String()] = name.String()
	}
	return nil
}

func parseOverrideRuleConfig(json gjson.Result, global KeyAuthConfig, config *KeyAuthConfig, log wrapper.Log) error {
	log.Debug("domain/route config")

	*config = global

	allow := json.Get("allow")
	if !allow.Exists() {
		return errors.New("allow is required")
	}
	if len(allow.Array()) == 0 {
		return errors.New("allow cannot be empty")
	}

	for _, item := range allow.Array() {
		config.allow = append(config.allow, item.String())
	}
	ruleSet = true

	return nil
}

// key-auth 插件认证逻辑：
// - global_auth == true 开启全局生效：
//   - 若当前 domain/route 未配置 allow 列表，即未配置该插件：则在所有 consumers 中查找，如果找到则认证通过，否则认证失败 (1*)
//   - 若当前 domain/route 配置了该插件：则在 allow 列表中查找，如果找到则认证通过，否则认证失败
//
// - global_auth == false 非全局生效：(2*)
//   - 若当前 domain/route 未配置该插件：则直接放行
//   - 若当前 domain/route 配置了该插件：则在 allow 列表中查找，如果找到则认证通过，否则认证失败
//
// - global_auth 未设置：
//   - 若没有一个 domain/route 配置该插件：则遵循 (1*)
//   - 若有至少一个 domain/route 配置该插件：则遵循 (2*)
func onHttpRequestHeaders(ctx wrapper.HttpContext, config KeyAuthConfig, log wrapper.Log) types.Action {
	var (
		noAllow            = len(config.allow) == 0 // 未配置 allow 列表，表示插件在该 domain/route 未生效
		globalAuthNoSet    = config.globalAuth == nil
		globalAuthSetTrue  = !globalAuthNoSet && *config.globalAuth
		globalAuthSetFalse = !globalAuthNoSet && !*config.globalAuth
	)
	// 不需要认证而直接放行的情况：
	// - global_auth == false 且 当前 domain/route 未配置该插件
	// - global_auth 未设置 且 有至少一个 domain/route 配置该插件 且 当前 domain/route 未配置该插件
	if globalAuthSetFalse || (globalAuthNoSet && ruleSet) {
		if noAllow {
			log.Info("authorization is not required")
			return types.ActionContinue
		}
	}

	// 以下需要认证：
	// - 从 header 中获取 tokens 信息
	// - 从 query 中获取 tokens 信息
	var tokens []string
	if config.InHeader {
		// 匹配keys中的 keyname
		for _, key := range config.Keys {
			value, err := proxywasm.GetHttpRequestHeader(key)
			if err == nil && value != "" {
				tokens = append(tokens, value)
			}
		}
	} else if config.InQuery {
		requestUrl, _ := proxywasm.GetHttpRequestHeader(":path")
		url, _ := url.Parse(requestUrl)
		queryValues := url.Query()
		for _, key := range config.Keys {
			values, ok := queryValues[key]
			if ok && len(values) > 0 {
				tokens = append(tokens, values...)
			}
		}
	}

	// header/query
	if len(tokens) > 1 {
		return deniedMutiKeyAuthData()
	} else if len(tokens) <= 0 {
		return deniedNoKeyAuthData()
	}

	// 验证token
	name, ok := config.credential2Name[tokens[0]]
	if !ok {
		log.Warnf("credential %q is not configured", tokens[0])
		return deniedUnauthorizedConsumer()
	}

	// 全局生效：
	// - global_auth == true 且 当前 domain/route 未配置该插件
	// - global_auth 未设置 且 没有任何一个 domain/route 配置该插件
	if (globalAuthSetTrue && noAllow) || (globalAuthNoSet && !ruleSet) {
		log.Infof("consumer %q authenticated", name)
		return authenticated(name)
	}

	// 全局生效，但当前 domain/route 配置了 allow 列表
	if globalAuthSetTrue && !noAllow {
		if !contains(config.allow, name) {
			log.Warnf("consumer %q is not allowed", name)
			return deniedUnauthorizedConsumer()
		}
		log.Infof("consumer %q authenticated", name)
		return authenticated(name)
	}

	// 非全局生效
	if globalAuthSetFalse || (globalAuthNoSet && ruleSet) {
		if !noAllow { // 配置了 allow 列表
			if !contains(config.allow, name) {
				log.Warnf("consumer %q is not allowed", name)
				return deniedUnauthorizedConsumer()
			}
			log.Infof("consumer %q authenticated", name)
			return authenticated(name)
		}
	}

	return types.ActionContinue
}

func deniedMutiKeyAuthData() types.Action {
	_ = proxywasm.SendHttpResponse(401, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by Key Auth check. Muti Key Authentication information found."), -1)
	return types.ActionContinue
}

func deniedNoKeyAuthData() types.Action {
	_ = proxywasm.SendHttpResponse(401, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by Key Auth check. No Key Authentication information found."), -1)
	return types.ActionContinue
}

func deniedUnauthorizedConsumer() types.Action {
	_ = proxywasm.SendHttpResponse(403, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by Key Auth check. Unauthorized consumer."), -1)
	return types.ActionContinue
}

func authenticated(name string) types.Action {
	_ = proxywasm.AddHttpRequestHeader("X-Mse-Consumer", name)
	return types.ActionContinue
}

func contains(arr []string, item string) bool {
	for _, i := range arr {
		if i == item {
			return true
		}
	}
	return false
}

func WWWAuthenticateHeader(realm string) [][2]string {
	return [][2]string{
		{"WWW-Authenticate", fmt.Sprintf("Key realm=%s", realm)},
	}
}
