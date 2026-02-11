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

// 测试配置：Vertex OpenAI 兼容模式配置
var vertexOpenAICompatibleModeConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":                   "vertex",
			"vertexOpenAICompatible": true,
			"vertexAuthKey":          `{"type":"service_account","client_email":"test@test.iam.gserviceaccount.com","private_key":"-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7k1v5C7y8L4SN\n-----END PRIVATE KEY-----\n","token_uri":"https://oauth2.googleapis.com/token"}`,
			"vertexRegion":           "us-central1",
			"vertexProjectId":        "test-project-id",
			"vertexAuthServiceName":  "test-auth-service",
		},
	})
	return data
}()

// 测试配置：Vertex OpenAI 兼容模式配置（含模型映射）
var vertexOpenAICompatibleModeWithModelMappingConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":                   "vertex",
			"vertexOpenAICompatible": true,
			"vertexAuthKey":          `{"type":"service_account","client_email":"test@test.iam.gserviceaccount.com","private_key":"-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7k1v5C7y8L4SN\n-----END PRIVATE KEY-----\n","token_uri":"https://oauth2.googleapis.com/token"}`,
			"vertexRegion":           "us-central1",
			"vertexProjectId":        "test-project-id",
			"vertexAuthServiceName":  "test-auth-service",
			"modelMapping": map[string]string{
				"gpt-4":         "gemini-2.0-flash",
				"gpt-3.5-turbo": "gemini-1.5-flash",
			},
		},
	})
	return data
}()

// 测试配置：无效配置 - Express Mode 与 OpenAI 兼容模式互斥
var invalidVertexExpressAndOpenAICompatibleConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":                   "vertex",
			"apiTokens":              []string{"test-api-key"},
			"vertexOpenAICompatible": true,
		},
	})
	return data
}()

// 测试配置：Vertex Raw 模式配置（Express Mode + 原生 Vertex API 路径）
var vertexRawModeExpressConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "vertex",
			"apiTokens": []string{"test-api-key-for-raw-mode"},
			"protocol":  "original",
		},
	})
	return data
}()

// 测试配置：Vertex Raw 模式配置（标准模式 + 原生 Vertex API 路径）
var vertexRawModeStandardConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":                  "vertex",
			"vertexAuthKey":         `{"type":"service_account","client_email":"test@test.iam.gserviceaccount.com","private_key":"-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7k1v5C7y8L4SN\n-----END PRIVATE KEY-----\n","token_uri":"https://oauth2.googleapis.com/token"}`,
			"vertexRegion":          "us-central1",
			"vertexProjectId":       "test-project-id",
			"vertexAuthServiceName": "test-auth-service",
			"protocol":              "original",
		},
	})
	return data
}()

// 测试配置：Vertex Raw 模式配置（Express Mode + basePath removePrefix）
var vertexRawModeWithBasePathConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":             "vertex",
			"apiTokens":        []string{"test-api-key-for-raw-mode"},
			"protocol":         "original",
			"basePath":         "/vertex-proxy",
			"basePathHandling": "removePrefix",
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

		// 测试 Vertex OpenAI 兼容模式配置解析
		t.Run("vertex openai compatible mode config", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试 Vertex OpenAI 兼容模式配置（含模型映射）
		t.Run("vertex openai compatible mode with model mapping config", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeWithModelMappingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试无效配置 - Express Mode 与 OpenAI 兼容模式互斥
		t.Run("invalid config - express mode and openai compatible mode conflict", func(t *testing.T) {
			host, status := test.NewTestHost(invalidVertexExpressAndOpenAICompatibleConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
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

func RunVertexOpenAICompatibleModeOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex OpenAI 兼容模式请求头处理
		t.Run("vertex openai compatible mode request headers", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeConfig)
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

			// 验证Host是否被改为 vertex 域名（带 region 前缀）
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "us-central1-aiplatform.googleapis.com"), "Host header should be changed to vertex domain with region prefix")
		})
	})
}

func RunVertexOpenAICompatibleModeOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex OpenAI 兼容模式请求体处理（不转换格式，保持 OpenAI 格式）
		t.Run("vertex openai compatible mode request body - no format conversion", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体（OpenAI 格式）
			requestBody := `{"model":"gemini-2.0-flash","messages":[{"role":"user","content":"test"}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// OpenAI 兼容模式需要等待 OAuth token，所以返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			// 验证请求体保持 OpenAI 格式（不转换为 Vertex 原生格式）
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// OpenAI 兼容模式应该保持 messages 字段，而不是转换为 contents
			require.Contains(t, string(processedBody), "messages", "Request should keep OpenAI format with messages field")
			require.NotContains(t, string(processedBody), "contents", "Request should NOT be converted to vertex native format")

			// 验证路径为 OpenAI 兼容端点
			requestHeaders := host.GetRequestHeaders()
			pathHeader := ""
			for _, header := range requestHeaders {
				if header[0] == ":path" {
					pathHeader = header[1]
					break
				}
			}
			require.Contains(t, pathHeader, "/v1beta1/projects/", "Path should use OpenAI compatible endpoint format")
			require.Contains(t, pathHeader, "/endpoints/openapi/chat/completions", "Path should contain openapi chat completions endpoint")
		})

		// 测试 Vertex OpenAI 兼容模式请求体处理（含模型映射）
		t.Run("vertex openai compatible mode with model mapping", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeWithModelMappingConfig)
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

			require.Equal(t, types.ActionPause, action)

			// 验证请求体中的模型名被映射
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 模型名应该被映射为 gemini-2.0-flash
			require.Contains(t, string(processedBody), "gemini-2.0-flash", "Model name should be mapped to gemini-2.0-flash")
		})

		// 测试 Vertex OpenAI 兼容模式不支持 Embeddings API
		t.Run("vertex openai compatible mode - embeddings not supported", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeConfig)
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

			// OpenAI 兼容模式只支持 chat completions，embeddings 应该返回错误
			require.Equal(t, types.ActionContinue, action)
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

func RunVertexOpenAICompatibleModeOnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex OpenAI 兼容模式响应体处理（直接透传，不转换格式）
		t.Run("vertex openai compatible mode response body - passthrough", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeConfig)
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
			requestBody := `{"model":"gemini-2.0-flash","messages":[{"role":"user","content":"test"}]}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应属性，确保IsResponseFromUpstream()返回true
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体（OpenAI 格式 - 因为 Vertex AI OpenAI-compatible API 返回的就是 OpenAI 格式）
			responseBody := `{
				"id": "chatcmpl-abc123",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! How can I help you today?"
					},
					"finish_reason": "stop"
				}],
				"created": 1729986750,
				"model": "gemini-2.0-flash",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 9,
					"completion_tokens": 12,
					"total_tokens": 21
				}
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体被直接透传（不进行格式转换）
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			// 响应应该保持原样
			responseStr := string(processedResponseBody)
			require.Contains(t, responseStr, "chatcmpl-abc123", "Response should be passed through unchanged")
			require.Contains(t, responseStr, "chat.completion", "Response should contain original object type")
		})

		// 测试 Vertex OpenAI 兼容模式流式响应处理（直接透传）
		t.Run("vertex openai compatible mode streaming response - passthrough", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeConfig)
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
			requestBody := `{"model":"gemini-2.0-flash","messages":[{"role":"user","content":"test"}],"stream":true}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置流式响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "text/event-stream"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟 OpenAI 格式的流式响应（Vertex AI OpenAI-compatible API 返回）
			chunk1 := `data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1729986750,"model":"gemini-2.0-flash","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`
			chunk2 := `data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1729986750,"model":"gemini-2.0-flash","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}`

			// 处理流式响应体 - 应该直接透传
			action1 := host.CallOnHttpStreamingResponseBody([]byte(chunk1), false)
			require.Equal(t, types.ActionContinue, action1)

			action2 := host.CallOnHttpStreamingResponseBody([]byte(chunk2), true)
			require.Equal(t, types.ActionContinue, action2)
		})

		// 测试 Vertex OpenAI 兼容模式流式响应处理（Unicode 转义解码）
		t.Run("vertex openai compatible mode streaming response - unicode escape decoding", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeConfig)
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
			requestBody := `{"model":"gemini-2.0-flash","messages":[{"role":"user","content":"test"}],"stream":true}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置流式响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "text/event-stream"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟带有 Unicode 转义的流式响应（Vertex AI OpenAI-compatible API 可能返回的格式）
			// \u4e2d\u6587 = 中文
			chunkWithUnicode := `data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1729986750,"model":"gemini-2.0-flash","choices":[{"index":0,"delta":{"role":"assistant","content":"\u4e2d\u6587\u6d4b\u8bd5"},"finish_reason":null}]}`

			// 处理流式响应体 - 应该解码 Unicode 转义
			action := host.CallOnHttpStreamingResponseBody([]byte(chunkWithUnicode), false)
			require.Equal(t, types.ActionContinue, action)

			// 验证响应体中的 Unicode 转义已被解码
			responseBody := host.GetResponseBody()
			require.NotNil(t, responseBody)

			responseStr := string(responseBody)
			// 应该包含解码后的中文字符，而不是 \uXXXX 转义序列
			require.Contains(t, responseStr, "中文测试", "Unicode escapes should be decoded to Chinese characters")
			require.NotContains(t, responseStr, `\u4e2d`, "Should not contain Unicode escape sequences")
		})

		// 测试 Vertex OpenAI 兼容模式非流式响应处理（Unicode 转义解码）
		t.Run("vertex openai compatible mode response body - unicode escape decoding", func(t *testing.T) {
			host, status := test.NewTestHost(vertexOpenAICompatibleModeConfig)
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
			requestBody := `{"model":"gemini-2.0-flash","messages":[{"role":"user","content":"test"}]}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 模拟带有 Unicode 转义的响应体
			// \u76c8\u5229\u80fd\u529b = 盈利能力
			responseBodyWithUnicode := `{"id":"chatcmpl-abc123","object":"chat.completion","created":1729986750,"model":"gemini-2.0-flash","choices":[{"index":0,"message":{"role":"assistant","content":"\u76c8\u5229\u80fd\u529b\u5206\u6790"},"finish_reason":"stop"}]}`

			// 处理响应体 - 应该解码 Unicode 转义
			action := host.CallOnHttpResponseBody([]byte(responseBodyWithUnicode))
			require.Equal(t, types.ActionContinue, action)

			// 验证响应体中的 Unicode 转义已被解码
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			responseStr := string(processedResponseBody)
			// 应该包含解码后的中文字符
			require.Contains(t, responseStr, "盈利能力分析", "Unicode escapes should be decoded to Chinese characters")
			require.NotContains(t, responseStr, `\u76c8`, "Should not contain Unicode escape sequences")
		})
	})
}

