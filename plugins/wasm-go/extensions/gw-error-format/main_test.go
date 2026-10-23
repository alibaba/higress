// Copyright (c) 2022 Alibaba Group Holding Ltd.
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

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本配置
var basicConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"match": map[string]interface{}{
					"statuscode":   "403",
					"responsebody": "RBAC: access denied",
				},
				"replace": map[string]interface{}{
					"statuscode":   "200",
					"responsebody": `{"code":401,"message":"User is not authenticated"}`,
				},
			},
		},
		"set_header": []map[string]interface{}{
			{"content-type": "application/json;charset=UTF-8"},
			{"custom-header": "test-value"},
		},
	})
	return data
}()

// 测试配置：多个规则配置
var multipleRulesConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"match": map[string]interface{}{
					"statuscode":   "403",
					"responsebody": "RBAC: access denied",
				},
				"replace": map[string]interface{}{
					"statuscode":   "200",
					"responsebody": `{"code":401,"message":"User is not authenticated"}`,
				},
			},
			{
				"match": map[string]interface{}{
					"statuscode":   "503",
					"responsebody": "no healthy upstream",
				},
				"replace": map[string]interface{}{
					"statuscode":   "200",
					"responsebody": `{"code":404,"message":"No Healthy Service"}`,
				},
			},
		},
		"set_header": []map[string]interface{}{
			{"content-type": "application/json;charset=UTF-8"},
			{"access-control-allow-origin": "*"},
			{"access-control-allow-methods": "GET,POST,PUT,DELETE"},
		},
	})
	return data
}()

// 测试配置：无效配置（缺少 match.statuscode）
var invalidConfigMissingStatusCode = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"match": map[string]interface{}{
					"responsebody": "RBAC: access denied",
					// 缺少 statuscode
				},
				"replace": map[string]interface{}{
					"statuscode":   "200",
					"responsebody": `{"code":401,"message":"User is not authenticated"}`,
				},
			},
		},
	})
	return data
}()

// 测试配置：无效配置（缺少 replace.statuscode）
var invalidConfigMissingReplaceStatusCode = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"match": map[string]interface{}{
					"statuscode":   "403",
					"responsebody": "RBAC: access denied",
				},
				"replace": map[string]interface{}{
					// 缺少 statuscode
					"responsebody": `{"code":401,"message":"User is not authenticated"}`,
				},
			},
		},
	})
	return data
}()

// 测试配置：空配置
var emptyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{})
	return data
}()

// 测试配置：只有规则，没有响应头
var rulesOnlyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"match": map[string]interface{}{
					"statuscode":   "403",
					"responsebody": "RBAC: access denied",
				},
				"replace": map[string]interface{}{
					"statuscode":   "200",
					"responsebody": `{"code":401,"message":"User is not authenticated"}`,
				},
			},
		},
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本配置解析
		t.Run("basic config", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试多个规则配置解析
		t.Run("multiple rules config", func(t *testing.T) {
			host, status := test.NewTestHost(multipleRulesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试无效配置 - 缺少 match.statuscode
		t.Run("invalid config - missing match.statuscode", func(t *testing.T) {
			host, status := test.NewTestHost(invalidConfigMissingStatusCode)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试无效配置 - 缺少 replace.statuscode
		t.Run("invalid config - missing replace.statuscode", func(t *testing.T) {
			host, status := test.NewTestHost(invalidConfigMissingReplaceStatusCode)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试空配置解析
		t.Run("empty config", func(t *testing.T) {
			host, status := test.NewTestHost(emptyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试只有规则的配置解析
		t.Run("rules only config", func(t *testing.T) {
			host, status := test.NewTestHost(rulesOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func TestOnHttpResponseHeader(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试状态码匹配 - 没有 x-envoy-upstream-service-time 头
		t.Run("status code match - no upstream service time header", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置响应头，状态码为 403，但没有 x-envoy-upstream-service-time 头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "403"},
				{"content-type", "text/plain"},
			})

			require.Equal(t, types.ActionContinue, action)

			// 验证状态码是否被替换
			responseHeaders := host.GetResponseHeaders()
			statusCodeFound := false
			for _, header := range responseHeaders {
				if header[0] == ":status" && header[1] == "200" {
					statusCodeFound = true
					break
				}
			}
			require.True(t, statusCodeFound, "Status code should be replaced to 200")

			// 验证自定义响应头是否被添加
			customHeaderFound := false
			contentTypeHeaderFound := false
			for _, header := range responseHeaders {
				if header[0] == "custom-header" && header[1] == "test-value" {
					customHeaderFound = true
				}
				if header[0] == "content-type" && header[1] == "application/json;charset=UTF-8" {
					contentTypeHeaderFound = true
				}
			}
			require.True(t, customHeaderFound, "Custom header should be added")
			require.True(t, contentTypeHeaderFound, "Content-Type header should be replaced")

			host.CompleteHttp()
		})

		// 测试状态码匹配 - 有 x-envoy-upstream-service-time 头（不生效）
		t.Run("status code match - with upstream service time header", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置响应头，状态码为 403，且有 x-envoy-upstream-service-time 头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "403"},
				{"content-type", "text/plain"},
				{"x-envoy-upstream-service-time", "123"},
			})

			require.Equal(t, types.ActionContinue, action)

			// 由于有 x-envoy-upstream-service-time 头，插件不应该生效
			// 状态码应该保持为 403
			responseHeaders := host.GetResponseHeaders()
			statusCodeFound := false
			for _, header := range responseHeaders {
				if header[0] == ":status" && header[1] == "403" {
					statusCodeFound = true
					break
				}
			}
			require.True(t, statusCodeFound, "Status code should remain 403 when upstream service time header exists")

			host.CompleteHttp()
		})

		// 测试状态码不匹配
		t.Run("status code no match", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置响应头，状态码为 404，不匹配规则
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "404"},
				{"content-type", "text/plain"},
			})

			require.Equal(t, types.ActionContinue, action)

			// 状态码应该保持为 404
			responseHeaders := host.GetResponseHeaders()
			statusCodeFound := false
			for _, header := range responseHeaders {
				if header[0] == ":status" && header[1] == "404" {
					statusCodeFound = true
					break
				}
			}
			require.True(t, statusCodeFound, "Status code should remain 404 when no rule matches")

			host.CompleteHttp()
		})

		// 测试多个规则配置
		t.Run("multiple rules config", func(t *testing.T) {
			host, status := test.NewTestHost(multipleRulesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 测试第一个规则：403 -> 200
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "403"},
				{"content-type", "text/plain"},
			})

			require.Equal(t, types.ActionContinue, action)

			// 验证状态码是否被替换
			responseHeaders := host.GetResponseHeaders()
			statusCodeFound := false
			for _, header := range responseHeaders {
				if header[0] == ":status" && header[1] == "200" {
					statusCodeFound = true
					break
				}
			}
			require.True(t, statusCodeFound, "Status code should be replaced to 200 for 403 match")

			host.CompleteHttp()
		})

		// 测试空配置
		t.Run("empty config", func(t *testing.T) {
			host, status := test.NewTestHost(emptyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "403"},
				{"content-type", "text/plain"},
			})

			require.Equal(t, types.ActionContinue, action)

			// 由于没有规则，状态码应该保持为 403
			responseHeaders := host.GetResponseHeaders()
			statusCodeFound := false
			for _, header := range responseHeaders {
				if header[0] == ":status" && header[1] == "403" {
					statusCodeFound = true
					break
				}
			}
			require.True(t, statusCodeFound, "Status code should remain 403 when no rules configured")

			host.CompleteHttp()
		})
	})
}

func TestOnHttpResponseBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试响应体匹配和替换
		t.Run("response body match and replace", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "403"},
				{"content-type", "text/plain"},
			})
			require.Equal(t, types.ActionContinue, action)

			// 处理响应体
			originalBody := []byte("RBAC: access denied")
			action = host.CallOnHttpResponseBody(originalBody)

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体是否被替换
			responseBody := host.GetResponseBody()
			expectedBody := `{"code":401,"message":"User is not authenticated"}`
			require.Equal(t, expectedBody, string(responseBody), "Response body should be replaced")

			host.CompleteHttp()
		})

		// 测试响应体不匹配
		t.Run("response body no match", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "403"},
				{"content-type", "text/plain"},
			})
			require.Equal(t, types.ActionContinue, action)

			// 处理不匹配的响应体
			originalBody := []byte("Different error message")
			action = host.CallOnHttpResponseBody(originalBody)

			require.Equal(t, types.ActionContinue, action)

			// 响应体应该保持不变
			responseBody := host.GetResponseBody()
			require.Equal(t, "Different error message", string(responseBody), "Response body should remain unchanged")

			host.CompleteHttp()
		})

		// 测试多个规则的响应体匹配
		t.Run("multiple rules response body match", func(t *testing.T) {
			host, status := test.NewTestHost(multipleRulesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "503"},
				{"content-type", "text/plain"},
			})
			require.Equal(t, types.ActionContinue, action)

			// 处理响应体
			originalBody := []byte("no healthy upstream")
			action = host.CallOnHttpResponseBody(originalBody)

			require.Equal(t, types.ActionContinue, action)

			// 验证响应体是否被替换
			responseBody := host.GetResponseBody()
			expectedBody := `{"code":404,"message":"No Healthy Service"}`
			require.Equal(t, expectedBody, string(responseBody), "Response body should be replaced for 503 match")

			host.CompleteHttp()
		})

		// 测试空配置的响应体处理
		t.Run("empty config response body", func(t *testing.T) {
			host, status := test.NewTestHost(emptyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "403"},
				{"content-type", "text/plain"},
			})
			require.Equal(t, types.ActionContinue, action)

			// 处理响应体
			originalBody := []byte("RBAC: access denied")
			action = host.CallOnHttpResponseBody(originalBody)

			require.Equal(t, types.ActionContinue, action)

			// 由于没有规则，响应体应该保持不变
			responseBody := host.GetResponseBody()
			require.Equal(t, "RBAC: access denied", string(responseBody), "Response body should remain unchanged when no rules configured")

			host.CompleteHttp()
		})
	})
}

func TestCompleteFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("complete response flow", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 1. 处理响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "403"},
				{"content-type", "text/plain"},
			})
			require.Equal(t, types.ActionContinue, action)

			// 2. 处理响应体
			originalBody := []byte("RBAC: access denied")
			action = host.CallOnHttpResponseBody(originalBody)
			require.Equal(t, types.ActionContinue, action)

			// 3. 验证完整的响应处理结果
			// 验证状态码
			responseHeaders := host.GetResponseHeaders()
			statusCodeFound := false
			for _, header := range responseHeaders {
				if header[0] == ":status" && header[1] == "200" {
					statusCodeFound = true
					break
				}
			}
			require.True(t, statusCodeFound, "Status code should be replaced to 200")

			// 验证响应体
			responseBody := host.GetResponseBody()
			expectedBody := `{"code":401,"message":"User is not authenticated"}`
			require.Equal(t, expectedBody, string(responseBody), "Response body should be replaced")

			// 验证自定义响应头
			customHeaderFound := false
			for _, header := range responseHeaders {
				if header[0] == "custom-header" && header[1] == "test-value" {
					customHeaderFound = true
					break
				}
			}
			require.True(t, customHeaderFound, "Custom header should be added")

			host.CompleteHttp()
		})
	})
}
