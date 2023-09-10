// Copyright (c) 2022 Alibaba Group Holding Ltd.
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

// The 'Basic' HTTP Authentication Scheme: https://datatracker.ietf.org/doc/html/rfc7617

package main

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/pkg/errors"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"basic-auth",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

// @Name basic-auth
// @Category auth
// @Phase AUTHN
// @Priority 320
// @Title zh-CN Basic Auth
// @Description zh-CN 本插件实现了基于 HTTP Basic Auth 标准进行认证鉴权的功能。
// @Description en-US This plugin implements an authentication function based on HTTP Basic Auth standard.
// @IconUrl https://img.alicdn.com/imgextra/i4/O1CN01BPFGlT1pGZ2VDLgaH_!!6000000005333-2-tps-42-42.png
// @Version 1.0.0
//
// @Contact.name Higress Team
// @Contact.url http://higress.io/
// @Contact.email admin@higress.io
//
// @Example
// global_auth: false
// consumers:
//   - name: consumer1
//     credential: admin:123456
//   - name: consumer2
//     credential: guest:abc
//
// @End
type BasicAuthConfig struct {
	// @Title 是否开启全局认证
	// @Title en-US Enable Global Auth
	// @Description 若不开启全局认证，则全局配置只提供凭证信息。只有在域名或路由上进行了配置才会启用认证。
	// @Description en-US If set to false, only consumer info will be accepted from the global config. Auth feature shall only be enabled if the corresponding domain or route is configured.
	// @Scope GLOBAL
	globalAuth bool `yaml:"global_auth"`

	// @Title 调用方列表
	// @Title en-US Consumer List
	// @Description 服务调用方列表，用于对请求进行认证。
	// @Description en-US List of service consumers which will be used in request authentication.
	// @Scope GLOBAL
	consumers []Consumer `yaml:"consumers"`

	// @Title 授权访问的调用方列表
	// @Title en-US Allowed Consumers
	// @Description 对于匹配上述条件的请求，允许访问的调用方列表。
	// @Description en-US Consumers to be allowed for matched requests.
	allow []string `yaml:"allow"`
}

// BasicAuthConfig.globalAuth 和 .consumers 实际上没什么用，仅用于生成 spec.yaml

type Consumer struct {
	// @Title 名称
	// @Title en-US Name
	// @Description 该调用方的名称。
	// @Description en-US The name of the consumer.
	name string `yaml:"name"`

	// @Title 访问凭证
	// @Title en-US Credential
	// @Description 该调用方的访问凭证。
	// @Description en-US The credential of the consumer.
	// @Scope GLOBAL
	credential string `yaml:"credential"`
}

// 保存全局配置
type globalConfig struct {
	user2Passwd      map[string]string // credential username -> credential password
	credential2Name  map[string]string // username:password -> consumer name
	globalAuth       bool
	globalAuthSet    bool // 是否配置了 global_auth 字段
	domainOrRouteSet bool // 插件是否至少在一个 domain 或 route 上生效
}

func newGlobalConfig() *globalConfig {
	return &globalConfig{
		user2Passwd:     make(map[string]string),
		credential2Name: make(map[string]string),
	}
}

var (
	gc              *globalConfig
	protectionSpace = "MSE Gateway"
)

// 非常依赖 pkg/matcher ParseRuleConfig 中先解析 global config，
// 再解析 domain/route config 的逻辑， 否则 gc 保存的全局变量就会出错
func parseConfig(json gjson.Result, config *BasicAuthConfig, log wrapper.Log) error {
	// global config
	consumers := json.Get("consumers")
	if consumers.Exists() {
		log.Debug("global config")
		gc = newGlobalConfig()
		for _, item := range consumers.Array() {
			name := item.Get("name")
			if !name.Exists() || name.String() == "" {
				return errors.New("consumer name is required")
			}
			credential := item.Get("credential")
			if !credential.Exists() || credential.String() == "" {
				return errors.New("consumer credential is required")
			}
			if _, ok := gc.credential2Name[credential.String()]; ok {
				return errors.Errorf("duplicate consumer credential: %s", credential.String())
			}

			consumer := Consumer{
				name:       name.String(),
				credential: credential.String(),
			}
			config.consumers = append(config.consumers, consumer)

			userAndPasswd := strings.Split(consumer.credential, ":")
			if len(userAndPasswd) != 2 {
				return errors.Errorf("invalid credential format: %s", consumer.credential)
			}
			gc.user2Passwd[userAndPasswd[0]] = userAndPasswd[1]
			gc.credential2Name[consumer.credential] = consumer.name
		}

		globalAuth := json.Get("global_auth")
		if globalAuth.Exists() {
			gc.globalAuthSet = true
			gc.globalAuth = globalAuth.Bool()
			config.globalAuth = globalAuth.Bool()
		}
	}

	if gc.globalAuth && len(gc.credential2Name) == 0 {
		return errors.New("global_auth is true, but no consumer is configured")
	}

	// domain/route config
	allow := json.Get("allow")
	if consumers.Exists() && allow.Exists() {
		return errors.New("'allow' (domain/route) and 'consumers' (global) cannot be configured at the same level")
	}
	if allow.Exists() {
		log.Debug("domain/route config")
		if len(allow.Array()) == 0 {
			return errors.New("allow list is empty")
		}

		gc.domainOrRouteSet = true
		for _, item := range allow.Array() {
			config.allow = append(config.allow, item.String())
		}
	}

	return nil
}

