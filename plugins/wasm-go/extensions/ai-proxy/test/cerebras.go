package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本 Cerebras 配置
var basicCerebrasConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "cerebras",
			"apiTokens": []string{"csk-cerebras-test123456789"},
			"modelMapping": map[string]string{
				"*": "llama3.1-8b",
			},
		},
	})
	return data
}()

// 测试配置：Cerebras 多模型配置
var cerebrasMultiModelConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "cerebras",
			"apiTokens": []string{"csk-cerebras-multi-model"},
			"modelMapping": map[string]string{
				"gpt-4":         "llama3.1-70b",
				"gpt-3.5-turbo": "llama3.1-8b",
				"*":             "llama3.1-8b",
			},
		},
	})
	return data
}()

// RunCerebrasParseConfigTests 测试 Cerebras 配置解析
func RunCerebrasParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本 Cerebras 配置解析
		t.Run("basic cerebras config", func(t *testing.T) {
			host, status := test.NewTestHost(basicCerebrasConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
		})

		// 测试多模型 Cerebras 配置解析
		t.Run("cerebras multi-model config", func(t *testing.T) {
			host, status := test.NewTestHost(cerebrasMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
		})
	})
}

// RunCerebrasOnHttpRequestHeadersTests 测试 Cerebras 请求头处理
func RunCerebrasOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Cerebras 请求头处理（聊天完成接口）
		t.Run("cerebras chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicCerebrasConfig)
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
		})
	})
}

// RunCerebrasOnHttpRequestBodyTests 测试 Cerebras 请求体处理
func RunCerebrasOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Cerebras 请求体处理（聊天完成接口）
		t.Run("cerebras chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicCerebrasConfig)
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
				"model": "gpt-3.5-turbo",
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

			// 验证请求体是否包含映射后的模型名
			actualRequestBody := host.GetRequestBody()
			require.Contains(t, string(actualRequestBody), "llama3.1-8b")
		})
	})
}

// RunCerebrasOnHttpResponseHeadersTests 测试 Cerebras 响应头处理
func RunCerebrasOnHttpResponseHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Cerebras 响应头处理（聊天完成接口）
		t.Run("cerebras chat completion response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicCerebrasConfig)
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
		})
	})
}

// RunCerebrasOnHttpResponseBodyTests 测试 Cerebras 响应体处理
func RunCerebrasOnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Cerebras 响应体处理（聊天完成接口）
		t.Run("cerebras chat completion response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicCerebrasConfig)
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
				"id": "cmpl-test123",
				"object": "chat.completion",
				"created": 1699123456,
				"model": "llama3.1-8b",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Hello! I am an AI assistant powered by Cerebras."
						},
						"finish_reason": "stop"
					}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 12,
					"total_tokens": 22
				}
			}`

			action := host.CallOnHttpResponseBody([]byte(responseBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证响应体格式
			actualResponseBody := host.GetResponseBody()
			require.Contains(t, string(actualResponseBody), "chat.completion")
			require.Contains(t, string(actualResponseBody), "assistant")
		})
	})
}

// RunCerebrasOnStreamingResponseBodyTests 测试 Cerebras 流式响应体处理
func RunCerebrasOnStreamingResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("cerebras streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicCerebrasConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求上下文
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 模拟流式响应
			streamChunks := []string{
				`data: {"id":"cmpl-test","object":"chat.completion.chunk","created":1699123456,"model":"llama3.1-8b","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
				`data: {"id":"cmpl-test","object":"chat.completion.chunk","created":1699123456,"model":"llama3.1-8b","choices":[{"index":0,"delta":{"content":" there!"},"finish_reason":null}]}`,
				`data: {"id":"cmpl-test","object":"chat.completion.chunk","created":1699123456,"model":"llama3.1-8b","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
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
