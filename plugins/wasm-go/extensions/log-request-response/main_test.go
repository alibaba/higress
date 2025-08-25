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
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本配置 - 只启用请求头部日志
var basicConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"request": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": true,
			},
			"body": map[string]interface{}{
				"enabled": false,
			},
		},
		"response": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": false,
			},
			"body": map[string]interface{}{
				"enabled": false,
			},
		},
	})
	return data
}()

// 测试配置：完整配置 - 启用所有日志功能
var fullConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"request": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": true,
			},
			"body": map[string]interface{}{
				"enabled":      true,
				"maxSize":      1024,
				"contentTypes": []string{"application/json", "text/plain"},
			},
		},
		"response": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": true,
			},
			"body": map[string]interface{}{
				"enabled":      true,
				"maxSize":      2048,
				"contentTypes": []string{"application/json", "text/html"},
			},
		},
	})
	return data
}()

// 测试配置：自定义内容类型配置
var customContentTypesConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"request": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": true,
			},
			"body": map[string]interface{}{
				"enabled":      true,
				"maxSize":      512,
				"contentTypes": []string{"application/xml", "text/csv"},
			},
		},
		"response": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": true,
			},
			"body": map[string]interface{}{
				"enabled":      true,
				"maxSize":      512,
				"contentTypes": []string{"application/xml", "text/csv"},
			},
		},
	})
	return data
}()

// 测试配置：大文件配置 - 测试大小限制
var largeFileConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"request": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": true,
			},
			"body": map[string]interface{}{
				"enabled":      true,
				"maxSize":      100,
				"contentTypes": []string{"text/plain"},
			},
		},
		"response": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": true,
			},
			"body": map[string]interface{}{
				"enabled":      true,
				"maxSize":      100,
				"contentTypes": []string{"text/plain"},
			},
		},
	})
	return data
}()

// 测试配置：默认值配置 - 不指定 maxSize 和 contentTypes
var defaultValuesConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"request": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": true,
			},
			"body": map[string]interface{}{
				"enabled": true,
			},
		},
		"response": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": true,
			},
			"body": map[string]interface{}{
				"enabled": true,
			},
		},
	})
	return data
}()

