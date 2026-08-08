// Copyright (c) 2026 Alibaba Group Holding Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

func testConfig(t *testing.T, legacy bool) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]interface{}{
		"protocolVersion": "1.0",
		"mode":            "enforce",
		"legacy03":        map[string]interface{}{"enabled": legacy},
		"agent":           map[string]interface{}{"id": "weather-agent"},
		"jsonrpc": map[string]interface{}{
			"maxRequestBytes": 4096,
		},
	})
	require.NoError(t, err)
	return data
}

func TestTrustedHeadersReplaceSpoofedInput(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(testConfig(t, false))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.InitHttp()
		require.Equal(t, types.ActionContinue, host.CallOnHttpRequestHeaders([][2]string{
			{":method", "POST"}, {":path", "/a2a"}, {":authority", "agent.example.com"},
			{"content-type", "application/a2a+json"}, {"a2a-version", "1.0"},
			{"x-higress-a2a-method", "CancelTask"}, {"x-higress-a2a-task-id", "spoofed"},
		}))
		require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody([]byte(`{"jsonrpc":"2.0","id":"r1","method":"GetTask","params":{"id":"real-task"}}`)))
		headers := map[string]string{}
		for _, pair := range host.GetRequestHeaders() {
			headers[pair[0]] = pair[1]
		}
		require.Equal(t, "GetTask", headers["x-higress-a2a-method"])
		require.Equal(t, "real-task", headers["x-higress-a2a-task-id"])
		require.Equal(t, "weather-agent", headers["x-higress-a2a-agent-id"])
		method, err := host.GetProperty([]string{"a2a", "method"})
		require.NoError(t, err)
		require.Equal(t, "GetTask", string(method))
		host.CompleteHttp()
	})
}

func TestLegacyAliasAndStrictUnknownMethod(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(testConfig(t, true))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.InitHttp()
		host.CallOnHttpRequestHeaders([][2]string{{":method", "POST"}, {":path", "/a2a"}, {":authority", "agent.example.com"}, {"content-type", "application/json"}})
		require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody([]byte(`{"jsonrpc":"2.0","id":1,"method":"tasks/cancel","params":{"id":"t1"}}`)))
		headers := map[string]string{}
		for _, pair := range host.GetRequestHeaders() {
			headers[pair[0]] = pair[1]
		}
		require.Equal(t, "CancelTask", headers["x-higress-a2a-method"])
		require.Equal(t, "legacy", headers["x-higress-a2a-parse-status"])
		host.CompleteHttp()
	})

	test.RunGoTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(testConfig(t, false))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.InitHttp()
		host.CallOnHttpRequestHeaders([][2]string{{":method", "POST"}, {":path", "/a2a"}, {":authority", "agent.example.com"}, {"content-type", "application/a2a+json"}, {"a2a-version", "1.0"}})
		require.Equal(t, types.ActionPause, host.CallOnHttpRequestBody([]byte(`{"jsonrpc":"2.0","id":1,"method":"unknown"}`)))
		host.CompleteHttp()
	})
}

func TestNonA2ARequestPassesWithoutTrustedHeaders(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(testConfig(t, false))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.InitHttp()
		require.Equal(t, types.ActionContinue, host.CallOnHttpRequestHeaders([][2]string{{":method", "GET"}, {":path", "/health"}, {":authority", "agent.example.com"}, {"x-higress-a2a-method", "spoofed"}}))
		for _, pair := range host.GetRequestHeaders() {
			require.NotEqual(t, "x-higress-a2a-method", pair[0])
		}
		host.CompleteHttp()
	})
}
