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
	"time"

	"github.com/alibaba/higress/hgctl/cmd/hgctl/config"
	"github.com/alibaba/higress/pkg/ingress/kube/configmap"
	"github.com/alibaba/higress/test/e2e/conformance/utils/envoy"
	"github.com/alibaba/higress/test/e2e/conformance/utils/kubernetes"
	cfg "github.com/alibaba/higress/test/e2e/conformance/utils/config"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/util/wait"
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
						Path:            "configs.#.dynamic_clusters.#.cluster",
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
				
				// Special debugging for the per_connection_buffer_limit_bytes assertion
				if assertion.Path == "configs.#.dynamic_clusters.#.cluster" {
					t.Logf("🔍 ConfigMapGlobalEnvoy: Special debugging for cluster buffer limit assertion")
					debugClusterConfig(t, suite.TimeoutConfig)
				}
				
				envoy.AssertEnvoyConfig(t, suite.TimeoutConfig, assertion)
				t.Logf("✅ ConfigMapGlobalEnvoy: Assertion %d passed", i+1)
			}
		}
	},
}

// debugClusterConfig dumps the actual cluster configuration for debugging
func debugClusterConfig(t *testing.T, timeoutConfig cfg.TimeoutConfig) {
	t.Logf("🔍 Debug: Starting cluster configuration debug")
	
	options := &config.GetEnvoyConfigOptions{
		PodName:         "",
		PodNamespace:    "higress-system",
		BindAddress:     "localhost",
		Output:          "json",
		EnvoyConfigType: config.AllEnvoyConfigType,
		IncludeEds:      true,
	}
	
	var allEnvoyConfig string
	err := wait.Poll(1*time.Second, 10*time.Second, func() (bool, error) {
		out, err := config.GetEnvoyConfig(options)
		if err != nil {
			return false, err
		}
		allEnvoyConfig = string(out)
		return true, nil
	})
	
	if err != nil {
		t.Logf("❌ Debug: Failed to get Envoy config: %v", err)
		return
	}
	
	t.Logf("🔍 Debug: Successfully retrieved Envoy config, length: %d bytes", len(allEnvoyConfig))
	
	// Try to parse and debug cluster configurations
	parsed := gjson.Parse(allEnvoyConfig)
	dynamicClusters := parsed.Get("configs.#.dynamic_clusters")
	
	if dynamicClusters.Exists() {
		t.Logf("🔍 Debug: Found dynamic_clusters in config")
		if dynamicClusters.IsArray() {
			t.Logf("🔍 Debug: dynamic_clusters is an array with %d elements", len(dynamicClusters.Array()))
			for i, cluster := range dynamicClusters.Array() {
				t.Logf("🔍 Debug: Cluster %d:", i)
				if cluster.IsObject() {
					t.Logf("🔍 Debug: Cluster %d keys:", i)
					cluster.ForEach(func(key, value gjson.Result) bool {
						t.Logf("  - %s", key.String())
						return true
					})
					
					// Check if cluster has nested cluster object
					nestedCluster := cluster.Get("cluster")
					if nestedCluster.Exists() {
						t.Logf("🔍 Debug: Cluster %d has nested cluster object:", i)
						nestedCluster.ForEach(func(key, value gjson.Result) bool {
							t.Logf("  - cluster.%s", key.String())
							return true
						})
						
						// Specifically check for per_connection_buffer_limit_bytes
						if bufferLimit := nestedCluster.Get("per_connection_buffer_limit_bytes"); bufferLimit.Exists() {
							t.Logf("🔍 Debug: FOUND per_connection_buffer_limit_bytes in cluster %d: %v", i, bufferLimit.Value())
						} else {
							t.Logf("🔍 Debug: per_connection_buffer_limit_bytes NOT FOUND in cluster %d", i)
						}
					} else {
						t.Logf("🔍 Debug: Cluster %d does NOT have nested cluster object", i)
						// Check if per_connection_buffer_limit_bytes exists directly
						if bufferLimit := cluster.Get("per_connection_buffer_limit_bytes"); bufferLimit.Exists() {
							t.Logf("🔍 Debug: FOUND per_connection_buffer_limit_bytes directly in cluster %d: %v", i, bufferLimit.Value())
						} else {
							t.Logf("🔍 Debug: per_connection_buffer_limit_bytes NOT FOUND directly in cluster %d", i)
						}
					}
				}
			}
		} else {
			t.Logf("🔍 Debug: dynamic_clusters is not an array: %T", dynamicClusters.Value())
		}
	} else {
		t.Logf("🔍 Debug: dynamic_clusters NOT found in config")
		
		// Let's see what's actually in configs
		configs := parsed.Get("configs")
		if configs.Exists() {
			t.Logf("🔍 Debug: configs exists, type: %T", configs.Value())
			if configs.IsArray() {
				t.Logf("🔍 Debug: configs is an array with %d elements", len(configs.Array()))
				for i, config := range configs.Array() {
					t.Logf("🔍 Debug: Config %d keys:", i)
					config.ForEach(func(key, value gjson.Result) bool {
						t.Logf("  - %s", key.String())
						return true
					})
				}
			}
		} else {
			t.Logf("🔍 Debug: configs NOT found in config")
		}
	}
	
	t.Logf("🔍 Debug: Cluster configuration debug completed")
}
