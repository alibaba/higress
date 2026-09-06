// Copyright (c) 2026 Alibaba Group Holding Ltd.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/alibaba/higress/plugins/wasm-go/pkg/a2a"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"strings"
	"testing"
)

func TestAffinityKeysDoNotAlias(t *testing.T) {
	task := bindingKeys("route-a", a2a.Metadata{TaskID: "same"})
	context := bindingKeys("route-a", a2a.Metadata{ContextID: "same"})
	other := bindingKeys("route-b", a2a.Metadata{TaskID: "same"})
	require.NotEqual(t, task, context)
	require.NotEqual(t, task, other)
	require.Len(t, bindingKeys("route-a", a2a.Metadata{TaskID: "same"}, a2a.Metadata{TaskID: "same"}), 1)
	require.NotContains(t, task[0], "same")
	require.False(t, affinityIDsValid(a2a.Metadata{TaskID: strings.Repeat("a", 256)}))
	require.True(t, affinityIDsValid(a2a.Metadata{ContextID: strings.Repeat("a", 255)}))
}
func TestAffinitySSEBoundaries(t *testing.T) {
	for _, sep := range []string{"\n\n", "\r\n\r\n", "\r\r"} {
		data := "data: {}" + sep + "data: {\"partial"
		require.Equal(t, len("data: {}"+sep), completeSSEPrefix([]byte(data)))
	}
	require.Zero(t, completeSSEPrefix([]byte("data: incomplete\n")))
}
func TestAffinityEndpointHeaderIsNeverClientControlled(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(testConfig(t, true))
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.InitHttp()
		host.CallOnHttpRequestHeaders([][2]string{{":method", "GET"}, {":path", "/"}, {":authority", "agent.example.com"}, {affinityHeader, "attacker"}})
		require.Empty(t, headerMap(host.GetRequestHeaders())[affinityHeader])
		host.CompleteHttp()
	})
}

func TestAffinityRequiresEnforcementAndStore(t *testing.T) {
	var cfg pluginConfig
	require.ErrorContains(t, parseConfig(gjson.Parse(`{"mode":"audit","agent":{"id":"a"},"affinity":{"enabled":true}}`), &cfg), "requires enforce")
	cfg = pluginConfig{}
	require.ErrorContains(t, parseConfig(gjson.Parse(`{"agent":{"id":"a"},"affinity":{"enabled":true}}`), &cfg), "requires a Redis")
}