// 测试配置：最小配置 - 只启用必要的功能
var minimalConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"request": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": false,
			},
			"body": map[string]interface{}{
				"enabled": false,
			},
		},
		"response": map[string]interface{}{
			"headers": map[string]interface{}{
				"enabled": false,
			},
			"body": map[string]interface{}{
				"enabled": false,
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

			pluginConfig := config.(*PluginConfig)
			require.True(t, pluginConfig.Request.Headers.Enabled)
			require.False(t, pluginConfig.Request.Body.Enabled)
			require.False(t, pluginConfig.Response.Headers.Enabled)
			require.False(t, pluginConfig.Response.Body.Enabled)
		})

		// 测试完整配置解析
		t.Run("full config", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			pluginConfig := config.(*PluginConfig)
			require.True(t, pluginConfig.Request.Headers.Enabled)
			require.True(t, pluginConfig.Request.Body.Enabled)
			require.Equal(t, 1024, pluginConfig.Request.Body.MaxSize)
			require.Len(t, pluginConfig.Request.Body.ContentTypes, 2)
			require.Equal(t, "application/json", pluginConfig.Request.Body.ContentTypes[0])
			require.Equal(t, "text/plain", pluginConfig.Request.Body.ContentTypes[1])

			require.True(t, pluginConfig.Response.Headers.Enabled)
			require.True(t, pluginConfig.Response.Body.Enabled)
			require.Equal(t, 2048, pluginConfig.Response.Body.MaxSize)
			require.Len(t, pluginConfig.Response.Body.ContentTypes, 2)
			require.Equal(t, "application/json", pluginConfig.Response.Body.ContentTypes[0])
			require.Equal(t, "text/html", pluginConfig.Response.Body.ContentTypes[1])
		})

		// 测试自定义内容类型配置
		t.Run("custom content types config", func(t *testing.T) {
			host, status := test.NewTestHost(customContentTypesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			pluginConfig := config.(*PluginConfig)
			require.Len(t, pluginConfig.Request.Body.ContentTypes, 2)
			require.Equal(t, "application/xml", pluginConfig.Request.Body.ContentTypes[0])
			require.Equal(t, "text/csv", pluginConfig.Request.Body.ContentTypes[1])

			require.Len(t, pluginConfig.Response.Body.ContentTypes, 2)
			require.Equal(t, "application/xml", pluginConfig.Response.Body.ContentTypes[0])
			require.Equal(t, "text/csv", pluginConfig.Response.Body.ContentTypes[1])
		})

		// 测试大文件配置
		t.Run("large file config", func(t *testing.T) {
			host, status := test.NewTestHost(largeFileConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			pluginConfig := config.(*PluginConfig)
			require.Equal(t, 100, pluginConfig.Request.Body.MaxSize)
			require.Equal(t, 100, pluginConfig.Response.Body.MaxSize)
		})

		// 测试默认值配置
		t.Run("default values config", func(t *testing.T) {
			host, status := test.NewTestHost(defaultValuesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			pluginConfig := config.(*PluginConfig)
			// 默认 maxSize 应该是 10KB
			require.Equal(t, 10*1024, pluginConfig.Request.Body.MaxSize)
			require.Equal(t, 10*1024, pluginConfig.Response.Body.MaxSize)

			// 默认内容类型
			require.Len(t, pluginConfig.Request.Body.ContentTypes, 4)
			require.Contains(t, pluginConfig.Request.Body.ContentTypes, "application/json")
			require.Contains(t, pluginConfig.Request.Body.ContentTypes, "text/plain")

			require.Len(t, pluginConfig.Response.Body.ContentTypes, 4)
			require.Contains(t, pluginConfig.Response.Body.ContentTypes, "application/json")
			require.Contains(t, pluginConfig.Response.Body.ContentTypes, "text/html")
		})

		// 测试最小配置
		t.Run("minimal config", func(t *testing.T) {
			host, status := test.NewTestHost(minimalConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			pluginConfig := config.(*PluginConfig)
			require.False(t, pluginConfig.Request.Headers.Enabled)
			require.False(t, pluginConfig.Request.Body.Enabled)
			require.False(t, pluginConfig.Response.Headers.Enabled)
			require.False(t, pluginConfig.Response.Body.Enabled)
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试请求头部日志 - 启用
		t.Run("request headers logging enabled", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "GET"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "application/json"},
				{"user-agent", "test-agent"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试请求头部日志 - 禁用
		t.Run("request headers logging disabled", func(t *testing.T) {
			host, status := test.NewTestHost(minimalConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "GET"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "application/json"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试请求体日志 - POST 请求，内容类型匹配
		t.Run("request body logging enabled - POST with matching content type", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "application/json"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试请求体日志 - POST 请求，内容类型不匹配
		t.Run("request body logging enabled - POST with non-matching content type", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "image/png"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试请求体日志 - GET 请求（不应该读取请求体）
		t.Run("request body logging enabled - GET request", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "GET"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "application/json"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试请求体日志 - PUT 请求，内容类型匹配
		t.Run("request body logging enabled - PUT with matching content type", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "PUT"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "text/plain"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试请求体日志 - PATCH 请求，内容类型匹配
		t.Run("request body logging enabled - PATCH with matching content type", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "PATCH"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "application/json"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})
	})
}

func TestOnHttpResponseHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试响应头部日志 - 启用
		t.Run("response headers logging enabled", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"content-length", "123"},
				{"server", "test-server"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试响应头部日志 - 禁用
		t.Run("response headers logging disabled", func(t *testing.T) {
			host, status := test.NewTestHost(minimalConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"content-length", "123"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试响应体日志 - 内容类型匹配
		t.Run("response body logging enabled - matching content type", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"content-length", "123"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试响应体日志 - 内容类型不匹配
		t.Run("response body logging enabled - non-matching content type", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "image/png"},
				{"content-length", "123"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})

		// 测试响应体日志 - 没有 content-type
		t.Run("response body logging enabled - no content type", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-length", "123"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			host.CompleteHttp()
		})
	})
}

func TestOnStreamingRequestBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试流式请求体处理 - 小数据
		t.Run("streaming request body - small data", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "application/json"},
			})

			// 测试流式请求体
			testData := []byte(`{"key": "value"}`)
			action := host.CallOnHttpStreamingRequestBody(testData, true)
			require.Equal(t, types.ActionContinue, action)
			result := host.GetRequestBody()
			require.Equal(t, testData, result, "Request body should be returned unchanged")
		})

		// 测试流式请求体处理 - 大数据（超过限制）
		t.Run("streaming request body - large data", func(t *testing.T) {
			host, status := test.NewTestHost(largeFileConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "text/plain"},
			})

			// 测试大数据（超过 100 字节限制）
			largeData := []byte(strings.Repeat("a", 200))
			action := host.CallOnHttpStreamingRequestBody(largeData, true)
			require.Equal(t, types.ActionContinue, action)
			result := host.GetRequestBody()
			require.Equal(t, largeData, result, "Request body should be returned unchanged even if large")
			host.CompleteHttp()
		})

		// 测试流式请求体处理 - 禁用
		t.Run("streaming request body - disabled", func(t *testing.T) {
			host, status := test.NewTestHost(minimalConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "application/json"},
			})

			// 测试流式请求体
			testData := []byte(`{"key": "value"}`)
			action := host.CallOnHttpStreamingRequestBody(testData, true)
			require.Equal(t, types.ActionContinue, action)
			result := host.GetRequestBody()
			require.Equal(t, testData, result, "Request body should be returned unchanged when disabled")
			host.CompleteHttp()
		})
	})
}

func TestOnStreamingResponseBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试流式响应体处理 - 小数据
		t.Run("streaming response body - small data", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"content-length", "123"},
			})

			// 测试流式响应体
			testData := []byte(`{"status": "success"}`)
			action := host.CallOnHttpStreamingResponseBody(testData, true)
			require.Equal(t, types.ActionContinue, action)
			result := host.GetResponseBody()
			require.Equal(t, testData, result, "Response body should be returned unchanged")
		})

		// 测试流式响应体处理 - 大数据（超过限制）
		t.Run("streaming response body - large data", func(t *testing.T) {
			host, status := test.NewTestHost(largeFileConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/plain"},
				{"content-length", "123"},
			})

			// 测试大数据（超过 100 字节限制）
			largeData := []byte(strings.Repeat("b", 200))
			action := host.CallOnHttpStreamingResponseBody(largeData, true)
			require.Equal(t, types.ActionContinue, action)
			result := host.GetResponseBody()
			require.Equal(t, largeData, result, "Response body should be returned unchanged even if large")
			host.CompleteHttp()
		})

		// 测试流式响应体处理 - 禁用
		t.Run("streaming response body - disabled", func(t *testing.T) {
			host, status := test.NewTestHost(minimalConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"content-length", "123"},
			})

			// 测试流式响应体
			testData := []byte(`{"status": "success"}`)
			action := host.CallOnHttpStreamingResponseBody(testData, true)
			require.Equal(t, types.ActionContinue, action)
			result := host.GetResponseBody()
			require.Equal(t, testData, result, "Response body should be returned unchanged when disabled")
			host.CompleteHttp()
		})
	})
}

func TestCompleteFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试完整的请求-响应流程
		t.Run("complete request-response flow", func(t *testing.T) {
			host, status := test.NewTestHost(fullConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 1. 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{":scheme", "https"},
				{"content-type", "application/json"},
				{"user-agent", "test-agent"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			// 2. 处理请求体
			requestBody := []byte(`{"name": "test", "value": "data"}`)
			action = host.CallOnHttpStreamingRequestBody(requestBody, true)
			require.Equal(t, types.ActionContinue, action)
			body := host.GetRequestBody()
			require.Equal(t, requestBody, body)

			// 3. 处理响应头
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"content-length", "45"},
				{"server", "test-server"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			// 4. 处理响应体
			responseBody := []byte(`{"status": "success", "message": "ok"}`)
			action = host.CallOnHttpStreamingResponseBody(responseBody, true)
			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())
			responseBodyResult := host.GetResponseBody()
			require.Equal(t, responseBody, responseBodyResult)

			host.CompleteHttp()
		})
	})
}
