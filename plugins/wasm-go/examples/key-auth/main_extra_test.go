// Copyright (c) 2025 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// === helpers =============================================================

// mustConfig marshals m to JSON and fails the test on error.
func mustConfig(t *testing.T, m map[string]interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	require.NoError(t, err)
	return b
}

// routeRuleConfig produces a config with two consumers and a single _rules_ entry
// scoped to "route-a". globalAuth=nil leaves global_auth unset; allow controls
// who is permitted on route-a.
func routeRuleConfig(t *testing.T, globalAuth interface{}, allow []string) json.RawMessage {
	t.Helper()
	cfg := map[string]interface{}{
		"consumers": []map[string]interface{}{
			{"name": "consumer1", "credential": "token1"},
			{"name": "consumer2", "credential": "token2"},
		},
		"keys":      []string{"x-api-key"},
		"in_header": true,
		"_rules_": []map[string]interface{}{
			{
				"_match_route_": []string{"route-a"},
				"allow":         allow,
			},
		},
	}
	if globalAuth != nil {
		cfg["global_auth"] = globalAuth
	}
	return mustConfig(t, cfg)
}

// === Module A — contains helper =========================================
//
// `contains` ships at 0% in the baseline; it is reachable only through the
// allow-list branches of onHttpRequestHeaders, which Module B drives. A direct
// unit test pins behavior independently in case the wasm dispatch path
// regresses or the helper is reused elsewhere (e.g. a future plugin pulling
// in the same util pattern, mirroring basic-auth's contains).

func TestContains_Hit(t *testing.T) {
	require.True(t, contains([]string{"a", "b", "c"}, "b"))
}

func TestContains_Miss(t *testing.T) {
	require.False(t, contains([]string{"a", "b", "c"}, "z"))
}

func TestContains_EmptySlice(t *testing.T) {
	require.False(t, contains([]string{}, "x"))
}

func TestContains_NilSlice(t *testing.T) {
	require.False(t, contains(nil, "x"))
}

// === Module B — onHttpRequestHeaders allow-list branches ================
//
// Baseline onHttpRequestHeaders is 68.9%. Existing main_test.go only drives
// the early-exit and "no allow list" paths plus token-extraction failures.
// The four allow-list dispatches at main.go:344-363 — global_auth=true vs
// global_auth=false, consumer in vs not in allow — are uncovered. Each of
// these reaches `contains`, so this module also exercises the helper through
// its production caller.

// global_auth=true + route-scoped allow + consumer in allow → authenticated
// + X-Mse-Consumer header injected (main.go:344-351 success branch).
func TestOnHttpRequestHeaders_GlobalAuthTrue_RouteAllow_ConsumerAllowed(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(routeRuleConfig(t, true, []string{"consumer1"}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"},
			{":path", "/api/test"},
			{":method", "GET"},
			{"x-api-key", "token1"},
		})
		require.Equal(t, types.ActionContinue, action)
		require.Nil(t, host.GetLocalResponse())
		require.True(t, test.HasHeaderWithValue(host.GetRequestHeaders(), "X-Mse-Consumer", "consumer1"))
	})
}

// global_auth=true + route-scoped allow + consumer not in allow → 403 via
// deniedUnauthorizedConsumer (main.go:344-348). The credential decodes and
// authenticates against credential2Consumer — `consumer2` exists but is not
// permitted on route-a — distinct from the "credential not configured"
// variant already covered by main_test.go's "invalid api key" case.
func TestOnHttpRequestHeaders_GlobalAuthTrue_RouteAllow_ConsumerNotAllowed(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(routeRuleConfig(t, true, []string{"consumer1"}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"},
			{":path", "/api/test"},
			{":method", "GET"},
			{"x-api-key", "token2"},
		})
		require.Equal(t, types.ActionContinue, action)

		resp := host.GetLocalResponse()
		require.NotNil(t, resp)
		require.Equal(t, uint32(http.StatusForbidden), resp.StatusCode)
		require.True(t, test.HasHeader(resp.Headers, "WWW-Authenticate"))
	})
}

