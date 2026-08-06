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

package annotations

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/alibaba/higress/v2/pkg/ingress/kube/mcpserver"
)

func TestMCPServer_Parse(t *testing.T) {
	parser := mcpServer{}
	testCases := []struct {
		skip   bool
		input  Annotations
		expect *mcpserver.McpServer
	}{
		{
			// No annotation
			input:  Annotations{},
			expect: nil,
		},
		{
			// Not enabled
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "false",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "www.foo.com",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "prefix",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "rest",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "",
			},
			expect: nil,
		},
		{
			// Enabled but no match rule type
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "www.foo.com",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "rest",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "",
			},
			expect: nil,
		},
		{
			// Enabled but empty match rule type
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "www.foo.com",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "rest",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "",
			},
			expect: nil,
		},
		{
			// Enabled but bad match rule type
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "www.foo.com",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "bad-type",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "rest",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "",
			},
			expect: nil,
		},
		{
			// Enabled but bad upstream type
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "www.foo.com",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "prefix",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "bad-type",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "",
			},
			expect: nil,
		},
		{
			// Enabled and rewrite not enabled
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "www.foo.com",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "prefix",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "rest",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "false",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "/",
			},
			expect: &mcpserver.McpServer{
				Name:              "default/route",
				Domains:           []string{"www.foo.com"},
				PathMatchType:     "prefix",
				PathMatchValue:    "/mcp",
				UpstreamType:      "rest",
				EnablePathRewrite: false,
				PathRewritePrefix: "/",
			},
		},
		{
			// Enabled and rewrite not enabled and empty domain
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "prefix",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "rest",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "false",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "/",
			},
			expect: &mcpserver.McpServer{
				Name:              "default/route",
				Domains:           []string{"*"},
				PathMatchType:     "prefix",
				PathMatchValue:    "/mcp",
				UpstreamType:      "rest",
				EnablePathRewrite: false,
				PathRewritePrefix: "/",
			},
		},
		{
			// Enabled and rewrite not enabled and wildcard domain
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "*",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "prefix",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "rest",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "false",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "/",
			},
			expect: &mcpserver.McpServer{
				Name:              "default/route",
				Domains:           []string{"*"},
				PathMatchType:     "prefix",
				PathMatchValue:    "/mcp",
				UpstreamType:      "rest",
				EnablePathRewrite: false,
				PathRewritePrefix: "/",
			},
		},
		{
			// Enabled and rewrite enabled with root
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "www.foo.com",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "prefix",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "rest",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "true",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "/",
			},
			expect: &mcpserver.McpServer{
				Name:              "default/route",
				Domains:           []string{"www.foo.com"},
				PathMatchType:     "prefix",
				PathMatchValue:    "/mcp",
				UpstreamType:      "rest",
				EnablePathRewrite: true,
				PathRewritePrefix: "/",
			},
		},
		{
			// Enabled and rewrite enabled with root
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "www.foo.com",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "prefix",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "rest",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "true",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "/mcp-api",
			},
			expect: &mcpserver.McpServer{
				Name:              "default/route",
				Domains:           []string{"www.foo.com"},
				PathMatchType:     "prefix",
				PathMatchValue:    "/mcp",
				UpstreamType:      "rest",
				EnablePathRewrite: true,
				PathRewritePrefix: "/mcp-api",
			},
		},
		{
			// Enabled and multiple domains
			input: Annotations{
				buildHigressAnnotationKey(enableMcpServer):            "true",
				buildHigressAnnotationKey(mcpServerMatchRuleDomains):  "www.foo.com,www.bar.com",
				buildHigressAnnotationKey(mcpServerMatchRuleType):     "exact",
				buildHigressAnnotationKey(mcpServerMatchRuleValue):    "/mcp",
				buildHigressAnnotationKey(mcpServerUpstreamType):      "sse",
				buildHigressAnnotationKey(mcpServerEnablePathRewrite): "true",
				buildHigressAnnotationKey(mcpServerPathRewritePrefix): "/",
			},
			expect: &mcpserver.McpServer{
				Name:              "default/route",
				Domains:           []string{"www.foo.com", "www.bar.com"},
				PathMatchType:     "exact",
				PathMatchValue:    "/mcp",
				UpstreamType:      "sse",
				EnablePathRewrite: true,
				PathRewritePrefix: "/",
			},
		},
	}

	for _, tt := range testCases {
		if tt.skip {
			return
		}

		t.Run("", func(t *testing.T) {
			config := &Ingress{Meta: Meta{
				Namespace: "default",
				Name:      "route",
			}}
			globalContext := &GlobalContext{}
			_ = parser.Parse(tt.input, config, globalContext)
			if tt.expect == nil {
				if len(globalContext.McpServers) != 0 {
					t.Fatalf("globalContext.McpServers is not empty: %v", globalContext.McpServers)
				}
				return
			}

			if len(globalContext.McpServers) != 1 {
				t.Fatalf("globalContext.McpServers length is not 1: %v", globalContext.McpServers)
			}

			if diff := cmp.Diff(tt.expect, globalContext.McpServers[0]); diff != "" {
				t.Fatalf("TestMCPServer_Parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
