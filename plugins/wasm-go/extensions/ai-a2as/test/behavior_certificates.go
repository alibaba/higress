package test

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 基本行为证书配置
var basicBehaviorCertificatesConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"behaviorCertificates": map[string]interface{}{
			"enabled":     true,
			"allowedTools": []string{"read_email", "search_documents"},
			"denyMessage": "Tool not permitted",
		},
	})
	return data
}()

// 空白名单配置（拒绝所有）
var emptyWhitelistConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"behaviorCertificates": map[string]interface{}{
			"enabled":     true,
			"allowedTools": []string{},
		},
	})
	return data
}()

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
	})
}

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
							"name": "read_email",
							"description": "Read an email message"
						}
					}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
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
							"name": "delete_file",
							"description": "Delete a file"
						}
					}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
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
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("empty whitelist - deny all tools", func(t *testing.T) {
			host, status := test.NewTestHost(emptyWhitelistConfig)
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
							"name": "any_tool"
						}
					}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionPause, action)
		})
	})
}

