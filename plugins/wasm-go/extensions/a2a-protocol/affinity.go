// Copyright (c) 2026 Alibaba Group Holding Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/a2a"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

// Consumed by the route-scoped, strict Envoy stateful_session filter. Never
// accept a client-provided endpoint or fall back to non-strict WASM selection.
const affinityHeader = "x-higress-a2a-affinity-endpoint"

type affinityConfig struct {
	Enabled    bool `json:"enabled"`
	TTLSeconds int  `json:"ttlSeconds"`
	Redis      struct {
		ServiceFQDN string `json:"serviceFQDN"`
		ServicePort int64  `json:"servicePort"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		Timeout     int64  `json:"timeout"`
	} `json:"redis"`
	client wrapper.RedisClient
}

func (c *affinityConfig) init() error {
	if !c.Enabled {
		return nil
	}
	if c.TTLSeconds == 0 {
		c.TTLSeconds = 3600
	}
	if c.TTLSeconds < 1 || c.TTLSeconds > 86400 {
		return fmt.Errorf("affinity.ttlSeconds must be between 1 and 86400")
	}
	if c.Redis.ServiceFQDN == "" || c.Redis.ServicePort < 1 || c.Redis.ServicePort > 65535 {
		return fmt.Errorf("affinity requires a Redis serviceFQDN and servicePort")
	}
	if c.Redis.Timeout == 0 {
		c.Redis.Timeout = 1000
	}
	if c.Redis.Timeout < 1 || c.Redis.Timeout > 10000 {
		return fmt.Errorf("affinity.redis.timeout must be between 1 and 10000 ms")
	}
	c.client = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{FQDN: c.Redis.ServiceFQDN, Port: c.Redis.ServicePort})
	return c.client.Init(c.Redis.Username, c.Redis.Password, c.Redis.Timeout, wrapper.WithDisableBuffer())
}

// All aliases must already exist when supplied by a client. An expired context
// cannot be distinguished from lost process state: reject rather than remap it.
const affinityLookup = `local host = false
for _, key in ipairs(KEYS) do
 local value = redis.call('GET', key)
 if not value then return redis.error_reply('A2A_BINDING_MISSING') end
 if host and host ~= value then return redis.error_reply('A2A_BINDING_CONFLICT') end
 host = value
end
for _, key in ipairs(KEYS) do redis.call('EXPIRE', key, ARGV[1]) end
return host`

// Preflight every key before writing anything, including aliases created in a
// different gateway worker. Never replace an existing binding with another host.
const affinityBind = `for _, key in ipairs(KEYS) do
 local value = redis.call('GET', key)
 if value and value ~= ARGV[1] then return redis.error_reply('A2A_BINDING_CONFLICT') end
end
for _, key in ipairs(KEYS) do redis.call('SET', key, ARGV[1], 'EX', ARGV[2]) end
return 'OK'`

type affinityState struct {
	scope          string
	host           string
	requestContext string
	pending        []byte
	busy           bool
	ended          bool
	trailers       bool
	failed         bool
}

func bindingKeys(scope string, metas ...a2a.Metadata) []interface{} {
	keys := []interface{}{}
	seen := map[string]bool{}
	for _, m := range metas {
		for kind, id := range map[string]string{"task": m.TaskID, "context": m.ContextID} {
			if id == "" {
				continue
			}
			key := fmt.Sprintf("a2a:{%s}:%s:%x", scope, kind, sha256.Sum256([]byte(id)))
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func affinityError(ctx wrapper.HttpContext, message string) types.Action {
	// Deliberately independent of protocol audit mode: a missing binding must not
	// turn a stateful operation into a best-effort request to an arbitrary Agent.
	body, _ := json.Marshal(map[string]interface{}{"jsonrpc": "2.0", "id": ctx.GetContext("a2a.affinity.request_id"), "error": map[string]interface{}{"code": -32010, "message": message}})
	_ = proxywasm.SendHttpResponse(503, [][2]string{{"content-type", "application/json"}}, body, -1)
	return types.ActionPause
}

func healthyAffinityHost(host string) bool {
	hosts, err := proxywasm.GetUpstreamHosts()
	if err != nil {
		return false
	}
	for _, h := range hosts {
		if h[0] == host && gjson.Get(h[1], "health_status").String() == "Healthy" {
			return true
		}
	}
	return false
}

func routeAffinity(ctx wrapper.HttpContext, c pluginConfig, m a2a.Metadata, body []byte) types.Action {
	if !c.Affinity.Enabled {
		return types.ActionContinue
	}
	// Preserve the JSON-RPC ID's type in failures.
	var request struct {
		ID json.RawMessage `json:"id"`
	}
	_ = json.Unmarshal(body, &request)
	ctx.SetContext("a2a.affinity.request_id", request.ID)
	if !affinityIDsValid(m) {
		return affinityError(ctx, "A2A affinity identifier exceeds limit")
	}
	route, e1 := proxywasm.GetProperty([]string{"route_name"})
	cluster, e2 := proxywasm.GetProperty([]string{"cluster_name"})
	if e1 != nil || e2 != nil || len(route) == 0 || len(cluster) == 0 {
		return affinityError(ctx, "A2A affinity route unavailable")
	}
	scope := fmt.Sprintf("%x", sha256.Sum256([]byte(c.Agent.ID+"\x00"+string(route)+"\x00"+string(cluster))))
	state := &affinityState{scope: scope, requestContext: m.ContextID}
	ctx.SetContext("a2a.affinity", state)
	selectHost := func(host string) bool {
		if !healthyAffinityHost(host) {
			affinityError(ctx, "A2A bound endpoint unavailable")
			return false
		}
		if err := proxywasm.ReplaceHttpRequestHeader(affinityHeader, base64.StdEncoding.EncodeToString([]byte(host))); err != nil {
			affinityError(ctx, "A2A affinity routing failed")
			return false
		}
		state.host = host
		return true
	}
	keys := bindingKeys(scope, m)
	if len(keys) == 0 {
		if m.Method != "SendMessage" && m.Method != "SendStreamingMessage" {
			return affinityError(ctx, "A2A task or context binding required")
		}
		hosts, err := proxywasm.GetUpstreamHosts()
		if err != nil {
			return affinityError(ctx, "A2A endpoints unavailable")
		}
		healthy := []string{}
		for _, h := range hosts {
			if gjson.Get(h[1], "health_status").String() == "Healthy" {
				healthy = append(healthy, h[0])
			}
		}
		if len(healthy) == 0 {
			return affinityError(ctx, "A2A endpoints unavailable")
		}
		if !selectHost(healthy[rand.Intn(len(healthy))]) {
			return types.ActionPause
		}
		return types.ActionContinue
	}
	err := c.Affinity.client.Eval(affinityLookup, len(keys), keys, []interface{}{c.Affinity.TTLSeconds}, func(v resp.Value) {
		if v.Error() != nil || v.IsNull() {
			affinityError(ctx, "A2A binding unavailable or inconsistent")
			return
		}
		if selectHost(v.String()) {
			_ = proxywasm.ResumeHttpRequest()
		}
	})
	if err != nil {
		return affinityError(ctx, "A2A affinity store unavailable")
	}
	return types.ActionPause
}

func bindAffinity(ctx wrapper.HttpContext, c pluginConfig, metas []a2a.Metadata, done func(bool)) {
	state, ok := ctx.GetContext("a2a.affinity").(*affinityState)
	if !ok {
		done(false)
		return
	}
	actual, err := proxywasm.GetProperty([]string{"upstream", "address"})
	if err != nil || string(actual) != state.host {
		done(false)
		return
	}
	for _, m := range metas {
		if !affinityIDsValid(m) {
			done(false)
			return
		}
		if state.requestContext != "" && m.ContextID != "" && m.ContextID != state.requestContext {
			done(false)
			return
		}
	}
	keys := bindingKeys(state.scope, metas...)
	err = c.Affinity.client.Eval(affinityBind, len(keys), keys, []interface{}{state.host, c.Affinity.TTLSeconds}, func(v resp.Value) { done(v.Error() == nil && v.String() == "OK") })
	if err != nil {
		done(false)
	}
}

func bindUnaryAffinity(ctx wrapper.HttpContext, c pluginConfig, m a2a.Metadata) types.Action {
	bindAffinity(ctx, c, []a2a.Metadata{m}, func(ok bool) {
		if !ok {
			affinityError(ctx, "A2A response binding failed")
			return
		}
		_ = proxywasm.ResumeHttpResponse()
	})
	return types.ActionPause
}

// Return the last complete SSE event boundary, preserving the original bytes.
func completeSSEPrefix(data []byte) int {
	end := 0
	for _, sep := range [][]byte{[]byte("\n\n"), []byte("\r\n\r\n"), []byte("\r\r")} {
		if i := bytes.LastIndex(data, sep); i >= 0 && i+len(sep) > end {
			end = i + len(sep)
		}
	}
	return end
}

func streamAffinity(ctx wrapper.HttpContext, c pluginConfig, data []byte, end bool) []byte {
	s, ok := ctx.GetContext("a2a.affinity").(*affinityState)
	if !ok {
		return nil
	}
	if s.failed {
		return nil
	}
	if len(s.pending)+len(data) > c.JSONRPC.MaxSSEEventBytes {
		failAffinityStream(s)
		return nil
	}
	s.pending = append(s.pending, data...)
	s.ended = s.ended || end
	pumpAffinityStream(ctx, c, s)
	return nil
}
func failAffinityStream(s *affinityState) {
	if s.failed {
		return
	}
	s.failed = true
	s.pending = nil
	_ = proxywasm.InjectEncodedDataToFilterChain([]byte("data: {\"jsonrpc\":\"2.0\",\"id\":null,\"error\":{\"code\":-32010,\"message\":\"A2A stream binding failed\"}}\n\n"), true)
}
func pumpAffinityStream(ctx wrapper.HttpContext, c pluginConfig, s *affinityState) {
	if s.busy || s.failed {
		return
	}
	n := completeSSEPrefix(s.pending)
	if s.ended || s.trailers {
		n = len(s.pending)
	}
	if n == 0 {
		if s.trailers {
			_ = proxywasm.ResumeHttpResponse()
		} else if s.ended {
			_ = proxywasm.InjectEncodedDataToFilterChain(nil, true)
		}
		return
	}
	data := bytes.Clone(s.pending[:n])
	s.pending = bytes.Clone(s.pending[n:])
	s.busy = true
	events := a2a.NewSSEParser(c.JSONRPC.MaxSSEEventBytes).Feed(data, true, ctx.GetStringContext("a2a.version", c.ProtocolVersion), ctx.GetStringContext("a2a.method", ""))
	metas := []a2a.Metadata{}
	for _, event := range events {
		if event.Oversized || event.Metadata.ParseStatus == "invalid" {
			s.busy = false
			failAffinityStream(s)
			return
		}
		publishMetadata(ctx, c, event.Metadata, false, false)
		metas = append(metas, event.Metadata)
	}
	bindAffinity(ctx, c, metas, func(ok bool) {
		s.busy = false
		if s.failed {
			return
		}
		if !ok {
			failAffinityStream(s)
			return
		}
		final := s.ended && len(s.pending) == 0 && !s.trailers
		_ = proxywasm.InjectEncodedDataToFilterChain(data, final)
		if !final {
			pumpAffinityStream(ctx, c, s)
		}
	})
}
func onAffinityTrailers(ctx wrapper.HttpContext, c pluginConfig) types.Action {
	if !c.Affinity.Enabled || ctx.GetContext("a2a.sse") == nil {
		return types.ActionContinue
	}
	s, ok := ctx.GetContext("a2a.affinity").(*affinityState)
	if !ok {
		return types.ActionContinue
	}
	if !s.busy && len(s.pending) == 0 {
		return types.ActionContinue
	}
	s.trailers = true
	pumpAffinityStream(ctx, c, s)
	return types.ActionPause
}

// Protocol metadata truncates diagnostic fields. Never route on a truncated
// identifier: reject the boundary value too, so different IDs cannot alias.
func affinityIDsValid(m a2a.Metadata) bool {
	return len(m.TaskID) < a2a.MaxMetadataValueBytes && len(m.ContextID) < a2a.MaxMetadataValueBytes
}