// global_auth=false + route-scoped allow + consumer in allow → authenticated
// path through the case-3 branch (main.go:354-362 success). Verifies
// X-Mse-Consumer is still injected when auth is enabled per-route only.
func TestOnHttpRequestHeaders_GlobalAuthFalse_RouteAllow_ConsumerAllowed(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(routeRuleConfig(t, false, []string{"consumer1"}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"},
			{":path", "/api/test"},
			{":method", "GET"},
			{"x-api-key", "token1"},
		})
		require.Equal(t, types.ActionContinue, action)
		require.Nil(t, host.GetLocalResponse())
		require.True(t, test.HasHeaderWithValue(host.GetRequestHeaders(), "X-Mse-Consumer", "consumer1"))
	})
}

// global_auth=false + route-scoped allow + consumer not in allow → 403 via
// deniedUnauthorizedConsumer (main.go:354-359 reject). Mirror of the
// global_auth=true rejection but exercises the case-3 entry condition
// `(globalAuthSetFalse || (globalAuthNoSet && ruleSet)) && !noAllow`.
func TestOnHttpRequestHeaders_GlobalAuthFalse_RouteAllow_ConsumerNotAllowed(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(routeRuleConfig(t, false, []string{"consumer1"}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"},
			{":path", "/api/test"},
			{":method", "GET"},
			{"x-api-key", "token2"},
		})
		require.Equal(t, types.ActionContinue, action)

		resp := host.GetLocalResponse()
		require.NotNil(t, resp)
		require.Equal(t, uint32(http.StatusForbidden), resp.StatusCode)
	})
}

// global_auth unset + at least one route configured + current route NOT
// configured → noAllow short-circuit through `(globalAuthNoSet && ruleSet)`
// at main.go:288-293. Existing tests use global_auth=false for this branch;
// this drives the unset path so the boolean expression's other operand is
// covered.
func TestOnHttpRequestHeaders_GlobalAuthUnset_RuleSet_OtherRoute_PassThrough(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// global_auth omitted; _rules_ entry on route-a only.
		host, status := test.NewTestHost(routeRuleConfig(t, nil, []string{"consumer1"}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		// Drive a request on a different route — current rule context has no
		// allow list, so auth must be skipped entirely.
		require.NoError(t, host.SetRouteName("route-b"))

		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"},
			{":path", "/api/test"},
			{":method", "GET"},
		})
		require.Equal(t, types.ActionContinue, action)
		require.Nil(t, host.GetLocalResponse())
	})
}

// === Module C — parse-time edge rejects =================================
//
// parseGlobalConfig is 93.2% and parseOverrideRuleConfig is 90.9%. The
// missing branches are the empty-string credential rejects (singular and
// inside the credentials array) and the "allow key absent entirely" branch
// at parseOverrideRuleConfig:251 which existing tests don't reach because
// every route-rule fixture supplies `allow` (possibly empty).

// credential: "" must be rejected at parseGlobalConfig:206-208 — distinct
// from "credential field absent" already covered by mixed-credential
// fixtures in main_test.go.
func TestParseGlobalConfig_EmptyCredentialString(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		cfg := mustConfig(t, map[string]interface{}{
			"consumers": []map[string]interface{}{
				{"name": "consumer1", "credential": ""},
			},
			"keys":      []string{"x-api-key"},
			"in_header": true,
		})
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusFailed, status)
	})
}

// An empty string inside the credentials array must be rejected at
// parseGlobalConfig:215-217 — separate branch from the "credentials array
// empty" reject already covered by invalidEmptyPluralCredentialsConfig.
func TestParseGlobalConfig_EmptyCredentialInArray(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		cfg := mustConfig(t, map[string]interface{}{
			"consumers": []map[string]interface{}{
				{"name": "consumer1", "credentials": []string{"token1", ""}},
			},
			"keys":      []string{"x-api-key"},
			"in_header": true,
		})
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusFailed, status)
	})
}

// _rules_ entry without an `allow` key at all — distinct from
// invalidRuleConfig which supplies `allow: []`. Hits the
// `if !allow.Exists()` branch at parseOverrideRuleConfig:251.
func TestParseOverrideRuleConfig_AllowMissing(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		cfg := mustConfig(t, map[string]interface{}{
			"consumers": []map[string]interface{}{
				{"name": "consumer1", "credential": "token1"},
			},
			"keys":      []string{"x-api-key"},
			"in_header": true,
			"_rules_": []map[string]interface{}{
				{
					"_match_route_": []string{"route-a"},
					// no "allow" key at all
				},
			},
		})
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusFailed, status)
	})
}

