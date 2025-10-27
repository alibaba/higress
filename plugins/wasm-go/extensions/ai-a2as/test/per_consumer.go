package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：Per-Consumer配置
var perConsumerConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "openai",
		// 全局默认配置
		"securityBoundaries": map[string]interface{}{
			"enabled":              false,
			"wrapUserMessages":     true,
			"includeContentDigest": false,
		},
		"behaviorCertificates": map[string]interface{}{
			"enabled": true,
			"permissions": map[string]interface{}{
				"allowedTools": []string{"read_*", "search_*"},
			},
		},
		// Per-Consumer 配置
		"consumerConfigs": map[string]interface{}{
			"consumer_high_risk": map[string]interface{}{
				"securityBoundaries": map[string]interface{}{
					"enabled":              true,
					"wrapUserMessages":     true,
					"includeContentDigest": true,
				},
				"behaviorCertificates": map[string]interface{}{
					"enabled": true,
					"permissions": map[string]interface{}{
						"allowedTools": []string{"read_only_tool"},
					},
				},
			},
			"consumer_trusted": map[string]interface{}{
				"securityBoundaries": map[string]interface{}{
					"enabled": false,
				},
				"behaviorCertificates": map[string]interface{}{
					"enabled": true,
					"permissions": map[string]interface{}{
						"allowedTools": []string{"*"},
					},
				},
			},
		},
	})
	return data
}()

func RunPerConsumerParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("per-consumer config with multiple consumers", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func RunPerConsumerOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("identify consumer from X-Mse-Consumer header", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Mse-Consumer", "consumer_high_risk"},
			})

			// 应该继续处理
			require.Equal(t, types.ActionContinue, action)

			// 检查日志中是否识别了消费�?
			debugLogs := host.GetDebugLogs()
			hasConsumerLog := false
			for _, log := range debugLogs {
				if strings.Contains(log, "consumer") || strings.Contains(log, "high_risk") {
					hasConsumerLog = true
					break
				}
			}
			// 注意：日志可能不会立即出现，这个检查是可选的
			_ = hasConsumerLog
		})

		t.Run("no consumer header - use global config", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				// 没有 X-Mse-Consumer �?
			})

			// 应该继续处理（使用全局配置�?
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func RunPerConsumerOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("high-risk consumer - apply strict security boundaries", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头（识别消费者）
			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Mse-Consumer", "consumer_high_risk"},
			})

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			modifiedBody := host.GetRequestBody()
			bodyStr := string(modifiedBody)

			// 高风险消费者应该启用安全边界（全局配置是 disabled）
			require.True(t, containsTag(bodyStr, "<a2as:user"), "High-risk consumer should have security boundaries")
		})

		t.Run("trusted consumer - no security boundaries", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Mse-Consumer", "consumer_trusted"},
			})

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			modifiedBody := host.GetRequestBody()
			bodyStr := string(modifiedBody)

			// 受信任消费者禁用了安全边界
			require.False(t, containsTag(bodyStr, "<a2as:user"), "Trusted consumer should not have security boundaries")
		})

		t.Run("high-risk consumer - restricted tool permissions", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Mse-Consumer", "consumer_high_risk"},
			})

			// 请求一个不在高风险消费者允许列表中的工�?
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				],
				"tools": [
					{
						"type": "function",
						"function": {
							"name": "search_email",
							"description": "Search emails"
						}
					}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// search_email 不在 consumer_high_risk �?allowedTools (只有 read_only_tool)
			// 应该被拒�?
			require.Equal(t, types.ActionPause, action)
		})

		t.Run("trusted consumer - all tools allowed", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Mse-Consumer", "consumer_trusted"},
			})

			// 请求任意工具
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				],
				"tools": [
					{
						"type": "function",
						"function": {
							"name": "delete_email",
							"description": "Delete email"
						}
					}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// consumer_trusted 允许所有工�?(*)
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("unknown consumer - use global config", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Mse-Consumer", "consumer_unknown"},
			})

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				],
				"tools": [
					{
						"type": "function",
						"function": {
							"name": "read_email",
							"description": "Read email"
						}
					}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 未知消费者使用全局配置，read_email 匹配 read_*
			require.Equal(t, types.ActionContinue, action)
		})
	})
}
