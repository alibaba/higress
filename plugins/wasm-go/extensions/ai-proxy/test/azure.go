package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本Azure OpenAI配置
var basicAzureConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "azure",
			"apiTokens": []string{
				"sk-azure-test123456789",
			},
			"azureServiceUrl": "https://test-resource.openai.azure.com/openai/deployments/test-deployment/chat/completions?api-version=2024-02-15-preview",
			"modelMapping": map[string]string{
				"*": "gpt-3.5-turbo",
			},
		},
	})
	return data
}()

// 测试配置：Azure OpenAI完整路径配置
var azureFullPathConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "azure",
			"apiTokens": []string{
				"sk-azure-fullpath",
			},
			"azureServiceUrl": "https://fullpath-resource.openai.azure.com/openai/deployments/fullpath-deployment/chat/completions?api-version=2024-02-15-preview",
			"modelMapping": map[string]string{
				"gpt-3.5-turbo": "gpt-3.5-turbo",
				"gpt-4":         "gpt-4",
			},
		},
	})
	return data
}()

// 测试配置：Azure OpenAI仅部署配置
var azureDeploymentOnlyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "azure",
			"apiTokens": []string{
				"sk-azure-deployment",
			},
			"azureServiceUrl": "https://deployment-resource.openai.azure.com/openai/deployments/deployment-only?api-version=2024-02-15-preview",
			"modelMapping": map[string]string{
				"*": "gpt-3.5-turbo",
			},
		},
	})
	return data
}()

// 测试配置：Azure OpenAI仅域名配置
var azureDomainOnlyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "azure",
			"apiTokens": []string{
				"sk-azure-domain",
			},
			"azureServiceUrl": "https://domain-resource.openai.azure.com?api-version=2024-02-15-preview",
			"modelMapping": map[string]string{
				"*": "gpt-3.5-turbo",
			},
		},
	})
	return data
}()

// 测试配置：Azure OpenAI多模型配置
var azureMultiModelConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "azure",
			"apiTokens": []string{
				"sk-azure-multi",
			},
			"azureServiceUrl": "https://multi-resource.openai.azure.com/openai/deployments/multi-deployment?api-version=2024-02-15-preview",
			"modelMapping": map[string]string{
				"gpt-3.5-turbo":          "gpt-3.5-turbo",
				"gpt-4":                  "gpt-4",
				"text-embedding-ada-002": "text-embedding-ada-002",
			},
		},
	})
	return data
}()

// 测试配置：Azure OpenAI无效配置（缺少azureServiceUrl）
var azureInvalidConfigMissingUrl = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "azure",
			"apiTokens": []string{
				"sk-azure-invalid",
			},
			"modelMapping": map[string]string{
				"*": "gpt-3.5-turbo",
			},
		},
	})
	return data
}()

// 测试配置：Azure OpenAI无效配置（缺少api-version）
var azureInvalidConfigMissingApiVersion = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "azure",
			"apiTokens": []string{
				"sk-azure-invalid",
			},
			"azureServiceUrl": "https://invalid-resource.openai.azure.com/openai/deployments/invalid-deployment/chat/completions",
			"modelMapping": map[string]string{
				"*": "gpt-3.5-turbo",
			},
		},
	})
	return data
}()

// 测试配置：Azure OpenAI无效配置（缺少apiToken）
var azureInvalidConfigMissingToken = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":            "azure",
			"azureServiceUrl": "https://invalid-resource.openai.azure.com/openai/deployments/invalid-deployment/chat/completions?api-version=2024-02-15-preview",
			"modelMapping": map[string]interface{}{
				"*": "gpt-3.5-turbo",
			},
		},
	})
	return data
}()

func RunAzureParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本Azure OpenAI配置解析
		t.Run("basic azure config", func(t *testing.T) {
			host, status := test.NewTestHost(basicAzureConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试Azure OpenAI完整路径配置解析
		t.Run("azure full path config", func(t *testing.T) {
			host, status := test.NewTestHost(azureFullPathConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试Azure OpenAI仅部署配置解析
		t.Run("azure deployment only config", func(t *testing.T) {
			host, status := test.NewTestHost(azureDeploymentOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试Azure OpenAI仅域名配置解析
		t.Run("azure domain only config", func(t *testing.T) {
			host, status := test.NewTestHost(azureDomainOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试Azure OpenAI多模型配置解析
		t.Run("azure multi model config", func(t *testing.T) {
			host, status := test.NewTestHost(azureMultiModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试Azure OpenAI无效配置（缺少azureServiceUrl）
		t.Run("azure invalid config missing url", func(t *testing.T) {
			host, status := test.NewTestHost(azureInvalidConfigMissingUrl)
			defer host.Reset()
			// 应该失败，因为缺少azureServiceUrl
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试Azure OpenAI无效配置（缺少api-version）
		t.Run("azure invalid config missing api version", func(t *testing.T) {
			host, status := test.NewTestHost(azureInvalidConfigMissingApiVersion)
			defer host.Reset()
			// 应该失败，因为缺少api-version
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试Azure OpenAI无效配置（缺少apiToken）
		t.Run("azure invalid config missing token", func(t *testing.T) {
			host, status := test.NewTestHost(azureInvalidConfigMissingToken)
			defer host.Reset()
			// 应该失败，因为缺少apiToken
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})
	})
}

func RunAzureOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试Azure OpenAI请求头处理（聊天完成接口）
		t.Run("azure chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicAzureConfig)
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

			// 验证Host是否被改为Azure服务域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Equal(t, "test-resource.openai.azure.com", hostValue, "Host should be changed to Azure service domain")

			// 验证api-key是否被设置
			apiKeyValue, hasApiKey := test.GetHeaderValue(requestHeaders, "api-key")
			require.True(t, hasApiKey, "api-key header should exist")
			require.Equal(t, "sk-azure-test123456789", apiKeyValue, "api-key should contain Azure API token")

			// 验证Path是否被正确处理
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/openai/deployments/test-deployment/chat/completions", "Path should contain Azure deployment path")

			// 验证Content-Length是否被删除
			_, hasContentLength := test.GetHeaderValue(requestHeaders, "Content-Length")
			require.False(t, hasContentLength, "Content-Length header should be deleted")

			// 检查是否有相关的处理日志
			debugLogs := host.GetDebugLogs()
			hasAzureLogs := false
			for _, log := range debugLogs {
				if strings.Contains(log, "azureProvider") {
					hasAzureLogs = true
					break
				}
			}
			assert.True(t, hasAzureLogs, "Should have Azure provider debug logs")
		})

		// 测试Azure OpenAI请求头处理（完整路径配置）
		t.Run("azure full path request headers", func(t *testing.T) {
			host, status := test.NewTestHost(azureFullPathConfig)
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

			// 验证请求头是否被正确处理
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// 验证Host是否被改为Azure服务域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Equal(t, "fullpath-resource.openai.azure.com", hostValue, "Host should be changed to Azure service domain")

			// 验证api-key是否被设置
			apiKeyValue, hasApiKey := test.GetHeaderValue(requestHeaders, "api-key")
			require.True(t, hasApiKey, "api-key header should exist")
			require.Equal(t, "sk-azure-fullpath", apiKeyValue, "api-key should contain Azure API token")
		})
	})
}

func RunAzureOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试Azure OpenAI请求体处理（聊天完成接口）
		t.Run("azure chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicAzureConfig)
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

			// 设置请求体
			requestBody := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{
						"role": "user",
						"content": "Hello, how are you?"
					}
				],
				"temperature": 0.7
			}`

			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确处理
			transformedBody := host.GetRequestBody()
			require.NotNil(t, transformedBody)

			// 验证模型映射是否生效
			var bodyMap map[string]interface{}
			err := json.Unmarshal(transformedBody, &bodyMap)
			require.NoError(t, err)

			model, exists := bodyMap["model"]
			require.True(t, exists, "Model should exist in request body")
			require.Equal(t, "gpt-3.5-turbo", model, "Model should be mapped correctly")

			// 验证请求路径是否被正确转换
			requestHeaders := host.GetRequestHeaders()
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/openai/deployments/test-deployment/chat/completions", "Path should contain Azure deployment path")
			require.Contains(t, pathValue, "api-version=2024-02-15-preview", "Path should contain API version")
		})

		// 测试Azure OpenAI请求体处理（不同模型）
		t.Run("azure different model request body", func(t *testing.T) {
			host, status := test.NewTestHost(azureMultiModelConfig)
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

			// 设置请求体
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{
						"role": "user",
						"content": "Explain quantum computing"
					}
				]
			}`

			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确处理
			transformedBody := host.GetRequestBody()
			require.NotNil(t, transformedBody)

			var bodyMap map[string]interface{}
			err := json.Unmarshal(transformedBody, &bodyMap)
			require.NoError(t, err)

			model, exists := bodyMap["model"]
			require.True(t, exists, "Model should exist in request body")
			require.Equal(t, "gpt-4", model, "Model should be mapped correctly")
		})

		// 测试Azure OpenAI请求体处理（仅部署配置）
		t.Run("azure deployment only request body", func(t *testing.T) {
			host, status := test.NewTestHost(azureDeploymentOnlyConfig)
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

			// 设置请求体
			requestBody := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{
						"role": "user",
						"content": "Test message"
					}
				]
			}`

			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求路径是否使用默认部署
			requestHeaders := host.GetRequestHeaders()
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/openai/deployments/deployment-only/chat/completions", "Path should use default deployment")
		})

		// 测试Azure OpenAI请求体处理（仅域名配置）
		t.Run("azure domain only request body", func(t *testing.T) {
			host, status := test.NewTestHost(azureDomainOnlyConfig)
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

			// 设置请求体
			requestBody := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{
						"role": "user",
						"content": "Test message"
					}
				]
			}`

			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求路径是否使用模型占位符
			requestHeaders := host.GetRequestHeaders()
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/openai/deployments/gpt-3.5-turbo/chat/completions", "Path should use model from request body")
		})
	})
}

func RunAzureOnHttpResponseHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试Azure OpenAI响应头处理
		t.Run("azure response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicAzureConfig)
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

			// 设置请求体
			requestBody := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{
						"role": "user",
						"content": "Hello"
					}
				]
			}`
			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 处理响应头
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.ActionContinue, action)

			// 验证响应头是否被正确处理
			responseHeaders := host.GetResponseHeaders()
			require.NotNil(t, responseHeaders)

			// 验证状态码
			statusValue, hasStatus := test.GetHeaderValue(responseHeaders, ":status")
			require.True(t, hasStatus, "Status header should exist")
			require.Equal(t, "200", statusValue, "Status should be 200")

			// 验证Content-Type
			contentTypeValue, hasContentType := test.GetHeaderValue(responseHeaders, "Content-Type")
			require.True(t, hasContentType, "Content-Type header should exist")
			require.Equal(t, "application/json", contentTypeValue, "Content-Type should be application/json")
		})
	})
}

func RunAzureOnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试Azure OpenAI响应体处理
		t.Run("azure response body", func(t *testing.T) {
			host, status := test.NewTestHost(basicAzureConfig)
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

			// 设置请求体
			requestBody := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{
						"role": "user",
						"content": "Hello"
					}
				]
			}`
			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 处理响应体
			responseBody := `{
				"choices": [
					{
						"message": {
							"content": "Hello! How can I help you?"
						}
					}
				]
			}`

			action = host.CallOnHttpResponseBody([]byte(responseBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证响应体是否被正确处理
			transformedResponseBody := host.GetResponseBody()
			require.NotNil(t, transformedResponseBody)

			// 验证响应体内容
			var responseMap map[string]interface{}
			err := json.Unmarshal(transformedResponseBody, &responseMap)
			require.NoError(t, err)

			choices, exists := responseMap["choices"]
			require.True(t, exists, "Choices should exist in response body")
			require.NotNil(t, choices, "Choices should not be nil")
		})
	})
}
