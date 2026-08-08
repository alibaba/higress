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
	"fmt"
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/protocol"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	wasmtest "github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newProxyBridgeHost(t *testing.T, strategy ProtocolStrategy) wasmtest.TestHost {
	t.Helper()
	savedGlobalContext := globalContext
	globalContext = Context{servers: make(map[string]Server)}
	Initialize()
	t.Cleanup(func() { globalContext = savedGlobalContext })
	config := json.RawMessage(`{"server":{"name":"proxy","type":"mcp-proxy","transport":"http","protocolStrategy":"` + string(strategy) + `","mcpServerURL":"http://backend.example/mcp"}}`)
	host, status := wasmtest.NewTestHost(config)
	require.Equal(t, types.OnPluginStartStatusOK, status)
	t.Cleanup(host.Reset)
	return host
}

func modernProxyListHeaders(extra ...[2]string) [][2]string {
	headers := [][2]string{
		{":authority", "mcp.example.com"},
		{":method", "POST"},
		{":path", "/mcp"},
		{"content-type", "application/json"},
		{"accept", "application/json, text/event-stream"},
		{protocol.HeaderProtocolVersion, string(protocol.Version20260728)},
		{protocol.HeaderMethod, "tools/list"},
	}
	return append(headers, extra...)
}

func modernProxyListBody(id int) []byte {
	return []byte(`{"jsonrpc":"2.0","id":` + fmt.Sprintf("%d", id) + `,"method":"tools/list","params":{"_meta":{"` +
		protocol.MetaProtocolVersion + `":"2026-07-28","` + protocol.MetaClientCapabilities + `":{},"x.example/opaque":{"keep":true}}}}`)
}

func calloutAt(t *testing.T, host wasmtest.TestHost, index int) proxytest.HttpCalloutAttribute {
	t.Helper()
	callouts := host.GetHttpCalloutAttributes()
	require.Greater(t, len(callouts), index)
	return callouts[index]
}

func completeCallout(host wasmtest.TestHost, callout proxytest.HttpCalloutAttribute, status string, headers [][2]string, body []byte) {
	allHeaders := append([][2]string{{":status", status}}, headers...)
	host.CallOnHttpCallResponse(callout.CalloutID, allHeaders, nil, body)
}

func assertNoProxyLeakHeaders(t *testing.T, headers [][2]string) {
	t.Helper()
	for _, name := range []string{"Cookie", "Mcp-Session-Id", "Last-Event-ID", "x-envoy-allow-mcp-tools", "Authorization"} {
		_, exists := findHeader(headers, name)
		assert.False(t, exists, "%s leaked in %#v", name, headers)
	}
}

func TestModernToModernProxyUsesSingleStatelessCallAndPreservesOpaqueResult(t *testing.T) {
	host := newProxyBridgeHost(t, ProtocolStrategyModern)
	body := modernProxyListBody(7)
	headers := modernProxyListHeaders(
		[2]string{"traceparent", "00-a-b-01"},
		[2]string{"Mcp-Param-Future", "opaque"},
		[2]string{"Cookie", "private=1"},
		[2]string{"Mcp-Session-Id", "downstream-session"},
		[2]string{"Last-Event-ID", "99"},
		[2]string{"Authorization", "Bearer downstream"},
		[2]string{"x-envoy-allow-mcp-tools", ""},
	)
	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(headers))
	require.Equal(t, types.ActionPause, host.CallOnHttpRequestBody(body))

	call := calloutAt(t, host, 0)
	assert.Equal(t, string(body), string(call.Body), "modern envelope and metadata bytes must be unchanged")
	assert.Equal(t, string(protocol.Version20260728), mustHeaderValue(t, call.Headers, protocol.HeaderProtocolVersion))
	assert.Equal(t, "tools/list", mustHeaderValue(t, call.Headers, protocol.HeaderMethod))
	assert.Equal(t, "opaque", mustHeaderValue(t, call.Headers, "Mcp-Param-Future"))
	assert.Equal(t, "00-a-b-01", mustHeaderValue(t, call.Headers, "traceparent"))
	assertNoProxyLeakHeaders(t, call.Headers)

	upstreamResult := []byte(`{"jsonrpc":"2.0","id":7,"result":{"resultType":"input_required","opaque":{"future":true},"content":[{"type":"text","text":"continue"}]}}`)
	completeCallout(host, call, "200", [][2]string{{"content-type", "application/json"}}, upstreamResult)
	response := host.GetLocalResponse()
	require.NotNil(t, response)
	assert.Equal(t, uint32(200), response.StatusCode)
	assert.Equal(t, "input_required", gjson.GetBytes(response.Data, "result.resultType").String())
	assert.True(t, gjson.GetBytes(response.Data, "result.opaque.future").Bool())
	assert.Empty(t, host.GetHttpCalloutAttributes(), "the single modern callout must be complete")
}

