// Copyright (c) 2024 Alibaba Group Holding Ltd.
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

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/utils"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基础安全配置
var basicConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"serviceName":               "security-service",
		"servicePort":               8080,
		"serviceHost":               "security.example.com",
		"accessKey":                 "test-ak",
		"secretKey":                 "test-sk",
		"checkRequest":              true,
		"checkResponse":             true,
		"contentModerationLevelBar": "high",
		"promptAttackLevelBar":      "high",
		"sensitiveDataLevelBar":     "S3",
		"timeout":                   2000,
		"bufferLimit":               1000,
	})
	return data
}()

// 测试配置：仅检查请求
var requestOnlyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"serviceName":               "security-service",
		"servicePort":               8080,
		"serviceHost":               "security.example.com",
		"accessKey":                 "test-ak",
		"secretKey":                 "test-sk",
		"checkRequest":              true,
		"checkResponse":             false,
		"contentModerationLevelBar": "high",
		"promptAttackLevelBar":      "high",
		"sensitiveDataLevelBar":     "S3",
		"timeout":                   1000,
		"bufferLimit":               500,
	})
	return data
}()

// 测试配置：缺少必需字段
var missingRequiredConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"accessKey": "test-ak",
		"secretKey": "test-sk",
		// 故意缺少必需字段：serviceName, servicePort, serviceHost
	})
	return data
}()

// 测试配置：缺少服务配置字段
var missingServiceConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"accessKey":     "test-ak",
		"secretKey":     "test-sk",
		"checkRequest":  true,
		"checkResponse": true,
		// 缺少 serviceName, servicePort, serviceHost
	})
	return data
}()

// 测试配置：缺少认证字段
var missingAuthConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"serviceName":   "security-service",
		"servicePort":   8080,
		"serviceHost":   "security.example.com",
		"checkRequest":  true,
		"checkResponse": true,
		// 缺少 accessKey, secretKey
	})
	return data
}()

