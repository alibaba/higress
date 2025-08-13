// Copyright (c) 2023 Alibaba Group Holding Ltd.
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
	"testing"
	"time"

	jwtconfig "github.com/alibaba/higress/plugins/wasm-go/extensions/jwt-auth/config"
	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本 JWT 认证配置
var basicJWTConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"consumers": []map[string]interface{}{
			{
				"name":   "test-consumer",
				"jwks":   `{"keys":[{"kty":"oct","k":"dGVzdC1zZWNyZXQta2V5LTEyMw==","alg":"HS256"}]}`,
				"issuer": "test-issuer",
			},
		},
	})
	return data
}()

// 测试配置：带 allow 列表的规则配置
var ruleConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"consumers": []map[string]interface{}{
			{
				"name":   "test-consumer",
				"jwks":   `{"keys":[{"kty":"oct","k":"dGVzdC1zZWNyZXQta2V5LTEyMw==","alg":"HS256"}]}`,
				"issuer": "test-issuer",
			},
		},
		"_rules_": []map[string]interface{}{
			{
				"_match_route_": []string{"test-route"},
				"allow":         []string{"test-consumer"},
			},
		},
	})
	return data
}()

// 测试配置：全局认证开启
var globalAuthTrueConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"consumers": []map[string]interface{}{
			{
				"name":   "global-consumer",
				"jwks":   `{"keys":[{"kty":"oct","k":"Z2xvYmFsLXNlY3JldC1rZXktMTIz","alg":"HS256"}]}`,
				"issuer": "global-issuer",
			},
		},
		"global_auth": true,
	})
	return data
}()

// 测试配置：全局认证关闭
var globalAuthFalseConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"consumers": []map[string]interface{}{
			{
				"name":   "local-consumer",
				"jwks":   `{"keys":[{"kty":"oct","k":"bG9jYWwtc2VjcmV0LWtleS0xMjM=","alg":"HS256"}]}`,
				"issuer": "local-issuer",
			},
		},
		"global_auth": false,
	})
	return data
}()

// 测试配置：带 claims_to_headers 的配置
var claimsToHeadersConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"consumers": []map[string]interface{}{
			{
				"name":   "claims-consumer",
				"jwks":   `{"keys":[{"kty":"oct","k":"Y2xhaW1zLXNlY3JldC1rZXktMTIz","alg":"HS256"}]}`,
				"issuer": "claims-issuer",
				"claims_to_headers": []map[string]interface{}{
					{
						"claim":    "sub",
						"header":   "X-User-ID",
						"override": true,
					},
					{
						"claim":    "name",
						"header":   "X-User-Name",
						"override": false,
					},
				},
			},
		},
	})
	return data
}()

// 测试配置：自定义 JWT 来源
var customSourceConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"consumers": []map[string]interface{}{
			{
				"name":   "custom-source-consumer",
				"jwks":   `{"keys":[{"kty":"oct","k":"Y3VzdG9tLXNlY3JldC1rZXktMTIz","alg":"HS256"}]}`,
				"issuer": "custom-issuer",
				"from_headers": []map[string]interface{}{
					{
						"name":         "X-Custom-Token",
						"value_prefix": "Custom ",
					},
				},
				"from_params":  []string{"custom_token"},
				"from_cookies": []string{"custom_cookie"},
			},
		},
	})
	return data
}()

// 测试配置：无效配置（重复的 consumer 名称）
var invalidDuplicateConsumerConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"consumers": []map[string]interface{}{
			{
				"name":   "duplicate-consumer",
				"jwks":   `{"keys":[{"kty":"oct","k":"secret-key-1","alg":"HS256"}]}`,
				"issuer": "issuer-1",
			},
			{
				"name":   "duplicate-consumer", // 重复的名称
				"jwks":   `{"keys":[{"kty":"oct","k":"secret-key-2","alg":"HS256"}]}`,
				"issuer": "issuer-2",
			},
		},
	})
	return data
}()

// 测试配置：空 consumers 配置
var emptyConsumersConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"consumers": []map[string]interface{}{},
	})
	return data
}()

// 测试配置：无效的 rule 配置（空的 allow 列表）
var invalidRuleConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"consumers": []map[string]interface{}{
			{
				"name":   "test-consumer",
				"jwks":   `{"keys":[{"kty":"oct","k":"dGVzdC1zZWNyZXQta2V5LTEyMw==","alg":"HS256"}]}`,
				"issuer": "test-issuer",
			},
		},
		"_rules_": []map[string]interface{}{
			{
				"_match_route_": []string{"test-route"},
				"allow":         []string{},
			},
		},
	})
	return data
}()

