// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package server

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/protocol"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/utils"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	wasmtest "github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type validationTestServer struct {
	base BaseMCPServer
}

func newValidationTestServer() *validationTestServer {
	return &validationTestServer{base: NewBaseMCPServer()}
}

func (s *validationTestServer) AddMCPTool(name string, tool Tool) Server {
	s.base.AddMCPTool(name, tool)
	return s
}

func (s *validationTestServer) GetMCPTools() map[string]Tool { return s.base.GetMCPTools() }
func (s *validationTestServer) SetConfig(config []byte)      { s.base.SetConfig(config) }
func (s *validationTestServer) GetConfig(value any)          { s.base.GetConfig(value) }
func (s *validationTestServer) Clone() Server {
	return &validationTestServer{base: s.base.CloneBase()}
}

type validationToolCounters struct {
	create int
	call   int
}

type validationTestTool struct {
	counters *validationToolCounters
	schema   map[string]any
	callErr  error
}

type validationOutputTool struct {
	*validationTestTool
}

func (t *validationOutputTool) OutputSchema() map[string]any {
	return map[string]any{"type": "object"}
}

func (t *validationTestTool) Create(_ []byte) Tool {
	t.counters.create++
	return &validationTestTool{counters: t.counters, schema: t.schema, callErr: t.callErr}
}

func (t *validationTestTool) Call(ctx HttpContext, _ Server) error {
	t.counters.call++
	if t.callErr != nil {
		return t.callErr
	}
	utils.SendMCPToolTextResult(ctx, "called")
	return nil
}

func (t *validationTestTool) Description() string         { return "validation test tool" }
func (t *validationTestTool) InputSchema() map[string]any { return t.schema }

func modernToolHeaders(name string) [][2]string {
	return [][2]string{
		{":authority", "mcp.example.com"},
		{":method", "POST"},
		{":path", "/mcp"},
		{"content-type", "application/json"},
		{"accept", "application/json, text/event-stream"},
		{protocol.HeaderProtocolVersion, string(protocol.Version20260728)},
		{protocol.HeaderMethod, "tools/call"},
		{protocol.HeaderName, name},
	}
}

func legacyToolHeaders() [][2]string {
	return [][2]string{
		{":authority", "mcp.example.com"},
		{":method", "POST"},
		{":path", "/mcp"},
		{"content-type", "application/json"},
		{"accept", "application/json, text/event-stream"},
	}
}

func modernToolCallBody(name, arguments string) []byte {
	return []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + name + `","arguments":` + arguments + `,"_meta":{"` +
		protocol.MetaProtocolVersion + `":"2026-07-28","` + protocol.MetaClientCapabilities + `":{}}}}`)
}

func newValidationHost(t *testing.T, config json.RawMessage) wasmtest.TestHost {
	t.Helper()
	host, status := wasmtest.NewTestHost(config)
	require.Equal(t, types.OnPluginStartStatusOK, status)
	t.Cleanup(host.Reset)
	return host
}

func installRegisteredValidationServer(t *testing.T, counters *validationToolCounters) {
	t.Helper()
	savedGlobalContext := globalContext
	globalContext = Context{servers: make(map[string]Server)}
	server := newValidationTestServer()
	server.AddMCPTool("count", &validationTestTool{
		counters: counters,
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"count": map[string]any{"type": "integer"},
			},
			"required":             []any{"count"},
			"additionalProperties": false,
		},
	})
	Load(AddMCPServer("registered", server))
	Initialize()
	t.Cleanup(func() { globalContext = savedGlobalContext })
}

func TestModernRegisteredToolValidatesBeforeCreateAndCall(t *testing.T) {
	counters := &validationToolCounters{}
	installRegisteredValidationServer(t, counters)
	host := newValidationHost(t, json.RawMessage(`{"server":{"name":"registered"}}`))

	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(modernToolHeaders("count")))
	require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody(modernToolCallBody("count", `{"count":"wrong","token":"secret-value"}`)))
	response := host.GetLocalResponse()
	require.NotNil(t, response)
	assert.Equal(t, uint32(200), response.StatusCode)
	assert.False(t, gjson.GetBytes(response.Data, "error").Exists())
	assert.True(t, gjson.GetBytes(response.Data, "result.isError").Bool())
	assert.Equal(t, resultTypeComplete, gjson.GetBytes(response.Data, "result.resultType").String())
	assert.Equal(t, "registered", gjson.GetBytes(response.Data, "result._meta.io\\.modelcontextprotocol/serverInfo.name").String())
	assert.NotContains(t, string(response.Data), "secret-value")
	assert.Zero(t, counters.create)
	assert.Zero(t, counters.call)
}

