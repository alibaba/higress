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

package protocol

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func modernTransport(method, name string) Transport {
	return Transport{
		Method:          "POST",
		Authority:       "mcp.example.com",
		ContentType:     "application/json; charset=utf-8",
		Accept:          "application/json, text/event-stream",
		ProtocolVersion: string(Version20260728),
		MCPMethod:       method,
		MCPName:         name,
	}
}

func available(method string) bool {
	switch method {
	case "server/discover", "tools/list", "tools/call":
		return true
	default:
		return false
	}
}

const modernListRequest = `{
  "jsonrpc":"2.0",
  "id":"request-1",
  "method":"tools/list",
  "params":{"_meta":{
    "io.modelcontextprotocol/protocolVersion":"2026-07-28",
    "io.modelcontextprotocol/clientCapabilities":{}
  }}
}`

const modernCallRequest = `{
  "jsonrpc":"2.0",
  "id":7,
  "method":"tools/call",
  "params":{
    "name":"echo",
    "arguments":{"text":"hello"},
    "_meta":{
      "io.modelcontextprotocol/protocolVersion":"2026-07-28",
      "io.modelcontextprotocol/clientCapabilities":{"tools":{}},
      "io.modelcontextprotocol/clientInfo":{"name":"tier-one-sdk","version":"1.2.3"}
    }
  }
}`

const modernListNotification = `{
  "jsonrpc":"2.0",
  "method":"tools/list",
  "params":{"_meta":{
    "io.modelcontextprotocol/protocolVersion":"2026-07-28",
    "io.modelcontextprotocol/clientCapabilities":{}
  }}
}`

func TestPrepareRequestAcceptsModernRequestAndNotification(t *testing.T) {
	tests := []struct {
		name           string
		transport      Transport
		body           string
		notification   bool
		wantClientInfo bool
		wantMethod     string
		wantID         string
	}{
		{
			name:       "stateless request without optional clientInfo",
			transport:  modernTransport("tools/list", ""),
			body:       modernListRequest,
			wantMethod: "tools/list",
			wantID:     `"request-1"`,
		},
		{
			name:           "tool request with clientInfo",
			transport:      modernTransport("tools/call", "echo"),
			body:           modernCallRequest,
			wantClientInfo: true,
			wantMethod:     "tools/call",
			wantID:         "7",
		},
		{
			name:         "valid notification",
			transport:    modernTransport("tools/list", ""),
			body:         modernListNotification,
			notification: true,
			wantMethod:   "tools/list",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lookups := 0
			request, protocolError := PrepareRequest(test.transport, []byte(test.body), func(method string) bool {
				lookups++
				return available(method)
			})
			if protocolError != nil {
				t.Fatalf("PrepareRequest() error = %+v", protocolError)
			}
			if request.Era != EraModern || request.Version != Version20260728 {
				t.Fatalf("unexpected profile: %+v", request)
			}
			if request.Envelope.Method != test.wantMethod || request.Envelope.Notification != test.notification {
				t.Fatalf("unexpected envelope: %+v", request.Envelope)
			}
			if got := string(request.Envelope.ID.Raw()); got != test.wantID {
				t.Fatalf("ID = %q, want %q", got, test.wantID)
			}
			if (request.Metadata.ClientInfo != nil) != test.wantClientInfo {
				t.Fatalf("ClientInfo = %+v", request.Metadata.ClientInfo)
			}
			if request.Cancellation == nil {
				t.Fatal("modern request must have request-scoped cancellation")
			}
			if lookups != 1 {
				t.Fatalf("method availability lookups = %d, want 1", lookups)
			}
		})
	}
}

