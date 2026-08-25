package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func getHeader(headers [][2]string, key string) (string, bool) {
	for _, h := range headers {
		if strings.EqualFold(h[0], key) {
			return h[1], true
		}
	}
	return "", false
}

// Basic configs for wasm test host
var (
	basicConfig = func() json.RawMessage {
		data, _ := json.Marshal(map[string]interface{}{
			"modelKey": "model",
			"modelMapping": map[string]string{
				"gpt-3.5-turbo": "gpt-4",
			},
			"enableOnPathSuffix": []string{
				"/v1/chat/completions",
			},
		})
		return data
	}()

	customConfig = func() json.RawMessage {
		data, _ := json.Marshal(map[string]interface{}{
			"modelKey": "request.model",
			"modelMapping": map[string]string{
				"*":          "gpt-4o",
				"gpt-3.5*":   "gpt-4-mini",
				"gpt-3.5-t":  "gpt-4-turbo",
				"gpt-3.5-t1": "gpt-4-turbo-1",
			},
			"enableOnPathSuffix": []string{
				"/v1/chat/completions",
				"/v1/embeddings",
			},
		})
		return data
	}()

	headerSyncConfig = func() json.RawMessage {
		data, _ := json.Marshal(map[string]interface{}{
			"modelKey": "model",
			"modelMapping": map[string]string{
				"gpt-3.5-turbo": "gpt-4",
			},
			"modelToHeader": "x-final-model",
			"enableOnPathSuffix": []string{
				"/v1/chat/completions",
			},
		})
		return data
	}()
)

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("basic config with defaults", func(t *testing.T) {
			var cfg Config
			jsonData := []byte(`{
				"modelMapping": {
					"gpt-3.5-turbo": "gpt-4",
					"gpt-4*": "gpt-4o-mini",
					"*": "gpt-4o"
				}
			}`)
			err := parseConfig(gjson.ParseBytes(jsonData), &cfg)
			require.NoError(t, err)

			// default modelKey
			require.Equal(t, "model", cfg.modelKey)
			// exact mapping
			require.Equal(t, "gpt-4", cfg.exactModelMapping["gpt-3.5-turbo"])
			// prefix mapping
			require.Len(t, cfg.prefixModelMapping, 1)
			require.Equal(t, "gpt-4", cfg.prefixModelMapping[0].Prefix)
			// default model
			require.Equal(t, "gpt-4o", cfg.defaultModel)
			// default enabled path suffixes
			require.Contains(t, cfg.enableOnPathSuffix, "/completions")
			require.Contains(t, cfg.enableOnPathSuffix, "/embeddings")
		})

		t.Run("custom modelKey and enableOnPathSuffix", func(t *testing.T) {
			var cfg Config
			jsonData := []byte(`{
				"modelKey": "request.model",
				"modelMapping": {
					"gpt-3.5-turbo": "gpt-4",
					"gpt-3.5*": "gpt-4-mini"
				},
				"enableOnPathSuffix": ["/v1/chat/completions", "/v1/embeddings"]
			}`)
			err := parseConfig(gjson.ParseBytes(jsonData), &cfg)
			require.NoError(t, err)

			require.Equal(t, "request.model", cfg.modelKey)
			require.Equal(t, "gpt-4", cfg.exactModelMapping["gpt-3.5-turbo"])
			require.Len(t, cfg.prefixModelMapping, 1)
			require.Equal(t, "gpt-3.5", cfg.prefixModelMapping[0].Prefix)
			require.Equal(t, "gpt-4-mini", cfg.prefixModelMapping[0].Target)
			require.Equal(t, 2, len(cfg.enableOnPathSuffix))
			require.Contains(t, cfg.enableOnPathSuffix, "/v1/chat/completions")
			require.Contains(t, cfg.enableOnPathSuffix, "/v1/embeddings")
		})

		t.Run("modelMapping must be object", func(t *testing.T) {
			var cfg Config
			jsonData := []byte(`{
				"modelMapping": "invalid"
			}`)
			err := parseConfig(gjson.ParseBytes(jsonData), &cfg)
			require.Error(t, err)
		})

		t.Run("enableOnPathSuffix must be array", func(t *testing.T) {
			var cfg Config
			jsonData := []byte(`{
				"enableOnPathSuffix": "not-array"
			}`)
			err := parseConfig(gjson.ParseBytes(jsonData), &cfg)
			require.Error(t, err)
		})

		t.Run("modelToHeader default and custom", func(t *testing.T) {
			var cfgDefault Config
			require.NoError(t, parseConfig(gjson.ParseBytes([]byte(`{"modelMapping":{}}`)), &cfgDefault))
			require.Equal(t, "x-higress-llm-model-final", cfgDefault.modelToHeader)

			var cfgCustom Config
			err := parseConfig(gjson.ParseBytes([]byte(`{
				"modelToHeader": "x-my-model",
				"modelMapping": {}
			}`)), &cfgCustom)
			require.NoError(t, err)
			require.Equal(t, "x-my-model", cfgCustom.modelToHeader)
		})

		t.Run("empty modelMapping", func(t *testing.T) {
			var cfg Config
			err := parseConfig(gjson.ParseBytes([]byte(`{"modelMapping": {}}`)), &cfg)
			require.NoError(t, err)
			require.Empty(t, cfg.exactModelMapping)
			require.Empty(t, cfg.prefixModelMapping)
			require.Equal(t, "", cfg.defaultModel)
		})

		t.Run("prefix rules sorted by key for stable iteration", func(t *testing.T) {
			var cfg Config
			// Object key order in JSON is z then a; after sort, prefix "a" is tried before "z".
			jsonData := []byte(`{
				"modelMapping": {
					"z*": "Z",
					"a*": "A"
				}
			}`)
			require.NoError(t, parseConfig(gjson.ParseBytes(jsonData), &cfg))
			require.Len(t, cfg.prefixModelMapping, 2)
			require.Equal(t, "a", cfg.prefixModelMapping[0].Prefix)
			require.Equal(t, "A", cfg.prefixModelMapping[0].Target)
			require.Equal(t, "z", cfg.prefixModelMapping[1].Prefix)
			require.Equal(t, "Z", cfg.prefixModelMapping[1].Target)
		})

		t.Run("exact mapping wins over prefix", func(t *testing.T) {
			var cfg Config
			jsonData := []byte(`{
				"modelKey": "model",
				"modelMapping": {
					"gpt-3.5*": "from-prefix",
					"gpt-3.5-turbo": "from-exact"
				}
			}`)
			require.NoError(t, parseConfig(gjson.ParseBytes(jsonData), &cfg))
			require.Equal(t, "from-exact", cfg.exactModelMapping["gpt-3.5-turbo"])
			require.Len(t, cfg.prefixModelMapping, 1)
			require.Equal(t, "gpt-3.5", cfg.prefixModelMapping[0].Prefix)
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
			_, foundContentLength := getHeader(newHeaders, "content-length")
			require.True(t, foundContentLength, "content-length should be kept when path is not enabled")
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
			_, foundCL := getHeader(newHeaders, "content-length")
			require.False(t, foundCL, "content-length should be removed when buffering body")
		})

		t.Run("path with query string still matches suffix", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions?trace=1"},
				{":method", "POST"},
				{"content-type", "application/json"},
				{"content-length", "99"},
			})
			require.Equal(t, types.HeaderStopIteration, action)
			_, foundCL := getHeader(host.GetRequestHeaders(), "content-length")
			require.False(t, foundCL)
		})
	})
}

