package test

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

var basicQiniuConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "qiniu",
			"apiTokens": []string{"qiniu-test-token-123"},
			"modelMapping": map[string]string{
				"*": "Qwen/Qwen2.5-7B-Instruct",
			},
		},
	})
	return data
}()

var invalidQiniuConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "qiniu",
			"apiTokens": []string{},
		},
	})
	return data
}()

func RunQiniuParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("basic qiniu config", func(t *testing.T) {
			host, status := test.NewTestHost(basicQiniuConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		t.Run("invalid qiniu config - missing apiToken", func(t *testing.T) {
			host, status := test.NewTestHost(invalidQiniuConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})
	})
}

func RunQiniuOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("qiniu chat completion request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicQiniuConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			requestHeaders := host.GetRequestHeaders()

			// 验证 Host 替换为七牛域名
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost)
			require.Equal(t, "api.qnaigc.com", hostValue)

			// 验证 Authorization
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth)
			require.Equal(t, "Bearer qiniu-test-token-123", authValue)

			// 验证 Path
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath)
			require.Equal(t, "/v1/chat/completions", pathValue)

			// 验证 Content-Length 被删除
			_, hasContentLength := test.GetHeaderValue(requestHeaders, "Content-Length")
			require.False(t, hasContentLength)
		})
	})
}

func RunQiniuOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("qiniu chat completion request body", func(t *testing.T) {
			host, status := test.NewTestHost(basicQiniuConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			requestBody := `{"model":"gpt-4","messages":[{"role":"user","content":"你好"}]}`
			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			processedBody := host.GetRequestBody()
			// 验证模型被映射
			require.Contains(t, string(processedBody), "Qwen/Qwen2.5-7B-Instruct")
		})
	})
}