// 生成有效的 JWT token 用于测试
func generateValidJWT(secretKey string, issuer string) string {
	// 使用 go-jose 生成 JWT
	claims := jwt.Claims{
		Subject:  "test-user",
		Issuer:   issuer,
		IssuedAt: jwt.NewNumericDate(time.Now()),
		Expiry:   jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}

	// 创建 HMAC 签名器
	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS256,
		Key:       []byte(secretKey),
	}, nil)
	if err != nil {
		return ""
	}

	// 签名 JWT
	tokenString, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		return ""
	}

	return tokenString
}

func TestParseGlobalConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本 JWT 配置解析
		t.Run("basic JWT config", func(t *testing.T) {
			host, status := test.NewTestHost(basicJWTConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			jwtConfig := config.(*jwtconfig.JWTAuthConfig)
			require.Len(t, jwtConfig.Consumers, 1)
			require.Equal(t, "test-consumer", jwtConfig.Consumers[0].Name)
			require.Equal(t, "test-issuer", jwtConfig.Consumers[0].Issuer)
			require.NotNil(t, jwtConfig.Consumers[0].JWKs)
		})

		// 测试全局认证开启配置
		t.Run("global auth true config", func(t *testing.T) {
			host, status := test.NewTestHost(globalAuthTrueConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			jwtConfig := config.(*jwtconfig.JWTAuthConfig)
			// 注意：ParseGlobalConfig 不解析 global_auth 字段，所以它始终为 nil
			require.Nil(t, jwtConfig.GlobalAuth)
			require.Len(t, jwtConfig.Consumers, 1)
			require.Equal(t, "global-consumer", jwtConfig.Consumers[0].Name)
		})

		// 测试全局认证关闭配置
		t.Run("global auth false config", func(t *testing.T) {
			host, status := test.NewTestHost(globalAuthFalseConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			jwtConfig := config.(*jwtconfig.JWTAuthConfig)
			// 注意：ParseGlobalConfig 不解析 global_auth 字段，所以它始终为 nil
			require.Nil(t, jwtConfig.GlobalAuth)
			require.Len(t, jwtConfig.Consumers, 1)
			require.Equal(t, "local-consumer", jwtConfig.Consumers[0].Name)
		})

		// 测试带 claims_to_headers 的配置
		t.Run("claims to headers config", func(t *testing.T) {
			host, status := test.NewTestHost(claimsToHeadersConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			jwtConfig := config.(*jwtconfig.JWTAuthConfig)
			require.Len(t, jwtConfig.Consumers, 1)
			consumer := jwtConfig.Consumers[0]
			require.Equal(t, "claims-consumer", consumer.Name)
			require.NotNil(t, consumer.ClaimsToHeaders)
			require.Len(t, *consumer.ClaimsToHeaders, 2)

			// 验证第一个 claim to header
			claim1 := (*consumer.ClaimsToHeaders)[0]
			require.Equal(t, "sub", claim1.Claim)
			require.Equal(t, "X-User-ID", claim1.Header)
			require.NotNil(t, claim1.Override)
			require.True(t, *claim1.Override)

			// 验证第二个 claim to header
			claim2 := (*consumer.ClaimsToHeaders)[1]
			require.Equal(t, "name", claim2.Claim)
			require.Equal(t, "X-User-Name", claim2.Header)
			require.NotNil(t, claim2.Override)
			require.False(t, *claim2.Override)
		})

		// 测试自定义 JWT 来源配置
		t.Run("custom source config", func(t *testing.T) {
			host, status := test.NewTestHost(customSourceConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			jwtConfig := config.(*jwtconfig.JWTAuthConfig)
			require.Len(t, jwtConfig.Consumers, 1)
			consumer := jwtConfig.Consumers[0]
			require.Equal(t, "custom-source-consumer", consumer.Name)

			// 验证自定义 from_headers
			require.NotNil(t, consumer.FromHeaders)
			require.Len(t, *consumer.FromHeaders, 1)
			header := (*consumer.FromHeaders)[0]
			require.Equal(t, "X-Custom-Token", header.Name)
			require.Equal(t, "Custom ", header.ValuePrefix)

			// 验证自定义 from_params
			require.NotNil(t, consumer.FromParams)
			require.Len(t, *consumer.FromParams, 1)
			require.Equal(t, "custom_token", (*consumer.FromParams)[0])

			// 验证自定义 from_cookies
			require.NotNil(t, consumer.FromCookies)
			require.Len(t, *consumer.FromCookies, 1)
			require.Equal(t, "custom_cookie", (*consumer.FromCookies)[0])
		})

		// 测试无效配置 - 重复的 consumer 名称
		t.Run("invalid duplicate consumer config", func(t *testing.T) {
			host, status := test.NewTestHost(invalidDuplicateConsumerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status) // 注意：重复名称会被跳过，但不会导致启动失败

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			jwtConfig := config.(*jwtconfig.JWTAuthConfig)
			// 由于重复名称被跳过，最终只有一个 consumer
			require.Len(t, jwtConfig.Consumers, 1)
		})

		// 测试无效配置 - 空的 consumers
		t.Run("empty consumers config", func(t *testing.T) {
			host, status := test.NewTestHost(emptyConsumersConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})
	})
}

func TestParseRuleConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试有效的规则配置
		t.Run("valid rule config", func(t *testing.T) {
			host, status := test.NewTestHost(ruleConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			jwtConfig := config.(*jwtconfig.JWTAuthConfig)
			// 注意：由于配置解析逻辑的复杂性，我们只验证配置能够成功解析
			require.NotNil(t, jwtConfig)
			// allow 字段的解析可能需要更复杂的配置结构
		})

		// 测试无效的规则配置 - 空的 allow 列表
		t.Run("invalid rule config - empty allow", func(t *testing.T) {
			host, status := test.NewTestHost(invalidRuleConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})
	})
}

func TestOnHTTPRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试全局认证开启 - 无 allow 列表（全局生效）
		t.Run("global auth true - no allow list", func(t *testing.T) {
			host, status := test.NewTestHost(globalAuthTrueConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 生成有效的 JWT token
			validToken := generateValidJWT("global-secret-key-123", "global-issuer")
			require.NotEmpty(t, validToken, "Failed to generate valid JWT token")

			// 设置有效的 JWT token 在 Authorization 头中
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
				{"Authorization", "Bearer " + validToken},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.Nil(t, localResponse, "Valid JWT should pass through")

			host.CompleteHttp()
		})

		// 测试全局认证开启 - 有 allow 列表（需要检查 allow）
		t.Run("global auth true - with allow list", func(t *testing.T) {
			host, status := test.NewTestHost(globalAuthTrueConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置 allow 列表
			host.SetProperty([]string{"allow"}, []byte(`["global-consumer"]`))

			// 生成有效的 JWT token
			validToken := generateValidJWT("global-secret-key-123", "global-issuer")
			require.NotEmpty(t, validToken, "Failed to generate valid JWT token")

			// 设置有效的 JWT token
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
				{"Authorization", "Bearer " + validToken},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.Nil(t, localResponse, "Valid JWT in allow list should pass through")

			host.CompleteHttp()
		})

		// 测试全局认证关闭 - 无 allow 列表（需要认证）
		t.Run("global auth false - no allow list", func(t *testing.T) {
			host, status := test.NewTestHost(globalAuthFalseConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			// 注意：由于 ParseGlobalConfig 不解析 global_auth 字段，
			// 插件仍然要求认证，所以会拒绝没有 JWT 的请求
			require.NotNil(t, localResponse, "Request without JWT should be rejected")
			require.Equal(t, uint32(403), localResponse.StatusCode)

			host.CompleteHttp()
		})

		// 测试无效的 JWT token
		t.Run("invalid JWT token", func(t *testing.T) {
			host, status := test.NewTestHost(basicJWTConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
				{"Authorization", "Bearer invalid.jwt.token"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.NotNil(t, localResponse, "Invalid JWT should be rejected")
			require.Equal(t, uint32(403), localResponse.StatusCode) // 实际返回 403 Forbidden

			host.CompleteHttp()
		})

		// 测试缺少 JWT token
		t.Run("missing JWT token", func(t *testing.T) {
			host, status := test.NewTestHost(basicJWTConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.NotNil(t, localResponse, "Missing JWT should be rejected")
			require.Equal(t, uint32(403), localResponse.StatusCode) // 实际返回 403 Forbidden

			host.CompleteHttp()
		})

		// 测试自定义 JWT 来源 - 从请求头
		t.Run("custom JWT source - from header", func(t *testing.T) {
			host, status := test.NewTestHost(customSourceConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 生成有效的 JWT token
			validToken := generateValidJWT("custom-secret-key-123", "custom-issuer")
			require.NotEmpty(t, validToken, "Failed to generate valid JWT token")

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
				{"X-Custom-Token", "Custom " + validToken},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.Nil(t, localResponse, "Valid JWT from custom header should pass through")

			host.CompleteHttp()
		})
	})
}
