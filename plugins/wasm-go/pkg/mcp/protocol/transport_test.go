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

import "testing"

func TestValidateModernTransportMappings(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*Transport)
		wantHTTP uint32
		wantCode int
	}{
		{"non-POST", func(v *Transport) { v.Method = "GET" }, 405, CodeInvalidRequest},
		{"missing Content-Type", func(v *Transport) { v.ContentType = "" }, 415, CodeInvalidRequest},
		{"wrong Content-Type", func(v *Transport) { v.ContentType = "text/plain" }, 415, CodeInvalidRequest},
		{"missing Accept", func(v *Transport) { v.Accept = "" }, 406, CodeInvalidRequest},
		{"Accept omits SSE", func(v *Transport) { v.Accept = "application/json" }, 406, CodeInvalidRequest},
		{"Accept disables JSON", func(v *Transport) { v.Accept = "application/json;q=0, text/event-stream" }, 406, CodeInvalidRequest},
		{"cross-origin host", func(v *Transport) { v.Origin = "https://evil.example" }, 403, CodeInvalidRequest},
		{"opaque origin", func(v *Transport) { v.Origin = "null" }, 403, CodeInvalidRequest},
		{"ambiguous header", func(v *Transport) { v.AmbiguousHeader = true }, 400, CodeHeaderMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := modernTransport("tools/list", "")
			test.mutate(&transport)
			protocolError := ValidateModernTransport(transport)
			if protocolError == nil || protocolError.HTTPStatus != test.wantHTTP || protocolError.Code != test.wantCode {
				t.Fatalf("error = %+v, want HTTP %d/code %d", protocolError, test.wantHTTP, test.wantCode)
			}
		})
	}
}

func TestValidateModernTransportAcceptsSameOriginAndNoOrigin(t *testing.T) {
	for _, origin := range []string{"", "https://mcp.example.com", "https://mcp.example.com:443"} {
		transport := modernTransport("tools/list", "")
		transport.Origin = origin
		if protocolError := ValidateModernTransport(transport); protocolError != nil {
			t.Fatalf("Origin %q rejected: %+v", origin, protocolError)
		}
	}
}

func TestNewTransportRejectsDuplicateSensitiveHeaders(t *testing.T) {
	transport := NewTransport("POST", "mcp.example.com", [][2]string{
		{"Content-Type", "application/json"},
		{"Accept", "application/json, text/event-stream"},
		{"Mcp-Method", "tools/list"},
		{"mcp-method", "tools/list"},
	})
	if !transport.AmbiguousHeader {
		t.Fatal("duplicate case-insensitive identity header was not marked ambiguous")
	}
}

func TestHasModernIdentityHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers [][2]string
		want    bool
	}{
		{
			name:    "modern method without protocol header",
			headers: [][2]string{{HeaderMethod, "tools/list"}},
			want:    true,
		},
		{
			name:    "modern name without protocol header",
			headers: [][2]string{{HeaderName, "weather"}},
			want:    true,
		},
		{
			name:    "present but empty modern method",
			headers: [][2]string{{HeaderMethod, " "}},
			want:    true,
		},
		{
			name:    "empty protocol identity",
			headers: [][2]string{{HeaderProtocolVersion, " "}},
			want:    true,
		},
		{
			name: "duplicate modern identity",
			headers: [][2]string{
				{HeaderMethod, "tools/list"},
				{HeaderMethod, "tools/list"},
			},
			want: true,
		},
		{
			name:    "legacy protocol only",
			headers: [][2]string{{HeaderProtocolVersion, string(Version20250618)}},
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := NewTransport("POST", "mcp.example.com", test.headers)
			if got := HasModernIdentityHeaders(transport); got != test.want {
				t.Fatalf("HasModernIdentityHeaders() = %v, want %v; transport = %+v", got, test.want, transport)
			}
		})
	}
}