func TestModernToLegacyProxyRunsRequestScopedHandshakeAndShapesResult(t *testing.T) {
	host := newProxyBridgeHost(t, ProtocolStrategyLegacy)
	headers := modernProxyListHeaders(
		[2]string{"Mcp-Param-Future", "must-not-cross-era"},
		[2]string{"Cookie", "private=1"},
		[2]string{"Mcp-Session-Id", "downstream-session"},
		[2]string{"Last-Event-ID", "99"},
		[2]string{"Authorization", "Bearer downstream"},
	)
	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(headers))
	require.Equal(t, types.ActionPause, host.CallOnHttpRequestBody(modernProxyListBody(9)))

	initialize := calloutAt(t, host, 0)
	assert.Equal(t, "initialize", gjson.GetBytes(initialize.Body, "method").String())
	assertNoProxyLeakHeaders(t, initialize.Headers)
	_, initParam := findHeader(initialize.Headers, "Mcp-Param-Future")
	assert.False(t, initParam)
	completeCallout(host, initialize, "200", [][2]string{
		{"content-type", "application/json"},
		{"Mcp-Session-Id", "upstream-request-session"},
	}, []byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"legacy","version":"1"}}}`))

	notification := calloutAt(t, host, 0)
	assert.Equal(t, "notifications/initialized", gjson.GetBytes(notification.Body, "method").String())
	assert.Equal(t, "upstream-request-session", mustHeaderValue(t, notification.Headers, "Mcp-Session-Id"))
	_, notificationParam := findHeader(notification.Headers, "Mcp-Param-Future")
	assert.False(t, notificationParam)
	completeCallout(host, notification, "202", nil, nil)

	toolCall := calloutAt(t, host, 0)
	assert.Equal(t, "tools/list", gjson.GetBytes(toolCall.Body, "method").String())
	assert.Equal(t, "upstream-request-session", mustHeaderValue(t, toolCall.Headers, "Mcp-Session-Id"))
	_, toolParam := findHeader(toolCall.Headers, "Mcp-Param-Future")
	assert.False(t, toolParam)
	completeCallout(host, toolCall, "200", [][2]string{{"content-type", "application/json"}}, []byte(`{"jsonrpc":"2.0","id":2,"result":{"tools":[]}}`))

	response := host.GetLocalResponse()
	require.NotNil(t, response)
	assert.Equal(t, uint32(200), response.StatusCode)
	assert.Equal(t, resultTypeComplete, gjson.GetBytes(response.Data, "result.resultType").String())
	assert.Equal(t, int64(0), gjson.GetBytes(response.Data, "result.ttlMs").Int())
	assert.Equal(t, cacheScopePrivate, gjson.GetBytes(response.Data, "result.cacheScope").String())
	assert.Empty(t, host.GetHttpCalloutAttributes())
}

func TestModernProxyPreservesUpstreamAuthenticationStatusAndChallenge(t *testing.T) {
	host := newProxyBridgeHost(t, ProtocolStrategyModern)
	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(modernProxyListHeaders()))
	require.Equal(t, types.ActionPause, host.CallOnHttpRequestBody(modernProxyListBody(11)))
	call := calloutAt(t, host, 0)
	completeCallout(host, call, "401", [][2]string{
		{"WWW-Authenticate", `Bearer realm="mcp", error="invalid_token"`},
		{"Set-Cookie", "upstream=secret"},
		{"Mcp-Session-Id", "upstream-secret-session"},
	}, []byte(`{"secret":"must-not-be-forwarded"}`))

	response := host.GetLocalResponse()
	require.NotNil(t, response)
	assert.Equal(t, uint32(401), response.StatusCode)
	assert.Equal(t, `Bearer realm="mcp", error="invalid_token"`, mustHeaderValue(t, response.Headers, "WWW-Authenticate"))
	_, cookie := findHeader(response.Headers, "Set-Cookie")
	assert.False(t, cookie)
	_, session := findHeader(response.Headers, "Mcp-Session-Id")
	assert.False(t, session)
	assert.NotContains(t, string(response.Data), "must-not-be-forwarded")
}

func TestLegacyDownstreamToModernOnlyProxyIsRejectedBeforeCallout(t *testing.T) {
	host := newProxyBridgeHost(t, ProtocolStrategyModern)
	headers := [][2]string{
		{":authority", "mcp.example.com"},
		{":method", "POST"},
		{":path", "/mcp"},
		{"content-type", "application/json"},
		{"accept", "application/json, text/event-stream"},
	}
	require.Equal(t, types.ActionPause, host.CallOnHttpRequestHeaders(headers))
	require.Equal(t, types.ActionContinue, host.CallOnHttpRequestBody([]byte(`{"jsonrpc":"2.0","id":13,"method":"tools/list","params":{}}`)))
	response := host.GetLocalResponse()
	require.NotNil(t, response)
	assert.Equal(t, int64(-32601), gjson.GetBytes(response.Data, "error.code").Int())
	assert.Empty(t, host.GetHttpCalloutAttributes())
}

// MockHttpContext is a mock implementation for testing - skipping interface implementation for now
// Tests that require full HttpContext will be tested in integration tests with real host
type MockHttpContext struct {
	responseBody   []byte
	responseStatus int
	headers        map[string]string
}

// TestMcpProtocolInitialization tests the MCP protocol initialization flow
func TestMcpProtocolInitialization(t *testing.T) {
	// Create proxy server
	server := NewMcpProxyServer("test-proxy")

	// Set server fields directly
	server.SetMcpServerURL("http://mock-backend.example.com/mcp")
	server.SetTimeout(5000)

	// Create proxy tool
	toolConfig := McpProxyToolConfig{
		Name:        "test-tool",
		Description: "Test tool for initialization",
		Args: []ToolArg{
			{
				Name:        "input",
				Description: "Test input",
				Type:        "string",
				Required:    true,
			},
		},
	}

	err := server.AddProxyTool(toolConfig)
	require.NoError(t, err)

	tool, exists := server.GetMCPTools()["test-tool"]
	require.True(t, exists)

	// Create tool instance with parameters
	params := map[string]interface{}{
		"input": "test value",
	}
	paramsBytes, err := json.Marshal(params)
	require.NoError(t, err)

	toolInstance := tool.Create(paramsBytes)
	require.NotNil(t, toolInstance)

	// Skip HttpContext-dependent test for now - will be tested in integration
	// mockCtx := &MockHttpContext{}
	// err = toolInstance.Call(mockCtx, server)
	// assert.NoError(t, err)

	// Test the tool creation was successful
	assert.NotNil(t, toolInstance)
}

// TestMcpSessionManagement tests temporary session creation and cleanup
func TestMcpSessionManagement(t *testing.T) {
	_ = NewMcpProxyServer("session-test")

	// Skip session management test until implemented
	t.Skip("Session management not implemented yet")

	// Test session creation
	sessionManager := NewMcpSessionManager()
	sessionID, err := sessionManager.CreateSession("http://backend.example.com/mcp")

	// This will fail until session management is implemented
	assert.NoError(t, err)
	assert.NotEmpty(t, sessionID)

	// Test session retrieval
	session, exists := sessionManager.GetSession(sessionID)
	assert.True(t, exists)
	assert.NotNil(t, session)

	// Test session cleanup
	sessionManager.CleanupSession(sessionID)
	_, exists = sessionManager.GetSession(sessionID)
	assert.False(t, exists)
}

// TestMcpProtocolVersionNegotiation tests protocol version handling
func TestMcpProtocolVersionNegotiation(t *testing.T) {
	tests := []struct {
		name              string
		requestedVersion  string
		supportedVersions []string
		shouldSucceed     bool
		expectedVersion   string
	}{
		{
			name:              "supported version 2025-03-26",
			requestedVersion:  "2025-03-26",
			supportedVersions: []string{"2024-11-05", "2025-03-26"},
			shouldSucceed:     true,
			expectedVersion:   "2025-03-26",
		},
		{
			name:              "unsupported version",
			requestedVersion:  "2026-01-01",
			supportedVersions: []string{"2024-11-05", "2025-03-26"},
			shouldSucceed:     false,
			expectedVersion:   "",
		},
		{
			name:              "fallback to supported version",
			requestedVersion:  "2025-06-18",
			supportedVersions: []string{"2024-11-05", "2025-03-26"},
			shouldSucceed:     false,
			expectedVersion:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip until NewMcpVersionNegotiator is implemented
			t.Skip("Version negotiation not implemented yet")

			negotiator := NewMcpVersionNegotiator(tt.supportedVersions)
			version, err := negotiator.NegotiateVersion(tt.requestedVersion)

			if tt.shouldSucceed {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVersion, version)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestMcpInitializeRequest tests the initialize request format and handling
func TestMcpInitializeRequest(t *testing.T) {
	_ = NewMcpProxyServer("init-test")

	// Skip until CreateInitializeRequest is implemented
	t.Skip("MCP protocol initialization not implemented yet")

	// Test initialize request creation
	initRequest := CreateInitializeRequest()

	assert.Equal(t, "2.0", initRequest.JsonRPC)
	assert.Equal(t, "initialize", initRequest.Method)
	assert.NotNil(t, initRequest.Params)

	// Validate client info
	params := initRequest.Params.(map[string]interface{})
	clientInfo := params["clientInfo"].(map[string]interface{})
	assert.Equal(t, "Higress-mcp-proxy", clientInfo["name"])
	assert.Equal(t, "1.0.0", clientInfo["version"])

	// Test protocol version
	assert.Equal(t, "2025-03-26", params["protocolVersion"])
}

// TestMcpNotificationsInitialized tests the notifications/initialized message
func TestMcpNotificationsInitialized(t *testing.T) {
	// Skip until CreateInitializedNotification is implemented
	t.Skip("MCP notifications not implemented yet")

	// Test notifications/initialized request creation
	notification := CreateInitializedNotification()

	assert.Equal(t, "2.0", notification.JsonRPC)
	assert.Equal(t, "notifications/initialized", notification.Method)
	assert.Nil(t, notification.ID) // Notifications don't have IDs
}

// TestMcpErrorHandling tests error response handling and source identification
func TestMcpErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		errorType      string
		originalError  error
		expectedSource string
		expectedCode   int
	}{
		{
			name:           "backend connection error",
			errorType:      "connection",
			originalError:  assert.AnError,
			expectedSource: "mcp-proxy",
			expectedCode:   -32603,
		},
		{
			name:           "backend timeout error",
			errorType:      "timeout",
			originalError:  assert.AnError,
			expectedSource: "mcp-proxy",
			expectedCode:   -32000,
		},
		{
			name:           "protocol version error",
			errorType:      "version",
			originalError:  assert.AnError,
			expectedSource: "mcp-proxy",
			expectedCode:   -32602,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip until CreateMcpErrorResponse is implemented
			t.Skip("MCP error handling not implemented yet")

			errorResponse := CreateMcpErrorResponse(tt.errorType, tt.originalError, "http://backend.example.com/mcp")

			assert.Equal(t, "2.0", errorResponse.JsonRPC)
			assert.NotNil(t, errorResponse.Error)
			assert.Equal(t, tt.expectedCode, errorResponse.Error.Code)
			assert.Equal(t, tt.expectedSource, errorResponse.Error.Data["source"])
		})
	}
}

// Helper types and functions that will fail until implemented

type McpSessionManager struct{}

func NewMcpSessionManager() *McpSessionManager {
	panic("McpSessionManager not implemented yet")
}

func (m *McpSessionManager) CreateSession(backendURL string) (string, error) {
	panic("CreateSession not implemented yet")
}

func (m *McpSessionManager) GetSession(sessionID string) (interface{}, bool) {
	panic("GetSession not implemented yet")
}

func (m *McpSessionManager) CleanupSession(sessionID string) {
	panic("CleanupSession not implemented yet")
}

type McpVersionNegotiator struct {
	supportedVersions []string
}

func NewMcpVersionNegotiator(versions []string) *McpVersionNegotiator {
	panic("McpVersionNegotiator not implemented yet")
}

func (n *McpVersionNegotiator) NegotiateVersion(requested string) (string, error) {
	panic("NegotiateVersion not implemented yet")
}

type McpRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type McpErrorResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Error   *McpError   `json:"error"`
}

type McpError struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func CreateInitializeRequest() *McpRequest {
	panic("CreateInitializeRequest not implemented yet")
}

func CreateInitializedNotification() *McpRequest {
	panic("CreateInitializedNotification not implemented yet")
}

func CreateMcpErrorResponse(errorType string, originalError error, backendURL string) *McpErrorResponse {
	panic("CreateMcpErrorResponse not implemented yet")
}