// basic-auth 插件认证逻辑：
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
func onHttpRequestHeaders(ctx wrapper.HttpContext, config BasicAuthConfig, log wrapper.Log) types.Action {
	// log.Debugf("global config: %+v", gc)
	// log.Debugf("allow: %v", config.allow)

	// 未配置 allow 列表，表示插件在该 domain/route 未生效
	noAllow := len(config.allow) == 0

	// 不需要认证而直接放行的情况：
	// - global_auth == false 且 当前 domain/route 未配置该插件
	// - global_auth 未设置 且 有至少一个 domain/route 配置该插件 且 当前 domain/route 未配置该插件
	if (gc.globalAuthSet && !gc.globalAuth) ||
		(!gc.globalAuthSet && gc.domainOrRouteSet) {
		if noAllow {
			log.Info("authorization is not required")
			return types.ActionContinue
		}
	}

	// 以下为需要认证的情况：
	auth, err := proxywasm.GetHttpRequestHeader("Authorization")
	if err != nil {
		log.Warnf("failed to get authorization: %v", err)
		return deniedNoBasicAuthData()
	}
	if auth == "" {
		log.Warnf("authorization is empty")
		return deniedNoBasicAuthData()
	}
	if !strings.HasPrefix(auth, "Basic ") {
		log.Warnf("authorization has no prefix 'Basic '")
		return deniedNoBasicAuthData()
	}

	encodedCredential := strings.TrimPrefix(auth, "Basic ")
	credentialByte, err := base64.StdEncoding.DecodeString(encodedCredential)
	if err != nil {
		log.Warnf("failed to decode authorization %q: %v", string(credentialByte), err)
		return deniedInvalidCredentials()
	}

	credential := string(credentialByte)
	userAndPasswd := strings.Split(credential, ":")
	if len(userAndPasswd) != 2 {
		log.Warnf("invalid credential format: %s", credential)
		return deniedInvalidCredentials()
	}

	user, passwd := userAndPasswd[0], userAndPasswd[1]
	if correctPasswd, ok := gc.user2Passwd[user]; !ok {
		log.Warnf("credential username %q is not configured", user)
		return deniedInvalidCredentials()
	} else {
		if passwd != correctPasswd {
			log.Warnf("credential password %q is not correct", passwd)
			return deniedInvalidCredentials()
		}
	}

	// 以下为 username 和 password 正确的情况：
	name, ok := gc.credential2Name[credential]
	if !ok { // 理论上该分支永远不可达，因为 username 和 password 都是从 credential 中获取的
		log.Warnf("credential %q is not configured", credential)
		return deniedUnauthorizedConsumer()
	}

	// 全局生效：
	// - global_auth == true 且 当前 domain/route 未配置该插件
	// - global_auth 未设置 且 没有任何一个 domain/route 配置该插件
	if (gc.globalAuth && noAllow) ||
		(!gc.globalAuthSet && !gc.domainOrRouteSet) {
		// log.Debug("case 1")
		log.Infof("consumer %q authenticated", name)
		return authenticated(name)
	}

	// 全局生效，但当前 domain/route 配置了 allow 列表
	if gc.globalAuth && !noAllow {
		if !contains(config.allow, name) {
			log.Warnf("consumer %q is not allowed", name)
			return deniedUnauthorizedConsumer()
		}
		// log.Debug("case 2")
		log.Infof("consumer %q authenticated", name)
		return authenticated(name)
	}

	// 非全局生效
	if (gc.globalAuthSet && !gc.globalAuth) ||
		(!gc.globalAuthSet && gc.domainOrRouteSet) {
		if !noAllow { // 配置了 allow 列表
			if !contains(config.allow, name) {
				log.Warnf("consumer %q is not allowed", name)
				return deniedUnauthorizedConsumer()
			}
			// log.Debug("case 3")
			log.Infof("consumer %q authenticated", name)
			return authenticated(name)
		}
	}

	return types.ActionContinue
}

func deniedNoBasicAuthData() types.Action {
	_ = proxywasm.SendHttpResponse(401, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by Basic Auth check. No Basic Authentication information found."), -1)
	return types.ActionContinue
}

func deniedInvalidCredentials() types.Action {
	_ = proxywasm.SendHttpResponse(401, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by Basic Auth check. Invalid username and/or password."), -1)
	return types.ActionContinue
}

func deniedUnauthorizedConsumer() types.Action {
	_ = proxywasm.SendHttpResponse(403, WWWAuthenticateHeader(protectionSpace),
		[]byte("Request denied by Basic Auth check. Unauthorized consumer."), -1)
	return types.ActionContinue
}

func authenticated(name string) types.Action {
	_ = proxywasm.AddHttpRequestHeader("X-Mse-Consumer", name)
	return types.ActionContinue
}

func WWWAuthenticateHeader(realm string) [][2]string {
	return [][2]string{
		{"WWW-Authenticate", fmt.Sprintf("Basic realm=%s", realm)},
	}
}

func contains(arr []string, item string) bool {
	for _, i := range arr {
		if i == item {
			return true
		}
	}
	return false
}
