package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本ai360配置
var basicAi360Config = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "ai360",
			"apiTokens": []string{"sk-ai360-test123456789"},
			"modelMapping": map[string]string{
				"*": "360GPT_S2_V9",
			},
		},
	})
	return data
}()

// 测试配置：ai360多模型配置
var ai360MultiModelConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "ai360",
			"apiTokens": []string{"sk-ai360-multi-model"},
			"modelMapping": map[string]string{
				"gpt-3.5-turbo":          "360GPT_S2_V9",
				"gpt-4":                  "360GPT_S2_V9",
				"text-embedding-ada-002": "360Embedding_Text_V1",
			},
		},
	})
	return data
}()

// 测试配置：无效ai360配置（缺少apiToken）
var invalidAi360Config = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "ai360",
			// 缺少apiTokens
		},
	})
	return data
}()

// 测试配置：ai360自定义域名配置
var ai360CustomDomainConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "ai360",
			"apiTokens": []string{"sk-ai360-custom-domain"},
			"modelMapping": map[string]string{
				"*": "360GPT_S2_V9",
			},
			"openaiCustomUrl": "https://custom.ai360.cn/v1",
		},
	})
	return data
}()

// 测试配置：ai360完整配置（包含failover等字段）
var completeAi360Config = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "ai360",
			"apiTokens": []string{"sk-ai360-complete"},
			"modelMapping": map[string]string{
				"*": "360GPT_S2_V9",
			},
			"failover": map[string]interface{}{
				"enabled": false,
			},
			"retryOnFailure": map[string]interface{}{
				"enabled": false,
			},
		},
	})
	return data
}()

func RunAi360ParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本ai360配置解析
		t.Run("basic ai360 config", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试ai360多模型配置解析
		t.Run("ai360 multi model config", func(t *testing.T) {
			host, status := test.NewTestHost(ai360MultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试无效ai360配置（缺少apiToken）
		t.Run("invalid ai360 config - missing api token", func(t *testing.T) {
			host, status := test.NewTestHost(invalidAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试ai360自定义域名配置解析
		t.Run("ai360 custom domain config", func(t *testing.T) {
			host, status := test.NewTestHost(ai360CustomDomainConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试ai360完整配置解析
		t.Run("ai360 complete config", func(t *testing.T) {
			host, status := test.NewTestHost(completeAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func RunAi360OnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试ai360请求头处理（聊天完成接口）
		t.Run("ai360 chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
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

			// 验证Host是否被改为ai360域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Equal(t, "api.360.cn", hostValue, "Host should be changed to ai360 domain")

			// 验证Authorization是否被设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist")
			require.Contains(t, authValue, "sk-ai360-test123456789", "Authorization should contain ai360 API token")

			// 验证Path是否被正确处理
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			// ai360应该支持聊天完成接口，路径可能被转换
			require.Contains(t, pathValue, "/v1/chat/completions", "Path should contain chat completions endpoint")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasAi360Logs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "ai360") {
					hasAi360Logs = true
					break
				}
			}
			require.True(t, hasAi360Logs, "Should have ai360 processing logs")
		})

		// 测试ai360请求头处理（嵌入接口）
		t.Run("ai360 embeddings request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/embeddings"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.HeaderStopIteration, action)

			// 验证嵌入接口的请求头处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证Host转换
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost)
			require.Equal(t, "api.360.cn", hostValue)

			// 验证Path转换
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath)
			require.Contains(t, pathValue, "/v1/embeddings", "Path should contain embeddings endpoint")

			// 验证Authorization设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist for embeddings")
			require.Contains(t, authValue, "sk-ai360-test123456789", "Authorization should contain ai360 API token")
		})

		// 测试ai360请求头处理（不支持的接口）
		t.Run("ai360 unsupported api request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/images/generations"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.HeaderStopIteration, action)

			// 验证不支持的接口处理
			// 即使是不支持的接口，基本的请求头转换仍然应该执行
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// Host仍然应该被转换
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost)
			require.Equal(t, "api.360.cn", hostValue)

		})
	})
}