// ==================== 图片生成测试 ====================

func RunVertexExpressModeImageGenerationRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex Express Mode 图片生成请求体处理
		t.Run("vertex express mode image generation request body", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/images/generations"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体（OpenAI 图片生成格式）
			requestBody := `{"model":"gemini-2.0-flash-exp","prompt":"A cute orange cat napping in the sunshine","size":"1024x1024"}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// Express Mode 不需要暂停等待 OAuth token
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确处理
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 验证请求体被转换为 Vertex 格式
			bodyStr := string(processedBody)
			require.Contains(t, bodyStr, "contents", "Request should be converted to vertex format with contents")
			require.Contains(t, bodyStr, "generationConfig", "Request should contain generationConfig")
			require.Contains(t, bodyStr, "responseModalities", "Request should contain responseModalities for image generation")
			require.Contains(t, bodyStr, "IMAGE", "Request should specify IMAGE in responseModalities")
			require.Contains(t, bodyStr, "imageConfig", "Request should contain imageConfig")

			// 验证路径包含 API Key 和正确的模型
			requestHeaders := host.GetRequestHeaders()
			pathHeader := ""
			for _, header := range requestHeaders {
				if header[0] == ":path" {
					pathHeader = header[1]
					break
				}
			}
			require.Contains(t, pathHeader, "key=test-api-key-123456789", "Path should contain API key as query parameter")
			require.Contains(t, pathHeader, "/v1/publishers/google/models/", "Path should use Express Mode format")
			require.Contains(t, pathHeader, "generateContent", "Path should use generateContent action for image generation")
			require.NotContains(t, pathHeader, "streamGenerateContent", "Path should NOT use streaming for image generation")
		})

		// 测试 Vertex Express Mode 图片生成请求体处理（自定义尺寸）
		t.Run("vertex express mode image generation with custom size", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/images/generations"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体（宽屏尺寸）
			requestBody := `{"model":"gemini-2.0-flash-exp","prompt":"A beautiful sunset over the ocean","size":"1792x1024"}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否正确处理尺寸映射
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			bodyStr := string(processedBody)
			// 1792x1024 应该映射为 16:9 宽高比
			require.Contains(t, bodyStr, "aspectRatio", "Request should contain aspectRatio in imageConfig")
			require.Contains(t, bodyStr, "16:9", "Request should map 1792x1024 to 16:9 aspect ratio")
		})

		// 测试 Vertex Express Mode 图片生成请求体处理（含安全设置）
		t.Run("vertex express mode image generation with safety settings", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeWithSafetyConfig)
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
			requestBody := `{"model":"gemini-2.0-flash-exp","prompt":"A mountain landscape"}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证请求体包含安全设置
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			bodyStr := string(processedBody)
			require.Contains(t, bodyStr, "safetySettings", "Request should contain safetySettings")
		})

		// 测试 Vertex Express Mode 图片生成请求体处理（含模型映射）
		t.Run("vertex express mode image generation with model mapping", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeWithModelMappingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/images/generations"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体（使用映射前的模型名称）
			requestBody := `{"model":"gpt-4","prompt":"A futuristic city"}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证路径中使用了映射后的模型名称
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

func RunVertexExpressModeImageGenerationResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex Express Mode 图片生成响应体处理
		t.Run("vertex express mode image generation response body", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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
			requestBody := `{"model":"gemini-2.0-flash-exp","prompt":"A cute cat"}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应属性，确保IsResponseFromUpstream()返回true
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体（Vertex 图片生成格式）
			responseBody := `{
				"candidates": [{
					"content": {
						"role": "model",
						"parts": [{
							"inlineData": {
								"mimeType": "image/png",
								"data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
							}
						}]
					},
					"finishReason": "STOP"
				}],
				"usageMetadata": {
					"promptTokenCount": 10,
					"candidatesTokenCount": 1024,
					"totalTokenCount": 1034
				}
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体是否被正确处理
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			responseStr := string(processedResponseBody)

			// 验证响应体被转换为 OpenAI 图片生成格式
			require.Contains(t, responseStr, "created", "Response should contain created field")
			require.Contains(t, responseStr, "data", "Response should contain data array")
			require.Contains(t, responseStr, "b64_json", "Response should contain b64_json field with base64 image data")
			require.Contains(t, responseStr, "usage", "Response should contain usage information")
			require.Contains(t, responseStr, "total_tokens", "Response should contain total_tokens in usage")
		})

		// 测试 Vertex Express Mode 图片生成响应体处理（跳过思考过程）
		t.Run("vertex express mode image generation response body - skip thinking", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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
			requestBody := `{"model":"gemini-3-pro-image-preview","prompt":"An Eiffel tower"}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应属性
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体（包含思考过程和图片）
			responseBody := `{
				"candidates": [{
					"content": {
						"role": "model",
						"parts": [
							{
								"text": "Considering visual elements...",
								"thought": true
							},
							{
								"inlineData": {
									"mimeType": "image/png",
									"data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
								}
							}
						]
					},
					"finishReason": "STOP"
				}],
				"usageMetadata": {
					"promptTokenCount": 13,
					"candidatesTokenCount": 1120,
					"totalTokenCount": 1356,
					"thoughtsTokenCount": 223
				}
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体是否被正确处理
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			responseStr := string(processedResponseBody)

			// 验证响应体只包含图片数据，不包含思考过程文本
			require.Contains(t, responseStr, "b64_json", "Response should contain b64_json field")
			require.NotContains(t, responseStr, "Considering visual elements", "Response should NOT contain thinking text")
			require.NotContains(t, responseStr, "thought", "Response should NOT contain thought field")
		})

		// 测试 Vertex Express Mode 图片生成响应体处理（空图片数据）
		t.Run("vertex express mode image generation response body - no image", func(t *testing.T) {
			host, status := test.NewTestHost(vertexExpressModeConfig)
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
			requestBody := `{"model":"gemini-2.0-flash-exp","prompt":"test"}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应属性
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置响应体（只有文本，没有图片）
			responseBody := `{
				"candidates": [{
					"content": {
						"role": "model",
						"parts": [{
							"text": "I cannot generate that image."
						}]
					},
					"finishReason": "SAFETY"
				}],
				"usageMetadata": {
					"promptTokenCount": 5,
					"candidatesTokenCount": 10,
					"totalTokenCount": 15
				}
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体是否被正确处理（即使没有图片）
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			responseStr := string(processedResponseBody)

			// 验证响应体结构正确，data 数组为空
			require.Contains(t, responseStr, "created", "Response should contain created field")
			require.Contains(t, responseStr, "data", "Response should contain data array")
		})
	})
}