func TestPrepareRequestRejectsModernBoundaryViolationsBeforeMethodLookup(t *testing.T) {
	oversized := append([]byte(modernListRequest), bytes.Repeat([]byte{' '}, int(ModernMaxBodyBytes))...)
	tests := []struct {
		name      string
		transport Transport
		body      string
		bodyBytes []byte
		wantHTTP  uint32
		wantCode  int
	}{
		{"batch", modernTransport("tools/list", ""), `[` + modernListRequest + `]`, nil, 400, CodeInvalidRequest},
		{"response envelope", modernTransport("tools/list", ""), `{"jsonrpc":"2.0","id":1,"result":{}}`, nil, 400, CodeInvalidRequest},
		{"trailing JSON", modernTransport("tools/list", ""), modernListRequest + `{}`, nil, 400, CodeParseError},
		{"malformed JSON", modernTransport("tools/list", ""), `{"jsonrpc":`, nil, 400, CodeParseError},
		{"null id", modernTransport("tools/list", ""), strings.Replace(modernListRequest, `"id":"request-1"`, `"id":null`, 1), nil, 400, CodeInvalidRequest},
		{"fractional id", modernTransport("tools/list", ""), strings.Replace(modernListRequest, `"id":"request-1"`, `"id":1.5`, 1), nil, 400, CodeInvalidRequest},
		{"boolean id", modernTransport("tools/list", ""), strings.Replace(modernListRequest, `"id":"request-1"`, `"id":true`, 1), nil, 400, CodeInvalidRequest},
		{"duplicate identity", modernTransport("tools/list", ""), strings.Replace(modernListRequest, `"method":"tools/list"`, `"method":"tools/list","method":"tools/list"`, 1), nil, 400, CodeParseError},
		{"oversized body", modernTransport("tools/list", ""), "", oversized, 413, CodeInvalidRequest},
		{"missing protocol header", func() Transport { v := modernTransport("tools/list", ""); v.ProtocolVersion = ""; return v }(), modernListRequest, nil, 400, CodeHeaderMismatch},
		{"legacy protocol header cannot downgrade modern body", func() Transport {
			v := modernTransport("tools/list", "")
			v.ProtocolVersion = string(Version20250618)
			return v
		}(), modernListRequest, nil, 400, CodeHeaderMismatch},
		{"mismatched method header", func() Transport { v := modernTransport("ping", ""); return v }(), modernListRequest, nil, 400, CodeHeaderMismatch},
		{"missing name header", modernTransport("tools/call", ""), modernCallRequest, nil, 400, CodeHeaderMismatch},
		{"mismatched name header", modernTransport("tools/call", "other"), modernCallRequest, nil, 400, CodeHeaderMismatch},
		{"unexpected name header", modernTransport("tools/list", "echo"), modernListRequest, nil, 400, CodeHeaderMismatch},
		{"missing metadata", modernTransport("tools/list", ""), `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`, nil, 400, CodeInvalidParams},
		{"missing capabilities", modernTransport("tools/list", ""), `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, nil, 400, CodeInvalidParams},
		{"invalid clientInfo", modernTransport("tools/list", ""), strings.Replace(modernListRequest, `"io.modelcontextprotocol/clientCapabilities":{}`, `"io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"sdk"}`, 1), nil, 400, CodeInvalidParams},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := []byte(test.body)
			if test.bodyBytes != nil {
				body = test.bodyBytes
			}
			lookups := 0
			_, protocolError := PrepareRequest(test.transport, body, func(string) bool {
				lookups++
				return true
			})
			if protocolError == nil {
				t.Fatal("PrepareRequest() unexpectedly succeeded")
			}
			if protocolError.HTTPStatus != test.wantHTTP || protocolError.Code != test.wantCode {
				t.Fatalf("error = %+v, want HTTP %d/code %d", protocolError, test.wantHTTP, test.wantCode)
			}
			if lookups != 0 {
				t.Fatalf("business method lookup ran %d times before validation completed", lookups)
			}
		})
	}
}