func TestOnHttpRequestBody_ModelMapping(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("exact mapping", func(t *testing.T) {
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
				"model": "gpt-3.5-turbo",
				"messages": [{"role": "user", "content": "hello"}]
			}`)
			action := host.CallOnHttpRequestBody(origBody)
			require.Equal(t, types.ActionContinue, action)

			processed := host.GetRequestBody()
			require.NotNil(t, processed)
			require.Equal(t, "gpt-4", gjson.GetBytes(processed, "model").String())
			v, ok := getHeader(host.GetRequestHeaders(), "x-higress-llm-model-final")
			require.True(t, ok)
			require.Equal(t, "gpt-4", v)
		})

		t.Run("default model when key missing", func(t *testing.T) {
			// use customConfig where default model is set with "*"
			host, status := test.NewTestHost(customConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			origBody := []byte(`{
				"request": {
					"messages": [{"role": "user", "content": "hello"}]
				}
			}`)
			action := host.CallOnHttpRequestBody(origBody)
			require.Equal(t, types.ActionContinue, action)

			processed := host.GetRequestBody()
			require.NotNil(t, processed)
			// default model should be set at request.model
			require.Equal(t, "gpt-4o", gjson.GetBytes(processed, "request.model").String())
			v, ok := getHeader(host.GetRequestHeaders(), "x-higress-llm-model-final")
			require.True(t, ok)
			require.Equal(t, "gpt-4o", v)
		})

		t.Run("prefix mapping takes effect", func(t *testing.T) {
			host, status := test.NewTestHost(customConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			origBody := []byte(`{
				"request": {
					"model": "gpt-3.5-turbo-16k",
					"messages": [{"role": "user", "content": "hello"}]
				}
			}`)
			action := host.CallOnHttpRequestBody(origBody)
			require.Equal(t, types.ActionContinue, action)

			processed := host.GetRequestBody()
			require.NotNil(t, processed)
			require.Equal(t, "gpt-4-mini", gjson.GetBytes(processed, "request.model").String())
			v, ok := getHeader(host.GetRequestHeaders(), "x-higress-llm-model-final")
			require.True(t, ok)
			require.Equal(t, "gpt-4-mini", v)
		})

		t.Run("exact mapping beats prefix for same family", func(t *testing.T) {
			host, status := test.NewTestHost(customConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/embeddings"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			origBody := []byte(`{
				"request": {
					"model": "gpt-3.5-t1",
					"input": "hello"
				}
			}`)
			action := host.CallOnHttpRequestBody(origBody)
			require.Equal(t, types.ActionContinue, action)

			processed := host.GetRequestBody()
			require.NotNil(t, processed)
			require.Equal(t, "gpt-4-turbo-1", gjson.GetBytes(processed, "request.model").String())
			v, ok := getHeader(host.GetRequestHeaders(), "x-higress-llm-model-final")
			require.True(t, ok)
			require.Equal(t, "gpt-4-turbo-1", v)
		})

		t.Run("empty request body is a no-op", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			action := host.CallOnHttpRequestBody(nil)
			require.Equal(t, types.ActionContinue, action)
			require.Nil(t, host.GetRequestBody())

			action = host.CallOnHttpRequestBody([]byte{})
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("invalid json body is skipped", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
				{"x-higress-llm-model-final", "should-not-change"},
			})

			bad := []byte(`not json`)
			action := host.CallOnHttpRequestBody(bad)
			require.Equal(t, types.ActionContinue, action)
			out := host.GetRequestBody()
			if out != nil {
				require.Equal(t, string(bad), string(out))
			}
			v, ok := getHeader(host.GetRequestHeaders(), "x-higress-llm-model-final")
			require.True(t, ok)
			require.Equal(t, "should-not-change", v, "invalid JSON must not refresh model header")
		})

		t.Run("no body rewrite when already mapped target but header still refreshed", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			origBody := []byte(`{"model":"gpt-4","messages":[]}`)
			action := host.CallOnHttpRequestBody(origBody)
			require.Equal(t, types.ActionContinue, action)
			out := host.GetRequestBody()
			if out != nil {
				require.Equal(t, string(origBody), string(out))
			}
			v, ok := getHeader(host.GetRequestHeaders(), "x-higress-llm-model-final")
			require.True(t, ok)
			require.Equal(t, "gpt-4", v)
		})

		t.Run("modelToHeader always set to resolved model", func(t *testing.T) {
			host, status := test.NewTestHost(headerSyncConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
				{"x-final-model", "gpt-3.5-turbo"},
			})

			origBody := []byte(`{"model":"gpt-3.5-turbo"}`)
			require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody(origBody))

			processed := host.GetRequestBody()
			require.NotNil(t, processed)
			require.Equal(t, "gpt-4", gjson.GetBytes(processed, "model").String())

			v, ok := getHeader(host.GetRequestHeaders(), "x-final-model")
			require.True(t, ok)
			require.Equal(t, "gpt-4", v)
		})

		t.Run("modelToHeader refreshed even when it already matches resolved model", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-type", "application/json"},
				{"x-higress-llm-model-final", "gpt-4"},
			})

			origBody := []byte(`{"model":"gpt-3.5-turbo","messages":[]}`)
			require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody(origBody))

			processed := host.GetRequestBody()
			require.NotNil(t, processed)
			require.Equal(t, "gpt-4", gjson.GetBytes(processed, "model").String())
			v, ok := getHeader(host.GetRequestHeaders(), "x-higress-llm-model-final")
			require.True(t, ok)
			require.Equal(t, "gpt-4", v)
		})
	})
}

// strictRuntimeIdentityTarget 构造含完整唯一 Service/Cluster 闭包的测试 ModelCard target。
// 输入约束：参数均为控制面冻结字段，函数只供本文件的 strict fixture 使用。
// 输出语义：返回与 C1 下发 JSON 形状一致的完整 target，确保测试不会退化为仅 Provider/model 校验。
// 边界场景：Service/Cluster 从 ModelCardID 派生仅是测试数据，生产选择始终来自 APIGO projection。
func strictRuntimeIdentityTarget(modelCardID, provider, upstreamModelName string) runtimeIdentityTarget {
	return runtimeIdentityTarget{
		ModelCardID:       modelCardID,
		Provider:          provider,
		UpstreamModelName: upstreamModelName,
		ServiceID:         "service-" + modelCardID,
		TargetCluster:     "outbound|443||" + modelCardID + ".internal",
	}
}

// strictRuntimeIdentityMapperConfig 构造一条由控制面冻结的 mapper/fallback 严格规则。
// 该 fixture 同时保留源卡和目标卡于 closure，确保测试覆盖跨卡与同卡两种 transition 语义。
func strictRuntimeIdentityMapperConfig(t *testing.T, targetKey string, target runtimeIdentityTarget, mappingTarget string) []byte {
	t.Helper()
	return strictRuntimeIdentityMapperConfigWithClosure(
		t,
		targetKey,
		target,
		map[string]string{"source-model": mappingTarget},
		map[string]runtimeIdentityTarget{
			"card-a":           strictRuntimeIdentityTarget("card-a", "provider-a", "source-model"),
			target.ModelCardID: target,
		},
	)
}

// strictRuntimeIdentityMapperConfigWithClosure 构造可指定映射和完整冻结闭包的 strict mapper 配置。
// 输入约束：mappings 的每个最终模型必须与 target.UpstreamModelName 一致；closure 要覆盖当前与目标卡。
// 输出语义：返回可直接用于 Proxy-Wasm component host 的同一 revision 配置。
// 边界情况：该 helper 只服务组件级序列断言，真实 Envoy redirect 重入仍由独立集成验证承担。
func strictRuntimeIdentityMapperConfigWithClosure(
	t *testing.T,
	targetKey string,
	target runtimeIdentityTarget,
	mappings map[string]string,
	closure map[string]runtimeIdentityTarget,
) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"modelKey":      "model",
		"modelToHeader": "x-higress-llm-model-final",
		"modelMapping":  mappings,
		"modelRuntimeIdentity": map[string]any{
			"mode":           runtimeIdentityMode,
			"configRevision": "revision-1",
			"scope": map[string]any{
				"gatewayId": "gw-1", "apiId": "api-1", "routeId": "route-1", "dataPlaneRouteName": "route-name-1",
			},
			"parser":                map[string]any{"source": runtimeIdentityJSONBody, "modelKey": "model"},
			"reservedAutoSelectors": []string{"auto", "auto/*"},
			"targetClosure":         closure,
		},
		"modelRuntimeIdentityTarget":    target,
		"modelRuntimeIdentityTargetKey": targetKey,
	})
	require.NoError(t, err)
	return raw
}

// TestRuntimeIdentityMapperDefersReservedAuto 验证 attached ModelMapper 在 Auto 最终候选 writer 之前保持完全无副作用。
func TestRuntimeIdentityMapperDefersReservedAuto(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		target := strictRuntimeIdentityTarget("card-b", "provider-b", "target-model")
		host, status := test.NewTestHost(strictRuntimeIdentityMapperConfigWithClosure(
			t,
			"mapper:policy-1:0",
			target,
			map[string]string{"*": "target-model", "auto": "", "auto/*": ""},
			map[string]runtimeIdentityTarget{
				"card-a": strictRuntimeIdentityTarget("card-a", "provider-a", "source-model"),
				"card-b": target,
			},
		))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		headers := [][2]string{
			{":authority", "example.com"}, {":path", "/v1/chat/completions"}, {":method", "POST"},
			{"content-type", "application/json"}, {"content-length", "42"}, {"x-higress-llm-model-final", "unchanged"},
		}
		require.Equal(t, types.HeaderStopIteration, host.CallOnHttpRequestHeaders(headers))
		body := []byte(`{"model":"auto/provider-a"}`)
		require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody(body))
		require.JSONEq(t, string(body), string(host.GetRequestBody()))
		value, found := getHeader(host.GetRequestHeaders(), "x-higress-llm-model-final")
		require.True(t, found)
		require.Equal(t, "unchanged", value)
		_, found = getHeader(host.GetRequestHeaders(), "content-length")
		require.True(t, found)
		_, err := host.GetProperty([]string{resolvedModelContextProperty})
		require.Error(t, err)
	})
}

// setStrictMapperContext 写入仅能由上游 Direct/Auto writer 建立的 request-lifespan 身份。
func setStrictMapperContext(t *testing.T, host test.TestHost, modelCardID, provider, upstreamModel string, sequence int64, revision string) {
	t.Helper()
	raw, err := json.Marshal(resolvedModelContext{
		SchemaVersion:       resolvedModelContextSchemaVersion,
		GatewayID:           "gw-1",
		APIID:               "api-1",
		RouteID:             "route-1",
		ConfigRevision:      revision,
		ResolvedModelCardID: modelCardID,
		TransitionSeq:       sequence,
		Source:              "direct",
	})
	require.NoError(t, err)
	require.NoError(t, host.SetProperty([]string{resolvedModelContextProperty}, raw))
}

// TestRuntimeIdentityMapperTransition 验证 mapper/fallback 只在完整改写成功后以目标卡提交下一代 context。
func TestRuntimeIdentityMapperTransition(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		for _, testCase := range []struct {
			name      string
			targetKey string
			source    string
		}{
			{name: "mapper cross card", targetKey: "mapper:policy-1:0", source: "mapper"},
			{name: "fallback cross card", targetKey: "fallback:route-1:0", source: "fallback"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				target := strictRuntimeIdentityTarget("card-b", "provider-b", "target-model")
				host, status := test.NewTestHost(strictRuntimeIdentityMapperConfig(t, testCase.targetKey, target, target.UpstreamModelName))
				defer host.Reset()
				require.Equal(t, types.OnPluginStartStatusOK, status)
				setStrictMapperContext(t, host, "card-a", "provider-a", "source-model", 1, "revision-1")

				require.Equal(t, types.HeaderStopIteration, host.CallOnHttpRequestHeaders([][2]string{
					{":authority", "example.com"}, {":path", "/v1/chat/completions"}, {":method", "POST"}, {"content-type", "application/json"}, {"content-length", "99"}, {runtimeIdentityTargetClusterHeader, "client-spoofed-cluster"},
				}))
				require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody([]byte(`{"model":"source-model"}`)))
				require.Equal(t, "target-model", gjson.GetBytes(host.GetRequestBody(), "model").String())
				raw, err := host.GetProperty([]string{resolvedModelContextProperty})
				require.NoError(t, err)
				var context resolvedModelContext
				require.NoError(t, json.Unmarshal(raw, &context))
				require.Equal(t, resolvedModelContextSchemaVersion, int(gjson.GetBytes(raw, "schemaVersion").Int()))
				require.Equal(t, "route-1", gjson.GetBytes(raw, "routeId").String())
				require.False(t, gjson.GetBytes(raw, "scope").Exists())
				require.Equal(t, "card-b", context.ResolvedModelCardID)
				require.Equal(t, int64(2), context.TransitionSeq)
				require.Equal(t, testCase.source, context.Source)
				clusterHeader, found := getHeader(host.GetRequestHeaders(), runtimeIdentityTargetClusterHeader)
				require.True(t, found)
				require.Equal(t, target.TargetCluster, clusterHeader)
			})
		}
	})
}

// TestRuntimeIdentityMapperSameCardAndConflict 验证同卡表示改写不伪造 generation，缺失或混 revision context 一律拒绝。
func TestRuntimeIdentityMapperSameCardAndConflict(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("same card keeps generation", func(t *testing.T) {
			target := strictRuntimeIdentityTarget("card-a", "provider-a", "source-model")
			host, status := test.NewTestHost(strictRuntimeIdentityMapperConfig(t, "mapper:policy-1:0", target, target.UpstreamModelName))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			setStrictMapperContext(t, host, "card-a", "provider-a", "source-model", 4, "revision-1")
			host.CallOnHttpRequestHeaders([][2]string{{":authority", "example.com"}, {":path", "/v1/chat/completions"}, {":method", "POST"}, {"content-type", "application/json"}, {"content-length", "99"}})
			require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody([]byte(`{"model":"source-model"}`)))
			raw, err := host.GetProperty([]string{resolvedModelContextProperty})
			require.NoError(t, err)
			var context resolvedModelContext
			require.NoError(t, json.Unmarshal(raw, &context))
			require.Equal(t, int64(4), context.TransitionSeq)
			require.Equal(t, "direct", context.Source)
		})

		for _, testCase := range []struct {
			name        string
			withContext bool
			revision    string
		}{
			{name: "missing context"},
			{name: "mixed revision", withContext: true, revision: "revision-old"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				target := strictRuntimeIdentityTarget("card-b", "provider-b", "target-model")
				host, status := test.NewTestHost(strictRuntimeIdentityMapperConfig(t, "mapper:policy-1:0", target, target.UpstreamModelName))
				defer host.Reset()
				require.Equal(t, types.OnPluginStartStatusOK, status)
				if testCase.withContext {
					setStrictMapperContext(t, host, "card-a", "provider-a", "source-model", 1, testCase.revision)
				}
				host.CallOnHttpRequestHeaders([][2]string{{":authority", "example.com"}, {":path", "/v1/chat/completions"}, {":method", "POST"}, {"content-type", "application/json"}, {"content-length", "99"}})
				require.Equal(t, types.ActionPause, host.CallOnHttpRequestBody([]byte(`{"model":"source-model"}`)))
			})
		}
	})
}

// TestRuntimeIdentityMapperSequenceABA 验证跨卡回切会继续递增 generation，不能把第一次 A 的身份当作当前 A 复用。
func TestRuntimeIdentityMapperSequenceABA(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		cardA := strictRuntimeIdentityTarget("card-a", "provider-a", "source-model")
		cardB := strictRuntimeIdentityTarget("card-b", "provider-b", "target-model")
		closure := map[string]runtimeIdentityTarget{"card-a": cardA, "card-b": cardB}
		headers := [][2]string{{":authority", "example.com"}, {":path", "/v1/chat/completions"}, {":method", "POST"}, {"content-type", "application/json"}}

		// 第一段模拟 Direct 后的 mapper A -> B；property 字节会作为 redirect 重新进入下一段的唯一上下文输入。
		first, status := test.NewTestHost(strictRuntimeIdentityMapperConfigWithClosure(
			t, "mapper:policy-1:0", cardB, map[string]string{"source-model": "target-model"}, closure,
		))
		require.Equal(t, types.OnPluginStartStatusOK, status)
		setStrictMapperContext(t, first, "card-a", "provider-a", "source-model", 1, "revision-1")
		require.Equal(t, types.HeaderStopIteration, first.CallOnHttpRequestHeaders(headers))
		require.Equal(t, types.ActionContinue, first.CallOnHttpRequestBody([]byte(`{"model":"source-model"}`)))
		raw, err := first.GetProperty([]string{resolvedModelContextProperty})
		require.NoError(t, err)
		require.Equal(t, "card-b", gjson.GetBytes(raw, "resolvedModelCardId").String())
		require.Equal(t, int64(2), gjson.GetBytes(raw, "transitionSeq").Int())
		first.Reset()

		// 第二段模拟同一请求在 shadow route 上回切 B -> A；这里不声称模拟真实 Envoy stream，只验证序列化 property 合同。
		second, status := test.NewTestHost(strictRuntimeIdentityMapperConfigWithClosure(
			t, "mapper:policy-1:1", cardA, map[string]string{"target-model": "source-model"}, closure,
		))
		defer second.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, second.SetProperty([]string{resolvedModelContextProperty}, raw))
		require.Equal(t, types.HeaderStopIteration, second.CallOnHttpRequestHeaders(headers))
		require.Equal(t, types.ActionContinue, second.CallOnHttpRequestBody([]byte(`{"model":"target-model"}`)))
		raw, err = second.GetProperty([]string{resolvedModelContextProperty})
		require.NoError(t, err)
		require.Equal(t, "card-a", gjson.GetBytes(raw, "resolvedModelCardId").String())
		require.Equal(t, int64(3), gjson.GetBytes(raw, "transitionSeq").Int())
		require.Equal(t, "mapper", gjson.GetBytes(raw, "source").String())
	})
}