// ==================== Vertex Raw 模式测试 ====================

func RunVertexRawModeOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex Raw 模式请求头处理（Express Mode + 原生 Vertex API 路径）
		t.Run("vertex raw mode express - request headers with native vertex path", func(t *testing.T) {
			host, status := test.NewTestHost(vertexRawModeExpressConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 使用原生 Vertex AI REST API 路径
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.0-flash:generateContent"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 应该返回 HeaderStopIteration，因为需要处理请求体
			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头是否被正确处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证 Host 是否被改为 vertex 域名（Express Mode 使用不带 region 前缀的域名）
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "aiplatform.googleapis.com"),
				"Host header should be changed to vertex domain without region prefix")
		})

		// 测试 Vertex Raw 模式请求头处理（标准模式 + 原生 Vertex API 路径）
		t.Run("vertex raw mode standard - request headers with native vertex path", func(t *testing.T) {
			host, status := test.NewTestHost(vertexRawModeStandardConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 使用原生 Vertex AI REST API 路径
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.0-flash:generateContent"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头是否被正确处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证 Host 是否被改为 vertex 域名（标准模式使用带 region 前缀的域名）
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "us-central1-aiplatform.googleapis.com"),
				"Host header should be changed to vertex domain with region prefix")
		})

		// 测试 Vertex Raw 模式请求头处理（带 basePath 前缀）
		t.Run("vertex raw mode with basePath - request headers", func(t *testing.T) {
			host, status := test.NewTestHost(vertexRawModeWithBasePathConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 使用带 basePath 前缀的原生 Vertex AI REST API 路径
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/vertex-proxy/v1/projects/test-project/locations/us-central1/publishers/google/models/imagen-4.0-generate-preview-06-06:predict"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头是否被正确处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证 Host 是否被改为 vertex 域名
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "aiplatform.googleapis.com"),
				"Host header should be changed to vertex domain")

			// 验证路径是否移除了 basePath 前缀
			pathHeader := ""
			for _, header := range requestHeaders {
				if header[0] == ":path" {
					pathHeader = header[1]
					break
				}
			}
			require.NotContains(t, pathHeader, "/vertex-proxy", "Path should have basePath prefix removed")
			require.Contains(t, pathHeader, "/v1/projects/", "Path should contain original vertex path after basePath removal")
		})

		// 测试 Vertex Raw 模式请求头处理（Anthropic 模型路径）
		t.Run("vertex raw mode express - request headers with anthropic model path", func(t *testing.T) {
			host, status := test.NewTestHost(vertexRawModeExpressConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 使用 Anthropic 模型的原生 Vertex AI REST API 路径
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/projects/test-project/locations/us-east5/publishers/anthropic/models/claude-sonnet-4@20250514:rawPredict"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.HeaderStopIteration, action)

			// 验证请求头是否被正确处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证 Host 是否被改为 vertex 域名
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "aiplatform.googleapis.com"),
				"Host header should be changed to vertex domain")
		})
	})
}

func RunVertexRawModeOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex Raw 模式请求体处理（Express Mode - 透传请求体）
		t.Run("vertex raw mode express - request body passthrough", func(t *testing.T) {
			host, status := test.NewTestHost(vertexRawModeExpressConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.0-flash:generateContent"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置原生 Vertex 格式的请求体
			requestBody := `{"contents":[{"role":"user","parts":[{"text":"Hello, world!"}]}],"generationConfig":{"temperature":0.7}}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// Express Mode 不需要暂停等待 OAuth token
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被透传（不做格式转换）
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 请求体应该保持原样
			require.Equal(t, requestBody, string(processedBody), "Request body should be passed through unchanged")
		})

		// 测试 Vertex Raw 模式请求体处理（标准模式 - 需要 OAuth token）
		// 注意：使用 countTokens action，因为 generateContent/predict 等会被识别为其他 API 类型
		// 注意：在单元测试环境中，由于测试配置使用的是无效的私钥，JWT 创建会失败，
		// 因此 getToken() 会返回错误，导致 ActionContinue 而不是 ActionPause。
		// 这个测试主要验证代码正确进入了 Vertex Raw 模式的处理分支，请求体被透传。
		t.Run("vertex raw mode standard - request body with oauth", func(t *testing.T) {
			host, status := test.NewTestHost(vertexRawModeStandardConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头 - 使用 countTokens action，这是一个不会被其他 API 类型匹配的原生 Vertex API
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.0-flash:countTokens"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置原生 Vertex 格式的请求体
			requestBody := `{"contents":[{"role":"user","parts":[{"text":"Hello, world!"}]}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 注意：在单元测试环境中，由于私钥无效，JWT 创建失败会返回 ActionContinue
			// 在真实环境中，如果 JWT 创建成功，会返回 ActionPause 等待 OAuth token
			// 这里我们只验证代码正确进入了 Vertex Raw 模式的处理分支
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被透传（不做格式转换）
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)

			// 请求体应该保持原样（这是 Vertex Raw 模式的核心功能）
			require.Equal(t, requestBody, string(processedBody), "Request body should be passed through unchanged")
		})

		// 测试 Vertex Raw 模式请求体处理（带 basePath 前缀 - 路径正确处理）
		t.Run("vertex raw mode with basePath - request body passthrough", func(t *testing.T) {
			host, status := test.NewTestHost(vertexRawModeWithBasePathConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头（带 basePath 前缀）
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/vertex-proxy/v1/projects/test-project/locations/us-central1/publishers/google/models/imagen-4.0-generate-preview-06-06:predict"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置原生 Vertex 格式的请求体（图片生成）
			requestBody := `{"instances":[{"prompt":"A beautiful sunset"}],"parameters":{"sampleCount":1}}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// Express Mode 不需要暂停等待 OAuth token
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被透传
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)
			require.Equal(t, requestBody, string(processedBody), "Request body should be passed through unchanged")

			// 验证路径已正确处理（移除 basePath）
			requestHeaders := host.GetRequestHeaders()
			pathHeader := ""
			for _, header := range requestHeaders {
				if header[0] == ":path" {
					pathHeader = header[1]
					break
				}
			}
			require.NotContains(t, pathHeader, "/vertex-proxy", "Path should have basePath prefix removed")
		})

		// 测试 Vertex Raw 模式请求体处理（流式请求）
		t.Run("vertex raw mode express - streaming request body passthrough", func(t *testing.T) {
			host, status := test.NewTestHost(vertexRawModeExpressConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头（流式端点）
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.0-flash:streamGenerateContent?alt=sse"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置原生 Vertex 格式的请求体
			requestBody := `{"contents":[{"role":"user","parts":[{"text":"Tell me a story"}]}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被透传
			processedBody := host.GetRequestBody()
			require.NotNil(t, processedBody)
			require.Equal(t, requestBody, string(processedBody), "Request body should be passed through unchanged")
		})
	})
}

func RunVertexRawModeOnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试 Vertex Raw 模式响应体处理（透传响应）
		t.Run("vertex raw mode express - response body passthrough", func(t *testing.T) {
			host, status := test.NewTestHost(vertexRawModeExpressConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/projects/test-project/locations/us-central1/publishers/google/models/gemini-2.0-flash:generateContent"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			// 设置请求体
			requestBody := `{"contents":[{"role":"user","parts":[{"text":"Hello"}]}]}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 设置响应属性
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// 设置响应头
			responseHeaders := [][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			}
			host.CallOnHttpResponseHeaders(responseHeaders)

			// 设置原生 Vertex 格式的响应体
			responseBody := `{
				"candidates": [{
					"content": {
						"role": "model",
						"parts": [{"text": "Hello! How can I help you?"}]
					},
					"finishReason": "STOP"
				}],
				"usageMetadata": {
					"promptTokenCount": 5,
					"candidatesTokenCount": 10,
					"totalTokenCount": 15
				}
			}`
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体被透传（不做格式转换）
			processedResponseBody := host.GetResponseBody()
			require.NotNil(t, processedResponseBody)

			responseStr := string(processedResponseBody)
			// 响应应该保持原生 Vertex 格式
			require.Contains(t, responseStr, "candidates", "Response should keep native vertex format with candidates")
			require.Contains(t, responseStr, "usageMetadata", "Response should keep native vertex format with usageMetadata")
		})
	})
}
