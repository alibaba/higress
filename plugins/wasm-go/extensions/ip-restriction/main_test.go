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

// 测试配置：白名单模式
var allowConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"ip_source_type": "origin-source",
		"allow":          []string{"192.168.1.0/24", "10.0.0.1"},
		"status":         403,
		"message":        "Access denied",
	})
	return data
}()

// 测试配置：黑名单模式
var denyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"ip_source_type": "header",
		"ip_header_name": "X-Real-IP",
		"deny":           []string{"192.168.2.0/24", "10.0.0.2"},
		"status":         429,
		"message":        "IP blocked",
	})
	return data
}()

// 测试配置：使用默认值
var defaultConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"allow": []string{"127.0.0.1"},
	})
	return data
}()

// 测试配置：无效配置（同时设置 allow 和 deny）
var invalidConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"allow": []string{"127.0.0.1"},
		"deny":  []string{"192.168.1.1"},
	})
	return data
}()

// 测试配置：空配置（没有 allow 和 deny）
var emptyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"ip_source_type": "origin-source",
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试白名单配置
		t.Run("allow list config", func(t *testing.T) {
			host, status := test.NewTestHost(allowConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			restrictionConfig := config.(*RestrictionConfig)
			require.Equal(t, "origin-source", restrictionConfig.IPSourceType)
			require.Equal(t, "X-Forwarded-For", restrictionConfig.IPHeaderName) // 默认值
			require.NotNil(t, restrictionConfig.Allow)
			require.Nil(t, restrictionConfig.Deny)
			require.Equal(t, uint32(403), restrictionConfig.Status)
			require.Equal(t, "Access denied", restrictionConfig.Message)
		})

		// 测试黑名单配置
		t.Run("deny list config", func(t *testing.T) {
			host, status := test.NewTestHost(denyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			restrictionConfig := config.(*RestrictionConfig)
			require.Equal(t, "header", restrictionConfig.IPSourceType)
			require.Equal(t, "X-Real-IP", restrictionConfig.IPHeaderName)
			require.Nil(t, restrictionConfig.Allow)
			require.NotNil(t, restrictionConfig.Deny)
			require.Equal(t, uint32(429), restrictionConfig.Status)
			require.Equal(t, "IP blocked", restrictionConfig.Message)
		})

		// 测试默认配置
		t.Run("default config", func(t *testing.T) {
			host, status := test.NewTestHost(defaultConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			restrictionConfig := config.(*RestrictionConfig)
			require.Equal(t, "origin-source", restrictionConfig.IPSourceType)   // 默认值
			require.Equal(t, "X-Forwarded-For", restrictionConfig.IPHeaderName) // 默认值
			require.NotNil(t, restrictionConfig.Allow)
			require.Nil(t, restrictionConfig.Deny)
			require.Equal(t, uint32(403), restrictionConfig.Status)                    // 默认值
			require.Equal(t, "Your IP address is blocked.", restrictionConfig.Message) // 默认值
		})

		// 测试无效配置（同时设置 allow 和 deny）
		t.Run("invalid config - both allow and deny", func(t *testing.T) {
			host, status := test.NewTestHost(invalidConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试空配置（没有 allow 和 deny）
		t.Run("empty config - no allow or deny", func(t *testing.T) {
			host, status := test.NewTestHost(emptyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试白名单模式 - IP 在白名单中（应该通过）
		t.Run("allow list - IP allowed", func(t *testing.T) {
			host, status := test.NewTestHost(allowConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置源 IP 地址（在白名单中）
			host.SetProperty([]string{"source", "address"}, []byte("192.168.1.100:8080"))

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.Nil(t, localResponse, "IP in allow list should pass through")

			host.CompleteHttp()
		})

		// 测试白名单模式 - IP 不在白名单中（应该被阻止）
		t.Run("allow list - IP not allowed", func(t *testing.T) {
			host, status := test.NewTestHost(allowConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置源 IP 地址（不在白名单中）
			host.SetProperty([]string{"source", "address"}, []byte("192.168.2.100:8080"))

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.NotNil(t, localResponse)
			require.Equal(t, uint32(403), localResponse.StatusCode)

			// 验证 JSON 响应格式
			var responseData map[string]string
			err := json.Unmarshal(localResponse.Data, &responseData)
			require.NoError(t, err)
			require.Equal(t, "Access denied", responseData["message"])

			host.CompleteHttp()
		})

		// 测试黑名单模式 - IP 在黑名单中（应该被阻止）
		t.Run("deny list - IP denied", func(t *testing.T) {
			host, status := test.NewTestHost(denyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
				{"X-Real-IP", "192.168.2.100"}, // IP 在黑名单中
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.NotNil(t, localResponse)
			require.Equal(t, uint32(429), localResponse.StatusCode)

			// 验证 JSON 响应格式
			var responseData map[string]string
			err := json.Unmarshal(localResponse.Data, &responseData)
			require.NoError(t, err)
			require.Equal(t, "IP blocked", responseData["message"])

			host.CompleteHttp()
		})

		// 测试黑名单模式 - IP 不在黑名单中（应该通过）
		t.Run("deny list - IP not denied", func(t *testing.T) {
			host, status := test.NewTestHost(denyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
				{"X-Real-IP", "192.168.3.100"}, // IP 不在黑名单中
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.Nil(t, localResponse, "IP not in deny list should pass through")

			host.CompleteHttp()
		})

		// 测试从请求头获取 IP - 多个 IP 的情况
		t.Run("header source - multiple IPs", func(t *testing.T) {
			host, status := test.NewTestHost(denyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
				{"X-Real-IP", "192.168.3.100, 10.0.0.1, 172.16.0.1"}, // 多个 IP，取第一个
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.Nil(t, localResponse, "First IP not in deny list should pass through")

			host.CompleteHttp()
		})

		// 测试无效 IP 地址
		t.Run("invalid IP address", func(t *testing.T) {
			host, status := test.NewTestHost(allowConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置无效的源 IP 地址
			host.SetProperty([]string{"source", "address"}, []byte("invalid-ip:8080"))

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.NotNil(t, localResponse)
			require.Equal(t, uint32(403), localResponse.StatusCode)

			host.CompleteHttp()
		})

		// 测试 IPv6 地址
		t.Run("IPv6 address", func(t *testing.T) {
			host, status := test.NewTestHost(allowConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置 IPv6 源地址
			host.SetProperty([]string{"source", "address"}, []byte("[2001:db8::1]:8080"))

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/test"},
				{":method", "GET"},
			})

			require.Equal(t, types.ActionContinue, action)
			require.Equal(t, types.ActionContinue, host.GetHttpStreamAction())

			localResponse := host.GetLocalResponse()
			require.NotNil(t, localResponse) // IPv6 不在白名单中，应该被阻止
			require.Equal(t, uint32(403), localResponse.StatusCode)

			host.CompleteHttp()
		})
	})
}

func TestParseIP(t *testing.T) {
	// 测试 parseIP 函数
	t.Run("IPv4 address", func(t *testing.T) {
		result := parseIP("192.168.1.100:8080", false)
		require.Equal(t, "192.168.1.100", result)
	})

	t.Run("IPv4 address without port", func(t *testing.T) {
		result := parseIP("192.168.1.100", false)
		require.Equal(t, "192.168.1.100", result)
	})

	t.Run("IPv6 address with port", func(t *testing.T) {
		result := parseIP("[2001:db8::1]:8080", false)
		require.Equal(t, "2001:db8::1", result)
	})

	t.Run("IPv6 address without port", func(t *testing.T) {
		result := parseIP("[2001:db8::1]", false)
		require.Equal(t, "2001:db8::1", result)
	})

	t.Run("IP from header - multiple IPs", func(t *testing.T) {
		result := parseIP("192.168.1.100, 10.0.0.1, 172.16.0.1", true)
		require.Equal(t, "192.168.1.100", result)
	})

	t.Run("IP from header - single IP", func(t *testing.T) {
		result := parseIP("192.168.1.100", true)
		require.Equal(t, "192.168.1.100", result)
	})

	t.Run("IP with spaces", func(t *testing.T) {
		result := parseIP("  192.168.1.100  ", false)
		require.Equal(t, "192.168.1.100", result)
	})

	t.Run("empty IP", func(t *testing.T) {
		result := parseIP("", false)
		require.Equal(t, "", result)
	})
}
