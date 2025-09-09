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

	"github.com/alibaba/higress/pkg/ingress/kube/configmap"
	"github.com/alibaba/higress/test/e2e/conformance/utils/envoy"
	"github.com/alibaba/higress/test/e2e/conformance/utils/kubernetes"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
	Register(ConfigMapGlobalEnvoy)
}

var ConfigMapGlobalEnvoy = suite.ConformanceTest{
	ShortName:   "ConfigMapGlobalEnvoy",
	Description: "The Envoy config should contain global config",
	Manifests:   []string{"tests/configmap-global.yaml"},
	Features:    []suite.SupportedFeature{suite.EnvoyConfigConformanceFeature},
	Parallel:    false,
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testCases := []struct {
			name           string
			higressConfig  *configmap.HigressConfig
			envoyAssertion []envoy.Assertion
		}{
			{
				name: "set config all",
				higressConfig: &configmap.HigressConfig{
					Downstream: &configmap.Downstream{
						IdleTimeout: 60,
					},
					Upstream: &configmap.Upstream{
						IdleTimeout:            10,
						ConnectionBufferLimits: 10485760,
					},
					AddXRealIpHeader: true,
				},
				envoyAssertion: []envoy.Assertion{
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"request_headers_to_add": []interface{}{
								map[string]interface{}{
									"append": false,
									"header": map[string]interface{}{
										"key":   "x-real-ip",
										"value": "%REQ(X-ENVOY-EXTERNAL-ADDRESS)%",
									},
								},
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"@type":       "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
							"stat_prefix": "outbound_0.0.0.0_80",
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config.http_filters",
						CheckType:       envoy.CheckTypeNotExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"typed_config": map[string]interface{}{
								"suppress_envoy_headers": true,
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"@type":       "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
							"stat_prefix": "outbound_0.0.0.0_80",
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "60s",
							},
						},
					},
					{
						Path:            "configs.#.dynamic_clusters.#",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 10485760,
						},
					},
				},
			},
		}

		for _, testcase := range testCases {
			t.Logf("📍 ConfigMapGlobalEnvoy: Applying test case configuration")
			err := kubernetes.ApplyConfigmapDataWithYaml(t, suite.Client, "higress-system", "higress-config", "higress", testcase.higressConfig)
			if err != nil {
				t.Logf("❌ ConfigMapGlobalEnvoy: Failed to apply configmap %s in namespace %s for data key %s", "higress-config", "higress-system", "higress")
				t.Logf("📍 ConfigMapGlobalEnvoy: ConfigMap application failed: %v", err)
				t.FailNow()
			}
			t.Logf("✅ ConfigMapGlobalEnvoy: Configuration applied successfully")
			
			for i, assertion := range testcase.envoyAssertion {
				t.Logf("📍 ConfigMapGlobalEnvoy: Running assertion %d with path: %s", i+1, assertion.Path)
				envoy.AssertEnvoyConfig(t, suite.TimeoutConfig, assertion)
				t.Logf("✅ ConfigMapGlobalEnvoy: Assertion %d passed", i+1)
			}
		}
	},
}