func TestModernRegisteredToolInvokesAfterValidation(t *testing.T) {
	counters := &validationToolCounters{}
	installRegisteredValidationServer(t, counters)
	host := newValidationHost(t, json.RawMessage(`{"server":{"name":"registered"}}`))

	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(modernToolHeaders("count")))
	require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody(modernToolCallBody("count", `{"count":2}`)))
	response := host.GetLocalResponse()
	require.NotNil(t, response)
	assert.False(t, gjson.GetBytes(response.Data, "result.isError").Bool())
	assert.Equal(t, 1, counters.create)
	assert.Equal(t, 1, counters.call)
}

func TestModernRegisteredToolFailureIsToolExecutionError(t *testing.T) {
	savedGlobalContext := globalContext
	globalContext = Context{servers: make(map[string]Server)}
	counters := &validationToolCounters{}
	registered := newValidationTestServer()
	registered.AddMCPTool("fail", &validationTestTool{
		counters: counters,
		schema:   map[string]any{"type": "object"},
		callErr:  errors.New("tool failed"),
	})
	Load(AddMCPServer("registered", registered))
	Initialize()
	t.Cleanup(func() { globalContext = savedGlobalContext })
	host := newValidationHost(t, json.RawMessage(`{"server":{"name":"registered"}}`))

	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(modernToolHeaders("fail")))
	require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody(modernToolCallBody("fail", `{}`)))
	response := host.GetLocalResponse()
	require.NotNil(t, response)
	assert.Equal(t, uint32(200), response.StatusCode)
	assert.False(t, gjson.GetBytes(response.Data, "error").Exists())
	assert.True(t, gjson.GetBytes(response.Data, "result.isError").Bool())
	assert.Equal(t, "tool failed", gjson.GetBytes(response.Data, "result.content.0.text").String())
	assert.Equal(t, resultTypeComplete, gjson.GetBytes(response.Data, "result.resultType").String())
	assert.Equal(t, 1, counters.create)
	assert.Equal(t, 1, counters.call)
}

