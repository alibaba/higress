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

// 测试配置：记录所有信息
var logAllConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"log_request_headers":  true,
		"log_request_body":     true,
		"log_response_headers": true,
		"log_response_body":    true,
	})
	return data
}()

// 测试配置：只记录请求头
var logRequestHeadersOnlyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"log_request_headers":  true,
		"log_request_body":     false,
		"log_response_headers": false,
		"log_response_body":    false,
	})
	return data
}()

// 测试配置：禁用所有记录
var logDisabledConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"log_request_headers":  false,
		"log_request_body":     false,
		"log_response_headers": false,
		"log_response_body":    false,
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试完整配置解析
		t.Run("log all config", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			pluginConfig := config.(*Config)
			require.True(t, pluginConfig.LogRequestHeaders)
			require.True(t, pluginConfig.LogRequestBody)
			require.True(t, pluginConfig.LogResponseHeaders)
			require.True(t, pluginConfig.LogResponseBody)
		})

		// 测试部分配置解析
		t.Run("log request headers only config", func(t *testing.T) {
			host, status := test.NewTestHost(logRequestHeadersOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			pluginConfig := config.(*Config)
			require.True(t, pluginConfig.LogRequestHeaders)
			require.False(t, pluginConfig.LogRequestBody)
			require.False(t, pluginConfig.LogResponseHeaders)
			require.False(t, pluginConfig.LogResponseBody)
		})

		// 测试禁用配置
		t.Run("log disabled config", func(t *testing.T) {
			host, status := test.NewTestHost(logDisabledConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			pluginConfig := config.(*Config)
			require.False(t, pluginConfig.LogRequestHeaders)
			require.False(t, pluginConfig.LogRequestBody)
			require.False(t, pluginConfig.LogResponseHeaders)
			require.False(t, pluginConfig.LogResponseBody)
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试请求头日志记录 - 启用
		t.Run("request headers logging enabled", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			headers := [][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{"content-type", "application/json"},
				{"user-agent", "test-agent"},
			}

			action := host.CallOnHttpRequestHeaders(headers)
			require.Equal(t, types.ActionContinue, action)

			// 验证日志是否包含请求头信息
			logs := host.GetInfoLogs()
			found := false
			for _, log := range logs {
				if strings.Contains(log, "request Headers: [") {
					found = true
					// 验证日志包含关键头信息
					require.Contains(t, log, ":method=POST")
					require.Contains(t, log, ":path=/test")
					require.Contains(t, log, "content-type=application/json")
					break
				}
			}
			require.True(t, found, "Should log request headers")
		})

		// 测试请求头日志记录 - 禁用
		t.Run("request headers logging disabled", func(t *testing.T) {
			host, status := test.NewTestHost(logDisabledConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			headers := [][2]string{
				{":authority", "example.com"},
				{":method", "GET"},
				{":path", "/test"},
				{"content-type", "application/json"},
			}

			action := host.CallOnHttpRequestHeaders(headers)
			require.Equal(t, types.ActionContinue, action)

			// 验证日志不包含请求头信息
			logs := host.GetInfoLogs()
			for _, log := range logs {
				require.NotContains(t, log, "request Headers:")
			}
		})

		// 测试 Content-Type 保存到 context
		t.Run("content-type saved to context", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			headers := [][2]string{
				{":method", "POST"},
				{"content-type", "application/json"},
			}

			host.CallOnHttpRequestHeaders(headers)
			action := host.CallOnHttpRequestHeaders(headers)
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestOnHttpRequestBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试请求体日志记录 - JSON 内容
		t.Run("request body logging - JSON content", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头（需要包含必需的 pseudo-headers）
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{"content-type", "application/json"},
			})

			// 调用请求体处理
			body := []byte(`{"name": "test", "message": "hello world"}`)
			action := host.CallOnHttpRequestBody(body)
			require.Equal(t, types.ActionContinue, action)

			// 验证日志包含请求体
			logs := host.GetInfoLogs()
			found := false
			for _, log := range logs {
				if strings.Contains(log, "request Body: [") {
					found = true
					require.Contains(t, log, `{"name": "test", "message": "hello world"}`)
					break
				}
			}
			require.True(t, found, "Should log request body")
		})

		// 测试请求体日志记录 - 表单内容
		t.Run("request body logging - form content", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头（需要包含必需的 pseudo-headers）
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{"content-type", "application/x-www-form-urlencoded"},
			})

			// 调用请求体处理
			body := []byte("name=test&message=hello")
			action := host.CallOnHttpRequestBody(body)
			require.Equal(t, types.ActionContinue, action)

			// 验证日志包含请求体
			logs := host.GetInfoLogs()
			found := false
			for _, log := range logs {
				if strings.Contains(log, "request Body: [") {
					found = true
					require.Contains(t, log, "name=test&message=hello")
					break
				}
			}
			require.True(t, found, "Should log request body")
		})

		// 测试请求体日志记录 - 不支持的 Content-Type
		t.Run("request body logging - unsupported content type", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头（需要包含必需的 pseudo-headers）
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{"content-type", "image/png"},
			})

			// 调用请求体处理
			body := []byte("binary data")
			action := host.CallOnHttpRequestBody(body)
			require.Equal(t, types.ActionContinue, action)

			// 验证不记录不支持的 content-type
			logs := host.GetInfoLogs()
			for _, log := range logs {
				require.NotContains(t, log, "request Body:")
			}
		})

		// 测试请求体日志记录 - 禁用
		t.Run("request body logging - disabled", func(t *testing.T) {
			host, status := test.NewTestHost(logRequestHeadersOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头（需要包含必需的 pseudo-headers）
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{"content-type", "application/json"},
			})

			// 调用请求体处理
			body := []byte(`{"test": "data"}`)
			action := host.CallOnHttpRequestBody(body)
			require.Equal(t, types.ActionContinue, action)

			// 验证不记录请求体
			logs := host.GetInfoLogs()
			for _, log := range logs {
				require.NotContains(t, log, "request Body:")
			}
		})

		// 测试请求体大小限制
		t.Run("request body - size limit", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头（需要包含必需的 pseudo-headers）
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":method", "POST"},
				{":path", "/test"},
				{"content-type", "text/plain"},
			})

			// 创建超过 1KB 的数据
			largeBody := []byte(strings.Repeat("a", 1500))
			action := host.CallOnHttpRequestBody(largeBody)
			require.Equal(t, types.ActionContinue, action)

			// 验证数据被截断
			logs := host.GetInfoLogs()
			found := false
			for _, log := range logs {
				if strings.Contains(log, "request Body: [") {
					found = true
					require.Contains(t, log, "<truncated>")
					break
				}
			}
			require.True(t, found, "Should log truncated body")
		})
	})
}

func TestOnHttpResponseHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试响应头日志记录 - 启用
		t.Run("response headers logging enabled", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			headers := [][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"content-length", "123"},
				{"server", "test-server"},
			}

			action := host.CallOnHttpResponseHeaders(headers)
			require.Equal(t, types.ActionContinue, action)

			// 验证日志包含响应头信息
			logs := host.GetInfoLogs()
			found := false
			for _, log := range logs {
				if strings.Contains(log, "response Headers: [") {
					found = true
					require.Contains(t, log, ":status=200")
					require.Contains(t, log, "content-type=application/json")
					break
				}
			}
			require.True(t, found, "Should log response headers")
		})

		// 测试响应头日志记录 - 禁用
		t.Run("response headers logging disabled", func(t *testing.T) {
			host, status := test.NewTestHost(logRequestHeadersOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			headers := [][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			}

			action := host.CallOnHttpResponseHeaders(headers)
			require.Equal(t, types.ActionContinue, action)

			// 验证不记录响应头
			logs := host.GetInfoLogs()
			for _, log := range logs {
				require.NotContains(t, log, "response Headers:")
			}
		})

		// 测试 Content-Encoding 检查
		t.Run("response headers - content encoding", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			headers := [][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"content-encoding", "gzip"},
			}

			action := host.CallOnHttpResponseHeaders(headers)
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})
	})
}

func TestOnHttpResponseBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试响应体日志记录 - JSON 内容
		t.Run("response body logging - JSON content", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 调用响应体处理
			body := []byte(`{"status": "success", "message": "ok"}`)
			action := host.CallOnHttpResponseBody(body)
			require.Equal(t, types.ActionContinue, action)

			// 验证日志包含响应体
			logs := host.GetInfoLogs()
			found := false
			for _, log := range logs {
				if strings.Contains(log, "response Body: [") {
					found = true
					require.Contains(t, log, `{"status": "success", "message": "ok"}`)
					break
				}
			}
			require.True(t, found, "Should log response body")
		})

		// 测试响应体日志记录 - 带有 Content-Encoding（不记录压缩内容）
		t.Run("response body logging - content encoding", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置响应头（带 content-encoding）
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"content-encoding", "gzip"},
			})

			// 调用响应体处理
			body := []byte(`{"data": "compressed"}`)
			action := host.CallOnHttpResponseBody(body)
			require.Equal(t, types.ActionContinue, action)

			// 验证不记录压缩内容
			logs := host.GetInfoLogs()
			for _, log := range logs {
				require.NotContains(t, log, "response Body:")
			}
		})

		// 测试响应体日志记录 - 不支持的 Content-Type
		t.Run("response body logging - unsupported content type", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "image/png"},
			})

			// 调用响应体处理
			body := []byte("binary data")
			action := host.CallOnHttpResponseBody(body)
			require.Equal(t, types.ActionContinue, action)

			// 验证不记录不支持的 content-type
			logs := host.GetInfoLogs()
			for _, log := range logs {
				require.NotContains(t, log, "response Body:")
			}
		})

		// 测试响应体日志记录 - 禁用
		t.Run("response body logging - disabled", func(t *testing.T) {
			host, status := test.NewTestHost(logRequestHeadersOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 调用响应体处理
			body := []byte(`{"status": "success"}`)
			action := host.CallOnHttpResponseBody(body)
			require.Equal(t, types.ActionContinue, action)

			// 验证不记录响应体
			logs := host.GetInfoLogs()
			for _, log := range logs {
				require.NotContains(t, log, "response Body:")
			}
		})

		// 测试响应体大小限制
		t.Run("response body - size limit", func(t *testing.T) {
			host, status := test.NewTestHost(logAllConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/plain"},
			})

			// 创建超过 1KB 的数据
			largeBody := []byte(strings.Repeat("b", 1500))
			action := host.CallOnHttpResponseBody(largeBody)
			require.Equal(t, types.ActionContinue, action)

			// 验证数据被截断
			logs := host.GetInfoLogs()
			found := false
			for _, log := range logs {
				if strings.Contains(log, "response Body: [") {
					found = true
					require.Contains(t, log, "<truncated>")
					break
				}
			}
			require.True(t, found, "Should log truncated body")
		})
	})
}
