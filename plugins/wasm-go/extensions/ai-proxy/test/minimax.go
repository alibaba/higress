package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：Minimax Pro API + basePath removePrefix + original 协议
var minimaxProBasePathRemovePrefixOriginalConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "minimax",
			"apiTokens": []string{
				"sk-minimax-test",
			},
			"minimaxApiType":   "pro",
			"minimaxGroupId":   "test-group-id",
			"basePath":         "/minimax-api",
			"basePathHandling": "removePrefix",
			"protocol":         "original",
		},
	})
	return data
}()

// 测试配置：Minimax Pro API + basePath removePrefix + 默认协议（openai）
var minimaxProBasePathRemovePrefixOpenAIConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "minimax",
			"apiTokens": []string{
				"sk-minimax-openai",
			},
			"minimaxApiType":   "pro",
			"minimaxGroupId":   "test-group-id",
			"basePath":         "/minimax-api",
			"basePathHandling": "removePrefix",
		},
	})
	return data
}()

// 测试配置：Minimax V2 API + basePath removePrefix + original 协议
var minimaxV2BasePathRemovePrefixOriginalConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "minimax",
			"apiTokens": []string{
				"sk-minimax-v2",
			},
			"minimaxApiType":   "v2",
			"basePath":         "/minimax-v2",
			"basePathHandling": "removePrefix",
			"protocol":         "original",
		},
	})
	return data
}()

// RunMinimaxBasePathHandlingTests 测试 Minimax basePath 处理在不同协议下的行为
func RunMinimaxBasePathHandlingTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 核心用例：测试 Minimax Pro API + basePath removePrefix + original 协议
		// 重要：此测试验证在 handleRequestBodyByChatCompletionPro 阶段后 path 仍然保持正确
		// 之前的 bug 是 handleRequestBodyByChatCompletionPro 无条件覆盖 path，
		// 导致在 Body 阶段 path 被重新覆盖为 minimaxChatCompletionProPath
		t.Run("minimax pro basePath removePrefix with original protocol after body processing", func(t *testing.T) {
			host, status := test.NewTestHost(minimaxProBasePathRemovePrefixOriginalConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 模拟带有 basePath 前缀的请求
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/minimax-api/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 在 Headers 阶段后验证 path（此时 handleRequestHeaders 已执行）
			headersAfterHeaderStage := host.GetRequestHeaders()
			pathAfterHeaders, _ := test.GetHeaderValue(headersAfterHeaderStage, ":path")
			// Headers 阶段后，basePath 应该已被移除
			require.NotContains(t, pathAfterHeaders, "/minimax-api",
				"After headers stage: basePath should be removed")

			// 执行 Body 阶段（此时 handleRequestBodyByChatCompletionPro 会被调用）
			requestBody := `{"model": "abab5.5-chat", "messages": [{"role": "user", "content": "Hello"}]}`
			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 核心验证：在 Body 阶段后验证 path
			// 这是关键测试点：确保 handleRequestBodyByChatCompletionPro
			// 不会将 path 重新覆盖为 minimaxChatCompletionProPath
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			// basePath "/minimax-api" 不应该出现在最终路径中
			require.NotContains(t, pathValue, "/minimax-api",
				"After body stage: basePath should still be removed")
			// original 协议下，path 不应该被覆盖为 minimaxChatCompletionProPath
			require.NotContains(t, pathValue, "chatcompletion_pro",
				"With original protocol: path should not be overwritten to minimax pro path")
			// 路径应该是移除 basePath 后的结果
			require.Equal(t, "/v1/chat/completions", pathValue,
				"Path should be the original path without basePath after full request processing")

			// 验证 Host 被正确设置
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Equal(t, "api.minimax.chat", hostValue)

			// 验证 Authorization 被正确设置
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist")
			require.Equal(t, "Bearer sk-minimax-test", authValue)
		})

		// 测试 Minimax Pro API + basePath removePrefix + 默认协议（openai）
		// 在 openai 协议下，path 应该被覆盖为 minimaxChatCompletionProPath
		t.Run("minimax pro basePath removePrefix with openai protocol after body processing", func(t *testing.T) {
			host, status := test.NewTestHost(minimaxProBasePathRemovePrefixOpenAIConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 模拟带有 basePath 前缀的请求
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/minimax-api/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 执行 Body 阶段
			requestBody := `{"model": "abab5.5-chat", "messages": [{"role": "user", "content": "Hello"}]}`
			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 在 Body 阶段后验证请求头
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			// basePath "/minimax-api" 不应该出现在最终路径中
			require.NotContains(t, pathValue, "/minimax-api",
				"After body stage: basePath should be removed from path")
			// 在 openai 协议下，path 应该被覆盖为 minimaxChatCompletionProPath
			require.True(t, strings.Contains(pathValue, "chatcompletion_pro"),
				"With openai protocol: path should be overwritten to minimax pro path")
			require.Contains(t, pathValue, "GroupId=test-group-id",
				"Path should contain GroupId parameter")
		})

		// 测试 Minimax V2 API + basePath removePrefix + original 协议
		// V2 API 使用 handleRequestBody 而不是 handleRequestBodyByChatCompletionPro
		t.Run("minimax v2 basePath removePrefix with original protocol after body processing", func(t *testing.T) {
			host, status := test.NewTestHost(minimaxV2BasePathRemovePrefixOriginalConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 模拟带有 basePath 前缀的请求
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/minimax-v2/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 执行 Body 阶段
			requestBody := `{"model": "abab5.5-chat", "messages": [{"role": "user", "content": "Hello"}]}`
			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 在 Body 阶段后验证请求头
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			// basePath "/minimax-v2" 不应该出现在最终路径中
			require.NotContains(t, pathValue, "/minimax-v2",
				"After body stage: basePath should be removed from path")
			// 路径应该是移除 basePath 后的结果
			require.Equal(t, "/v1/chat/completions", pathValue,
				"Path should be the original path without basePath")
		})

		// 测试 original 协议下请求体保持原样
		t.Run("minimax pro original protocol preserves request body and path", func(t *testing.T) {
			host, status := test.NewTestHost(minimaxProBasePathRemovePrefixOriginalConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/minimax-api/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 设置请求体（包含自定义字段）
			requestBody := `{
				"model": "custom-model",
				"messages": [{"role": "user", "content": "Hello"}],
				"custom_field": "custom_value"
			}`
			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体被保持原样
			transformedBody := host.GetRequestBody()
			require.NotNil(t, transformedBody)

			var bodyMap map[string]interface{}
			err := json.Unmarshal(transformedBody, &bodyMap)
			require.NoError(t, err)

			// model 应该保持原样（original 协议不做模型映射）
			model, exists := bodyMap["model"]
			require.True(t, exists, "Model should exist")
			require.Equal(t, "custom-model", model, "Model should remain unchanged")

			// 自定义字段应该保持原样
			customField, exists := bodyMap["custom_field"]
			require.True(t, exists, "Custom field should exist")
			require.Equal(t, "custom_value", customField, "Custom field should remain unchanged")

			// 同时验证 path 在 Body 阶段后仍然正确
			requestHeaders := host.GetRequestHeaders()
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.NotContains(t, pathValue, "/minimax-api",
				"After body stage: basePath should be removed")
			require.Equal(t, "/v1/chat/completions", pathValue,
				"Path should be correct after body processing")
		})
	})
}
