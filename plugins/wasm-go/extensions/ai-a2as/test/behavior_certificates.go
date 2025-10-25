package test

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本行为证书配置
var basicBehaviorCertificatesConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "openai",
		"behaviorCertificates": map[string]interface{}{
			"enabled": true,
			"permissions": map[string]interface{}{
				"allowedTools": []string{"email.read_message", "email.search"},
				"deniedTools":  []string{"email.send_message", "email.delete_message"},
			},
			"denyMessage": "This operation is not permitted",
		},
	})
	return data
}()

// 测试配置：通配符行为证书配置
var wildcardBehaviorCertificatesConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "openai",
		"behaviorCertificates": map[string]interface{}{
			"enabled": true,
			"permissions": map[string]interface{}{
				"allowedTools": []string{"read_*", "search_*"},
				"deniedTools":  []string{"delete_*", "write_*"},
			},
		},
	})
	return data
}()

// Runbehavior certificates tests
func RunBehaviorCertificatesParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("basic behavior certificates config", func(t *testing.T) {
			host, status := test.NewTestHost(basicBehaviorCertificatesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		t.Run("wildcard behavior certificates config", func(t *testing.T) {
			host, status := test.NewTestHost(wildcardBehaviorCertificatesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

// Runbehavior certificates tests
func RunBehaviorCertificatesOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("allowed tool - should pass", func(t *testing.T) {
			host, status := test.NewTestHost(basicBehaviorCertificatesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				],
				"tools": [
					{
						"type": "function",
						"function": {
							"name": "email.read_message",
							"description": "Read an email message"
						}
					}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// email.read_message �?allowedTools 中，应该允许
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("denied tool - should reject", func(t *testing.T) {
			host, status := test.NewTestHost(basicBehaviorCertificatesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				],
				"tools": [
					{
						"type": "function",
						"function": {
							"name": "email.send_message",
							"description": "Send an email"
						}
					}
				]
			}`

		action := host.CallOnHttpRequestBody([]byte(requestBody))

		// email.send_message 在 deniedTools 中，应该被拒绝
		require.Equal(t, types.ActionPause, action)
		})

		t.Run("wildcard allowed tool - should pass", func(t *testing.T) {
			host, status := test.NewTestHost(wildcardBehaviorCertificatesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

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

			// read_email 匹配 read_* 通配符，应该允许
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("wildcard denied tool - should reject", func(t *testing.T) {
			host, status := test.NewTestHost(wildcardBehaviorCertificatesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

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

			// delete_email 匹配 delete_* 通配符，应该被拒�?
			require.Equal(t, types.ActionPause, action)
		})

		t.Run("multiple tools - one denied - should reject", func(t *testing.T) {
			host, status := test.NewTestHost(basicBehaviorCertificatesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				],
				"tools": [
					{
						"type": "function",
						"function": {
							"name": "email.read_message",
							"description": "Read email"
						}
					},
					{
						"type": "function",
						"function": {
							"name": "email.send_message",
							"description": "Send email"
						}
					}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 即使有一个工具被拒绝，整个请求也应该被拒�?
			require.Equal(t, types.ActionPause, action)
		})

		t.Run("no tools - should pass", func(t *testing.T) {
			host, status := test.NewTestHost(basicBehaviorCertificatesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 没有工具调用，应该允�?
			require.Equal(t, types.ActionContinue, action)
		})
	})
}
