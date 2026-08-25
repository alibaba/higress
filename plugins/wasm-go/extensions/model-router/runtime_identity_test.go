package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// runtimeIdentityTestHost 为旧 wasm-go test helper 补齐 strict 插件启动所需的 declare_property foreign function。
// 输入约束：只在 Go-mode 单元测试中使用；每个实例独占一个 proxytest Host，不能跨用例复用。
// 输出语义：保留现有 Header/Body/property 断言能力，并记录 strict property declaration 的 wire payload。
// 边界场景：该夹具不模拟 Envoy internal redirect 的真实 FilterState 生命周期，真实链路由独立 Envoy 验证覆盖。
type runtimeIdentityTestHost struct {
	proxytest.HostEmulator
	currentContextID uint32
	contextActive    bool
	declarationCalls [][]byte
	reset            func()
}

// newRuntimeIdentityTestHost 创建已注册 declare_property 的 strict 单元测试宿主。
// 输入约束：config 必须是完整 strict ModelRouter JSON；调用方在断言结束后必须调用 Reset 释放全局 mock host。
// 输出语义：返回已执行 OnPluginStart 的宿主，且每次 declaration payload 都被保留以验证 wire 协议。
// 边界场景：每次重建 VM context，避免其它 legacy 单测 Reset SDK 全局状态后让 strict 用例依赖执行顺序；foreign function 返回一个占位字节以适配旧 proxytest 的非空返回缓冲实现，生产 Envoy 不依赖该返回值。
func newRuntimeIdentityTestHost(t *testing.T, config []byte) (*runtimeIdentityTestHost, types.OnPluginStartStatus) {
	t.Helper()
	wrapper.SetCtx(
		"model-router",
		wrapper.PrePluginStartOrReload[ModelRouterConfig](ensureRuntimeIdentityRequestProperty),
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.WithRebuildMaxMemBytes[ModelRouterConfig](200*1024*1024),
	)
	vmContext := proxywasm.GetVMContext()
	require.NotNil(t, vmContext, "runtime identity test VM context must be initialized by wrapper.SetCtx")
	host, reset := proxytest.NewHostEmulator(
		proxytest.NewEmulatorOption().
			WithPluginConfiguration(config).
			WithVMContext(vmContext),
	)
	result := &runtimeIdentityTestHost{HostEmulator: host, reset: reset}
	host.RegisterForeignFunction("declare_property", func(payload []byte) []byte {
		result.declarationCalls = append(result.declarationCalls, append([]byte(nil), payload...))
		return []byte{0}
	})
	// 旧 wrapper 在插件启动日志路径读取 get_log_level；该返回值不属于本协议，但必须在宿主启动前提供。
	host.RegisterForeignFunction("get_log_level", func([]byte) []byte { return []byte{0, 0, 0, 0} })
	status := host.StartPlugin()
	require.NoError(t, result.SetRouteName("test-route-default"))
	require.NoError(t, result.SetProperty([]string{"cluster_name"}, []byte("test-cluster-default")))
	require.NoError(t, result.SetProperty([]string{"x_request_id"}, []byte("test-request-id-default")))
	return result, status
}

// SetRouteName 设置 wrapper `_match_route_` 所读取的 route_name property。
// 输入约束：routeName 是测试固定字符串，不能在已开始的 HTTP context 中模拟生产重新路由。
// 输出语义：后续创建的 HTTP context 读取该路由名并选择对应 strict `_rules_` entry。
// 边界场景：property 写入失败直接返回，让用例以 require 失败而不是产生错误的 rule 命中结论。
func (h *runtimeIdentityTestHost) SetRouteName(routeName string) error {
	return h.SetProperty([]string{"route_name"}, []byte(routeName))
}

// CallOnHttpRequestHeaders 在当前或新建 HTTP context 上执行请求头回调。
// 输入约束：headers 必须先于 Body 调用，以保持 strict parser 的缓冲生命周期与真实插件一致。
// 输出语义：返回插件 action，并保留同一 context 供后续 Body 与 property 断言使用。
// 边界场景：夹具不自动跨请求复用 context，避免一次用例的 property 被误认作新请求状态。
func (h *runtimeIdentityTestHost) CallOnHttpRequestHeaders(headers [][2]string) types.Action {
	if !h.contextActive {
		h.currentContextID = h.InitializeHttpContext()
		h.contextActive = true
	}
	return h.HostEmulator.CallOnRequestHeaders(h.currentContextID, headers, false)
}

