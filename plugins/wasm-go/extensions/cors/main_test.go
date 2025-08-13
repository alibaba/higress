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

package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本 CORS 配置
var basicCorsConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"allow_origins": []string{
			"http://example.com",
			"https://example.com",
		},
		"allow_methods": []string{
			"GET",
			"POST",
			"OPTIONS",
		},
		"allow_headers": []string{
			"Content-Type",
			"Authorization",
		},
		"expose_headers": []string{
			"X-Custom-Header",
		},
		"allow_credentials": false,
		"max_age":           3600,
	})
	return data
}()

// 测试配置：允许所有 Origin 的配置
var allowAllOriginsConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"allow_origins": []string{
			"*",
		},
		"allow_methods": []string{
			"*",
		},
		"allow_headers": []string{
			"*",
		},
		"expose_headers": []string{
			"*",
		},
		"allow_credentials": false,
		"max_age":           7200,
	})
	return data
}()

// 测试配置：带模式匹配的配置
var patternMatchConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"allow_origin_patterns": []string{
			"http://*.example.com",
			"http://*.example.org:[8080,9090]",
		},
		"allow_methods": []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
		},
		"allow_headers": []string{
			"Content-Type",
			"Token",
			"Authorization",
		},
		"expose_headers": []string{
			"X-Custom-Header",
			"X-Env-UTM",
		},
		"allow_credentials": true,
		"max_age":           1800,
	})
	return data
}()

// 测试配置：允许凭据的配置
var credentialsConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"allow_origin_patterns": []string{
			"*",
		},
		"allow_methods": []string{
			"GET",
			"POST",
		},
		"allow_headers": []string{
			"Content-Type",
			"Authorization",
		},
		"expose_headers": []string{
			"X-Custom-Header",
		},
		"allow_credentials": true,
		"max_age":           86400,
	})
	return data
}()

