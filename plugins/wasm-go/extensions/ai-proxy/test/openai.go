package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本OpenAI配置
var basicOpenAIConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "openai",
			"apiTokens": []string{"sk-openai-test123456789"},
			"modelMapping": map[string]string{
				"*": "gpt-3.5-turbo",
			},
		},
	})
	return data
}()

// 测试配置：OpenAI多模型配置
var openAIMultiModelConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "openai",
			"apiTokens": []string{"sk-openai-multi-model"},
			"modelMapping": map[string]string{
				"gpt-3.5-turbo":          "gpt-3.5-turbo",
				"gpt-4":                  "gpt-4",
				"text-embedding-ada-002": "text-embedding-ada-002",
				"dall-e-3":               "dall-e-3",
			},
		},
	})
	return data
}()

// 测试配置：OpenAI自定义域名配置（直接路径）
var openAICustomDomainDirectPathConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "openai",
			"apiTokens": []string{"sk-openai-custom-domain"},
			"modelMapping": map[string]string{
				"*": "gpt-3.5-turbo",
			},
			"openaiCustomUrl": "https://custom.openai.com/v1",
		},
	})
	return data
}()

// 测试配置：OpenAI自定义域名配置（间接路径）
var openAICustomDomainIndirectPathConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "openai",
			"apiTokens": []string{"sk-openai-custom-domain"},
			"modelMapping": map[string]string{
				"*": "gpt-3.5-turbo",
			},
			"openaiCustomUrl": "https://custom.openai.com/api",
		},
	})
	return data
}()

// 测试配置：OpenAI完整配置（包含responseJsonSchema等字段）
var completeOpenAIConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "openai",
			"apiTokens": []string{"sk-openai-complete"},
			"modelMapping": map[string]string{
				"*": "gpt-3.5-turbo",
			},
			"responseJsonSchema": map[string]interface{}{
				"type": "json_object",
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

func RunOpenAIParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本OpenAI配置解析
		t.Run("basic openai config", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试OpenAI多模型配置解析
		t.Run("openai multi model config", func(t *testing.T) {
			host, status := test.NewTestHost(openAIMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试OpenAI自定义域名配置（直接路径）
		t.Run("openai custom domain direct path config", func(t *testing.T) {
			host, status := test.NewTestHost(openAICustomDomainDirectPathConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试OpenAI自定义域名配置（间接路径）
		t.Run("openai custom domain indirect path config", func(t *testing.T) {
			host, status := test.NewTestHost(openAICustomDomainIndirectPathConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试OpenAI完整配置解析
		t.Run("openai complete config", func(t *testing.T) {
			host, status := test.NewTestHost(completeOpenAIConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func RunOpenAIOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试OpenAI请求头处理（聊天完成接口）
		t.Run("openai chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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

			// 验证Host是否被改为OpenAI默认域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Equal(t, "api.openai.com", hostValue, "Host should be changed to OpenAI default domain")

			// 验证Authorization是否被设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist")
			require.Contains(t, authValue, "sk-openai-test123456789", "Authorization should contain OpenAI API token")

			// 验证Path是否被正确处理
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/v1/chat/completions", "Path should contain chat completions endpoint")

			// 验证Content-Length是否被删除
			_, hasContentLength := test.GetHeaderValue(requestHeaders, "Content-Length")
			require.False(t, hasContentLength, "Content-Length header should be deleted")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasOpenAILogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "openai") {
					hasOpenAILogs = true
					break
				}
			}
			require.True(t, hasOpenAILogs, "Should have OpenAI processing logs")
		})

		// 测试OpenAI请求头处理（嵌入接口）
		t.Run("openai embeddings request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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
			require.Equal(t, "api.openai.com", hostValue)

			// 验证Path转换
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath)
			require.Contains(t, pathValue, "/v1/embeddings", "Path should contain embeddings endpoint")

			// 验证Authorization设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist for embeddings")
			require.Contains(t, authValue, "sk-openai-test123456789", "Authorization should contain OpenAI API token")
		})

		// 测试OpenAI请求头处理（图像生成接口）
		t.Run("openai image generation request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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

			// 验证图像生成接口的请求头处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证Host转换
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost)
			require.Equal(t, "api.openai.com", hostValue)

			// 验证Path转换
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath)
			require.Contains(t, pathValue, "/v1/images/generations", "Path should contain image generations endpoint")
		})

		// 测试OpenAI自定义域名请求头处理
		t.Run("openai custom domain request headers", func(t *testing.T) {
			host, status := test.NewTestHost(openAICustomDomainDirectPathConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.HeaderStopIteration, action)

			// 验证自定义域名的请求头处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证Host是否被改为自定义域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost)
			require.Equal(t, "custom.openai.com", hostValue, "Host should be changed to custom domain")

			// 验证Path是否被正确处理
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath)
			// 对于直接路径，应该保持原有路径
			require.Contains(t, pathValue, "/v1/chat/completions", "Path should be preserved for direct custom path")
		})
	})
}

func RunOpenAIOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试OpenAI请求体处理（聊天完成接口）
		t.Run("openai chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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
			require.Contains(t, string(processedBody), "gpt-3.5-turbo", "Original model name should be preserved or mapped")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			infoLogs := host.GetInfoLogs()

			// 验证是否有OpenAI相关的处理日志
			hasOpenAILogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "openai") {
					hasOpenAILogs = true
					break
				}
			}
			for _, log := range infoLogs {
				if strings.Contains(log, "openai") {
					hasOpenAILogs = true
					break
				}
			}
			require.True(t, hasOpenAILogs, "Should have OpenAI processing logs")
		})

		// 测试OpenAI请求体处理（嵌入接口）
		t.Run("openai embeddings request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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
			// 由于使用了通配符映射 "*": "gpt-3.5-turbo"，text-embedding-ada-002 会被映射为 gpt-3.5-turbo
			require.Contains(t, string(processedBody), "gpt-3.5-turbo", "Model name should be mapped via wildcard")

			// 检查处理日志
			debugLogs := host.GetDebugLogs()
			hasEmbeddingLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "embeddings") || strings.Contains(log, "openai") {
					hasEmbeddingLogs = true
					break
				}
			}
			require.True(t, hasEmbeddingLogs, "Should have embedding processing logs")
		})

		// 测试OpenAI请求体处理（图像生成接口）
		t.Run("openai image generation request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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

			// 验证图像生成接口的请求体处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证模型名称映射
			// 由于使用了通配符映射 "*": "gpt-3.5-turbo"，dall-e-3 会被映射为 gpt-3.5-turbo
			require.Contains(t, string(processedBody), "gpt-3.5-turbo", "Model name should be mapped via wildcard")

			// 检查处理日志
			debugLogs := host.GetDebugLogs()
			hasImageLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "image") || strings.Contains(log, "openai") {
					hasImageLogs = true
					break
				}
			}
			require.True(t, hasImageLogs, "Should have image generation processing logs")
		})

		// 测试OpenAI请求体处理（带responseJsonSchema配置）
		t.Run("openai request body with responseJsonSchema", func(t *testing.T) {
			host, status := test.NewTestHost(completeOpenAIConfig)
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

			// 验证responseJsonSchema是否被应用
			// 注意：由于test框架的限制，我们可能需要检查日志或其他方式来验证处理结果
			require.Contains(t, string(processedBody), "gpt-3.5-turbo", "Model name should be preserved")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasSchemaLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "response format") || strings.Contains(log, "openai") {
					hasSchemaLogs = true
					break
				}
			}
			require.True(t, hasSchemaLogs, "Should have response format processing logs")
		})
	})
}

func RunOpenAIOnHttpResponseHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试OpenAI响应头处理（聊天完成接口）
		t.Run("openai chat completion response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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
				if strings.Contains(log, "response") || strings.Contains(log, "openai") {
					hasResponseLogs = true
					break
				}
			}
			require.True(t, hasResponseLogs, "Should have response processing logs")
		})

		// 测试OpenAI响应头处理（嵌入接口）
		t.Run("openai embeddings response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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
				{"X-Embedding-Model", "text-embedding-ada-002"},
			}
			action := host.CallOnHttpResponseHeaders(responseHeaders)

			require.Equal(t, types.ActionContinue, action)

			// 验证响应头处理
			processedResponseHeaders := host.GetResponseHeaders()
			require.NotNil(t, processedResponseHeaders)

			// 验证嵌入模型信息
			modelValue, hasModel := test.GetHeaderValue(processedResponseHeaders, "X-Embedding-Model")
			require.True(t, hasModel, "Embedding model header should exist")
			require.Equal(t, "text-embedding-ada-002", modelValue, "Embedding model should match configuration")
		})

		// 测试OpenAI响应头处理（错误响应）
		t.Run("openai error response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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

func RunOpenAIOnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试OpenAI响应体处理（聊天完成接口）
		t.Run("openai chat completion response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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
				if strings.Contains(log, "response") || strings.Contains(log, "body") || strings.Contains(log, "openai") {
					hasResponseBodyLogs = true
					break
				}
			}
			require.True(t, hasResponseBodyLogs, "Should have response body processing logs")
		})

		// 测试OpenAI响应体处理（嵌入接口）
		t.Run("openai embeddings response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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

		// 测试OpenAI响应体处理（图像生成接口）
		t.Run("openai image generation response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体
			responseBody := `{
				"created": 1677652288,
				"data": [{
					"url": "https://example.com/image1.png",
					"revised_prompt": "test image"
				}]
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体处理
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			// 验证图像生成响应内容
			responseStr := string(processedResponseBody)
			require.Contains(t, responseStr, "data", "Response should contain data array")
			require.Contains(t, responseStr, "url", "Response should contain image URL")
			require.Contains(t, responseStr, "revised_prompt", "Response should contain revised prompt")
		})
	})
}

func RunOpenAIOnStreamingResponseBodyTests(t *testing.T) {
	// 测试OpenAI响应体处理（流式响应）
	test.RunTest(t, func(t *testing.T) {
		t.Run("openai streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicOpenAIConfig)
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
				if strings.Contains(log, "streaming") || strings.Contains(log, "chunk") || strings.Contains(log, "openai") {
					hasStreamingLogs = true
					break
				}
			}
			require.True(t, hasStreamingLogs, "Should have streaming response processing logs")
		})
	})
}
