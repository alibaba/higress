package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：Vertex 标准模式配置
var basicVertexConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":                  "vertex",
			"vertexAuthKey":         `{"type":"service_account","client_email":"test@test.iam.gserviceaccount.com","private_key":"-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7k1v5C7y8L4SN\n-----END PRIVATE KEY-----\n","token_uri":"https://oauth2.googleapis.com/token"}`,
			"vertexRegion":          "us-central1",
			"vertexProjectId":       "test-project-id",
			"vertexAuthServiceName": "test-auth-service",
		},
	})
	return data
}()

// 测试配置：Vertex Express Mode 配置（使用 apiTokens）
var vertexExpressModeConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "vertex",
			"apiTokens": []string{"test-api-key-123456789"},
		},
	})
	return data
}()

// 测试配置：Vertex Express Mode 配置（含模型映射）
var vertexExpressModeWithModelMappingConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "vertex",
			"apiTokens": []string{"test-api-key-123456789"},
			"modelMapping": map[string]string{
				"gpt-4":                  "gemini-2.5-flash",
				"gpt-3.5-turbo":          "gemini-2.5-flash-lite",
				"text-embedding-ada-002": "text-embedding-001",
			},
		},
	})
	return data
}()

// 测试配置：Vertex Express Mode 配置（含安全设置）
var vertexExpressModeWithSafetyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "vertex",
			"apiTokens": []string{"test-api-key-123456789"},
			"geminiSafetySetting": map[string]string{
				"HARM_CATEGORY_HARASSMENT":        "BLOCK_MEDIUM_AND_ABOVE",
				"HARM_CATEGORY_HATE_SPEECH":       "BLOCK_LOW_AND_ABOVE",
				"HARM_CATEGORY_SEXUALLY_EXPLICIT": "BLOCK_NONE",
			},
		},
	})
	return data
}()

// 测试配置：无效 Vertex 标准模式配置（缺少 vertexAuthKey）
var invalidVertexStandardModeConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "vertex",
			// 缺少必需的标准模式配置
		},
	})
	return data
}()

func RunVertexParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试 Vertex 标准模式配置解析
		t.Run("vertex standard mode config", func(t *testing.T) {
			host, status := test.NewTestHost(basicVertexConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试 Vertex Express Mode 配置解析
		t.Run("vertex express mode config", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试 Vertex Express Mode 配置（含模型映射）
		t.Run("vertex express mode with model mapping config", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeWithModelMappingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试无效 Vertex 标准模式配置（缺少 vertexAuthKey）
		t.Run("invalid vertex standard mode config - missing auth key", func(t *testing.T) {
			host, status := test.NewTestHost(invalidVertexStandardModeConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试 Vertex Express Mode 配置（含安全设置）
		t.Run("vertex express mode with safety setting config", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeWithSafetyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func RunVertexExpressModeOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex Express Mode 请求头处理（聊天完成接口）
		t.Run("vertex express mode chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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

			// 验证Host是否被改为 vertex 域名（Express Mode 使用不带 region 前缀的域名）
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "aiplatform.googleapis.com"), "Host header should be changed to vertex domain without region prefix")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasVertexLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "vertex") {
					hasVertexLogs = true
					break
				}
			}
			require.True(t, hasVertexLogs, "Should have vertex processing logs")
		})

		// 测试 Vertex Express Mode 请求头处理（嵌入接口）
		t.Run("vertex express mode embeddings request headers", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "aiplatform.googleapis.com"), "Host header should be changed to vertex domain")
		})
	})
}

func RunVertexExpressModeOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex Express Mode 请求体处理（聊天完成接口）
		t.Run("vertex express mode chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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
			requestBody := `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"test"}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// Express Mode 不需要暂停等待 OAuth token
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证请求体被转换为 Vertex 格式
			require.Contains(t, string(processedBody), "contents", "Request should be converted to vertex format")
			require.Contains(t, string(processedBody), "generationConfig", "Request should contain vertex generation config")

			// 验证路径包含 API Key
			requestHeaders := host.GetRequestHeaders()
			pathHeader := ""
			for _, header := range requestHeaders {
				if header[0] == ":path" {
					pathHeader = header[1]
					break
				}
			}
			require.Contains(t, pathHeader, "key=test-api-key-123456789", "Path should contain API key as query parameter")
			require.Contains(t, pathHeader, "/v1/publishers/google/models/", "Path should use Express Mode format without project/location")

			// 验证没有 Authorization header（Express Mode 使用 URL 参数）
			hasAuthHeader := false
			for _, header := range requestHeaders {
				if header[0] == "Authorization" && header[1] != "" {
					hasAuthHeader = true
					break
				}
			}
			require.False(t, hasAuthHeader, "Authorization header should be removed in Express Mode")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasVertexLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "vertex") {
					hasVertexLogs = true
					break
				}
			}
			require.True(t, hasVertexLogs, "Should have vertex processing logs")
		})

		// 测试 Vertex Express Mode 请求体处理（嵌入接口）
		t.Run("vertex express mode embeddings request body", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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

			// 验证请求体被转换为 Vertex 格式
			require.Contains(t, string(processedBody), "instances", "Request should be converted to vertex format")

			// 验证路径包含 API Key
			requestHeaders := host.GetRequestHeaders()
			pathHeader := ""
			for _, header := range requestHeaders {
				if header[0] == ":path" {
					pathHeader = header[1]
					break
				}
			}
			require.Contains(t, pathHeader, "key=test-api-key-123456789", "Path should contain API key as query parameter")
		})

		// 测试 Vertex Express Mode 请求体处理（流式请求）
		t.Run("vertex express mode streaming request body", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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
			requestBody := `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"test"}],"stream":true}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证路径包含流式 action
			requestHeaders := host.GetRequestHeaders()
			pathHeader := ""
			for _, header := range requestHeaders {
				if header[0] == ":path" {
					pathHeader = header[1]
					break
				}
			}
			require.Contains(t, pathHeader, "streamGenerateContent", "Path should contain streaming action")
			require.Contains(t, pathHeader, "key=test-api-key-123456789", "Path should contain API key")
		})

		// 测试 Vertex Express Mode 请求体处理（含模型映射）
		t.Run("vertex express mode with model mapping request body", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeWithModelMappingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体（使用 OpenAI 模型名）
			requestBody := `{"model":"gpt-4","messages":[{"role":"user","content":"test"}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证路径包含映射后的模型名
			requestHeaders := host.GetRequestHeaders()
			pathHeader := ""
			for _, header := range requestHeaders {
				if header[0] == ":path" {
					pathHeader = header[1]
					break
				}
			}
			require.Contains(t, pathHeader, "gemini-2.5-flash", "Path should contain mapped model name")
		})
	})
}

func RunVertexExpressModeOnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex Express Mode 响应体处理（聊天完成接口）
		t.Run("vertex express mode chat completion response body", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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
			requestBody := `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"test"}]}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应属性，确保IsResponseFromUpstream()返回true
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体（Vertex 格式）
			responseBody := `{
				"candidates": [{
					"content": {
						"parts": [{
							"text": "Hello! How can I help you today?"
						}]
					},
					"finishReason": "STOP",
					"index": 0
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

			// 检查响应体是否被转换
			if strings.Contains(responseStr, "chat.completion") {
				require.Contains(t, responseStr, "assistant", "Response should contain assistant role")
				require.Contains(t, responseStr, "usage", "Response should contain usage information")
			}

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasResponseBodyLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "response") || strings.Contains(log, "body") || strings.Contains(log, "vertex") {
					hasResponseBodyLogs = true
					break
				}
			}
			require.True(t, hasResponseBodyLogs, "Should have response body processing logs")
		})
	})
}

func RunVertexExpressModeOnStreamingResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex Express Mode 流式响应处理
		t.Run("vertex express mode streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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
			requestBody := `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"test"}],"stream":true}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置流式响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "text/event-stream"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟流式响应体
			chunk1 := `data: {"candidates":[{"content":{"parts":[{"text":"Hello"}],"role":"model"},"finishReason":"","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":5,"totalTokenCount":14}}`
			chunk2 := `data: {"candidates":[{"content":{"parts":[{"text":"Hello! How can I help you today?"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":12,"totalTokenCount":21}}`

			// 处理流式响应体
			action1 := host.CallOnHttpStreamingResponseBody([]byte(chunk1), false)
			require.Equal(t, types.ActionContinue, action1)

			action2 := host.CallOnHttpStreamingResponseBody([]byte(chunk2), true)
			require.Equal(t, types.ActionContinue, action2)

			// 验证流式响应处理
			debugLogs := host.GetDebugLogs()
			hasStreamingLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "streaming") || strings.Contains(log, "chunk") || strings.Contains(log, "vertex") {
					hasStreamingLogs = true
					break
				}
			}
			require.True(t, hasStreamingLogs, "Should have streaming response processing logs")
		})
	})
}