func TestLegacyRegisteredToolKeepsExistingArgumentBehavior(t *testing.T) {
	counters := &validationToolCounters{}
	installRegisteredValidationServer(t, counters)
	host := newValidationHost(t, json.RawMessage(`{"server":{"name":"registered"}}`))

	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(legacyToolHeaders()))
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"count","arguments":{"count":"legacy"}}}`)
	require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody(body))
	response := host.GetLocalResponse()
	require.NotNil(t, response)
	assert.False(t, gjson.GetBytes(response.Data, "result.isError").Bool())
	assert.False(t, gjson.GetBytes(response.Data, "result.resultType").Exists())
	assert.Equal(t, 1, counters.create)
	assert.Equal(t, 1, counters.call)
}

func TestModernRESTToolValidationPreventsUpstreamInvocation(t *testing.T) {
	savedGlobalContext := globalContext
	globalContext = Context{servers: make(map[string]Server)}
	Initialize()
	t.Cleanup(func() { globalContext = savedGlobalContext })

	config := json.RawMessage(`{
		"server":{"name":"rest"},
		"tools":[{
			"name":"lookup",
			"description":"lookup",
			"args":[{"name":"count","description":"count","type":"integer","required":true}],
			"requestTemplate":{"url":"http://backend.example/items","method":"POST","argsToJsonBody":true}
		}]
	}`)
	host := newValidationHost(t, config)
	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(modernToolHeaders("lookup")))
	require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody(modernToolCallBody("lookup", `{"count":"wrong"}`)))
	response := host.GetLocalResponse()
	require.NotNil(t, response)
	assert.True(t, gjson.GetBytes(response.Data, "result.isError").Bool())
	assert.Empty(t, host.GetHttpCalloutAttributes())
}

func TestModernRESTToolAcceptsValidArguments(t *testing.T) {
	savedGlobalContext := globalContext
	globalContext = Context{servers: make(map[string]Server)}
	Initialize()
	t.Cleanup(func() { globalContext = savedGlobalContext })

	config := json.RawMessage(`{
		"server":{"name":"rest"},
		"tools":[{
			"name":"lookup",
			"description":"lookup",
			"args":[{"name":"count","description":"count","type":"integer","required":true}],
			"requestTemplate":{"url":"http://backend.example/items","method":"POST","argsToJsonBody":true}
		}]
	}`)
	host := newValidationHost(t, config)
	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(modernToolHeaders("lookup")))
	require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody(modernToolCallBody("lookup", `{"count":2}`)))
	assert.Nil(t, host.GetLocalResponse(), "valid arguments must not be rejected at the protocol boundary")
}

func TestDirectToolSnapshotsKeepProfileCompatibleDescriptors(t *testing.T) {
	inputSchema := map[string]any{"type": "object"}
	newTool := func() Tool {
		return &validationOutputTool{&validationTestTool{
			counters: &validationToolCounters{},
			schema:   inputSchema,
		}}
	}

	registered := newValidationTestServer()
	registered.AddMCPTool("zeta", newTool())
	registered.AddMCPTool("alpha", newTool())

	rest := NewRestMCPServer("rest")
	for _, name := range []string{"zeta", "alpha"} {
		require.NoError(t, rest.AddRestTool(RestTool{
			Name:            name,
			Description:     name,
			OutputSchema:    map[string]any{"type": "object"},
			RequestTemplate: RestToolRequestTemplate{URL: "/" + name, Method: "GET"},
		}))
	}

	registry := &GlobalToolRegistry{}
	registry.Initialize()
	registry.RegisterTool("registered", "zeta", newTool())
	registry.RegisterTool("registered", "alpha", newTool())
	composed := NewComposedMCPServer("composed", []ServerToolConfig{{
		ServerName: "registered",
		Tools:      []string{"zeta", "alpha"},
	}}, registry)

	tests := []struct {
		name      string
		server    Server
		wantNames []string
	}{
		{name: "registered", server: registered, wantNames: []string{"alpha", "zeta"}},
		{name: "rest", server: rest, wantNames: []string{"alpha", "zeta"}},
		{name: "composed", server: composed, wantNames: []string{"registered___alpha", "registered___zeta"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot, err := compileDirectToolSnapshot(test.server)
			require.NoError(t, err)
			modern := snapshot.buildModernToolList(nil)
			require.Len(t, modern, 2)
			for i, descriptor := range modern {
				assert.Equal(t, test.wantNames[i], descriptor["name"])
				assert.Equal(t, "object", descriptor["inputSchema"].(map[string]any)["type"])
				assert.NotContains(t, descriptor, "outputSchema")
			}

			legacy := buildToolList(test.server, nil, true)
			require.Len(t, legacy, 2)
			for _, descriptor := range legacy {
				assert.Contains(t, descriptor, "outputSchema")
			}
		})
	}
}

func TestDirectToolSchemasAreRejectedBeforeServing(t *testing.T) {
	unsupported := map[string]any{
		"type":  "object",
		"oneOf": []any{map[string]any{"type": "object"}},
	}

	t.Run("registered", func(t *testing.T) {
		registered := newValidationTestServer()
		registered.AddMCPTool("bad", &validationTestTool{counters: &validationToolCounters{}, schema: unsupported})
		config := &McpServerConfig{}
		opts := &ConfigOptions{
			Servers:      map[string]Server{"registered": registered},
			ToolRegistry: &GlobalToolRegistry{},
		}
		opts.ToolRegistry.Initialize()
		err := ParseConfigCore(gjson.Parse(`{"server":{"name":"registered"}}`), config, opts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unsupported schema keyword "oneOf"`)
	})

	t.Run("rest", func(t *testing.T) {
		rest := NewRestMCPServer("rest")
		err := rest.AddRestTool(RestTool{
			Name:        "bad",
			Description: "bad",
			Args: []RestToolArg{{
				Name:        "items",
				Description: "items",
				Type:        "array",
				Items:       map[string]any{"oneOf": []any{}},
			}},
			RequestTemplate: RestToolRequestTemplate{URL: "/items", Method: "POST"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unsupported schema keyword "oneOf"`)
		assert.Empty(t, rest.GetMCPTools())
	})

	t.Run("composed", func(t *testing.T) {
		registry := &GlobalToolRegistry{}
		registry.Initialize()
		registry.RegisterTool("registered", "bad", &validationTestTool{counters: &validationToolCounters{}, schema: unsupported})
		config := &McpServerConfig{}
		opts := &ConfigOptions{Servers: map[string]Server{}, ToolRegistry: registry}
		err := ParseConfigCore(gjson.Parse(`{
			"toolSet":{"name":"composed","serverTools":[{"serverName":"registered","tools":["bad"]}]}
		}`), config, opts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unsupported schema keyword "oneOf"`)
	})
}
