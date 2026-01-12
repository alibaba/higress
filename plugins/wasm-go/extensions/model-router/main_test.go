package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Basic configs for wasm test host
var (
	basicConfig = func() json.RawMessage {
		data, _ := json.Marshal(map[string]interface{}{
			"modelKey":          "model",
			"addProviderHeader": "x-provider",
			"modelToHeader":     "x-model",
			"enableOnPathSuffix": []string{
				"/v1/chat/completions",
			},
		})
		return data
	}()

	defaultSuffixConfig = func() json.RawMessage {
		data, _ := json.Marshal(map[string]interface{}{
			"modelKey":          "model",
			"addProviderHeader": "x-provider",
			"modelToHeader":     "x-model",
		})
		return data
	}()
)

func getHeader(headers [][2]string, key string) (string, bool) {
	for _, h := range headers {
		if strings.EqualFold(h[0], key) {
			return h[1], true
		}
	}
	return "", false
}

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("basic config with defaults", func(t *testing.T) {
			var cfg ModelRouterConfig
			err := parseConfig(gjson.ParseBytes(defaultSuffixConfig), &cfg)
			require.NoError(t, err)

			// default modelKey
			require.Equal(t, "model", cfg.modelKey)
			// headers
			require.Equal(t, "x-provider", cfg.addProviderHeader)
			require.Equal(t, "x-model", cfg.modelToHeader)
			// default enabled path suffixes should contain common openai paths
			require.Contains(t, cfg.enableOnPathSuffix, "/completions")
			require.Contains(t, cfg.enableOnPathSuffix, "/embeddings")
		})

		t.Run("custom enableOnPathSuffix", func(t *testing.T) {
			jsonData := []byte(`{
				"modelKey": "my_model",
				"addProviderHeader": "x-prov",
				"modelToHeader": "x-mod",
				"enableOnPathSuffix": ["/foo", "/bar"]
			}`)
			var cfg ModelRouterConfig
			err := parseConfig(gjson.ParseBytes(jsonData), &cfg)
			require.NoError(t, err)

			require.Equal(t, "my_model", cfg.modelKey)
			require.Equal(t, "x-prov", cfg.addProviderHeader)
			require.Equal(t, "x-mod", cfg.modelToHeader)
			require.Equal(t, []string{"/foo", "/bar"}, cfg.enableOnPathSuffix)
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("skip when path not matched", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			originalHeaders := [][2]string{
				{":authority", "example.com"},
				{":path", "/v1/other"},
				{":method", "POST"},
				{"content-type", "application/json"},
				{"content-length", "123"},
			}
			action := host.CallOnHttpRequestHeaders(originalHeaders)
			require.Equal(t, types.ActionContinue, action)

			newHeaders := host.GetRequestHeaders()
			_, found := getHeader(newHeaders, "content-length")
			require.True(t, found, "content-length should be kept when path not enabled")
		})

		t.Run("process when path and content-type match", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			originalHeaders := [][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
				{"content-length", "123"},
			}
			action := host.CallOnHttpRequestHeaders(originalHeaders)
			require.Equal(t, types.HeaderStopIteration, action)

			newHeaders := host.GetRequestHeaders()
			_, found := getHeader(newHeaders, "content-length")
			require.False(t, found, "content-length should be removed when buffering body")
		})

		t.Run("do not process for unsupported content-type", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			originalHeaders := [][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "text/plain"},
				{"content-length", "123"},
			}
			action := host.CallOnHttpRequestHeaders(originalHeaders)
			require.Equal(t, types.ActionContinue, action)

			newHeaders := host.GetRequestHeaders()
			_, found := getHeader(newHeaders, "content-length")
			require.True(t, found, "content-length should not be removed for unsupported content-type")
		})
	})
}

func TestOnHttpRequestBody_JSON(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("set headers and rewrite model when provider/model format", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			origBody := []byte(`{
				"model": "openai/gpt-4o",
				"messages": [{"role": "user", "content": "hello"}]
			}`)
			action := host.CallOnHttpRequestBody(origBody)
			require.Equal(t, types.ActionContinue, action)

			processed := host.GetRequestBody()
			require.NotNil(t, processed)
			// model should be rewritten to only the model part
			require.Equal(t, "gpt-4o", gjson.GetBytes(processed, "model").String())

			headers := host.GetRequestHeaders()
			hv, found := getHeader(headers, "x-model")
			require.True(t, found)
			require.Equal(t, "openai/gpt-4o", hv)
			pv, found := getHeader(headers, "x-provider")
			require.True(t, found)
			require.Equal(t, "openai", pv)
		})

		t.Run("no change when model not provided", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			origBody := []byte(`{
				"messages": [{"role": "user", "content": "hello"}]
			}`)
			action := host.CallOnHttpRequestBody(origBody)
			require.Equal(t, types.ActionContinue, action)

			processed := host.GetRequestBody()
			// body should remain nil or unchanged as plugin does nothing
			if processed != nil {
				require.JSONEq(t, string(origBody), string(processed))
			}
			_, found := getHeader(host.GetRequestHeaders(), "x-provider")
			require.False(t, found)
		})
	})
}

func TestOnHttpRequestBody_Multipart(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(basicConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// model field
		modelWriter, err := writer.CreateFormField("model")
		require.NoError(t, err)
		_, err = modelWriter.Write([]byte("openai/gpt-4o"))
		require.NoError(t, err)

		// another field to ensure others are preserved
		fileWriter, err := writer.CreateFormField("prompt")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte("hello"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		contentType := "multipart/form-data; boundary=" + writer.Boundary()

		host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"},
			{":path", "/v1/chat/completions"},
			{":method", "POST"},
			{"content-type", contentType},
		})

		action := host.CallOnHttpRequestBody(buf.Bytes())
		require.Equal(t, types.ActionContinue, action)

		processed := host.GetRequestBody()
		require.NotNil(t, processed)

		// Parse multipart body again to verify fields
		reader := multipart.NewReader(bytes.NewReader(processed), writer.Boundary())

		foundModel := false
		foundPrompt := false
		for {
			part, err := reader.NextPart()
			if err != nil {
				break
			}
			name := part.FormName()
			data, err := io.ReadAll(part)
			require.NoError(t, err)

			switch name {
			case "model":
				foundModel = true
				require.Equal(t, "gpt-4o", string(data))
			case "prompt":
				foundPrompt = true
				require.Equal(t, "hello", string(data))
			}
		}

		require.True(t, foundModel)
		require.True(t, foundPrompt)

		headers := host.GetRequestHeaders()
		hv, found := getHeader(headers, "x-model")
		require.True(t, found)
		require.Equal(t, "openai/gpt-4o", hv)
		pv, found := getHeader(headers, "x-provider")
		require.True(t, found)
		require.Equal(t, "openai", pv)
	})
}
