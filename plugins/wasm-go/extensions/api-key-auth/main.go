// Copyright (c) 2024 Alibaba Group Holding Ltd.
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

// API Key 验证 WASM 插件
// 从 Nginx Lua 脚本转换而来
package main

import (
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"api-key-auth",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

// 配置结构
type ApiKeyAuthConfig struct {
	// 有效的 API Keys 列表
	ValidKeys []string `required:"true" yaml:"valid_keys" json:"valid_keys"`
	// API Key 请求头名称（默认: X-API-Key）
	HeaderName string `required:"false" yaml:"header_name" json:"header_name"`
}

// 解析配置
func parseConfig(json gjson.Result, config *ApiKeyAuthConfig, log wrapper.Log) error {
	// 解析有效的 API Keys
	validKeys := json.Get("valid_keys")
	if validKeys.Exists() && validKeys.IsArray() {
		for _, key := range validKeys.Array() {
			config.ValidKeys = append(config.ValidKeys, key.String())
		}
	}

	// 如果配置中没有提供 valid_keys，使用默认值（对应 Lua 脚本中的硬编码值）
	if len(config.ValidKeys) == 0 {
		config.ValidKeys = []string{"key123", "key456", "key789"}
	}

	// 解析请求头名称
	headerName := json.Get("header_name").String()
	if headerName == "" {
		config.HeaderName = "X-API-Key"
	} else {
		config.HeaderName = headerName
	}

	log.Infof("API Key Auth 配置加载成功，有效密钥数量: %d, 请求头: %s",
		len(config.ValidKeys), config.HeaderName)

	return nil
}

// 处理请求头 - 实现 API Key 验证逻辑
func onHttpRequestHeaders(ctx wrapper.HttpContext, config ApiKeyAuthConfig, log wrapper.Log) types.Action {
	// 获取 API Key (对应 Lua: ngx.req.get_headers()["X-API-Key"])
	apiKey, err := proxywasm.GetHttpRequestHeader(config.HeaderName)

	// 如果没有 API Key，返回 401 (对应 Lua: if not api_key then)
	if err != nil || apiKey == "" {
		log.Warnf("缺少 API Key 请求头: %s", config.HeaderName)
		// 返回 401 未授权 (对应 Lua: ngx.status = 401, ngx.say, ngx.exit)
		err := proxywasm.SendHttpResponse(401,
			[][2]string{
				{"Content-Type", "application/json"},
			},
			[]byte(`{"error": "Missing API Key"}`), -1)
		if err != nil {
			log.Errorf("发送响应失败: %v", err)
		}
		return types.ActionPause
	}

	// 验证 API Key 是否在有效列表中 (对应 Lua: if not valid_keys[api_key] then)
	valid := false
	for _, validKey := range config.ValidKeys {
		if apiKey == validKey {
			valid = true
			break
		}
	}

	if !valid {
		log.Warnf("无效的 API Key: %s", apiKey)
		// 返回 403 禁止访问 (对应 Lua: ngx.status = 403, ngx.say, ngx.exit)
		err := proxywasm.SendHttpResponse(403,
			[][2]string{
				{"Content-Type", "application/json"},
			},
			[]byte(`{"error": "Invalid API Key"}`), -1)
		if err != nil {
			log.Errorf("发送响应失败: %v", err)
		}
		return types.ActionPause
	}

	// API Key 验证成功，添加认证标记头 (对应 Lua: ngx.req.set_header)
	err = proxywasm.AddHttpRequestHeader("X-Authenticated", "true")
	if err != nil {
		log.Warnf("添加 X-Authenticated 请求头失败: %v", err)
	}

	err = proxywasm.AddHttpRequestHeader("X-Auth-Method", "api-key")
	if err != nil {
		log.Warnf("添加 X-Auth-Method 请求头失败: %v", err)
	}

	// 记录日志 (对应 Lua: ngx.log(ngx.INFO, ...))
	log.Infof("API Key 验证成功: %s", apiKey)

	return types.ActionContinue
}