// 测试配置：消费者级别特殊配置
var consumerSpecificConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"serviceName":                "security-service",
		"servicePort":                8080,
		"serviceHost":                "security.example.com",
		"accessKey":                  "test-ak",
		"secretKey":                  "test-sk",
		"checkRequest":               true,
		"checkResponse":              false,
		"contentModerationLevelBar":  "high",
		"promptAttackLevelBar":       "high",
		"sensitiveDataLevelBar":      "S3",
		"maliciousUrlLevelBar":       "high",
		"modelHallucinationLevelBar": "high",
		"timeout":                    1000,
		"bufferLimit":                500,
		"consumerRequestCheckService": map[string]interface{}{
			"name":                "aaa",
			"matchType":           "exact",
			"requestCheckService": "llm_query_moderation_1",
		},
		"consumerResponseCheckService": map[string]interface{}{
			"name":                 "bbb",
			"matchType":            "prefix",
			"responseCheckService": "llm_response_moderation_1",
		},
		"consumerRiskLevel": map[string]interface{}{
			"name":                 "ccc.*",
			"matchType":            "regexp",
			"maliciousUrlLevelBar": "low",
		},
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基础配置解析
		t.Run("basic config", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			securityConfig := config.(*cfg.AISecurityConfig)
			require.Equal(t, "test-ak", securityConfig.AK)
			require.Equal(t, "test-sk", securityConfig.SK)
			require.Equal(t, true, securityConfig.CheckRequest)
			require.Equal(t, true, securityConfig.CheckResponse)
			require.Equal(t, "high", securityConfig.ContentModerationLevelBar)
			require.Equal(t, "high", securityConfig.PromptAttackLevelBar)
			require.Equal(t, "S3", securityConfig.SensitiveDataLevelBar)
			require.Equal(t, uint32(2000), securityConfig.Timeout)
			require.Equal(t, 1000, securityConfig.BufferLimit)
		})

		// 测试仅检查请求的配置
		t.Run("request only config", func(t *testing.T) {
			host, status := test.NewTestHost(requestOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			securityConfig := config.(*cfg.AISecurityConfig)
			require.Equal(t, true, securityConfig.CheckRequest)
			require.Equal(t, false, securityConfig.CheckResponse)
			require.Equal(t, "high", securityConfig.ContentModerationLevelBar)
			require.Equal(t, "high", securityConfig.PromptAttackLevelBar)
			require.Equal(t, "S3", securityConfig.SensitiveDataLevelBar)
		})

		// 测试缺少必需字段的配置
		t.Run("missing required config", func(t *testing.T) {
			host, status := test.NewTestHost(missingRequiredConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试缺少服务配置字段
		t.Run("missing service config", func(t *testing.T) {
			host, status := test.NewTestHost(missingServiceConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试缺少认证字段
		t.Run("missing auth config", func(t *testing.T) {
			host, status := test.NewTestHost(missingAuthConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试消费者级别配置
		t.Run("consumer specific config", func(t *testing.T) {
			host, status := test.NewTestHost(consumerSpecificConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			securityConfig := config.(*cfg.AISecurityConfig)
			require.Equal(t, "llm_query_moderation", securityConfig.GetRequestCheckService("aaaa"))
			require.Equal(t, "llm_query_moderation_1", securityConfig.GetRequestCheckService("aaa"))
			require.Equal(t, "llm_response_moderation", securityConfig.GetResponseCheckService("bb"))
			require.Equal(t, "llm_response_moderation_1", securityConfig.GetResponseCheckService("bbb-prefix-test"))
			require.Equal(t, "high", securityConfig.GetMaliciousUrlLevelBar("cc"))
			require.Equal(t, "low", securityConfig.GetMaliciousUrlLevelBar("ccc-regexp-test"))
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试启用请求检查的情况
		t.Run("request checking enabled", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)
		})

		// 测试禁用请求检查的情况
		t.Run("request checking disabled", func(t *testing.T) {
			host, status := test.NewTestHost(requestOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestOnHttpRequestBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试请求体安全检查通过
		t.Run("request body security check pass", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置请求体
			body := `{"messages": [{"role": "user", "content": "Hello, how are you?"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))

			// 应该返回ActionPause，等待安全检查结果
			require.Equal(t, types.ActionPause, action)

			// 模拟安全检查服务响应（通过）
			securityResponse := `{"Code": 200, "Message": "Success", "RequestId": "req-123", "Data": {"RiskLevel": "low"}}`
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			}, []byte(securityResponse))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试空请求内容
		t.Run("empty request content", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置空内容的请求体
			body := `{"messages": [{"role": "user", "content": ""}]}`
			action := host.CallOnHttpRequestBody([]byte(body))

			// 空内容应该直接通过
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestOnHttpResponseHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试启用响应检查的情况
		t.Run("response checking enabled", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回HeaderStopIteration
			require.Equal(t, types.HeaderStopIteration, action)
		})

		// 测试禁用响应检查的情况
		t.Run("response checking disabled", func(t *testing.T) {
			host, status := test.NewTestHost(requestOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)
		})

		// 测试非200状态码
		t.Run("non-200 status code", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置非200响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "500"},
				{"content-type", "application/json"},
			})

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestOnHttpResponseBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试响应体安全检查通过
		t.Run("response body security check pass", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 设置响应体
			body := `{"choices": [{"message": {"role": "assistant", "content": "Hello, how can I help you?"}}]}`
			action := host.CallOnHttpResponseBody([]byte(body))

			// 应该返回ActionPause，等待安全检查结果
			require.Equal(t, types.ActionPause, action)

			// 模拟安全检查服务响应（通过）
			securityResponse := `{"Code": 200, "Message": "Success", "RequestId": "req-123", "Data": {"RiskLevel": "low"}}`
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			}, []byte(securityResponse))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试空响应内容
		t.Run("empty response content", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 设置空内容的响应体
			body := `{"choices": [{"message": {"role": "assistant", "content": ""}}]}`
			action := host.CallOnHttpResponseBody([]byte(body))

			// 空内容应该直接通过
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestRiskLevelFunctions(t *testing.T) {
	// 测试风险等级转换函数
	t.Run("risk level conversion", func(t *testing.T) {
		require.Equal(t, 4, cfg.LevelToInt(cfg.MaxRisk))
		require.Equal(t, 3, cfg.LevelToInt(cfg.HighRisk))
		require.Equal(t, 2, cfg.LevelToInt(cfg.MediumRisk))
		require.Equal(t, 1, cfg.LevelToInt(cfg.LowRisk))
		require.Equal(t, 0, cfg.LevelToInt(cfg.NoRisk))
		require.Equal(t, -1, cfg.LevelToInt("invalid"))
	})

	// 测试风险等级比较
	t.Run("risk level comparison", func(t *testing.T) {
		require.True(t, cfg.LevelToInt(cfg.HighRisk) >= cfg.LevelToInt(cfg.MediumRisk))
		require.True(t, cfg.LevelToInt(cfg.MediumRisk) >= cfg.LevelToInt(cfg.LowRisk))
		require.True(t, cfg.LevelToInt(cfg.LowRisk) >= cfg.LevelToInt(cfg.NoRisk))
		require.False(t, cfg.LevelToInt(cfg.LowRisk) >= cfg.LevelToInt(cfg.HighRisk))
	})
}

func TestUtilityFunctions(t *testing.T) {
	// 测试十六进制ID生成函数
	t.Run("hex id generation", func(t *testing.T) {
		id, err := utils.GenerateHexID(16)
		require.NoError(t, err)
		require.Len(t, id, 16)
		require.Regexp(t, "^[0-9a-f]+$", id)
	})

	// 测试随机ID生成函数
	t.Run("random id generation", func(t *testing.T) {
		id := utils.GenerateRandomChatID()
		require.NotEmpty(t, id)
		require.Contains(t, id, "chatcmpl-")
		require.Len(t, id, 38) // "chatcmpl-" + 29 random chars
	})
}
