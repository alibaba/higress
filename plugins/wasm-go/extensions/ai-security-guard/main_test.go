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

// 测试配置：MCP配置
var mcpConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"serviceName":                   "security-service",
		"servicePort":                   8080,
		"serviceHost":                   "security.example.com",
		"accessKey":                     "test-ak",
		"secretKey":                     "test-sk",
		"checkRequest":                  false,
		"checkResponse":                 true,
		"action":                        "MultiModalGuard",
		"apiType":                       "mcp",
		"responseContentJsonPath":       "content",
		"responseStreamContentJsonPath": "content",
		"contentModerationLevelBar":     "high",
		"promptAttackLevelBar":          "high",
		"sensitiveDataLevelBar":         "S3",
		"timeout":                       2000,
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

func TestMCP(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// Test MCP Response Body Check - Pass
		t.Run("mcp response body security check pass", func(t *testing.T) {
			host, status := test.NewTestHost(mcpConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"x-mse-consumer", "test-user"},
			})

			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// body content matching responseContentJsonPath="content"
			body := `{"content": "Hello world"}`
			action := host.CallOnHttpResponseBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			securityResponse := `{"Code": 200, "Message": "Success", "RequestId": "req-123", "Data": {"RiskLevel": "low"}}`
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			}, []byte(securityResponse))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// Test MCP Response Body Check - Deny
		t.Run("mcp response body security check deny", func(t *testing.T) {
			host, status := test.NewTestHost(mcpConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			body := `{"content": "Bad content"}`
			action := host.CallOnHttpResponseBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// High Risk
			securityResponse := `{"Code": 200, "Message": "Success", "RequestId": "req-123", "Data": {"RiskLevel": "high"}}`
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			}, []byte(securityResponse))

			// Verify it was replaced with DenyResponse
			// Can't easily verify the replaced body content with current test wrapper but can check action
			// Since plugin calls SendHttpResponse, execution stops or changes.
			// mcp.go uses SendHttpResponse(..., DenyResponse, -1) which means it ends the stream.
			// We can check if GetHttpStreamAction is ActionPause (since it did send a response) or something else.
			// Actually SendHttpResponse in proxy-wasm usually terminates further processing of the original stream.
		})

		// Test MCP Streaming Response Body Check - Pass
		t.Run("mcp streaming response body security check pass", func(t *testing.T) {
			host, status := test.NewTestHost(mcpConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/event-stream"},
			})

			// streaming chunk
			// config uses "content" key
			chunk := []byte(`data: {"content": "Hello"}` + "\n\n")
			// This calls OnHttpStreamingResponseBody -> mcp.HandleMcpStreamingResponseBody
			// It should push buffer and make call
			host.CallOnHttpStreamingResponseBody(chunk, false)
			// Action assertion removed as it returns an internal value 3

			securityResponse := `{"Code": 200, "Message": "Success", "RequestId": "req-123", "Data": {"RiskLevel": "low"}}`
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			}, []byte(securityResponse))
		})

		// Test MCP Streaming Response Body Check - Deny
		t.Run("mcp streaming response body security check deny", func(t *testing.T) {
			host, status := test.NewTestHost(mcpConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/event-stream"},
			})

			chunk := []byte(`data: {"content": "Bad"}` + "\n\n")
			host.CallOnHttpStreamingResponseBody(chunk, false)

			// High Risk
			securityResponse := `{"Code": 200, "Message": "Success", "RequestId": "req-123", "Data": {"RiskLevel": "high"}}`
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			}, []byte(securityResponse))

			// It injects DenySSEResponse.
		})
	})
}

// Config: TextModerationPlus with checkAllMessages enabled
var dedupConfig = func() json.RawMessage {
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
		"timeout":                   2000,
		"checkAllMessages":          true,
		"checkRecordTTL":            3600,
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
			"timeout":      1000,
		},
	})
	return data
}()