// CallOnHttpRequestBody 在当前 HTTP context 上执行最终 Body 回调。
// 输入约束：调用前必须完成 Header 回调；Body 由 proxytest 模拟为 end-of-stream。
// 输出语义：返回插件 action，供 Direct 选择、拒绝和重入路径断言。
// 边界场景：未初始化 context 时直接 panic，防止测试绕过 strict Header parser 而得到不可信的结论。
func (h *runtimeIdentityTestHost) CallOnHttpRequestBody(body []byte) types.Action {
	if !h.contextActive {
		panic("runtime identity test must call request headers before request body")
	}
	return h.HostEmulator.CallOnRequestBody(h.currentContextID, body, true)
}

// GetRequestBody 返回当前 HTTP context 被插件改写后的请求 Body。
// 输入约束：只应在至少一次 Header 回调之后读取。
// 输出语义：Direct rewrite 断言读取当前流，而不是测试输入副本。
// 边界场景：context 未初始化视为测试顺序错误，避免零值 context 被当成真实请求。
func (h *runtimeIdentityTestHost) GetRequestBody() []byte {
	if !h.contextActive {
		panic("runtime identity test request context is not initialized")
	}
	return h.HostEmulator.GetCurrentRequestBody(h.currentContextID)
}

// GetRequestHeaders 返回当前 HTTP context 被插件改写后的请求头。
// 输入约束：只应在 Header/Body 回调后的同一请求中读取。
// 输出语义：供 strict modelToHeader 的投影断言使用。
// 边界场景：不从全局 Host 状态拼装 Header，避免相邻用例交叉污染。
func (h *runtimeIdentityTestHost) GetRequestHeaders() [][2]string {
	if !h.contextActive {
		panic("runtime identity test request context is not initialized")
	}
	return h.HostEmulator.GetCurrentRequestHeaders(h.currentContextID)
}

// Reset 关闭当前 HTTP context 并释放该用例注册的全局 proxy-wasm mock host。
// 输入约束：每个测试宿主只能 Reset 一次；调用方通常通过 defer 保证释放。
// 输出语义：清除 SDK VM state，避免下一条严格配置继承本用例的 property 或 foreign function。
// 边界场景：HTTP context 未创建时仅释放 root host，仍然是合法的启动失败清理路径。
func (h *runtimeIdentityTestHost) Reset() {
	if h.contextActive {
		h.CompleteHttpContext(h.currentContextID)
		h.contextActive = false
	}
	h.reset()
}

// TestRuntimeIdentityRequestPropertyDeclaration 验证 strict 检测和 Envoy declare_property wire 协议没有退回默认 FilterChain span。
func TestRuntimeIdentityRequestPropertyDeclaration(t *testing.T) {
	t.Run("strict config detection", func(t *testing.T) {
		require.False(t, runtimeIdentityConfigRequiresRequestProperty(gjson.Parse(`{"modelKey":"model"}`)))
		require.True(t, runtimeIdentityConfigRequiresRequestProperty(gjson.Parse(`{"modelRuntimeIdentity":{}}`)))
		require.True(t, runtimeIdentityConfigRequiresRequestProperty(gjson.Parse(`{"_rules_":[{"modelRuntimeIdentity":{}}]}`)))
	})
	t.Run("downstream request bytes payload", func(t *testing.T) {
		want := append([]byte{0x0a, byte(len(resolvedModelContextProperty))}, []byte(resolvedModelContextProperty)...)
		want = append(want, 0x18, 0x00, 0x28, 0x01)
		require.True(t, bytes.Equal(want, runtimeIdentityRequestPropertyDeclarationPayload()))
	})
}

func strictModelRouterConfig(t *testing.T) []byte {
	t.Helper()
	return strictModelRouterConfigWithRule(t, strictModelRouterRule())
}

