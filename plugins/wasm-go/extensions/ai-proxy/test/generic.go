package test

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 通用测试配置：最简配置，覆盖 host 与 token 注入。
var genericBasicConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":        "generic",
			"apiTokens":   []string{"sk-generic-basic"},
			"genericHost": "generic.backend.internal",
		},
	})
	return data
}()

// 通用测试配置：开启 basePath removePrefix。
var genericBasePathConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":             "generic",
			"apiTokens":        []string{"sk-generic-basepath"},
			"genericHost":      "basepath.backend.internal",
			"basePath":         "/proxy",
			"basePathHandling": "removePrefix",
		},
	})
	return data
}()

// 通用测试配置：开启 basePath prepend。
var genericPrependBasePathConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":             "generic",
			"apiTokens":        []string{"sk-generic-prepend"},
			"genericHost":      "prepend.backend.internal",
			"basePath":         "/custom",
			"basePathHandling": "prepend",
		},
	})
	return data
}()

// 通用测试配置：覆盖 firstByteTimeout，用于流式能力验证。
var genericStreamingConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":             "generic",
			"apiTokens":        []string{"sk-generic-stream"},
			"genericHost":      "stream.backend.internal",
			"firstByteTimeout": 1500,
		},
	})
	return data
}()

// 通用测试配置：无 token，也不设置 host。
var genericNoTokenConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "generic",
		},
	})
	return data
}()

func RunGenericParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("generic basic config", func(t *testing.T) {
			host, status := test.NewTestHost(genericBasicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		t.Run("generic config without token", func(t *testing.T) {
			host, status := test.NewTestHost(genericNoTokenConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
		})

		t.Run("generic config with streaming options", func(t *testing.T) {
			host, status := test.NewTestHost(genericStreamingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func RunGenericOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("generic injects token and custom host", func(t *testing.T) {
			host, status := test.NewTestHost(genericBasicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "client.local"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			requestHeaders := host.GetRequestHeaders()
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "generic.backend.internal"))
			require.True(t, test.HasHeaderWithValue(requestHeaders, "Authorization", "Bearer sk-generic-basic"))

			_, hasContentLength := test.GetHeaderValue(requestHeaders, "Content-Length")
			require.False(t, hasContentLength, "generic provider should remove Content-Length")
		})

		t.Run("generic removes basePath prefix", func(t *testing.T) {
			host, status := test.NewTestHost(genericBasePathConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "client.local"},
				{":path", "/proxy/service/echo"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			requestHeaders := host.GetRequestHeaders()
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":path", "/service/echo"))
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":authority", "basepath.backend.internal"))
		})

		t.Run("generic prepends basePath when configured", func(t *testing.T) {
			host, status := test.NewTestHost(genericPrependBasePathConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "client.local"},
				{":path", "/v1/echo"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			requestHeaders := host.GetRequestHeaders()
			require.True(t, test.HasHeaderWithValue(requestHeaders, ":path", "/custom/v1/echo"))
		})
	})
}

func RunGenericOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("generic stream true injects SSE headers", func(t *testing.T) {
			host, status := test.NewTestHost(genericStreamingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "client.local"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			body := `{"model":"gpt-any","stream":true}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionContinue, action)

			requestHeaders := host.GetRequestHeaders()
			require.True(t, test.HasHeaderWithValue(requestHeaders, "Accept", "text/event-stream"))
			require.True(t, test.HasHeaderWithValue(requestHeaders, "x-envoy-upstream-rq-first-byte-timeout-ms", "1500"))

			processedBody := host.GetRequestBody()
			require.JSONEq(t, body, string(processedBody))
		})

		t.Run("generic stream options trigger SSE headers", func(t *testing.T) {
			host, status := test.NewTestHost(genericStreamingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "client.local"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			body := `{"model":"gpt-any","stream_options":{"stream":true}}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionContinue, action)

			requestHeaders := host.GetRequestHeaders()
			require.True(t, test.HasHeaderWithValue(requestHeaders, "Accept", "text/event-stream"))
			require.True(t, test.HasHeaderWithValue(requestHeaders, "x-envoy-upstream-rq-first-byte-timeout-ms", "1500"))
		})

		t.Run("generic non streaming request keeps headers untouched", func(t *testing.T) {
			host, status := test.NewTestHost(genericStreamingConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "client.local"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			action := host.CallOnHttpRequestBody([]byte(`{"model":"gpt-any"}`))
			require.Equal(t, types.ActionContinue, action)

			requestHeaders := host.GetRequestHeaders()
			_, hasAccept := test.GetHeaderValue(requestHeaders, "Accept")
			require.False(t, hasAccept, "Accept header should remain untouched for non streaming requests")

			_, hasTimeout := test.GetHeaderValue(requestHeaders, "x-envoy-upstream-rq-first-byte-timeout-ms")
			require.False(t, hasTimeout, "timeout header should not be added when request is not streaming")
		})
	})
}