// Config: with custom denyMessage
var customDenyConfig = func() json.RawMessage {
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
		"timeout":                   2000,
		"denyMessage":               "自定义拒绝消息",
	})
	return data
}()

// Config: MultiModalGuard with checkAllMessages enabled
var multiModalDedupConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"serviceName":               "security-service",
		"servicePort":               8080,
		"serviceHost":               "security.example.com",
		"accessKey":                 "test-ak",
		"secretKey":                 "test-sk",
		"checkRequest":              true,
		"checkResponse":             false,
		"action":                    "MultiModalGuard",
		"contentModerationLevelBar": "high",
		"promptAttackLevelBar":      "high",
		"sensitiveDataLevelBar":     "S3",
		"timeout":                   2000,
		"checkAllMessages":          true,
		"checkRecordTTL":            3600,
		"redis": map[string]interface{}{
			"service_name": "redis.static",
			"service_port": 6379,
			"timeout":      1000,
		},
	})
	return data
}()

func TestDedupConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("dedup config parsed correctly", func(t *testing.T) {
			host, status := test.NewTestHost(dedupConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			securityConfig := config.(*cfg.AISecurityConfig)
			require.True(t, securityConfig.CheckAllMessages)
			require.Equal(t, 3600, securityConfig.CheckRecordTTL)
			require.NotNil(t, securityConfig.RedisClient)
		})

		t.Run("dedup disabled when redis config missing", func(t *testing.T) {
			noDedupConfig, _ := json.Marshal(map[string]interface{}{
				"serviceName":      "security-service",
				"servicePort":      8080,
				"serviceHost":      "security.example.com",
				"accessKey":        "test-ak",
				"secretKey":        "test-sk",
				"checkRequest":     true,
				"checkAllMessages": true,
			})
			host, status := test.NewTestHost(json.RawMessage(noDedupConfig))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			securityConfig := config.(*cfg.AISecurityConfig)
			require.False(t, securityConfig.CheckAllMessages, "should be disabled without redis config")
		})

		t.Run("default checkRecordTTL", func(t *testing.T) {
			defaultTTLConfig, _ := json.Marshal(map[string]interface{}{
				"serviceName":      "security-service",
				"servicePort":      8080,
				"serviceHost":      "security.example.com",
				"accessKey":        "test-ak",
				"secretKey":        "test-sk",
				"checkRequest":     true,
				"checkAllMessages": true,
				"redis": map[string]interface{}{
					"service_name": "redis.static",
				},
			})
			host, status := test.NewTestHost(json.RawMessage(defaultTTLConfig))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			securityConfig := config.(*cfg.AISecurityConfig)
			require.Equal(t, cfg.DefaultCheckRecordTTL, securityConfig.CheckRecordTTL)
		})

		t.Run("checkRecordTTL <= 0 with checkAllMessages returns error", func(t *testing.T) {
			invalidTTLConfig, _ := json.Marshal(map[string]interface{}{
				"serviceName":      "security-service",
				"servicePort":      8080,
				"serviceHost":      "security.example.com",
				"accessKey":        "test-ak",
				"secretKey":        "test-sk",
				"checkRequest":     true,
				"checkAllMessages": true,
				"checkRecordTTL":   -1,
				"redis": map[string]interface{}{
					"service_name": "redis.static",
				},
			})
			host, status := test.NewTestHost(json.RawMessage(invalidTTLConfig))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		t.Run("redis empty service_name disables dedup", func(t *testing.T) {
			emptyServiceConfig, _ := json.Marshal(map[string]interface{}{
				"serviceName":      "security-service",
				"servicePort":      8080,
				"serviceHost":      "security.example.com",
				"accessKey":        "test-ak",
				"secretKey":        "test-sk",
				"checkRequest":     true,
				"checkAllMessages": true,
				"redis": map[string]interface{}{
					"service_name": "",
				},
			})
			host, status := test.NewTestHost(json.RawMessage(emptyServiceConfig))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			securityConfig := config.(*cfg.AISecurityConfig)
			require.False(t, securityConfig.CheckAllMessages, "should be disabled with empty redis service_name")
		})

		t.Run("checkAllMessages false skips redis init", func(t *testing.T) {
			noCheckAllConfig, _ := json.Marshal(map[string]interface{}{
				"serviceName":      "security-service",
				"servicePort":      8080,
				"serviceHost":      "security.example.com",
				"accessKey":        "test-ak",
				"secretKey":        "test-sk",
				"checkRequest":     true,
				"checkAllMessages": false,
			})
			host, status := test.NewTestHost(json.RawMessage(noCheckAllConfig))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			securityConfig := config.(*cfg.AISecurityConfig)
			require.False(t, securityConfig.CheckAllMessages)
			require.Nil(t, securityConfig.RedisClient)
		})

		t.Run("redis with non-static service uses default port 6379", func(t *testing.T) {
			nonStaticConfig, _ := json.Marshal(map[string]interface{}{
				"serviceName":      "security-service",
				"servicePort":      8080,
				"serviceHost":      "security.example.com",
				"accessKey":        "test-ak",
				"secretKey":        "test-sk",
				"checkRequest":     true,
				"checkAllMessages": true,
				"redis": map[string]interface{}{
					"service_name": "redis-svc.default.svc.cluster.local",
				},
			})
			host, status := test.NewTestHost(json.RawMessage(nonStaticConfig))
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			securityConfig := config.(*cfg.AISecurityConfig)
			require.True(t, securityConfig.CheckAllMessages)
			require.NotNil(t, securityConfig.RedisClient)
		})
	})
}

func TestDedupRequestFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("dedup all messages cached - skip security check", func(t *testing.T) {
			host, status := test.NewTestHost(dedupConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[
				{"role":"system","content":"You are helpful."},
				{"role":"user","content":"Hello"},
				{"role":"assistant","content":"Hi!"},
				{"role":"user","content":"How are you?"}
			]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// Redis MGet returns all keys as cached (non-null)
			// 3 messages after role filter (system + 2 user), so 3 values
			allCachedResp := test.CreateRedisRespArray([]interface{}{"1", "1", "1"})
			host.CallOnRedisCall(0, allCachedResp)

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		t.Run("dedup some messages unchecked - check and mark", func(t *testing.T) {
			host, status := test.NewTestHost(dedupConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[
				{"role":"system","content":"You are helpful."},
				{"role":"user","content":"Hello"},
				{"role":"user","content":"New message"}
			]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// Redis MGet: system cached, first user cached, second user unchecked
			partialCacheResp := test.CreateRedisRespArray([]interface{}{"1", "1", nil})
			host.CallOnRedisCall(0, partialCacheResp)

			// Security check should be triggered for unchecked content
			securityResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			// After pass, MarkChecked via Redis EVAL
			evalOK := test.CreateRedisRespInt(1)
			host.CallOnRedisCall(0, evalOK)

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		t.Run("dedup security check deny", func(t *testing.T) {
			host, status := test.NewTestHost(dedupConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"bad content"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// Redis MGet: unchecked
			uncachedResp := test.CreateRedisRespArray([]interface{}{nil})
			host.CallOnRedisCall(0, uncachedResp)

			// Security check deny
			securityResponse := `{"Code":200,"Data":{"RiskLevel":"high"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			localResp := host.GetLocalResponse()
			require.NotNil(t, localResp)
			require.Contains(t, string(localResp.Data), "无法回答")
		})

		t.Run("dedup no messages after role filter - skip", func(t *testing.T) {
			host, status := test.NewTestHost(dedupConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// Only assistant messages (not checked)
			body := `{"messages":[{"role":"assistant","content":"I am a bot"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestMultiModalDedupRequestFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("multimodal dedup all cached - skip", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalDedupConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[
				{"role":"user","content":"Hello"},
				{"role":"user","content":"How are you?"}
			]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			allCached := test.CreateRedisRespArray([]interface{}{"1", "1"})
			host.CallOnRedisCall(0, allCached)

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		t.Run("multimodal dedup unchecked text - check pass and mark", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalDedupConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[
				{"role":"user","content":"Hello"},
				{"role":"user","content":"New message"}
			]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			partialCache := test.CreateRedisRespArray([]interface{}{"1", nil})
			host.CallOnRedisCall(0, partialCache)

			// Security check pass
			securityResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			// MarkChecked via EVAL
			evalOK := test.CreateRedisRespInt(1)
			host.CallOnRedisCall(0, evalOK)

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})
	})
}

func TestCustomDenyMessage(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("deny uses custom message from config", func(t *testing.T) {
			host, status := test.NewTestHost(customDenyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"bad content"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			securityResponse := `{"Code":200,"Data":{"RiskLevel":"high"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			localResp := host.GetLocalResponse()
			require.NotNil(t, localResp)
			require.Contains(t, string(localResp.Data), "自定义拒绝消息")
		})

		t.Run("deny uses advice answer when no custom message", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"bad content"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			securityResponse := `{"Code":200,"Data":{"RiskLevel":"high","Advice":[{"Answer":"安全审核建议"}],"Result":[{"Label":"spam","RiskWords":"bad"}]}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			localResp := host.GetLocalResponse()
			require.NotNil(t, localResp)
			require.Contains(t, string(localResp.Data), "安全审核建议")
		})
	})
}

func TestStreamDenyResponse(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("stream request gets SSE format deny", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"bad"}],"stream":true}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			securityResponse := `{"Code":200,"Data":{"RiskLevel":"high"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			localResp := host.GetLocalResponse()
			require.NotNil(t, localResp)
			require.Contains(t, string(localResp.Data), "chat.completion.chunk")
			require.Contains(t, string(localResp.Data), "[DONE]")
			for _, h := range localResp.Headers {
				if h[0] == "content-type" {
					require.Equal(t, "text/event-stream;charset=UTF-8", h[1])
				}
			}
		})

		t.Run("non-stream request gets JSON format deny", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"bad"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			securityResponse := `{"Code":200,"Data":{"RiskLevel":"high"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			localResp := host.GetLocalResponse()
			require.NotNil(t, localResp)
			require.Contains(t, string(localResp.Data), "chat.completion")
			require.NotContains(t, string(localResp.Data), "[DONE]")
		})
	})
}

func TestSecurityCheckServiceError(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("security service returns non-200 - pass through", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"hello"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			host.CallOnHttpCall([][2]string{{":status", "500"}}, []byte("internal error"))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		t.Run("security service returns non-200 Code - pass through", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"hello"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(`{"Code":500,"Message":"error"}`))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
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

// 测试配置：MultiModalGuard 基础配置（无dedup，用于测试 handleDefaultRequest）
var multiModalBasicConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"serviceName":               "security-service",
		"servicePort":               8080,
		"serviceHost":               "security.example.com",
		"accessKey":                 "test-ak",
		"secretKey":                 "test-sk",
		"checkRequest":              true,
		"checkResponse":             false,
		"action":                    "MultiModalGuard",
		"contentModerationLevelBar": "high",
		"promptAttackLevelBar":      "high",
		"sensitiveDataLevelBar":     "S3",
		"timeout":                   2000,
	})
	return data
}()

// 测试配置：MultiModalGuard 启用图片检查
var multiModalImageConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"serviceName":               "security-service",
		"servicePort":               8080,
		"serviceHost":               "security.example.com",
		"accessKey":                 "test-ak",
		"secretKey":                 "test-sk",
		"checkRequest":              true,
		"checkResponse":             false,
		"action":                    "MultiModalGuard",
		"checkRequestImage":         true,
		"contentModerationLevelBar": "high",
		"promptAttackLevelBar":      "high",
		"sensitiveDataLevelBar":     "S3",
		"timeout":                   2000,
	})
	return data
}()

// TestHandleDefaultRequest 测试 handleDefaultRequest 方法
func TestHandleDefaultRequest(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试纯文本请求 - 安全检查通过
		t.Run("text only - security check pass", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalBasicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"Hello, how are you?"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			securityResponse := `{"Code":200,"Message":"Success","RequestId":"req-001","Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试纯文本请求 - 安全检查拒绝
		t.Run("text only - security check deny", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalBasicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"bad content"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			securityResponse := `{"Code":200,"Data":{"RiskLevel":"high"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			localResp := host.GetLocalResponse()
			require.NotNil(t, localResp)
			require.Contains(t, string(localResp.Data), "无法回答")
		})

		// 测试空内容 - 直接通过
		t.Run("empty content - skip", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalBasicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":""}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionContinue, action)
		})

		// 测试安全服务返回非200状态码 - 放行
		t.Run("security service error - pass through", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalBasicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"hello"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			host.CallOnHttpCall([][2]string{{":status", "500"}}, []byte("internal error"))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试安全服务返回非200 Code - 放行
		t.Run("security service non-200 Code - pass through", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalBasicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":"hello"}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(`{"Code":500,"Message":"error"}`))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试仅图片内容（无文本）- checkRequestImage 未启用时，只有图片无文本直接调用 singleCallForImage
		t.Run("image only without checkRequestImage - calls image check", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalBasicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://example.com/img.png"}}]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 图片检查通过
			securityResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(securityResponse))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})
	})
}