// strictModelRouterConfigWithRule 将指定冻结 rule 包装成 model-router 的最小 strict 配置。
// 输入约束：rule 必须满足生产 parser 的完整性要求，调用方可仅在自身测试内修改其副本。
// 输出语义：返回不含 legacy 路由猜测字段的独立 strict 配置。
// 边界情况：这只模拟单插件的配置消费，不能替代真实 Envoy filter-chain 验证。
func strictModelRouterConfigWithRule(t *testing.T, rule map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"modelToHeader":        "x-higress-llm-model",
		"addProviderHeader":    "x-higress-llm-provider",
		"modelRuntimeIdentity": rule,
	})
	require.NoError(t, err)
	return raw
}

// strictModelRouterRule 返回由 APIGO control-plane 下发到单个 strict rule 的冻结身份字段。
// 责任：让直接配置与 `_rules_` 载体测试复用同一份真实 C1 输入，避免两套 fixture 漂移。
// 输入约束：返回 map 仅供测试 JSON 序列化使用，调用方不得在一个用例中改写后复用于其它用例。
// 输出语义：包含 Direct 选择所需的 route scope、body parser、selector target 和闭包。
// 边界场景：`modelToHeader` 属于 rule 外层字段，因为 Higress wrapper 会对每条 `_rules_` entry 独立执行 parseConfig。
func strictModelRouterRule() map[string]any {
	return map[string]any{
		"mode":           "ChooseModelV1",
		"configRevision": "revision-1",
		"scope": map[string]any{
			"gatewayId": "gw-1", "apiId": "api-1", "routeId": "route-1", "dataPlaneRouteName": "route-name-1",
		},
		"parser": map[string]any{"source": "json_body", "modelKey": "model"},
		"selectorTargets": map[string]any{
			"provider/model": map[string]any{"modelCardId": "card-1", "provider": "provider", "upstreamModelName": "model", "serviceId": "service-1", "targetCluster": "outbound|443||provider.internal"},
		},
		"reservedAutoSelectors": []string{"auto", "auto/*"},
		"targetClosure": map[string]any{
			"card-1": map[string]any{"modelCardId": "card-1", "provider": "provider", "upstreamModelName": "model", "serviceId": "service-1", "targetCluster": "outbound|443||provider.internal"},
		},
	}
}

// strictModelRouterCarrierConfig 构造 APIGO 专有 strict carrier 的实际 WasmPlugin 配置形态。
// 责任：验证全局层只有 strict-only 开关、route 匹配和模型 Header 规则字段均落在单条 `_rules_` entry。
// 输入约束：route 名必须与冻结 rule scope 的 dataPlaneRouteName 一致，否则 wrapper 不会选择该 rule。
// 输出语义：返回可直接喂给 Higress test host 的 JSON 配置。
// 边界场景：全局没有 legacy `modelToHeader` 或路径后缀，因此未命中 route 不能触发 body 缓冲或 legacy model-router 行为。
func strictModelRouterCarrierConfig(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"strictRuntimeIdentityOnly": true,
		"_rules_": []map[string]any{{
			"_match_route_":        []string{"route-name-1"},
			"modelToHeader":        "x-higress-llm-model",
			"addProviderHeader":    "x-higress-llm-provider",
			"modelRuntimeIdentity": strictModelRouterRule(),
		}},
	})
	require.NoError(t, err)
	return raw
}

