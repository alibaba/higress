package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本 Galadriel 配置
var basicGaladrielConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "galadriel",
			"apiTokens": []string{"gal-test123456789"},
			"modelMapping": map[string]string{
				"*": "llama3.1",
			},
		},
	})
	return data
}()

// 测试配置：Galadriel 多模型配置
var galadrielMultiModelConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "galadriel",
			"apiTokens": []string{"gal-multi-model"},
			"modelMapping": map[string]string{
				"gpt-3.5-turbo": "llama3.1",
				"gpt-4":         "llama3.1",
			},
		},
	})
	return data
}()

// 测试配置：无效 Galadriel 配置（缺少apiToken）
var invalidGaladrielConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "galadriel",
			// 缺少apiTokens
		},
	})
	return data
}()

// 测试配置：Galadriel 完整配置
var completeGaladrielConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "galadriel",
			"apiTokens": []string{"gal-complete-test"},
			"timeout":   30000,
			"modelMapping": map[string]string{
				"gpt-3.5-turbo": "llama3.1",
				"gpt-4":         "llama3.1",
				"*":             "llama3.1",
			},
		},
	})
	return data
}()

func RunGaladrielParseConfigTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试基本Galadriel配置解析
		t.Run("basic galadriel config", func(t *testing.T) {
			host, status := test.NewTestHost(basicGaladrielConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
		})

		// 测试Galadriel多模型配置解析
		t.Run("galadriel multi model config", func(t *testing.T) {
			host, status := test.NewTestHost(galadrielMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
		})

		// 测试无效Galadriel配置（缺少apiToken）
		t.Run("invalid galadriel config - missing apiToken", func(t *testing.T) {
			host, status := test.NewTestHost(invalidGaladrielConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试完整Galadriel配置解析
		t.Run("complete galadriel config", func(t *testing.T) {
			host, status := test.NewTestHost(completeGaladrielConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
		})
	})
}

func RunGaladrielOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Galadriel 请求头处理（聊天完成接口）
		t.Run("galadriel chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicGaladrielConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 应该返回HeaderStopIteration，因为需要处理请求体
			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头是否被正确处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证Host是否被改为Galadriel默认域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Equal(t, "api.galadriel.com", hostValue, "Host should be changed to Galadriel default domain")

			// 验证Authorization是否被设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist")
			require.Contains(t, authValue, "gal-test123456789", "Authorization should contain Galadriel API token")

			// 验证Path是否被正确处理
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/v1/chat/completions", "Path should contain chat completions endpoint")

			// 验证Content-Length是否被删除
			_, hasContentLength := test.GetHeaderValue(requestHeaders, "Content-Length")
			require.False(t, hasContentLength, "Content-Length header should be deleted")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasProviderLog := false
			for _, log := range debugLogs {
				if strings.Contains(log, "provider=galadriel") {
					hasProviderLog = true
					break
				}
			}
			require.True(t, hasProviderLog, "Should have debug log with provider=galadriel")
		})

		// 测试 Galadriel 模型接口请求头处理
		t.Run("galadriel models request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicGaladrielConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/models"},
				{":method", "GET"},
			})

			// GET请求没有请求体，应该直接继续
			require.Equal(t, types.ActionContinue, action)

			// 验证请求头是否被正确处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证Host是否被改为Galadriel默认域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Equal(t, "api.galadriel.com", hostValue, "Host should be changed to Galadriel default domain")

			// 验证Authorization是否被设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist")
			require.Contains(t, authValue, "gal-test123456789", "Authorization should contain Galadriel API token")
		})
	})
}

func RunGaladrielOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Galadriel 请求体处理（聊天完成接口）
		t.Run("galadriel chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGaladrielConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{
						"role": "user",
						"content": "Hello, who are you?"
					}
				],
				"temperature": 0.7,
				"max_tokens": 100
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确处理
			actualRequestBody := host.GetRequestBody()
			require.NotNil(t, actualRequestBody)

			// 验证模型映射是否正确应用
			bodyStr := string(actualRequestBody)
			require.Contains(t, bodyStr, `"model": "llama3.1"`, "Model should be mapped to llama3.1")

			// 验证其他字段是否保持不变
			require.Contains(t, bodyStr, `"temperature": 0.7`, "Temperature should be preserved")
			require.Contains(t, bodyStr, `"max_tokens": 100`, "Max tokens should be preserved")
			require.Contains(t, bodyStr, `"Hello, who are you?"`, "Content should be preserved")
		})

		// 测试 Galadriel 多模型映射
		t.Run("galadriel multi model mapping", func(t *testing.T) {
			host, status := test.NewTestHost(galadrielMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 测试gpt-3.5-turbo模型映射
			requestBody := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{
						"role": "user",
						"content": "Test message"
					}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证模型映射
			actualRequestBody := host.GetRequestBody()
			bodyStr := string(actualRequestBody)
			require.Contains(t, bodyStr, `"model": "llama3.1"`, "gpt-3.5-turbo should be mapped to llama3.1")
		})
	})
}

func RunGaladrielOnHttpResponseHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Galadriel 响应头处理（聊天完成接口）
		t.Run("galadriel chat completion response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicGaladrielConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求上下文
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"x-ratelimit-limit", "100"},
			})

			require.Equal(t, types.ActionContinue, action)

			// 验证响应头是否被正确处理
			responseHeaders := host.GetResponseHeaders()
			require.NotNil(t, responseHeaders)

			// 验证基本响应头字段
			statusValue, hasStatus := test.GetHeaderValue(responseHeaders, ":status")
			require.True(t, hasStatus, "Status header should exist")
			require.Equal(t, "200", statusValue, "Status should be 200")

			contentTypeValue, hasContentType := test.GetHeaderValue(responseHeaders, "content-type")
			require.True(t, hasContentType, "Content-Type header should exist")
			require.Equal(t, "application/json", contentTypeValue, "Content-Type should be application/json")
		})
	})
}

func RunGaladrielOnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Galadriel 响应体处理（聊天完成接口）
		t.Run("galadriel chat completion response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGaladrielConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求上下文
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置响应体
			responseBody := `{
				"id": "id",
				"object": "chat.completion",
				"created": 1728558433,
				"model": "neuralmagic/Meta-Llama-3.1-8B-Instruct-FP8",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Hello! I am an AI assistant."
						},
						"finish_reason": "stop"
					}
				]
			}`

			action := host.CallOnHttpResponseBody([]byte(responseBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证响应体处理
			actualResponseBody := host.GetResponseBody()
			require.NotNil(t, actualResponseBody)

			// 验证响应体内容
			bodyStr := string(actualResponseBody)
			require.Contains(t, bodyStr, `"object": "chat.completion"`, "Response should contain chat.completion object")
			require.Contains(t, bodyStr, `"Hello! I am an AI assistant."`, "Response should contain the assistant message")
		})
	})
}

func RunGaladrielOnStreamingResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("galadriel streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGaladrielConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求上下文
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置流式响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/event-stream"},
			})

			// 模拟流式响应
			streamChunks := []string{
				`data: {"id":"cmpl-test","object":"chat.completion.chunk","created":1699123456,"model":"llama3.1","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
				`data: {"id":"cmpl-test","object":"chat.completion.chunk","created":1699123456,"model":"llama3.1","choices":[{"index":0,"delta":{"content":" there!"},"finish_reason":null}]}`,
				`data: {"id":"cmpl-test","object":"chat.completion.chunk","created":1699123456,"model":"llama3.1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
				`data: [DONE]`,
			}

			for _, chunk := range streamChunks {
				chunk = chunk + "\n\n"
				action := host.CallOnHttpResponseBody([]byte(chunk))
				require.Equal(t, types.ActionContinue, action)
			}

			// 验证流式响应处理 - 检查是否包含流式数据或DONE标记
			actualResponseBody := host.GetResponseBody()
			responseStr := string(actualResponseBody)
			// 应该包含流式数据或结束标记
			require.True(t, strings.Contains(responseStr, "chat.completion.chunk") || strings.Contains(responseStr, "[DONE]"))
		})
	})
}