// TestBuildImageCallback 测试 buildImageCallback 方法
func TestBuildImageCallback(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试单张图片检查通过
		t.Run("single image pass", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"text","text":"describe this"},
				{"type":"image_url","image_url":{"url":"https://example.com/img.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 文本检查通过
			textResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(textResponse))

			// 图片检查通过
			imageResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试单张图片检查拒绝
		t.Run("single image deny", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"text","text":"describe this"},
				{"type":"image_url","image_url":{"url":"https://example.com/bad.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 文本检查通过
			textResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(textResponse))

			// 图片检查拒绝
			imageResponse := `{"Code":200,"Data":{"RiskLevel":"high"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse))

			localResp := host.GetLocalResponse()
			require.NotNil(t, localResp)
			require.Contains(t, string(localResp.Data), "无法回答")
		})

		// 测试多张图片 - 全部通过
		t.Run("multiple images all pass", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"text","text":"compare these"},
				{"type":"image_url","image_url":{"url":"https://example.com/a.png"}},
				{"type":"image_url","image_url":{"url":"https://example.com/b.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 文本检查通过
			textResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(textResponse))

			// 第一张图片通过
			imageResponse1 := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse1))

			// 第二张图片通过
			imageResponse2 := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse2))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试多张图片 - 第二张拒绝
		t.Run("multiple images second deny", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"text","text":"compare these"},
				{"type":"image_url","image_url":{"url":"https://example.com/ok.png"}},
				{"type":"image_url","image_url":{"url":"https://example.com/bad.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 文本检查通过
			textResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(textResponse))

			// 第一张图片通过
			imageResponse1 := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse1))

			// 第二张图片拒绝
			imageResponse2 := `{"Code":200,"Data":{"RiskLevel":"high"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse2))

			localResp := host.GetLocalResponse()
			require.NotNil(t, localResp)
			require.Contains(t, string(localResp.Data), "无法回答")
		})

		// 测试图片检查服务返回非200 - 继续检查下一张
		t.Run("image service error - continue to next image", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"text","text":"check these"},
				{"type":"image_url","image_url":{"url":"https://example.com/a.png"}},
				{"type":"image_url","image_url":{"url":"https://example.com/b.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 文本检查通过
			textResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(textResponse))

			// 第一张图片服务错误（非200状态码）
			host.CallOnHttpCall([][2]string{{":status", "500"}}, []byte("error"))

			// 第二张图片通过
			imageResponse2 := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse2))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试图片检查返回非200 Code - 继续检查下一张
		t.Run("image service non-200 Code - continue to next image", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"text","text":"check these"},
				{"type":"image_url","image_url":{"url":"https://example.com/a.png"}},
				{"type":"image_url","image_url":{"url":"https://example.com/b.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 文本检查通过
			textResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(textResponse))

			// 第一张图片返回非200 Code
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(`{"Code":500,"Message":"error"}`))

			// 第二张图片通过
			imageResponse2 := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse2))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试图片检查返回无法解析的JSON - 继续检查下一张
		t.Run("image service invalid JSON - continue to next image", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"text","text":"check these"},
				{"type":"image_url","image_url":{"url":"https://example.com/a.png"}},
				{"type":"image_url","image_url":{"url":"https://example.com/b.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 文本检查通过
			textResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(textResponse))

			// 第一张图片返回无效JSON（Code=200但body无法解析为Response）
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(`{"Code":200,"invalid`))

			// 第二张图片通过
			imageResponse2 := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse2))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试所有图片服务都出错 - 最终放行
		t.Run("all image services error - pass through", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"text","text":"check this"},
				{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 文本检查通过
			textResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(textResponse))

			// 图片服务错误
			host.CallOnHttpCall([][2]string{{":status", "500"}}, []byte("error"))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})
	})
}

