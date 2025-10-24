// Copyright (c) 2022 Alibaba Group Holding Ltd.
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

package tests

import (
	"testing"

	"github.com/alibaba/higress/v2/test/e2e/conformance/utils/envoy"
	"github.com/alibaba/higress/v2/test/e2e/conformance/utils/suite"
)

func init() {
	Register(IngressLoadBalanceMcpSse)
}

var IngressLoadBalanceMcpSse = suite.ConformanceTest{
	ShortName:   "IngressLoadBalanceMcpSse",
	Description: "The Envoy config should contain MCP SSE stateful session filter when load-balance annotation is set to mcp-sse",
	Manifests:   []string{"tests/ingress-loadbalance-mcp-sse.yaml"},
	Features:    []suite.SupportedFeature{suite.EnvoyConfigConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testCases := []struct {
			name           string
			envoyAssertion []envoy.Assertion
		}{
			{
				name: "MCP SSE stateful session global filter should be added",
				envoyAssertion: []envoy.Assertion{
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config.http_filters",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"name": "envoy.filters.http.mcp_sse_stateful_session",
							"typed_config": map[string]interface{}{
								"@type":    "type.googleapis.com/udpa.type.v1.TypedStruct",
								"type_url": "type.googleapis.com/envoy.extensions.filters.http.mcp_sse_stateful_session.v3alpha.McpSseStatefulSession",
							},
						},
					},
				},
			},
			// TODO: add per router filter check
		}

		t.Run("Ingress LoadBalance MCP SSE", func(t *testing.T) {
			for _, testcase := range testCases {
				t.Logf("Test Case %s", testcase.name)
				for _, assertion := range testcase.envoyAssertion {
					envoy.AssertEnvoyConfig(t, suite.TimeoutConfig, assertion)
				}
			}
		})
	},
}