func RunAi360OnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试ai360请求体处理（聊天完成接口）
		t.Run("ai360 chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证模型名称是否被正确映射
			// ai360 provider会将模型名称从gpt-3.5-turbo映射为360GPT_S2_V9
			require.Contains(t, string(processedBody), "360GPT_S2_V9", "Model name should be mapped to ai360 format")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			infoLogs := host.GetInfoLogs()

			// 验证是否有ai360相关的处理日志
			hasAi360Logs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "ai360") {
					hasAi360Logs = true
					break
				}
			}
			for _, log := range infoLogs {
				if strings.Contains(log, "ai360") {
					hasAi360Logs = true
					break
				}
			}
			require.True(t, hasAi360Logs, "Should have ai360 processing logs")
		})

		// 测试ai360请求体处理（嵌入接口）
		t.Run("ai360 embeddings request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/embeddings"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{"model":"text-embedding-ada-002","input":"test text"}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证嵌入接口的请求体处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证模型名称映射
			// ai360 provider会将模型名称从text-embedding-ada-002映射为360GPT_S2_V9
			require.Contains(t, string(processedBody), "360GPT_S2_V9", "Model name should be mapped to ai360 format")

			// 检查处理日志
			debugLogs := host.GetDebugLogs()
			hasEmbeddingLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "embeddings") || strings.Contains(log, "ai360") {
					hasEmbeddingLogs = true
					break
				}
			}
			require.True(t, hasEmbeddingLogs, "Should have embedding processing logs")
		})

		// 测试ai360请求体处理（不支持的接口）
		t.Run("ai360 unsupported api request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/images/generations"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{"model":"dall-e-3","prompt":"test image"}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证不支持的接口处理

			// 验证请求体没有被意外修改
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)
			require.Contains(t, string(processedBody), "dall-e-3", "Request body should not be modified for unsupported APIs")
		})
	})
}

func RunAi360OnHttpResponseHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试ai360响应头处理（聊天完成接口）
		t.Run("ai360 chat completion response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}]}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
				{"X-Request-Id", "req-123"},
			}
			action := host.CallOnHttpResponseHeaders(responseHeaders)

			require.Equal(t, types.ActionContinue, action)

			// 验证响应头是否被正确处理
			processedResponseHeaders := host.GetResponseHeaders()
			require.NotNil(t, processedResponseHeaders)

			// 验证状态码
			statusValue, hasStatus := test.GetHeaderValue(processedResponseHeaders, ":status")
			require.True(t, hasStatus, "Status header should exist")
			require.Equal(t, "200", statusValue, "Status should be 200")

			// 验证Content-Type
			contentTypeValue, hasContentType := test.GetHeaderValue(processedResponseHeaders, "Content-Type")
			require.True(t, hasContentType, "Content-Type header should exist")
			require.Equal(t, "application/json", contentTypeValue, "Content-Type should be application/json")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasResponseLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "response") || strings.Contains(log, "ai360") {
					hasResponseLogs = true
					break
				}
			}
			require.True(t, hasResponseLogs, "Should have response processing logs")
		})

		// 测试ai360响应头处理（嵌入接口）
		t.Run("ai360 embeddings response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/embeddings"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{"model":"text-embedding-ada-002","input":"test text"}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
				{"X-Embedding-Model", "360Embedding_Text_V1"},
			}
			action := host.CallOnHttpResponseHeaders(responseHeaders)

			require.Equal(t, types.ActionContinue, action)

			// 验证响应头处理
			processedResponseHeaders := host.GetResponseHeaders()
			require.NotNil(t, processedResponseHeaders)

			// 验证嵌入模型信息
			modelValue, hasModel := test.GetHeaderValue(processedResponseHeaders, "X-Embedding-Model")
			require.True(t, hasModel, "Embedding model header should exist")
			require.Equal(t, "360Embedding_Text_V1", modelValue, "Embedding model should match configuration")
		})

		// 测试ai360响应头处理（错误响应）
		t.Run("ai360 error response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}]}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置错误响应头
			errorResponseHeaders := [][2]string{
				{":status", "429"},
				{"Content-Type", "application/json"},
				{"Retry-After", "60"},
			}
			action := host.CallOnHttpResponseHeaders(errorResponseHeaders)

			require.Equal(t, types.ActionContinue, action)

			// 验证错误响应头处理
			processedResponseHeaders := host.GetResponseHeaders()
			require.NotNil(t, processedResponseHeaders)

			// 验证错误状态码
			statusValue, hasStatus := test.GetHeaderValue(processedResponseHeaders, ":status")
			require.True(t, hasStatus, "Status header should exist")
			require.Equal(t, "429", statusValue, "Status should be 429 (Too Many Requests)")

			// 验证重试信息
			retryValue, hasRetry := test.GetHeaderValue(processedResponseHeaders, "Retry-After")
			require.True(t, hasRetry, "Retry-After header should exist")
			require.Equal(t, "60", retryValue, "Retry-After should be 60 seconds")
		})
	})
}