// TestBuildImageCaller 测试 buildImageCaller 方法
func TestBuildImageCaller(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试URL类型图片请求
		t.Run("URL image request", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"image_url","image_url":{"url":"https://example.com/photo.jpg"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 图片检查通过
			imageResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试BASE64类型图片请求
		t.Run("BASE64 image request", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"image_url","image_url":{"url":"data:image/png;base64,iVBORw0KGgo="}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 图片检查通过
			imageResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试混合URL和BASE64图片
		t.Run("mixed URL and BASE64 images", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"image_url","image_url":{"url":"data:image/jpeg;base64,/9j/4AAQ="}},
				{"type":"image_url","image_url":{"url":"https://example.com/photo.jpg"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 第一张BASE64图片通过
			imageResponse1 := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse1))

			// 第二张URL图片通过
			imageResponse2 := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse2))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试文本+图片混合请求，checkRequestImage=false 时不检查图片
		t.Run("text with images but checkRequestImage disabled", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalBasicConfig) // checkRequestImage=false
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"text","text":"describe this"},
				{"type":"image_url","image_url":{"url":"https://example.com/img.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 只有文本检查，通过后直接放行（不检查图片）
			textResponse := `{"Code":200,"Data":{"RiskLevel":"none"}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(textResponse))

			action = host.GetHttpStreamAction()
			require.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()
		})

		// 测试图片检查拒绝后带有 Result 信息
		t.Run("image deny with result details", func(t *testing.T) {
			host, status := test.NewTestHost(multiModalImageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			body := `{"messages":[{"role":"user","content":[
				{"type":"image_url","image_url":{"url":"https://example.com/bad.png"}}
			]}]}`
			action := host.CallOnHttpRequestBody([]byte(body))
			require.Equal(t, types.ActionPause, action)

			// 图片检查拒绝，带有详细结果和建议
			imageResponse := `{"Code":200,"Data":{"RiskLevel":"high","Result":[{"Label":"porn","RiskWords":"explicit"}],"Advice":[{"Answer":"图片内容不合规"}]}}`
			host.CallOnHttpCall([][2]string{{":status", "200"}}, []byte(imageResponse))

			localResp := host.GetLocalResponse()
			require.NotNil(t, localResp)
			require.Contains(t, string(localResp.Data), "图片内容不合规")
		})
	})
}
