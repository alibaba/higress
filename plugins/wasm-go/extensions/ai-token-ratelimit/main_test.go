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
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：全局限流配置
var globalThresholdConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "ai-token-global-limit",
		"global_threshold": map[string]interface{}{
			"token_per_minute": 1000,
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
			"timeout":      1000,
		},
		"rejected_code": 429,
		"rejected_msg":  "Too many AI token requests",
	})
	return data
}()

// 测试配置：基于请求头的限流配置
var headerLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "ai-token-header-limit",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_header": "x-api-key",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "test-key-123",
						"token_per_minute": 100,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"rejected_code": 429,
		"rejected_msg":  "API key rate limit exceeded",
	})
	return data
}()

// 测试配置：基于请求参数的限流配置
var paramLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "ai-token-param-limit",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_param": "apikey",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "param-key-456",
						"token_per_minute": 50,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"rejected_code": 429,
		"rejected_msg":  "Parameter rate limit exceeded",
	})
	return data
}()

// 测试配置：基于 Consumer 的限流配置
var consumerLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "ai-token-consumer-limit",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_consumer": "",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "consumer1",
						"token_per_minute": 200,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"rejected_code": 429,
		"rejected_msg":  "Consumer rate limit exceeded",
	})
	return data
}()

// 测试配置：基于 Cookie 的限流配置
var cookieLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "ai-token-cookie-limit",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_cookie": "session-id",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "session-789",
						"token_per_minute": 75,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"rejected_code": 429,
		"rejected_msg":  "Session rate limit exceeded",
	})
	return data
}()

// 测试配置：基于 IP 的限流配置
var ipLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "ai-token-ip-limit",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_per_ip": "from-remote-addr",
				"limit_keys": []map[string]interface{}{
					{
						"key":              "192.168.1.0/24",
						"token_per_minute": 300,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"rejected_code": 429,
		"rejected_msg":  "IP rate limit exceeded",
	})
	return data
}()

// 测试配置：正则表达式限流配置
var regexpLimitConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rule_name": "ai-token-regexp-limit",
		"rule_items": []map[string]interface{}{
			{
				"limit_by_per_header": "x-user-id",
				"limit_keys": []map[string]interface{}{
					{
						"key":                "regexp:^user-\\d+$",
						"token_per_minute":   150,
					},
				},
			},
		},
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
		},
		"rejected_code": 429,
		"rejected_msg":  "User ID rate limit exceeded",
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

		// 测试基于请求头的限流配置解析
		t.Run("header limit config", func(t *testing.T) {
			host, status := test.NewTestHost(headerLimitConfig)
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
				{":method", "POST"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			// 返回 [count, remaining, ttl] 格式
			resp := test.CreateRedisRespArray([]interface{}{1000, 999, 60})
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
				{":method", "POST"},
				{"x-api-key", "test-key-123"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			resp := test.CreateRedisRespArray([]interface{}{100, 99, 60})
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
				{":path", "/api/test?apikey=param-key-456"},
				{":method", "POST"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			resp := test.CreateRedisRespArray([]interface{}{50, 49, 60})
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
				{":method", "POST"},
				{"x-mse-consumer", "consumer1"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			resp := test.CreateRedisRespArray([]interface{}{200, 199, 60})
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
				{":method", "POST"},
				{"cookie", "session-id=session-789; other=value"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（允许请求）
			resp := test.CreateRedisRespArray([]interface{}{75, 74, 60})
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
				{":method", "POST"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 模拟 Redis 调用响应（触发限流）
			// 返回 [count, remaining, ttl] 格式，remaining < 0 表示触发限流
			resp := test.CreateRedisRespArray([]interface{}{1000, -1, 60})
			host.CallOnRedisCall(0, resp)

			// 检查是否发送了限流响应
			localResponse := host.GetLocalResponse()
			require.NotNil(t, localResponse)
			require.Equal(t, uint32(429), localResponse.StatusCode)
			require.Contains(t, string(localResponse.Data), "Too many AI token requests")

			host.CompleteHttp()
		})

		// 测试没有匹配到限流规则的情况
		t.Run("no matching limit rule", func(t *testing.T) {
			host, status := test.NewTestHost(headerLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，但不包含限流键
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "POST"},
				// 不包含 x-api-key 头
			})

			// 应该返回 ActionContinue，因为没有匹配到限流规则
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestOnHttpStreamingBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试流式响应体处理（包含 token 统计）
		t.Run("streaming body with token usage", func(t *testing.T) {
			host, status := test.NewTestHost(globalThresholdConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "POST"},
			})

			// 模拟 Redis 调用响应
			resp := test.CreateRedisRespArray([]interface{}{1000, 999, 60})
			host.CallOnRedisCall(0, resp)

			// 处理流式响应体
			// 模拟包含 token 统计信息的响应体
			responseBody := []byte(`{"choices":[{"message":{"content":"Hello, how can I help you?"}}],"usage":{"prompt_tokens":10,"completion_tokens":15,"total_tokens":25}}`)
			result := host.CallOnHttpStreamingRequestBody(responseBody, false) // 不是最后一个块

			// 应该返回原始数据
			require.Equal(t, responseBody, result)

			// 处理最后一个块
			result = host.CallOnHttpStreamingRequestBody([]byte(""), true) // 最后一个块

			// 应该返回空数据
			require.Equal(t, []byte(""), result)

			host.CompleteHttp()
		})

		// 测试流式响应体处理（不包含 token 统计）
		t.Run("streaming body without token usage", func(t *testing.T) {
			host, status := test.NewTestHost(globalThresholdConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "POST"},
			})

			// 模拟 Redis 调用响应
			resp := test.CreateRedisRespArray([]interface{}{1000, 999, 60})
			host.CallOnRedisCall(0, resp)

			// 处理流式响应体
			// 模拟不包含 token 统计信息的响应体
			responseBody := []byte(`{"message": "Hello, world!"}`)
			result := host.CallOnHttpStreamingRequestBody(responseBody, true) // 最后一个块

			// 应该返回原始数据
			require.Equal(t, responseBody, result)

			host.CompleteHttp()
		})
	})
}

func TestCompleteFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试完整的限流流程
		t.Run("complete rate limit flow", func(t *testing.T) {
			host, status := test.NewTestHost(headerLimitConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 1. 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/test"},
				{":method", "POST"},
				{"x-api-key", "test-key-123"},
			})

			// 由于需要调用 Redis，应该返回 HeaderStopAllIterationAndWatermark
			require.Equal(t, types.HeaderStopAllIterationAndWatermark, action)

			// 2. 模拟 Redis 调用响应
			resp := test.CreateRedisRespArray([]interface{}{100, 99, 60})
			host.CallOnRedisCall(0, resp)

			// 3. 处理流式响应体
			responseBody := []byte(`{"choices":[{"message":{"content":"AI response"}}],"usage":{"prompt_tokens":5,"completion_tokens":8,"total_tokens":13}}`)
			result := host.CallOnHttpStreamingRequestBody(responseBody, true)

			// 应该返回原始数据
			require.Equal(t, responseBody, result)

			// 4. 完成请求
			host.CompleteHttp()
		})
	})
}
