// Copyright (c) 2025 Alibaba Group Holding Ltd.
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

package mcpserver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMcpServerCache_GetSet(t *testing.T) {
	testCases := []struct {
		name    string
		skip    bool
		init    []*McpServer
		input   []*McpServer
		expect  []*McpServer
		changed bool
	}{
		{
			name:    "nil",
			init:    nil,
			input:   nil,
			changed: false,
			expect:  nil,
		},
		{
			name: "nil to non-nil",
			init: nil,
			input: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			changed: true,
			expect: []*McpServer{
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
			},
		},
		{
			name: "non-nil to non-nil (length increase)",
			init: []*McpServer{
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
			},
			input: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			changed: true,
			expect: []*McpServer{
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
			},
		},
		{
			name: "non-nil to non-nil (length decrease)",
			init: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			input: []*McpServer{
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
			},
			changed: true,
			expect: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
			},
		},
		{
			name: "non-nil to non-nil (length unchanged + name field changed)",
			init: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			input: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3-1",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			changed: true,
			expect: []*McpServer{
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3-1",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
			},
		},
		{
			name: "non-nil to non-nil (length unchanged + non-name field changed)",
			init: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			input: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar-2.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test4",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			changed: true,
			expect: []*McpServer{
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar-2.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test4",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
			},
		},
		{
			name: "non-nil to non-nil (content unchanged + order unchanged)",
			init: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			input: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			changed: false,
			expect: []*McpServer{
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
			},
		},
		{
			name: "non-nil to non-nil (content unchanged + order changed)",
			init: []*McpServer{
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
			},
			input: []*McpServer{
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
			},
			changed: false,
			expect: []*McpServer{
				{
					Name:              "test1",
					Domains:           nil,
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test1",
					UpstreamType:      UpstreamTypeRest,
					EnablePathRewrite: false,
					PathRewritePrefix: "",
				},
				{
					Name:              "test2",
					Domains:           []string{"www.foo.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test2",
					UpstreamType:      UpstreamTypeSSE,
					EnablePathRewrite: true,
					PathRewritePrefix: "/test",
				},
				{
					Name:              "test3",
					Domains:           []string{"www.bar.com"},
					PathMatchType:     ExactMatchType,
					PathMatchValue:    "/mcp/test3",
					UpstreamType:      UpstreamTypeStreamable,
					EnablePathRewrite: true,
					PathRewritePrefix: "/",
				},
			},
		},
	}

	for _, tt := range testCases {
		if tt.skip {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			provider := &McpServerCache{}

			if provider.GetMcpServers() != nil {
				t.Fatalf("GetMcpServers doesn't return nil before testing.")
			}

			_ = provider.SetMcpServers(tt.init)

			changed := provider.SetMcpServers(tt.input)
			if changed != tt.changed {
				t.Fatalf("actual changed %t != expect changed %t", changed, tt.changed)
				return
			}

			actual := provider.GetMcpServers()

			if len(actual) != len(tt.expect) {
				t.Fatalf("actual length %d != expect length %d", len(actual), len(tt.expect))
			}
			for i := range actual {
				if diff := cmp.Diff(tt.expect[i], actual[i]); diff != "" {
					t.Fatalf("TestMcpServerCache_GetSet() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
