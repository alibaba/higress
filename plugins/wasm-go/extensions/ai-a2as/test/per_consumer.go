package test

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// Per-Consumer配置测试
var perConsumerConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"behaviorCertificates": map[string]interface{}{
			"enabled":     true,
			"allowedTools": []string{"read_email", "search_documents"},
		},
		"inContextDefenses": map[string]interface{}{
			"enabled":  true,
			"template": "default",
		},
		"consumerConfigs": map[string]interface{}{
			"premium_user": map[string]interface{}{
				"behaviorCertificates": map[string]interface{}{
					"enabled":     true,
					"allowedTools": []string{"read_email", "send_email", "search_documents"},
				},
			},
			"basic_user": map[string]interface{}{
				"behaviorCertificates": map[string]interface{}{
					"enabled":     true,
					"allowedTools": []string{"read_email"},
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
				{"X-Mse-Consumer", "premium_user"},
			})

			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func RunPerConsumerOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("premium user - extended tool permissions", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Mse-Consumer", "premium_user"},
			})

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				],
				"tools": [
					{"type": "function", "function": {"name": "send_email"}}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			// Premium用户可以使用send_email
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("basic user - restricted tool permissions", func(t *testing.T) {
			host, status := test.NewTestHost(perConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Mse-Consumer", "basic_user"},
			})

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "test"}
				],
				"tools": [
					{"type": "function", "function": {"name": "send_email"}}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			// Basic用户不能使用send_email
			require.Equal(t, types.ActionPause, action)
		})
	})
}

