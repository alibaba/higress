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

	"github.com/alibaba/higress/test/e2e/conformance/utils/envoy"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
	Register(EnvoyConfigTracing)
}

var EnvoyConfigTracing = suite.ConformanceTest{
	ShortName:   "EnvoyConfigTracing",
	Description: "The Envoy config should contain tracing config",
	Manifests:   []string{"tests/configmap-tracing.yaml"},
	Features:    []suite.SupportedFeature{suite.EnvoyConfigConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testCase := []envoy.Assertion{
			{
				Path:            "configs.#.bootstrap.tracing",
				TargetNamespace: "higress-system",
				ExceptContainEnvoyConfig: map[string]interface{}{
					"http": map[string]interface{}{
						"name": "envoy.tracers.zipkin",
						"typed_config": map[string]interface{}{
							"@type":                      "type.googleapis.com/envoy.config.trace.v3.ZipkinConfig",
							"collector_cluster":          "zipkin",
							"collector_endpoint":         "/api/v2/spans",
							"trace_id_128bit":            true,
							"shared_span_context":        false,
							"collector_endpoint_version": "HTTP_JSON",
						},
					},
				},
			},
		}
		for _, test := range testCase {
			if err := envoy.AssertEnvoyConfig(t, test); err != nil {
				t.Errorf("failed to assert envoy config: %v", err)
			}
		}
	},
}