func TestPrepareRequestExactVersionAndMethodErrors(t *testing.T) {
	tests := []struct {
		name      string
		transport Transport
		body      string
		wantCode  int
	}{
		{
			name: "unsupported version",
			transport: func() Transport {
				value := modernTransport("tools/list", "")
				value.ProtocolVersion = "2025-11-25"
				return value
			}(),
			body:     strings.ReplaceAll(modernListRequest, "2026-07-28", "2025-11-25"),
			wantCode: CodeUnsupportedVersion,
		},
		{
			name:      "removed lifecycle method",
			transport: modernTransport("initialize", ""),
			body:      strings.Replace(modernListRequest, "tools/list", "initialize", 1),
			wantCode:  CodeMethodNotFound,
		},
		{
			name:      "unsupported subscription",
			transport: modernTransport("subscriptions/listen", ""),
			body:      strings.Replace(modernListRequest, "tools/list", "subscriptions/listen", 1),
			wantCode:  CodeMethodNotFound,
		},
		{
			name:      "known surface not registered",
			transport: modernTransport("server/discover", ""),
			body:      strings.Replace(modernListRequest, "tools/list", "server/discover", 1),
			wantCode:  CodeMethodNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, protocolError := PrepareRequest(test.transport, []byte(test.body), func(method string) bool {
				return method != "server/discover"
			})
			if protocolError == nil || protocolError.HTTPStatus != 400 && protocolError.HTTPStatus != 404 || protocolError.Code != test.wantCode {
				t.Fatalf("error = %+v, want code %d", protocolError, test.wantCode)
			}
			if test.wantCode == CodeMethodNotFound && protocolError.HTTPStatus != 404 {
				t.Fatalf("method error HTTP status = %d, want 404", protocolError.HTTPStatus)
			}
		})
	}
}

func TestPrepareRequestPreservesAllLegacyProfiles(t *testing.T) {
	legacyBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}} trailing-legacy-data`)
	for _, version := range LegacyVersions() {
		t.Run(string(version), func(t *testing.T) {
			transport := Transport{Method: "PUT", ProtocolVersion: string(version)}
			request, protocolError := PrepareRequest(transport, legacyBody, func(string) bool {
				t.Fatal("modern method lookup must not run for legacy")
				return false
			})
			if protocolError != nil {
				t.Fatalf("legacy request rejected: %+v", protocolError)
			}
			if request.Era != EraLegacy || request.Version != version {
				t.Fatalf("unexpected legacy profile: %+v", request)
			}
			if !bytes.Equal(request.Envelope.Raw, legacyBody) {
				t.Fatal("legacy bytes changed at compatibility boundary")
			}
		})
	}

	initialize := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	request, protocolError := PrepareRequest(Transport{Method: "POST"}, initialize, available)
	if protocolError != nil || request.Era != EraLegacy {
		t.Fatalf("headerless legacy initialize was not preserved: request=%+v error=%+v", request, protocolError)
	}
}

func TestModernCancellationIsRequestScopedAndIdempotent(t *testing.T) {
	request, protocolError := PrepareRequest(modernTransport("tools/list", ""), []byte(modernListRequest), available)
	if protocolError != nil {
		t.Fatalf("PrepareRequest() error = %+v", protocolError)
	}
	request.Cancel()
	request.Cancel()
	select {
	case <-request.Cancellation.Done():
	default:
		t.Fatal("request cancellation was not released")
	}
	requestType := reflect.TypeOf(*request)
	for i := 0; i < requestType.NumField(); i++ {
		if strings.Contains(strings.ToLower(requestType.Field(i).Name), "session") {
			t.Fatalf("modern request context contains protocol session field %q", requestType.Field(i).Name)
		}
	}
}

func TestMarshalErrorResponsePreservesValidID(t *testing.T) {
	request, protocolError := PrepareRequest(
		func() Transport { value := modernTransport("ping", ""); return value }(),
		[]byte(strings.Replace(modernListRequest, "tools/list", "initialize", 1)),
		available,
	)
	if protocolError == nil {
		t.Fatal("expected method-not-found error")
	}
	body := MarshalErrorResponse(request.Envelope.ID, protocolError)
	var decoded struct {
		ID    json.RawMessage `json:"id"`
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}
	if string(decoded.ID) != `"request-1"` || decoded.Error.Code != CodeHeaderMismatch {
		t.Fatalf("unexpected error response: %s", body)
	}
}
