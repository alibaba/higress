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

package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：全局限流配置
var globalThresholdConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "routeA-global-limit-rule",
		"global_threshold": map[string]interface{}{
			"query_per_minute": 1000,
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
			"timeout":      1000,
		},
		"show_limit_quota_header": true,
		"rejected_code":           429,
		"rejected_msg":            "Too many requests",
	})
	return data
}()

// 测试配置：基于请求参数的限流配置
var paramLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "routeA-request-param-limit-rule",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_param": "apikey",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "9a342114-ba8a-11ec-b1bf-00163e1250b5",
						"query_per_minute": 10,
					},
					{
						"key":            "a6a6d7f2-ba8a-11ec-bec2-00163e1250b5",
						"query_per_hour": 100,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"show_limit_quota_header": true,
	})
	return data
}()

// 测试配置：基于请求头的限流配置
var headerLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "routeA-request-header-limit-rule",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_header": "x-ca-key",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "102234",
						"query_per_minute": 10,
					},
					{
						"key":            "308239",
						"query_per_hour": 10,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"show_limit_quota_header": true,
	})
	return data
}()

// 测试配置：基于 Consumer 的限流配置
var consumerLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "routeA-consumer-limit-rule",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_consumer": "",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "consumer1",
						"query_per_second": 10,
					},
					{
						"key":            "consumer2",
						"query_per_hour": 100,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"show_limit_quota_header": true,
	})
	return data
}()

// 测试配置：基于 Cookie 的限流配置
var cookieLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "routeA-cookie-limit-rule",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_cookie": "key1",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "value1",
						"query_per_minute": 10,
					},
					{
						"key":            "value2",
						"query_per_hour": 100,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"show_limit_quota_header": true,
		"rejected_code":           200,
		"rejected_msg":            `{"code":-1,"msg":"Too many requests"}`,
	})
	return data
}()

// 测试配置：基于 IP 的限流配置
var ipLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "routeA-client-ip-limit-rule",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_per_ip": "from-header-x-forwarded-for",
				"limit_keys": []map[string]interface{}{
					{
						"key":           "1.1.1.1",
						"query_per_day": 10,
					},
					{
						"key":           "1.1.1.0/24",
						"query_per_day": 100,
					},
					{
						"key":           "0.0.0.0/0",
						"query_per_day": 1000,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"show_limit_quota_header": true,
	})
	return data
}()

