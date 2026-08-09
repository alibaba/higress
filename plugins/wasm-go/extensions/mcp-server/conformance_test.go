// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	wasmtest "github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const officialFixtureDir = "testdata/conformance/official"

var conformanceServerConfig = json.RawMessage(`{
	"server":{"name":"fixture-server","type":"rest"},
	"tools":[{
		"name":"get_weather",
		"description":"Return deterministic fixture weather",
		"args":[{"name":"location","description":"City name","type":"string","required":true}],
		"responseTemplate":{"body":"weather for {{.args.location}}"}
	}]
}`)

func TestOfficial20260728FixturesArePinned(t *testing.T) {
	var provenance struct {
		Commit string            `json:"commit"`
		SHA256 map[string]string `json:"sha256"`
	}
	bytes := readOfficialFixture(t, "PROVENANCE.json")
	require.NoError(t, json.Unmarshal(bytes, &provenance))
	require.Equal(t, "f817239f4d6b1efff2c4dfc2f7af85c985d73076", provenance.Commit)
	require.Len(t, provenance.SHA256, 9)
	for name, want := range provenance.SHA256 {
		sum := sha256.Sum256(readOfficialFixture(t, name))
		require.Equal(t, want, hex.EncodeToString(sum[:]), name)
	}
}

func TestOfficial20260728PositiveRequests(t *testing.T) {
	wasmtest.RunTest(t, func(t *testing.T) {
		for _, name := range []string{"discover-request.json", "list-tools-request.json", "call-tool-request.json"} {
			t.Run(name, func(t *testing.T) {
				body := readOfficialFixture(t, name)
				response := conformanceExchange(t, body, nil)
				require.Equal(t, uint32(200), response.StatusCode)
				require.False(t, gjson.GetBytes(response.Data, "error").Exists(), string(response.Data))
				require.Equal(t, "complete", gjson.GetBytes(response.Data, "result.resultType").String())
				switch gjson.GetBytes(body, "method").String() {
				case "server/discover":
					officialResult := readOfficialFixture(t, "discover-result.json")
					require.Equal(t, gjson.GetBytes(officialResult, "resultType").String(), gjson.GetBytes(response.Data, "result.resultType").String())
					require.True(t, gjson.GetBytes(response.Data, "result.capabilities.tools").Exists())
					require.True(t, gjson.GetBytes(response.Data, "result.supportedVersions.#(==\"2026-07-28\")").Exists())
				case "tools/list":
					require.Equal(t, "get_weather", gjson.GetBytes(response.Data, "result.tools.0.name").String())
				case "tools/call":
					require.Contains(t, string(response.Data), "weather for New York")
				}
			})
		}
	})
}

func TestOfficial20260728NegativeContracts(t *testing.T) {
	wasmtest.RunTest(t, func(t *testing.T) {
		call := readOfficialFixture(t, "call-tool-request.json")
		list := readOfficialFixture(t, "list-tools-request.json")
		cases := []struct {
			name         string
			body         []byte
			extraHeaders [][2]string
			fixture      string
		}{
			{"header mismatch", call, [][2]string{{"Mcp-Name", "different_tool"}}, "header-mismatch-error.json"},
			{"unsupported version", []byte(strings.ReplaceAll(string(list), "2026-07-28", "1900-01-01")), nil, "unsupported-version-error.json"},
			{"invalid tool arguments", []byte(strings.Replace(string(call), `"name": "get_weather"`, `"name": 7`, 1)), nil, "invalid-tool-arguments-error.json"},
			{"method not found", []byte(strings.ReplaceAll(string(list), "tools/list", "prompts/list")), nil, "method-not-found-error.json"},
			{"parse error", []byte(`{"jsonrpc":"2.0",`), nil, "parse-error.json"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				response := conformanceExchange(t, tc.body, tc.extraHeaders)
				want := officialErrorCode(t, tc.fixture)
				require.Equal(t, want, gjson.GetBytes(response.Data, "error.code").Int(), string(response.Data))
			})
		}
	})
}

func conformanceExchange(t *testing.T, body []byte, extra [][2]string) *wasmtestLocalResponse {
	t.Helper()
	host, status := wasmtest.NewTestHost(conformanceServerConfig)
	require.Equal(t, types.OnPluginStartStatusOK, status)
	t.Cleanup(host.Reset)
	host.InitHttp()
	t.Cleanup(host.CompleteHttp)
	method := gjson.GetBytes(body, "method").String()
	version := gjson.GetBytes(body, "params._meta.io\\.modelcontextprotocol/protocolVersion").String()
	headers := [][2]string{
		{":authority", "mcp.example.com"}, {":method", "POST"}, {":path", "/mcp"},
		{"content-type", "application/json"}, {"accept", "application/json, text/event-stream"},
	}
	if version != "" {
		headers = append(headers, [2]string{"MCP-Protocol-Version", version})
	}
	if method != "" {
		headers = append(headers, [2]string{"Mcp-Method", method})
	}
	if name := gjson.GetBytes(body, "params.name").String(); name != "" {
		headers = append(headers, [2]string{"Mcp-Name", name})
	}
	headers = append(headers, extra...)
	host.CallOnHttpRequestHeaders(headers)
	host.CallOnHttpRequestBody(body)
	response := host.GetLocalResponse()
	require.NotNil(t, response)
	return &wasmtestLocalResponse{StatusCode: response.StatusCode, Data: append([]byte(nil), response.Data...)}
}

// wasmtestLocalResponse copies the fields used by assertions before TestHost
// cleanup releases the process-global proxy-wasm emulator.
type wasmtestLocalResponse struct {
	StatusCode uint32
	Data       []byte
}

func readOfficialFixture(t *testing.T, name string) []byte {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join(officialFixtureDir, name))
	require.NoError(t, err)
	return bytes
}

func officialErrorCode(t *testing.T, name string) int64 {
	t.Helper()
	fixture := readOfficialFixture(t, name)
	if code := gjson.GetBytes(fixture, "error.code"); code.Exists() {
		return code.Int()
	}
	return gjson.GetBytes(fixture, "code").Int()
}
