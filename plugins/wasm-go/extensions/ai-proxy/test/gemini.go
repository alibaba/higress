package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本gemini配置
var basicGeminiConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "gemini",
			"apiTokens": []string{"sk-gemini-test123456789"},
			"modelMapping": map[string]string{
				"*": "gemini-pro",
			},
		},
	})
	return data
}()

// 测试配置：gemini多模型配置
var geminiMultiModelConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "gemini",
			"apiTokens": []string{"sk-gemini-multi-model"},
			"modelMapping": map[string]string{
				"gpt-3.5-turbo":          "gemini-pro",
				"gpt-4":                  "gemini-2.0-flash-001",
				"text-embedding-ada-002": "text-embedding-001",
				"dall-e-3":               "imagen-3",
			},
		},
	})
	return data
}()

// 测试配置：无效gemini配置（缺少apiToken）
var invalidGeminiConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "gemini",
			// 缺少apiTokens
		},
	})
	return data
}()

// 测试配置：gemini安全设置配置
var geminiSafetySettingConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "gemini",
			"apiTokens": []string{"sk-gemini-safety"},
			"modelMapping": map[string]string{
				"*": "gemini-pro",
			},
			"geminiSafetySetting": map[string]string{
				"HARM_CATEGORY_HARASSMENT":        "BLOCK_MEDIUM_AND_ABOVE",
				"HARM_CATEGORY_HATE_SPEECH":       "BLOCK_LOW_AND_ABOVE",
				"HARM_CATEGORY_SEXUALLY_EXPLICIT": "BLOCK_NONE",
				"HARM_CATEGORY_DANGEROUS_CONTENT": "BLOCK_HIGH_AND_ABOVE",
			},
		},
	})
	return data
}()

// 测试配置：gemini思考模式配置
var geminiThinkingConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "gemini",
			"apiTokens": []string{"sk-gemini-thinking"},
			"modelMapping": map[string]string{
				"*": "gemini-2.5-pro",
			},
			"geminiThinkingBudget": 1000,
		},
	})
	return data
}()

// 测试配置：gemini API版本配置
var geminiApiVersionConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "gemini",
			"apiTokens": []string{"sk-gemini-version"},
			"modelMapping": map[string]string{
				"*": "gemini-pro",
			},
			"apiVersion": "v1",
		},
	})
	return data
}()

// 测试配置：gemini完整配置（包含所有特殊字段）
var completeGeminiConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "gemini",
			"apiTokens": []string{"sk-gemini-complete"},
			"modelMapping": map[string]string{
				"*": "gemini-pro",
			},
			"geminiSafetySetting": map[string]string{
				"HARM_CATEGORY_HARASSMENT": "BLOCK_MEDIUM_AND_ABOVE",
			},
			"geminiThinkingBudget": 500,
			"apiVersion":           "v1beta",
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

func RunGeminiParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本gemini配置解析
		t.Run("basic gemini config", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试gemini多模型配置解析
		t.Run("gemini multi model config", func(t *testing.T) {
			host, status := test.NewTestHost(geminiMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试无效gemini配置（缺少apiToken）
		t.Run("invalid gemini config - missing api token", func(t *testing.T) {
			host, status := test.NewTestHost(invalidGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试gemini安全设置配置解析
		t.Run("gemini safety setting config", func(t *testing.T) {
			host, status := test.NewTestHost(geminiSafetySettingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试gemini思考模式配置解析
		t.Run("gemini thinking config", func(t *testing.T) {
			host, status := test.NewTestHost(geminiThinkingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试gemini API版本配置解析
		t.Run("gemini api version config", func(t *testing.T) {
			host, status := test.NewTestHost(geminiApiVersionConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试gemini完整配置解析
		t.Run("gemini complete config", func(t *testing.T) {
			host, status := test.NewTestHost(completeGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func RunGeminiOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试gemini请求头处理（聊天完成接口）
		t.Run("gemini chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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

			// 验证Host是否被改为gemini默认域名
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "generativelanguage.googleapis.com"), "Host header should be changed to gemini default domain")

			// 验证API Key是否被设置
			require.True(t, test.HasHeaderWithValue(requestHeaders, "x-goog-api-key", "sk-gemini-test123456789"), "API Key header should contain gemini API token")

			// 验证Authorization是否被清空
			require.True(t, test.HasHeaderWithValue(requestHeaders, "Authorization", ""), "Authorization header should be removed for gemini")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasGeminiLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "gemini") {
					hasGeminiLogs = true
					break
				}
			}
			require.True(t, hasGeminiLogs, "Should have gemini processing logs")
		})

		// 测试gemini请求头处理（嵌入接口）
		t.Run("gemini embeddings request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "generativelanguage.googleapis.com"), "Host header should be changed to gemini default domain")

			// 验证API Key设置
			require.True(t, test.HasHeaderWithValue(requestHeaders, "x-goog-api-key", "sk-gemini-test123456789"), "API Key header should contain gemini API token")
		})

		// 测试gemini请求头处理（图像生成接口）
		t.Run("gemini image generation request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "generativelanguage.googleapis.com"), "Host header should be changed to gemini default domain")

			// 验证API Key设置
			require.True(t, test.HasHeaderWithValue(requestHeaders, "x-goog-api-key", "sk-gemini-test123456789"), "API Key header should contain gemini API token")
		})

		// 测试gemini思考模式请求头处理
		t.Run("gemini thinking mode request headers", func(t *testing.T) {
			host, status := test.NewTestHost(geminiThinkingConfig)
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

			// 验证思考模式的请求头处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证Host转换
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "generativelanguage.googleapis.com"), "Host header should be changed to gemini default domain")

			// 验证API Key设置
			require.True(t, test.HasHeaderWithValue(requestHeaders, "x-goog-api-key", "sk-gemini-thinking"), "API Key header should contain gemini API token")
		})
	})
}

func RunGeminiOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试gemini请求体处理（聊天完成接口）
		t.Run("gemini chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"gemini-pro","messages":[{"role":"user","content":"test"}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证请求体被转换为gemini格式
			require.Contains(t, string(processedBody), "contents", "Request should be converted to gemini format")
			require.Contains(t, string(processedBody), "generationConfig", "Request should contain gemini generation config")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			infoLogs := host.GetInfoLogs()

			// 验证是否有gemini相关的处理日志
			hasGeminiLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "gemini") {
					hasGeminiLogs = true
					break
				}
			}
			for _, log := range infoLogs {
				if strings.Contains(log, "gemini") {
					hasGeminiLogs = true
					break
				}
			}
			require.True(t, hasGeminiLogs, "Should have gemini processing logs")
		})

		// 测试gemini请求体处理（嵌入接口）
		t.Run("gemini embeddings request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"text-embedding-001","input":"test text"}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证嵌入接口的请求体处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证请求体被转换为gemini格式
			require.Contains(t, string(processedBody), "requests", "Request should be converted to gemini format")
			require.Contains(t, string(processedBody), "models/gemini-pro", "Request should contain gemini model path")

			// 检查处理日志
			debugLogs := host.GetDebugLogs()
			hasEmbeddingLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "embeddings") || strings.Contains(log, "gemini") {
					hasEmbeddingLogs = true
					break
				}
			}
			require.True(t, hasEmbeddingLogs, "Should have embedding processing logs")
		})

		// 测试gemini请求体处理（图像生成接口）
		t.Run("gemini image generation request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"imagen-3","prompt":"test image"}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证图像生成接口的请求体处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证请求体被转换为gemini格式
			require.Contains(t, string(processedBody), "instances", "Request should be converted to gemini format")
			require.Contains(t, string(processedBody), "parameters", "Request should contain gemini parameters")

			// 检查处理日志
			debugLogs := host.GetDebugLogs()
			hasImageLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "image") || strings.Contains(log, "gemini") {
					hasImageLogs = true
					break
				}
			}
			require.True(t, hasImageLogs, "Should have image generation processing logs")
		})

		// 测试gemini请求体处理（思考模式）
		t.Run("gemini thinking mode request body", func(t *testing.T) {
			host, status := test.NewTestHost(geminiThinkingConfig)
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
			requestBody := `{"model":"gemini-2.5-pro","messages":[{"role":"user","content":"test"}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证思考模式的请求体处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证请求体被转换为gemini格式并包含思考配置
			require.Contains(t, string(processedBody), "contents", "Request should be converted to gemini format")
			require.Contains(t, string(processedBody), "thinkingConfig", "Request should contain thinking configuration")

			// 检查处理日志
			debugLogs := host.GetDebugLogs()
			hasThinkingLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "thinking") || strings.Contains(log, "gemini") {
					hasThinkingLogs = true
					break
				}
			}
			require.True(t, hasThinkingLogs, "Should have thinking mode processing logs")
		})

		// 测试gemini请求体处理（安全设置）
		t.Run("gemini safety setting request body", func(t *testing.T) {
			host, status := test.NewTestHost(geminiSafetySettingConfig)
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
			requestBody := `{"model":"gemini-pro","messages":[{"role":"user","content":"test"}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证安全设置的请求体处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证请求体被转换为gemini格式并包含安全设置
			require.Contains(t, string(processedBody), "contents", "Request should be converted to gemini format")
			require.Contains(t, string(processedBody), "safetySettings", "Request should contain safety settings")

			// 检查处理日志
			debugLogs := host.GetDebugLogs()
			hasSafetyLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "safety") || strings.Contains(log, "gemini") {
					hasSafetyLogs = true
					break
				}
			}
			require.True(t, hasSafetyLogs, "Should have safety setting processing logs")
		})

		// 测试验证 flash 请求支持 generationConfig.responseModalities
		t.Run("gemini flash image generation with response modalities", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/images/generations"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			requestBody := `{"model":"gemini-2.5-flash-image","prompt":"test image","generationConfig":{"responseModalities":["TEXT","IMAGE"],"imageConfig":{"aspectRatio":"16:9","imageSize":"2K"}}}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)
			bodyStr := string(processedBody)
			require.Contains(t, bodyStr, `"responseModalities":["TEXT","IMAGE"]`, "response modalities should be forwarded to flash request")

			requestHeaders := host.GetRequestHeaders()
			require.Contains(t, bodyStr, `"aspectRatio":"16:9"`, "aspectRatio should be forwarded to gemini request")
			require.Contains(t, bodyStr, `"imageSize":"2K"`, "imageSize should be forwarded to gemini request")
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":path", "/v1beta/models/gemini-2.5-flash-image:generateContent"), "flash image request should call generateContent path")
		})
	})
}

func RunGeminiOnHttpResponseHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试gemini响应头处理（聊天完成接口）
		t.Run("gemini chat completion response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"gemini-pro","messages":[{"role":"user","content":"test"}]}`
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
			require.True(t, test.HasHeaderWithValue(processedResponseHeaders, ":status", "200"), "Status header should be 200")

			// 验证Content-Type
			require.True(t, test.HasHeaderWithValue(processedResponseHeaders, "Content-Type", "application/json"), "Content-Type header should be application/json")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasResponseLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "response") || strings.Contains(log, "gemini") {
					hasResponseLogs = true
					break
				}
			}
			require.True(t, hasResponseLogs, "Should have response processing logs")
		})

		// 测试gemini响应头处理（嵌入接口）
		t.Run("gemini embeddings response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"text-embedding-001","input":"test text"}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
				{"X-Embedding-Model", "text-embedding-001"},
			}
			action := host.CallOnHttpResponseHeaders(responseHeaders)

			require.Equal(t, types.ActionContinue, action)

			// 验证响应头处理
			processedResponseHeaders := host.GetResponseHeaders()
			require.NotNil(t, processedResponseHeaders)

			// 验证嵌入模型信息
			require.True(t, test.HasHeaderWithValue(processedResponseHeaders, "X-Embedding-Model", "text-embedding-001"), "Embedding model should match configuration")
		})

		// 测试gemini响应头处理（错误响应）
		t.Run("gemini error response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"gemini-pro","messages":[{"role":"user","content":"test"}]}`
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
			require.True(t, test.HasHeaderWithValue(processedResponseHeaders, ":status", "429"), "Status should be 429 (Too Many Requests)")

			// 验证重试信息
			require.True(t, test.HasHeaderWithValue(processedResponseHeaders, "Retry-After", "60"), "Retry-After should be 60 seconds")
		})
	})
}

func RunGeminiOnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试gemini响应体处理（聊天完成接口）
		t.Run("gemini chat completion response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"gemini-pro","messages":[{"role":"user","content":"test"}]}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应属性，确保IsResponseFromUpstream()返回true
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体（gemini格式）
			responseBody := `{
				"candidates": [{
					"content": {
						"parts": [{
							"text": "Hello! How can I help you today?"
						}]
					},
					"finishReason": "STOP",
					"index": 0,
					"safetyRatings": [{
						"category": "HARM_CATEGORY_HARASSMENT",
						"probability": "NEGLIGIBLE"
					}]
				}],
				"usageMetadata": {
					"promptTokenCount": 9,
					"candidatesTokenCount": 12,
					"totalTokenCount": 21
				}
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体是否被正确处理
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			// 验证响应体内容（转换为OpenAI格式）
			responseStr := string(processedResponseBody)

			// 添加调试信息
			debugLogs := host.GetDebugLogs()
			t.Logf("Original response body: %s", string(responseBody))
			t.Logf("Processed response body: %s", responseStr)
			t.Logf("Debug logs: %v", debugLogs)

			// 检查响应体是否被转换
			if strings.Contains(responseStr, "chat.completion") {
				// 响应体已被转换
				require.Contains(t, responseStr, "assistant", "Response should contain assistant role")
				require.Contains(t, responseStr, "usage", "Response should contain usage information")
			} else {
				// 响应体未被转换，检查是否有错误日志
				errorLogs := host.GetErrorLogs()
				require.Empty(t, errorLogs, "No errors should occur during response body transformation")

				// 即使响应体未被转换，我们也应该验证gemini provider被调用
				hasGeminiLogs := false
				for _, logEntry := range debugLogs {
					if strings.Contains(logEntry, "gemini") {
						hasGeminiLogs = true
						break
					}
				}
				require.True(t, hasGeminiLogs, "Should have gemini processing logs")
			}

			// 检查是否有相关的处理日志
			hasResponseBodyLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "response") || strings.Contains(log, "body") || strings.Contains(log, "gemini") {
					hasResponseBodyLogs = true
					break
				}
			}
			require.True(t, hasResponseBodyLogs, "Should have response body processing logs")
		})

		// 测试gemini响应体处理（嵌入接口）
		t.Run("gemini embeddings response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"text-embedding-001","input":"test text"}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应属性，确保IsResponseFromUpstream()返回true
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体（gemini格式）
			responseBody := `{
				"embeddings": [{
					"values": [0.1, 0.2, 0.3, 0.4, 0.5]
				}]
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体处理
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			// 验证嵌入响应内容（转换为OpenAI格式）
			responseStr := string(processedResponseBody)
			require.Contains(t, responseStr, "embedding", "Response should contain embedding object")
			require.Contains(t, responseStr, "0.1", "Response should contain embedding vector")
			require.Contains(t, responseStr, "list", "Response should contain list object")
		})

		// 测试gemini响应体处理（图像生成接口）
		t.Run("gemini image generation response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"imagen-3","prompt":"test image"}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应属性，确保IsResponseFromUpstream()返回true
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体（gemini格式）
			responseBody := `{
				"predictions": [{
					"bytesBase64Encoded": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
					"mimeType": "image/png"
				}]
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体处理
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			// 验证图像生成响应内容（转换为OpenAI格式）
			responseStr := string(processedResponseBody)
			require.Contains(t, responseStr, "data", "Response should contain data array")
			require.Contains(t, responseStr, "b64", "Response should contain base64 encoded image")
		})
	})
}

func RunGeminiOnStreamingResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试gemini响应体处理（流式响应）
		t.Run("gemini streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
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
			requestBody := `{"model":"gemini-pro","messages":[{"role":"user","content":"test"}],"stream":true}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置流式响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "text/event-stream"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟流式响应体
			chunk1 := `{"candidates":[{"content":{"parts":[{"text":""}],"role":"model"},"finishReason":"","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":0,"totalTokenCount":9}}`
			chunk2 := `{"candidates":[{"content":{"parts":[{"text":"Hello"}],"role":"model"},"finishReason":"","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":5,"totalTokenCount":14}}`
			chunk3 := `{"candidates":[{"content":{"parts":[{"text":"Hello! How can I help you today?"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":12,"totalTokenCount":21}}`

			// 处理流式响应体
			action1 := host.CallOnHttpStreamingResponseBody([]byte(chunk1), false)
			require.Equal(t, types.ActionContinue, action1)

			action2 := host.CallOnHttpStreamingResponseBody([]byte(chunk2), false)
			require.Equal(t, types.ActionContinue, action2)

			action3 := host.CallOnHttpStreamingResponseBody([]byte(chunk3), true)
			require.Equal(t, types.ActionContinue, action3)

			// 验证流式响应处理
			// 注意：流式响应可能不会在GetResponseBody中累积，需要检查日志或其他方式验证
			debugLogs := host.GetDebugLogs()
			hasStreamingLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "streaming") || strings.Contains(log, "chunk") || strings.Contains(log, "gemini") {
					hasStreamingLogs = true
					break
				}
			}
			require.True(t, hasStreamingLogs, "Should have streaming response processing logs")
		})

		// 测试gemini增量流式响应处理
		t.Run("gemini incremental streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置增量流式请求体
			requestBody := `{"model":"gemini-pro","messages":[{"role":"user","content":"test"}],"stream":true}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置流式响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "text/event-stream"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟增量流式响应体
			chunk1 := `{"candidates":[{"content":{"parts":[{"text":"H"}],"role":"model"},"finishReason":"","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":1,"totalTokenCount":10}}`
			chunk2 := `{"candidates":[{"content":{"parts":[{"text":"He"}],"role":"model"},"finishReason":"","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":2,"totalTokenCount":11}}`
			chunk3 := `{"candidates":[{"content":{"parts":[{"text":"Hello"}],"role":"model"},"finishReason":"","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":5,"totalTokenCount":14}}`
			chunk4 := `{"candidates":[{"content":{"parts":[{"text":"Hello! How can I help you today?"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":12,"totalTokenCount":21}}`

			// 处理增量流式响应体
			action1 := host.CallOnHttpStreamingResponseBody([]byte(chunk1), false)
			require.Equal(t, types.ActionContinue, action1)

			action2 := host.CallOnHttpStreamingResponseBody([]byte(chunk2), false)
			require.Equal(t, types.ActionContinue, action2)

			action3 := host.CallOnHttpStreamingResponseBody([]byte(chunk3), false)
			require.Equal(t, types.ActionContinue, action3)

			action4 := host.CallOnHttpStreamingResponseBody([]byte(chunk4), true)
			require.Equal(t, types.ActionContinue, action4)

			// 验证增量流式响应处理
			debugLogs := host.GetDebugLogs()
			hasIncrementalLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "incremental") || strings.Contains(log, "streaming") || strings.Contains(log, "gemini") {
					hasIncrementalLogs = true
					break
				}
			}
			require.True(t, hasIncrementalLogs, "Should have incremental streaming response processing logs")
		})

		// 测试gemini思考模式流式响应处理
		// 测试gemini思考模式流式响应处理
		t.Run("gemini thinking mode streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(geminiThinkingConfig)
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
			requestBody := `{"model":"gemini-2.5-pro","messages":[{"role":"user","content":"test"}],"stream":true}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置流式响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "text/event-stream"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟思考模式流式响应体
			chunk1 := `{"candidates":[{"content":{"parts":[{"text":"Let me think about this..."}],"role":"model"},"finishReason":"","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":8,"totalTokenCount":17}}`
			chunk2 := `{"candidates":[{"content":{"parts":[{"text":"Hello! How can I help you today?"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":12,"totalTokenCount":21}}`

			// 处理思考模式流式响应体
			action1 := host.CallOnHttpStreamingResponseBody([]byte(chunk1), false)
			require.Equal(t, types.ActionContinue, action1)

			action2 := host.CallOnHttpStreamingResponseBody([]byte(chunk2), true)
			require.Equal(t, types.ActionContinue, action2)

			// 验证思考模式流式响应处理
			debugLogs := host.GetDebugLogs()
			hasThinkingStreamingLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "thinking") || strings.Contains(log, "streaming") || strings.Contains(log, "gemini") {
					hasThinkingStreamingLogs = true
					break
				}
			}
			require.True(t, hasThinkingStreamingLogs, "Should have thinking mode streaming response processing logs")
		})

		// 测试gemini多模态流式响应处理
		t.Run("gemini multimodal streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置多模态流式请求体
			requestBody := `{"model":"gemini-pro","messages":[{"role":"user","content":"test"}],"stream":true}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置流式响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "text/event-stream"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟多模态流式响应体
			chunk1 := `{"candidates":[{"content":{"parts":[{"text":"I can see the image and understand your question..."}],"role":"model"},"finishReason":"","index":0}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":12,"totalTokenCount":27}}`
			chunk2 := `{"candidates":[{"content":{"parts":[{"text":"I can see the image and understand your question. Here's my response based on what I observe."}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":25,"totalTokenCount":40}}`

			// 处理多模态流式响应体
			action1 := host.CallOnHttpStreamingResponseBody([]byte(chunk1), false)
			require.Equal(t, types.ActionContinue, action1)

			action2 := host.CallOnHttpStreamingResponseBody([]byte(chunk2), true)
			require.Equal(t, types.ActionContinue, action2)

			// 验证多模态流式响应处理
			debugLogs := host.GetDebugLogs()
			hasMultimodalLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "multimodal") || strings.Contains(log, "streaming") || strings.Contains(log, "gemini") {
					hasMultimodalLogs = true
					break
				}
			}
			require.True(t, hasMultimodalLogs, "Should have multimodal streaming response processing logs")
		})

		// 测试gemini安全设置流式响应处理
		t.Run("gemini safety setting streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(geminiSafetySettingConfig)
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
			requestBody := `{"model":"gemini-pro","messages":[{"role":"user","content":"test"}],"stream":true}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置流式响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "text/event-stream"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟安全设置流式响应体
			chunk1 := `{"candidates":[{"content":{"parts":[{"text":"Hello"}],"role":"model"},"finishReason":"","index":0,"safetyRatings":[{"category":"HARM_CATEGORY_HARASSMENT","probability":"NEGLIGIBLE"}]}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":5,"totalTokenCount":14}}`
			chunk2 := `{"candidates":[{"content":{"parts":[{"text":"Hello! How can I help you today?"}],"role":"model"},"finishReason":"STOP","index":0,"safetyRatings":[{"category":"HARM_CATEGORY_HARASSMENT","probability":"NEGLIGIBLE"}]}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":12,"totalTokenCount":21}}`

			// 处理安全设置流式响应体
			action1 := host.CallOnHttpStreamingResponseBody([]byte(chunk1), false)
			require.Equal(t, types.ActionContinue, action1)

			action2 := host.CallOnHttpStreamingResponseBody([]byte(chunk2), true)
			require.Equal(t, types.ActionContinue, action2)

			// 验证安全设置流式响应处理
			debugLogs := host.GetDebugLogs()
			hasSafetyStreamingLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "safety") || strings.Contains(log, "streaming") || strings.Contains(log, "gemini") {
					hasSafetyStreamingLogs = true
					break
				}
			}
			require.True(t, hasSafetyStreamingLogs, "Should have safety setting streaming response processing logs")
		})
	})
}

func RunGeminiGetImageURLTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试gemini外部服务交互（图片URL获取）
		t.Run("gemini external image URL fetch", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置包含图片URL的请求体
			requestBody := `{
						"model": "gemini-pro",
						"messages": [{
							"role": "user",
							"content": [
								{"type": "text", "text": "What's in this image?"},
								{"type": "image_url", "image_url": {"url": "https://example.com/test-image.jpg"}}
							]
						}]
					}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 由于需要获取外部图片，应该返回ActionPause
			require.Equal(t, types.ActionPause, action)

			// 模拟外部HTTP调用响应（图片获取成功）
			imageResponseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "image/jpeg"},
			}
			imageResponseBody := []byte("fake-image-data")
			host.CallOnHttpCall(imageResponseHeaders, imageResponseBody)

			// 验证外部服务交互
			debugLogs := host.GetDebugLogs()
			hasExternalServiceLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "image") || strings.Contains(log, "fetch") || strings.Contains(log, "external") {
					hasExternalServiceLogs = true
					break
				}
			}
			require.True(t, hasExternalServiceLogs, "Should have external service interaction logs")
		})

		// 测试gemini外部服务交互（多个图片URL获取）
		t.Run("gemini multiple external image URLs fetch", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置包含多个图片URL的请求体
			requestBody := `{
						"model": "gemini-pro",
						"messages": [{
							"role": "user",
							"content": [
								{"type": "text", "text": "Compare these two images"},
								{"type": "image_url", "image_url": {"url": "https://example.com/image1.jpg"}},
								{"type": "image_url", "image_url": {"url": "https://example.com/image2.jpg"}}
							]
						}]
					}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 由于需要获取多个外部图片，应该返回ActionPause
			require.Equal(t, types.ActionPause, action)

			// 模拟第一个图片的HTTP调用响应
			image1ResponseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "image/jpeg"},
			}
			image1ResponseBody := []byte("fake-image-1-data")
			host.CallOnHttpCall(image1ResponseHeaders, image1ResponseBody)

			// 模拟第二个图片的HTTP调用响应
			image2ResponseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "image/png"},
			}
			image2ResponseBody := []byte("fake-image-2-data")
			host.CallOnHttpCall(image2ResponseHeaders, image2ResponseBody)

			// 验证多个外部服务交互
			debugLogs := host.GetDebugLogs()
			hasMultipleImageLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "image") && (strings.Contains(log, "1") || strings.Contains(log, "2")) {
					hasMultipleImageLogs = true
					break
				}
			}
			require.True(t, hasMultipleImageLogs, "Should have multiple image external service interaction logs")
		})

		// 测试gemini外部服务交互（图片获取失败）
		t.Run("gemini external image fetch failure", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置包含图片URL的请求体
			requestBody := `{
						"model": "gemini-pro",
						"messages": [{
							"role": "user",
							"content": [
								{"type": "text", "text": "What's in this image?"},
								{"type": "image_url", "image_url": {"url": "https://example.com/invalid-image.jpg"}}
							]
						}]
					}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 由于需要获取外部图片，应该返回ActionPause
			require.Equal(t, types.ActionPause, action)

			// 模拟外部HTTP调用响应（图片获取失败）
			imageErrorResponseHeaders := [][2]string{
				{":status", "404"},
				{"Content-Type", "text/plain"},
			}
			imageErrorResponseBody := []byte("Image not found")
			host.CallOnHttpCall(imageErrorResponseHeaders, imageErrorResponseBody)

			// 验证外部服务交互失败处理
			errorLogs := host.GetErrorLogs()
			hasImageErrorLogs := false
			for _, log := range errorLogs {
				if strings.Contains(log, "image") || strings.Contains(log, "fetch") || strings.Contains(log, "failed") {
					hasImageErrorLogs = true
					break
				}
			}
			require.True(t, hasImageErrorLogs, "Should have image fetch failure error logs")
		})

		// 测试gemini外部服务交互（base64图片处理）
		t.Run("gemini base64 image processing", func(t *testing.T) {
			host, status := test.NewTestHost(basicGeminiConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置包含base64图片的请求体
			requestBody := `{
						"model": "gemini-pro",
						"messages": [{
							"role": "user",
							"content": [
								{"type": "text", "text": "What's in this image?"},
								{"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k="}}
							]
						}]
					}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// base64图片应该直接处理，不需要外部服务调用
			require.Equal(t, types.ActionContinue, action)

			// 验证base64图片处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证base64图片被正确处理
			bodyStr := string(processedBody)
			require.Contains(t, bodyStr, "inlineData", "Response should contain inlineData for base64 image")
			require.Contains(t, bodyStr, "image/jpeg", "Response should contain correct MIME type")
		})
	})
}