// TestRuntimeIdentityDirectSelection 验证 Direct 只使用 frozen selector map，并把唯一卡身份写入 property。
func TestRuntimeIdentityDirectSelection(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		for _, testCase := range []struct {
			name     string
			selector string
			rule     map[string]any
		}{
			{name: "canonical selector", selector: "provider/model", rule: strictModelRouterRule()},
			{
				name:     "explicit alias selector",
				selector: "model-alias",
				rule: func() map[string]any {
					rule := strictModelRouterRule()
					selectors := rule["selectorTargets"].(map[string]any)
					// 便捷别名是否可用由 APIGO compiler 决定；数据面仅按已冻结的精确字面量表查找。
					selectors["model-alias"] = selectors["provider/model"]
					return rule
				}(),
			},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				host, status := newRuntimeIdentityTestHost(t, strictModelRouterConfigWithRule(t, testCase.rule))
				defer host.Reset()
				require.Equal(t, types.OnPluginStartStatusOK, status)
				require.Equal(t, [][]byte{runtimeIdentityRequestPropertyDeclarationPayload()}, host.declarationCalls)

				require.Equal(t, types.HeaderStopIteration, host.CallOnHttpRequestHeaders([][2]string{
					{":authority", "example.com"}, {":path", "/any/path"}, {":method", "POST"}, {"content-type", "application/json"}, {"content-length", "99"}, {runtimeIdentityTargetClusterHeader, "client-spoofed-cluster"},
				}))
				require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody([]byte(`{"model":"`+testCase.selector+`"}`)))
				require.Equal(t, "model", gjson.GetBytes(host.GetRequestBody(), "model").String())
				// selector 改写为不同长度的 upstream model 后，不得继续携带请求入站时的旧长度。
				_, hasContentLength := getHeader(host.GetRequestHeaders(), "content-length")
				require.False(t, hasContentLength)
				// APIGO-CONTRACT: modelcard-runtime-identity
				// 客户端传入的 cluster 不能穿透 Direct writer；只有 selector target 中同 revision 的目标可写入 Route。
				clusterHeader, found := getHeader(host.GetRequestHeaders(), runtimeIdentityTargetClusterHeader)
				require.True(t, found)
				require.Equal(t, "outbound|443||provider.internal", clusterHeader)

				raw, err := host.GetProperty([]string{resolvedModelContextProperty})
				require.NoError(t, err)
				var context resolvedModelContext
				require.NoError(t, json.Unmarshal(raw, &context))
				require.Equal(t, resolvedModelContextSchemaVersion, int(gjson.GetBytes(raw, "schemaVersion").Int()))
				require.Equal(t, "gw-1", gjson.GetBytes(raw, "gatewayId").String())
				require.False(t, gjson.GetBytes(raw, "scope").Exists())
				require.False(t, gjson.GetBytes(raw, "version").Exists())
				require.Equal(t, "card-1", context.ResolvedModelCardID)
				require.Equal(t, int64(1), context.TransitionSeq)
				require.Equal(t, "direct", context.Source)
			})
		}
	})
}

// TestRuntimeIdentityDedicatedCarrierRule 验证专有 carrier 的 `_rules_` entry 能被 wrapper 正确选择并保留 strict-only 隔离。
func TestRuntimeIdentityDedicatedCarrierRule(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		host, status := newRuntimeIdentityTestHost(t, strictModelRouterCarrierConfig(t))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-name-1"))

		require.Equal(t, types.HeaderStopIteration, host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"}, {":path", "/any/path"}, {":method", "POST"}, {"content-type", "application/json"}, {"content-length", "99"},
		}))
		require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody([]byte(`{"model":"provider/model"}`)))
		// Header 与 Body 都使用冻结 target 的 upstreamModelName；可信身份只通过 stream property 传播。
		header, found := getHeader(host.GetRequestHeaders(), "x-higress-llm-model")
		require.True(t, found)
		require.Equal(t, "model", header)
		providerHeader, found := getHeader(host.GetRequestHeaders(), "x-higress-llm-provider")
		require.True(t, found)
		require.Equal(t, "provider", providerHeader)
		clusterHeader, found := getHeader(host.GetRequestHeaders(), runtimeIdentityTargetClusterHeader)
		require.True(t, found)
		require.Equal(t, "outbound|443||provider.internal", clusterHeader)

		raw, err := host.GetProperty([]string{resolvedModelContextProperty})
		require.NoError(t, err)
		require.Equal(t, "card-1", gjson.GetBytes(raw, "resolvedModelCardId").String())
	})
}