func RunAi360OnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试ai360响应体处理（聊天完成接口）
		t.Run("ai360 chat completion response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}]}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体
			responseBody := `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-3.5-turbo",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! How can I help you today?"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 9,
					"completion_tokens": 12,
					"total_tokens": 21
				}
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体是否被正确处理
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			// 验证响应体内容
			responseStr := string(processedResponseBody)
			require.Contains(t, responseStr, "chat.completion", "Response should contain chat completion object")
			require.Contains(t, responseStr, "assistant", "Response should contain assistant role")
			require.Contains(t, responseStr, "usage", "Response should contain usage information")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasResponseBodyLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "response") || strings.Contains(log, "body") || strings.Contains(log, "ai360") {
					hasResponseBodyLogs = true
					break
				}
			}
			require.True(t, hasResponseBodyLogs, "Should have response body processing logs")
		})

		// 测试ai360响应体处理（嵌入接口）
		t.Run("ai360 embeddings response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/embeddings"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{"model":"text-embedding-ada-002","input":"test text"}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体
			responseBody := `{
				"object": "list",
				"data": [{
					"object": "embedding",
					"embedding": [0.1, 0.2, 0.3, 0.4, 0.5],
					"index": 0
				}],
				"model": "text-embedding-ada-002",
				"usage": {
					"prompt_tokens": 5,
					"total_tokens": 5
				}
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体处理
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			// 验证嵌入响应内容
			responseStr := string(processedResponseBody)
			require.Contains(t, responseStr, "embedding", "Response should contain embedding object")
			require.Contains(t, responseStr, "0.1", "Response should contain embedding vector")
			require.Contains(t, responseStr, "text-embedding-ada-002", "Response should contain model name")
		})

	})
}

func RunAi360OnStreamingResponseBodyTests(t *testing.T) {
	// 测试ai360响应体处理（流式响应）
	test.RunTest(t, func(t *testing.T) {
		t.Run("ai360 streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicAi360Config)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置流式请求体
			requestBody := `{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}],"stream":true}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置流式响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "text/event-stream"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟流式响应体
			chunk1 := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"delta":{"role":"assistant"},"index":0}]}

`
			chunk2 := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"},"index":0}]}

`
			chunk3 := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"delta":{"content":"!"},"index":0}]}

`
			chunk4 := `data: [DONE]

`

			// 处理流式响应体
			action1 := host.CallOnHttpStreamingResponseBody([]byte(chunk1), false)
			require.Equal(t, types.ActionContinue, action1)

			action2 := host.CallOnHttpStreamingResponseBody([]byte(chunk2), false)
			require.Equal(t, types.ActionContinue, action2)

			action3 := host.CallOnHttpStreamingResponseBody([]byte(chunk3), false)
			require.Equal(t, types.ActionContinue, action3)

			action4 := host.CallOnHttpStreamingResponseBody([]byte(chunk4), true)
			require.Equal(t, types.ActionContinue, action4)

			// 验证流式响应处理
			// 注意：流式响应可能不会在GetResponseBody中累积，需要检查日志或其他方式验证
			debugLogs := host.GetDebugLogs()
			hasStreamingLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "streaming") || strings.Contains(log, "chunk") || strings.Contains(log, "ai360") {
					hasStreamingLogs = true
					break
				}
			}
			require.True(t, hasStreamingLogs, "Should have streaming response processing logs")
		})
	})
}
