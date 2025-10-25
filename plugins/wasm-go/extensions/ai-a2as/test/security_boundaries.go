package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 检查标签是否存在（处理 JSON 转义）
func containsTag(body, tag string) bool {
	unescaped := tag
	escaped := strings.ReplaceAll(strings.ReplaceAll(tag, "<", "\\u003c"), ">", "\\u003e")
	return strings.Contains(body, unescaped) || strings.Contains(body, escaped)
}

var basicSecurityBoundariesConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "openai",
		"securityBoundaries": map[string]interface{}{
			"enabled":          true,
			"wrapUserMessages": true,
			"wrapToolOutputs":  true,
			"wrapSystemMessages": false,
			"includeContentDigest": false,
		},
	})
	return data
}()

var securityBoundariesWithDigestConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "openai",
		"securityBoundaries": map[string]interface{}{
			"enabled":              true,
			"wrapUserMessages":     true,
			"wrapToolOutputs":      true,
			"wrapSystemMessages":   true,
			"includeContentDigest": true,
		},
	})
	return data
}()

func RunSecurityBoundariesParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("basic security boundaries config", func(t *testing.T) {
			host, status := test.NewTestHost(basicSecurityBoundariesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		t.Run("security boundaries with content digest", func(t *testing.T) {
			host, status := test.NewTestHost(securityBoundariesWithDigestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func RunSecurityBoundariesOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("basic request headers - no signature required", func(t *testing.T) {
			host, status := test.NewTestHost(basicSecurityBoundariesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func RunSecurityBoundariesOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("wrap user messages with security tags", func(t *testing.T) {
			host, status := test.NewTestHost(basicSecurityBoundariesConfig)
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

		modifiedBody := host.GetRequestBody()
		require.NotNil(t, modifiedBody)

		bodyStr := string(modifiedBody)
		require.True(t, containsTag(bodyStr, "<a2as:user>"))
		require.True(t, containsTag(bodyStr, "</a2as:user>"))
		})

		t.Run("wrap user messages with content digest", func(t *testing.T) {
			host, status := test.NewTestHost(securityBoundariesWithDigestConfig)
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

		modifiedBody := host.GetRequestBody()
		bodyStr := string(modifiedBody)

		// Check for content digest tag format <a2as:user:DIGEST>
		require.True(t, 
			containsTag(bodyStr, "<a2as:user:") || strings.Contains(bodyStr, "\\u003ca2as:user:"))
		})

		t.Run("handle multiple user messages", func(t *testing.T) {
			host, status := test.NewTestHost(basicSecurityBoundariesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "first"},
					{"role": "assistant", "content": "response"},
					{"role": "user", "content": "second"}
				]
			}`

			action := host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

		modifiedBody := host.GetRequestBody()
		bodyStr := string(modifiedBody)

		// Count both escaped and unescaped tags
		userTagCount := strings.Count(bodyStr, "<a2as:user>") + strings.Count(bodyStr, "\\u003ca2as:user\\u003e")
		require.Equal(t, 2, userTagCount)
		})
	})
}