// TestRuntimeIdentityRejectsUnknownOrSpoofedContext 验证客户端 selector 和损坏 property 都不会被静默接受。
func TestRuntimeIdentityRejectsUnknownOrSpoofedContext(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("unknown selector", func(t *testing.T) {
			host, status := newRuntimeIdentityTestHost(t, strictModelRouterConfig(t))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			host.CallOnHttpRequestHeaders([][2]string{{":authority", "example.com"}, {":path", "/any"}, {":method", "POST"}, {"content-type", "application/json"}})
			require.Equal(t, types.ActionPause, host.CallOnHttpRequestBody([]byte(`{"model":"untrusted"}`)))
		})

		t.Run("malformed property", func(t *testing.T) {
			host, status := newRuntimeIdentityTestHost(t, strictModelRouterConfig(t))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			require.NoError(t, host.SetProperty([]string{resolvedModelContextProperty}, []byte(`not-json`)))
			host.CallOnHttpRequestHeaders([][2]string{{":authority", "example.com"}, {":path", "/any"}, {":method", "POST"}, {"content-type", "application/json"}})
			require.Equal(t, types.ActionPause, host.CallOnHttpRequestBody([]byte(`{"model":"provider/model"}`)))
		})
	})
}

// TestRuntimeIdentityLiteralSelectorAndReentry 验证 slash selector 不拆段，并且 fallback 重入不会重置已有 generation。
func TestRuntimeIdentityLiteralSelectorAndReentry(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("slash selector remains one literal", func(t *testing.T) {
			rule := strictModelRouterRule()
			selectorTargets := rule["selectorTargets"].(map[string]any)
			targetClosure := rule["targetClosure"].(map[string]any)
			delete(selectorTargets, "provider/model")
			selectorTargets["provider/model/with/slash"] = map[string]any{
				"modelCardId": "card-slash", "provider": "provider", "upstreamModelName": "model/with/slash", "serviceId": "service-slash", "targetCluster": "outbound|443||provider-slash.internal",
			}
			delete(targetClosure, "card-1")
			targetClosure["card-slash"] = map[string]any{
				"modelCardId": "card-slash", "provider": "provider", "upstreamModelName": "model/with/slash", "serviceId": "service-slash", "targetCluster": "outbound|443||provider-slash.internal",
			}

			host, status := newRuntimeIdentityTestHost(t, strictModelRouterConfigWithRule(t, rule))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			require.Equal(t, types.HeaderStopIteration, host.CallOnHttpRequestHeaders(runtimeIdentityTestHeaders()))
			require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody([]byte(`{"model":"provider/model/with/slash"}`)))
			require.Equal(t, "model/with/slash", gjson.GetBytes(host.GetRequestBody(), "model").String())

			raw, err := host.GetProperty([]string{resolvedModelContextProperty})
			require.NoError(t, err)
			require.Equal(t, "card-slash", gjson.GetBytes(raw, "resolvedModelCardId").String())
		})

		t.Run("valid reentry preserves previous generation", func(t *testing.T) {
			host, status := newRuntimeIdentityTestHost(t, strictModelRouterConfig(t))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			context, err := json.Marshal(resolvedModelContext{
				SchemaVersion:       resolvedModelContextSchemaVersion,
				GatewayID:           "gw-1",
				APIID:               "api-1",
				RouteID:             "route-1",
				ConfigRevision:      "revision-1",
				ResolvedModelCardID: "card-1",
				TransitionSeq:       2,
				Source:              "fallback",
			})
			require.NoError(t, err)
			require.NoError(t, host.SetProperty([]string{resolvedModelContextProperty}, context))

			require.Equal(t, types.HeaderStopIteration, host.CallOnHttpRequestHeaders(runtimeIdentityTestHeaders()))
			require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody([]byte(`{"model":"model"}`)))
			raw, err := host.GetProperty([]string{resolvedModelContextProperty})
			require.NoError(t, err)
			require.Equal(t, int64(2), gjson.GetBytes(raw, "transitionSeq").Int())
			require.Equal(t, "fallback", gjson.GetBytes(raw, "source").String())
		})
	})
}

// runtimeIdentityTestHeaders 返回 strict JSON Body 路径所需的最小请求头。
// 输入约束：调用方随后必须提供 JSON Body；该 helper 不模拟真实 Envoy 路由或 internal redirect。
// 输出语义：返回新的切片，避免不同 TestHost 共享可变 header。
// 边界情况：content-length 故意省略，交由插件在 Header 阶段按真实行为处理。
func runtimeIdentityTestHeaders() [][2]string {
	return [][2]string{
		{":authority", "example.com"}, {":path", "/v1/chat/completions"}, {":method", "POST"}, {"content-type", "application/json"},
	}
}