// === 路由级 allow_groups / allow_credentials 解析 ========================

// routeRuleConfigWith 用例工厂：consumers 固定两个（consumer1 带
// group=team-a），路由规则字段按需组合。globalAuth 传 nil 表示不设置
// global_auth；lists 内某列表传 nil 表示该字段不配置。
func routeRuleConfigWith(t *testing.T, globalAuth interface{}, lists map[string][]string) json.RawMessage {
	t.Helper()
	rule := map[string]interface{}{
		"_match_route_": []string{"route-a"},
	}
	for k, v := range lists {
		if v != nil {
			rule[k] = v
		}
	}
	cfg := map[string]interface{}{
		"consumers": []map[string]interface{}{
			{"name": "consumer1", "credential": "token1", "group": "team-a"},
			{"name": "consumer2", "credential": "token2"},
		},
		"keys":      []string{"x-api-key"},
		"in_header": true,
		"_rules_":   []map[string]interface{}{rule},
	}
	if globalAuth != nil {
		cfg["global_auth"] = globalAuth
	}
	return mustConfig(t, cfg)
}

func TestParseOverrideRuleConfig_OnlyAllowGroups(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		cfg := routeRuleConfigWith(t, nil, map[string][]string{
			"allow_groups": {"team-a"},
		})
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
	})
}

func TestParseOverrideRuleConfig_OnlyAllowCredentials(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		cfg := routeRuleConfigWith(t, nil, map[string][]string{
			"allow_credentials": {"token2"},
		})
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
	})
}

func TestParseOverrideRuleConfig_AllThreeLists(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		cfg := routeRuleConfigWith(t, nil, map[string][]string{
			"allow":             {"consumer1"},
			"allow_groups":      {"team-a"},
			"allow_credentials": {"token2"},
		})
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
	})
}

func TestParseOverrideRuleConfig_MissingAllLists(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		cfg := routeRuleConfigWith(t, nil, nil)
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusFailed, status)
	})
}

func TestParseOverrideRuleConfig_EmptyAllowGroups(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		cfg := routeRuleConfigWith(t, nil, map[string][]string{
			"allow_groups": {},
		})
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusFailed, status)
	})
}

func TestParseOverrideRuleConfig_EmptyAllowCredentials(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		cfg := routeRuleConfigWith(t, nil, map[string][]string{
			"allow_credentials": {},
		})
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusFailed, status)
	})
}

// === 路由级三列表 OR 匹配 ================================================

// requestWithToken 以 x-api-key 头发起一次请求；token 为空串表示不带头。
func requestWithToken(host test.TestHost, token string) types.Action {
	headers := [][2]string{
		{":authority", "example.com"},
		{":path", "/api/test"},
		{":method", "GET"},
	}
	if token != "" {
		headers = append(headers, [2]string{"x-api-key", token})
	}
	return host.CallOnHttpRequestHeaders(headers)
}

func TestOnHttpRequestHeaders_GroupHit_Allowed(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(routeRuleConfigWith(t, true, map[string][]string{
			"allow_groups": {"team-a"},
		}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		require.Equal(t, types.ActionContinue, requestWithToken(host, "token1"))
		require.Nil(t, host.GetLocalResponse())
		require.True(t, test.HasHeaderWithValue(host.GetRequestHeaders(), "X-Mse-Consumer", "consumer1"))
		require.True(t, test.HasHeaderWithValue(host.GetRequestHeaders(), "X-Mse-Consumer-Group", "team-a"))
		require.True(t, test.HasHeaderWithValue(host.GetRequestHeaders(), "X-Mse-Credential", "token1"))
	})
}

func TestOnHttpRequestHeaders_GroupMiss_Denied(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(routeRuleConfigWith(t, true, map[string][]string{
			"allow_groups": {"team-b"},
		}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		require.Equal(t, types.ActionContinue, requestWithToken(host, "token1"))
		resp := host.GetLocalResponse()
		require.NotNil(t, resp)
		require.Equal(t, uint32(http.StatusForbidden), resp.StatusCode)
	})
}

func TestOnHttpRequestHeaders_NoGroupConsumer_NotMatchedByGroup(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// consumer2 无组，不能被 allow_groups 放行
		host, status := test.NewTestHost(routeRuleConfigWith(t, true, map[string][]string{
			"allow_groups": {"team-a"},
		}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		require.Equal(t, types.ActionContinue, requestWithToken(host, "token2"))
		resp := host.GetLocalResponse()
		require.NotNil(t, resp)
		require.Equal(t, uint32(http.StatusForbidden), resp.StatusCode)
	})
}