// 测试配置：默认值配置
var defaultConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本 CORS 配置解析
		t.Run("basic cors config", func(t *testing.T) {
			host, status := test.NewTestHost(basicCorsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试允许所有 Origin 的配置解析
		t.Run("allow all origins config", func(t *testing.T) {
			host, status := test.NewTestHost(allowAllOriginsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试带模式匹配的配置解析
		t.Run("pattern match config", func(t *testing.T) {
			host, status := test.NewTestHost(patternMatchConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试允许凭据的配置解析
		t.Run("credentials config", func(t *testing.T) {
			host, status := test.NewTestHost(credentialsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试默认值配置解析
		t.Run("default config", func(t *testing.T) {
			host, status := test.NewTestHost(defaultConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试简单 CORS 请求头处理
		t.Run("simple cors request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicCorsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含 Origin
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"origin", "http://example.com"},
			})

			// 有效的 CORS 请求应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试预检请求头处理
		t.Run("preflight request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicCorsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置预检请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "OPTIONS"},
				{"origin", "http://example.com"},
				{"access-control-request-method", "POST"},
				{"access-control-request-headers", "Content-Type, Authorization"},
			})

			// 预检请求应该返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			host.CompleteHttp()
		})

		// 测试无效 Origin 的请求头处理
		t.Run("invalid origin request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicCorsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含无效的 Origin
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"origin", "http://invalid.com"},
			})

			// 无效的 CORS 请求应该返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			host.CompleteHttp()
		})

		// 测试允许所有 Origin 的请求头处理
		t.Run("allow all origins request headers", func(t *testing.T) {
			host, status := test.NewTestHost(allowAllOriginsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含任意 Origin
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"origin", "http://any-domain.com"},
			})

			// 允许所有 Origin 的配置应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试模式匹配的请求头处理
		t.Run("pattern match request headers", func(t *testing.T) {
			host, status := test.NewTestHost(patternMatchConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含匹配模式的 Origin
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"origin", "http://sub.example.com"},
			})

			// 匹配模式的 Origin 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试非 CORS 请求头处理
		t.Run("non-cors request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicCorsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，不包含 Origin
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
			})

			// 非 CORS 请求应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})
	})
}

func TestOnHttpResponseHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 CORS 响应头处理
		t.Run("cors response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicCorsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"origin", "http://example.com"},
			})

			// 处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证是否添加了 CORS 响应头
			responseHeaders := host.GetResponseHeaders()
			allowOriginFound := false
			exposeHeadersFound := false

			for _, header := range responseHeaders {
				if strings.EqualFold(header[0], "access-control-allow-origin") {
					allowOriginFound = true
				}
				if strings.EqualFold(header[0], "access-control-expose-headers") {
					exposeHeadersFound = true
				}
			}

			// 对于简单请求，只添加必要的 CORS 头
			require.True(t, allowOriginFound, "Access-Control-Allow-Origin header should be added")
			require.True(t, exposeHeadersFound, "Access-Control-Expose-Headers header should be added")

			// 对于简单请求，不添加 AllowMethods 和 AllowHeaders（这些只在预检请求时添加）
			allowMethodsFound := false
			allowHeadersFound := false
			for _, header := range responseHeaders {
				if strings.EqualFold(header[0], "access-control-allow-methods") {
					allowMethodsFound = true
				}
				if strings.EqualFold(header[0], "access-control-allow-headers") {
					allowHeadersFound = true
				}
			}
			require.False(t, allowMethodsFound, "Access-Control-Allow-Methods header should not be added for simple requests")
			require.False(t, allowHeadersFound, "Access-Control-Allow-Headers header should not be added for simple requests")

			host.CompleteHttp()
		})

		// 测试非 CORS 请求的响应头处理
		t.Run("non-cors response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicCorsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理请求头，不包含 Origin
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
			})

			// 处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证是否没有添加 CORS 响应头
			responseHeaders := host.GetResponseHeaders()
			corsHeaderFound := false
			for _, header := range responseHeaders {
				if strings.HasPrefix(strings.ToLower(header[0]), "access-control-") {
					corsHeaderFound = true
					break
				}
			}
			require.False(t, corsHeaderFound, "CORS headers should not be added for non-CORS requests")

			host.CompleteHttp()
		})

		// 测试允许凭据的响应头处理
		t.Run("credentials response headers", func(t *testing.T) {
			host, status := test.NewTestHost(credentialsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"origin", "http://any-domain.com"},
			})

			// 处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证是否添加了允许凭据的响应头
			responseHeaders := host.GetResponseHeaders()
			allowCredentialsFound := false
			for _, header := range responseHeaders {
				if strings.EqualFold(header[0], "access-control-allow-credentials") {
					allowCredentialsFound = true
					require.Equal(t, "true", header[1])
					break
				}
			}
			require.True(t, allowCredentialsFound, "Access-Control-Allow-Credentials header should be added")

			host.CompleteHttp()
		})

		// 测试预检请求的响应头处理
		t.Run("preflight response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicCorsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理预检请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "OPTIONS"},
				{"origin", "http://example.com"},
				{"access-control-request-method", "POST"},
				{"access-control-request-headers", "Content-Type, Authorization"},
			})

			// 预检请求应该返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			host.CompleteHttp()
		})
	})
}

func TestCompleteFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("complete cors request flow", func(t *testing.T) {
			host, status := test.NewTestHost(basicCorsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 1. 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"origin", "http://example.com"},
			})

			// 有效的 CORS 请求应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 2. 处理响应头
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证完整的 CORS 流程
			responseHeaders := host.GetResponseHeaders()

			// 验证是否添加了必要的 CORS 响应头
			allowOriginFound := false
			exposeHeadersFound := false

			for _, header := range responseHeaders {
				if strings.EqualFold(header[0], "access-control-allow-origin") {
					allowOriginFound = true
				}
				if strings.EqualFold(header[0], "access-control-expose-headers") {
					exposeHeadersFound = true
				}
			}

			require.True(t, allowOriginFound, "Access-Control-Allow-Origin header should be added")
			require.True(t, exposeHeadersFound, "Access-Control-Allow-Headers header should be added")

			// 对于简单请求，不添加 AllowMethods 和 AllowHeaders（这些只在预检请求时添加）
			allowMethodsFound := false
			allowHeadersFound := false
			for _, header := range responseHeaders {
				if strings.EqualFold(header[0], "access-control-allow-methods") {
					allowMethodsFound = true
				}
				if strings.EqualFold(header[0], "access-control-allow-headers") {
					allowHeadersFound = true
				}
			}
			require.False(t, allowMethodsFound, "Access-Control-Allow-Methods header should not be added for simple requests")
			require.False(t, allowHeadersFound, "Access-Control-Allow-Headers header should not be added for simple requests")

			host.CompleteHttp()
		})
	})
}