// 测试配置：正则表达式限流配置
var regexpLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "routeA-regexp-limit-rule",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_per_param": "apikey",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "regexp:^a.*",
						"query_per_second": 10,
					},
					{
						"key":              "regexp:^b.*",
						"query_per_minute": 100,
					},
					{
						"key":            "*",
						"query_per_hour": 1000,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"show_limit_quota_header": true,
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试全局限流配置解析
		t.Run("global threshold config", func(t *testing.T) {
			host, status := test.NewTestHost(globalThresholdConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试基于请求参数的限流配置解析
		t.Run("param limit config", func(t *testing.T) {
			host, status := test.NewTestHost(paramLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试基于请求头的限流配置解析
		t.Run("header limit config", func(t *testing.T) {
			host, status := test.NewTestHost(headerLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试基于 Consumer 的限流配置解析
		t.Run("consumer limit config", func(t *testing.T) {
			host, status := test.NewTestHost(consumerLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试基于 Cookie 的限流配置解析
		t.Run("cookie limit config", func(t *testing.T) {
			host, status := test.NewTestHost(cookieLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试基于 IP 的限流配置解析
		t.Run("ip limit config", func(t *testing.T) {
			host, status := test.NewTestHost(ipLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试正则表达式限流配置解析
		t.Run("regexp limit config", func(t *testing.T) {
			host, status := test.NewTestHost(regexpLimitConfig)
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
		// 测试全局限流请求头处理
		t.Run("global threshold request headers", func(t *testing.T) {
			host, status := test.NewTestHost(globalThresholdConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			resp := test.CreateRedisRespArray([]interface{}{1000, 999, 60})
			// 模拟 Redis 调用响应（允许请求）
			host.CallOnRedisCall(0, resp)

			host.CompleteHttp()
		})

		// 测试基于请求参数的限流请求头处理
		t.Run("param limit request headers", func(t *testing.T) {
			host, status := test.NewTestHost(paramLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含查询参数
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test?apikey=9a342114-ba8a-11ec-b1bf-00163e1250b5"},
				{":method", "GET"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			resp := test.CreateRedisRespArray([]interface{}{10, 9, 60})
			host.CallOnRedisCall(0, resp)

			host.CompleteHttp()
		})

		// 测试基于请求头的限流请求头处理
		t.Run("header limit request headers", func(t *testing.T) {
			host, status := test.NewTestHost(headerLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含限流键
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"x-ca-key", "102234"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			resp := test.CreateRedisRespArray([]interface{}{10, 9, 60})
			host.CallOnRedisCall(0, resp)

			host.CompleteHttp()
		})

		// 测试基于 Consumer 的限流请求头处理
		t.Run("consumer limit request headers", func(t *testing.T) {
			host, status := test.NewTestHost(consumerLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含 consumer 信息
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"x-mse-consumer", "consumer1"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			resp := test.CreateRedisRespArray([]interface{}{10, 9, 1})
			host.CallOnRedisCall(0, resp)

			host.CompleteHttp()
		})

		// 测试基于 Cookie 的限流请求头处理
		t.Run("cookie limit request headers", func(t *testing.T) {
			host, status := test.NewTestHost(cookieLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含 cookie
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"cookie", "key1=value1; other=value"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			resp := test.CreateRedisRespArray([]interface{}{10, 9, 60})
			host.CallOnRedisCall(0, resp)

			host.CompleteHttp()
		})

		// 测试基于 IP 的限流请求头处理
		t.Run("ip limit request headers", func(t *testing.T) {
			host, status := test.NewTestHost(ipLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含 IP 信息
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
				{"x-forwarded-for", "1.1.1.1"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			resp := test.CreateRedisRespArray([]interface{}{10, 9, 86400})
			host.CallOnRedisCall(0, resp)

			host.CompleteHttp()
		})

		// 测试限流触发的情况
		t.Run("rate limit exceeded", func(t *testing.T) {
			host, status := test.NewTestHost(globalThresholdConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（触发限流）
			resp := test.CreateRedisRespArray([]interface{}{1000, -1, 60})
			host.CallOnRedisCall(0, resp)

			host.CompleteHttp()
		})
	})
}

func TestOnHttpResponseHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试显示限流配额的响应头处理
		t.Run("show limit quota headers", func(t *testing.T) {
			host, status := test.NewTestHost(globalThresholdConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
			})

			// 模拟 Redis 调用响应
			resp := test.CreateRedisRespArray([]interface{}{1000, 999, 60})
			host.CallOnRedisCall(0, resp)

			// 处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证是否添加了限流配额响应头
			responseHeaders := host.GetResponseHeaders()
			limitHeaderFound := false
			remainingHeaderFound := false

			for _, header := range responseHeaders {
				if strings.EqualFold(header[0], "x-ratelimit-limit") {
					limitHeaderFound = true
				}
				if strings.EqualFold(header[0], "x-ratelimit-remaining") {
					remainingHeaderFound = true
				}
			}

			require.True(t, limitHeaderFound, "X-RateLimit-Limit header should be added")
			require.True(t, remainingHeaderFound, "X-RateLimit-Remaining header should be added")

			host.CompleteHttp()
		})

		// 测试不显示限流配额的响应头处理
		t.Run("hide limit quota headers", func(t *testing.T) {
			// 创建不显示限流配额的配置
			hideQuotaConfig := func() json.RawMessage {
				data, _ := json.Marshal(map[string]interface{}{
					"rule_name": "routeA-global-limit-rule",
					"global_threshold": map[string]interface{}{
						"query_per_minute": 1000,
					},
					"redis": map[string]interface{}{
						"service_name": "redis.static",
						"service_port": 6379,
					},
					"show_limit_quota_header": false,
				})
				return data
			}()

			host, status := test.NewTestHost(hideQuotaConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
			})

			// 模拟 Redis 调用响应
			resp := test.CreateRedisRespArray([]interface{}{1000, 999, 60})
			host.CallOnRedisCall(0, resp)

			// 处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证是否没有添加限流配额响应头
			responseHeaders := host.GetResponseHeaders()
			limitHeaderFound := false
			remainingHeaderFound := false

			for _, header := range responseHeaders {
				if strings.EqualFold(header[0], "x-ratelimit-limit") {
					limitHeaderFound = true
				}
				if strings.EqualFold(header[0], "x-ratelimit-remaining") {
					remainingHeaderFound = true
				}
			}

			require.False(t, limitHeaderFound, "X-RateLimit-Limit header should not be added")
			require.False(t, remainingHeaderFound, "X-RateLimit-Remaining header should not be added")

			host.CompleteHttp()
		})
	})
}

func TestCompleteFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("complete rate limit flow", func(t *testing.T) {
			host, status := test.NewTestHost(globalThresholdConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 1. 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "GET"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 2. 模拟 Redis 调用响应
			resp := test.CreateRedisRespArray([]interface{}{1000, 999, 60})
			host.CallOnRedisCall(0, resp)

			// 3. 处理响应头
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证完整的限流流程
			responseHeaders := host.GetResponseHeaders()

			// 验证是否添加了必要的限流响应头
			limitHeaderFound := false
			remainingHeaderFound := false

			for _, header := range responseHeaders {
				if strings.EqualFold(header[0], "x-ratelimit-limit") {
					limitHeaderFound = true
				}
				if strings.EqualFold(header[0], "x-ratelimit-remaining") {
					remainingHeaderFound = true
				}
			}

			require.True(t, limitHeaderFound, "X-RateLimit-Limit header should be added")
			require.True(t, remainingHeaderFound, "X-RateLimit-Remaining header should be added")

			host.CompleteHttp()
		})
	})
}