func TestOnHttpRequestHeaders_CredentialHit_Allowed(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(routeRuleConfigWith(t, true, map[string][]string{
			"allow_credentials": {"token2"},
		}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		require.Equal(t, types.ActionContinue, requestWithToken(host, "token2"))
		require.Nil(t, host.GetLocalResponse())
		require.True(t, test.HasHeaderWithValue(host.GetRequestHeaders(), "X-Mse-Consumer", "consumer2"))
		require.True(t, test.HasHeaderWithValue(host.GetRequestHeaders(), "X-Mse-Credential", "token2"))
	})
}

func TestOnHttpRequestHeaders_BearerCredential_StripsPrefixInIdentityHeader(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// BEARER 来源：注册表凭证带 "Bearer " 传输前缀（与请求原文匹配），
		// X-Mse-Credential 身份头应输出剥掉前缀的逻辑凭证值，
		// 下游限流/配额配置无需感知凭证来源
		cfg := mustConfig(t, map[string]interface{}{
			"consumers": []map[string]interface{}{
				{
					"name":        "consumer1",
					"credentials": []string{"Bearer token1"},
					"keys":        []string{"Authorization"},
					"in_header":   true,
				},
			},
			"global_auth": true,
		})
		host, status := test.NewTestHost(cfg)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		headers := [][2]string{
			{":authority", "example.com"},
			{":path", "/api/test"},
			{":method", "GET"},
			{"Authorization", "Bearer token1"},
		}
		require.Equal(t, types.ActionContinue, host.CallOnHttpRequestHeaders(headers))
		require.Nil(t, host.GetLocalResponse())
		require.True(t, test.HasHeaderWithValue(host.GetRequestHeaders(), "X-Mse-Consumer", "consumer1"))
		require.True(t, test.HasHeaderWithValue(host.GetRequestHeaders(), "X-Mse-Credential", "token1"))
	})
}

func TestOnHttpRequestHeaders_CredentialMiss_Denied(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(routeRuleConfigWith(t, true, map[string][]string{
			"allow_credentials": {"token3"},
		}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		require.Equal(t, types.ActionContinue, requestWithToken(host, "token1"))
		resp := host.GetLocalResponse()
		require.NotNil(t, resp)
		require.Equal(t, uint32(http.StatusForbidden), resp.StatusCode)
	})
}

func TestOnHttpRequestHeaders_OrCombination(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// allow 命中 consumer2、allow_groups 命中 team-a：两者都应放行
		host, status := test.NewTestHost(routeRuleConfigWith(t, true, map[string][]string{
			"allow":        {"consumer2"},
			"allow_groups": {"team-a"},
		}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		require.Equal(t, types.ActionContinue, requestWithToken(host, "token1"))
		require.Nil(t, host.GetLocalResponse())
		require.Equal(t, types.ActionContinue, requestWithToken(host, "token2"))
		require.Nil(t, host.GetLocalResponse())
	})
}

func TestOnHttpRequestHeaders_GlobalAuthFalse_GroupAllowed(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(routeRuleConfigWith(t, false, map[string][]string{
			"allow_groups": {"team-a"},
		}))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.NoError(t, host.SetRouteName("route-a"))

		require.Equal(t, types.ActionContinue, requestWithToken(host, "token1"))
		require.Nil(t, host.GetLocalResponse())
	})
}

func TestConsumerAllowed_Direct(t *testing.T) {
	cfg := KeyAuthConfig{
		allow:            []string{"c1"},
		allowGroups:      []string{"team-a"},
		allowCredentials: []string{"tok3"},
	}
	require.True(t, consumerAllowed(cfg, Consumer{Name: "c1"}, "other"))
	require.True(t, consumerAllowed(cfg, Consumer{Name: "c2", Group: "team-a"}, "other"))
	require.True(t, consumerAllowed(cfg, Consumer{Name: "c2"}, "tok3"))
	require.False(t, consumerAllowed(cfg, Consumer{Name: "c2"}, "other"))
	require.False(t, consumerAllowed(cfg, Consumer{Name: "c2", Group: ""}, "other"))
}
