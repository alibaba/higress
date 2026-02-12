package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本 Fireworks 配置
var basicFireworksConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "fireworks",
			"apiTokens": []string{"fw-test123456789"},
			"modelMapping": map[string]string{
				"*": "accounts/fireworks/models/llama-v3p1-8b-instruct",
			},
		},
	})
	return data
}()

// 测试配置：Fireworks 多模型配置
var fireworksMultiModelConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "fireworks",
			"apiTokens": []string{"fw-multi-model"},
			"modelMapping": map[string]string{
				"gpt-4":         "accounts/fireworks/models/llama-v3p1-70b-instruct",
				"gpt-3.5-turbo": "accounts/fireworks/models/llama-v3p1-8b-instruct",
				"*":             "accounts/fireworks/models/llama-v3p1-8b-instruct",
			},
		},
	})
	return data
}()

// 测试配置：无效 Fireworks 配置（缺少 apiToken）
var invalidFireworksConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":         "fireworks",
			"apiTokens":    []string{},
			"modelMapping": map[string]string{},
		},
	})
	return data
}()

// 测试配置：完整 Fireworks 配置
var completeFireworksConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "fireworks",
			"apiTokens": []string{"fw-complete-test"},
			"modelMapping": map[string]string{
				"gpt-4":         "accounts/fireworks/models/llama-v3p1-70b-instruct",
				"gpt-3.5-turbo": "accounts/fireworks/models/llama-v3p1-8b-instruct",
				"*":             "accounts/fireworks/models/llama-v3p1-8b-instruct",
			},
		},
	})
	return data
}()

// RunFireworksParseConfigTests 测试 Fireworks 配置解析
func RunFireworksParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本 Fireworks 配置解析
		t.Run("basic fireworks config", func(t *testing.T) {
			host, status := test.NewTestHost(basicFireworksConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试 Fireworks 多模型配置解析
		t.Run("fireworks multi model config", func(t *testing.T) {
			host, status := test.NewTestHost(fireworksMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试无效 Fireworks 配置（缺少 apiToken）
		t.Run("invalid fireworks config - missing apiToken", func(t *testing.T) {
			host, status := test.NewTestHost(invalidFireworksConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试完整 Fireworks 配置解析
		t.Run("fireworks complete config", func(t *testing.T) {
			host, status := test.NewTestHost(completeFireworksConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

// RunFireworksOnHttpRequestHeadersTests 测试 Fireworks 请求头处理
func RunFireworksOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Fireworks 聊天完成请求头处理
		t.Run("fireworks chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicFireworksConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 应该返回 HeaderStopIteration，因为需要处理请求体
			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头是否被正确处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证 Host 是否被改为 Fireworks 域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Equal(t, "api.fireworks.ai", hostValue, "Host should be changed to Fireworks domain")

			// 验证 Authorization 是否被设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist")
			require.Contains(t, authValue, "Bearer fw-test123456789", "Authorization should contain Fireworks API token with Bearer prefix")

			// 验证 Path 保持 OpenAI 兼容格式
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Equal(t, "/v1/chat/completions", pathValue, "Path should remain OpenAI compatible")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasFireworksLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "fireworks") || strings.Contains(log, "ai-proxy") {
					hasFireworksLogs = true
					break
				}
			}
			require.True(t, hasFireworksLogs, "Should have Fireworks or ai-proxy processing logs")
		})

		// 测试 Fireworks 文本完成请求头处理
		t.Run("fireworks completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicFireworksConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证 Host 转换
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost)
			require.Equal(t, "api.fireworks.ai", hostValue)

			// 验证 Path 保持 OpenAI 兼容格式
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath)
			require.Equal(t, "/v1/completions", pathValue, "Path should remain OpenAI compatible for completions")

			// 验证 Authorization 设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist for completions")
			require.Contains(t, authValue, "Bearer fw-test123456789", "Authorization should contain Fireworks API token")
		})

		// 测试 Fireworks 模型列表请求头处理
		t.Run("fireworks models request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicFireworksConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/models"},
				{":method", "GET"},
			})

			// TODO: Due to the limitations of the test framework, we just treat it as a request with body here.
			//require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证 Host 转换
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost)
			require.Equal(t, "api.fireworks.ai", hostValue)

			// 验证 Path 保持 OpenAI 兼容格式
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath)
			require.Equal(t, "/v1/models", pathValue, "Path should remain OpenAI compatible for models")

			// 验证 Authorization 设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist for models")
			require.Contains(t, authValue, "Bearer fw-test123456789", "Authorization should contain Fireworks API token")
		})
	})
}

// RunFireworksOnHttpRequestBodyTests 测试 Fireworks 请求体处理
func RunFireworksOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Fireworks 聊天完成请求体处理
		t.Run("fireworks chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicFireworksConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 测试请求体
			requestBody := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "user", "content": "Hello, world!"}
				],
				"stream": false
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被正确处理
			actualRequestBody := host.GetRequestBody()
			require.NotNil(t, actualRequestBody)

			// 验证模型映射
			require.Contains(t, string(actualRequestBody), "accounts/fireworks/models/llama-v3p1-8b-instruct",
				"Model should be mapped to Fireworks model")
			require.Contains(t, string(actualRequestBody), "Hello, world!",
				"Request content should be preserved")
		})

		// 测试 Fireworks 流式聊天完成请求体处理
		t.Run("fireworks streaming chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(fireworksMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 测试流式请求体
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "Write a poem about AI"}
				],
				"stream": true
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被正确处理
			actualRequestBody := host.GetRequestBody()
			require.NotNil(t, actualRequestBody)

			// 验证模型映射（gpt-4 应该映射到 70b 模型）
			require.Contains(t, string(actualRequestBody), "accounts/fireworks/models/llama-v3p1-70b-instruct",
				"GPT-4 should be mapped to Fireworks 70b model")
			require.Contains(t, string(actualRequestBody), "Write a poem about AI",
				"Request content should be preserved")
			require.Contains(t, string(actualRequestBody), `"stream": true`,
				"Stream flag should be preserved")
		})

		// 测试 Fireworks 文本完成请求体处理
		t.Run("fireworks completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicFireworksConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 测试完成请求体
			requestBody := `{
				"model": "gpt-3.5-turbo",
				"prompt": "The future of AI is",
				"max_tokens": 100
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被正确处理
			actualRequestBody := host.GetRequestBody()
			require.NotNil(t, actualRequestBody)

			// 验证模型映射
			require.Contains(t, string(actualRequestBody), "accounts/fireworks/models/llama-v3p1-8b-instruct",
				"Model should be mapped to Fireworks model")
			require.Contains(t, string(actualRequestBody), "The future of AI is",
				"Prompt should be preserved")
		})
	})
}
