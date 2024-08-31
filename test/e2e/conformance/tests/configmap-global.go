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
						IdleTimeout:            180,
						MaxRequestHeadersKb:    60,
						ConnectionBufferLimits: 32768,
						Http2: &configmap.Http2{
							MaxConcurrentStreams:        100,
							InitialStreamWindowSize:     65535,
							InitialConnectionWindowSize: 1048576,
						},
						RouteTimeout: 15,
					},
					Upstream: &configmap.Upstream{
						IdleTimeout:            10,
						ConnectionBufferLimits: 10485760,
					},
					DisableXEnvoyHeaders: true,
					AddXRealIpHeader:     true,
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
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"name": "envoy.filters.http.router",
							"typed_config": map[string]interface{}{
								"@type":                  "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
								"suppress_envoy_headers": true,
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"http2_protocol_options": map[string]interface{}{
								"max_concurrent_streams":         100,
								"initial_stream_window_size":     65535,
								"initial_connection_window_size": 1048576,
							},
							"stream_idle_timeout":    "180s",
							"max_request_headers_kb": 60,
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "180s",
							},
						},
					},
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config.virtual_hosts.#.routes.#.route",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"timeout": "15s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
							"per_connection_buffer_limit_bytes": 10485760,
						},
					},
				},
			},
			{
				name: "did not set AddXRealIpHeader",
				higressConfig: &configmap.HigressConfig{
					Downstream: &configmap.Downstream{
						IdleTimeout:            180,
						MaxRequestHeadersKb:    60,
						ConnectionBufferLimits: 32768,
						Http2: &configmap.Http2{
							MaxConcurrentStreams:        100,
							InitialStreamWindowSize:     65535,
							InitialConnectionWindowSize: 1048576,
						},
						RouteTimeout: 15,
					},
					Upstream: &configmap.Upstream{
						IdleTimeout:            10,
						ConnectionBufferLimits: 10485760,
					},
					DisableXEnvoyHeaders: true,
				},
				envoyAssertion: []envoy.Assertion{
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config.request_headers_to_add.#.header",
						CheckType:       envoy.CheckTypeNotExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"key":   "x-real-ip",
							"value": "%REQ(X-ENVOY-EXTERNAL-ADDRESS)%",
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
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"name": "envoy.filters.http.router",
							"typed_config": map[string]interface{}{
								"@type":                  "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
								"suppress_envoy_headers": true,
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"http2_protocol_options": map[string]interface{}{
								"max_concurrent_streams":         100,
								"initial_stream_window_size":     65535,
								"initial_connection_window_size": 1048576,
							},
							"stream_idle_timeout":    "180s",
							"max_request_headers_kb": 60,
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "180s",
							},
						},
					},
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config.virtual_hosts.#.routes.#.route",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"timeout": "15s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
							"per_connection_buffer_limit_bytes": 10485760,
						},
					},
				},
			},
			{
				name: "did not set DisableXEnvoyHeaders",
				higressConfig: &configmap.HigressConfig{
					Downstream: &configmap.Downstream{
						IdleTimeout:            180,
						MaxRequestHeadersKb:    60,
						ConnectionBufferLimits: 32768,
						Http2: &configmap.Http2{
							MaxConcurrentStreams:        100,
							InitialStreamWindowSize:     65535,
							InitialConnectionWindowSize: 1048576,
						},
						RouteTimeout: 15,
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
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"http2_protocol_options": map[string]interface{}{
								"max_concurrent_streams":         100,
								"initial_stream_window_size":     65535,
								"initial_connection_window_size": 1048576,
							},
							"stream_idle_timeout":    "180s",
							"max_request_headers_kb": 60,
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "180s",
							},
						},
					},
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config.virtual_hosts.#.routes.#.route",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"timeout": "15s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
							"per_connection_buffer_limit_bytes": 10485760,
						},
					},
				},
			},
			{
				name: "did not set AddXRealIpHeader and DisableXEnvoyHeaders",
				higressConfig: &configmap.HigressConfig{
					Downstream: &configmap.Downstream{
						IdleTimeout:            180,
						MaxRequestHeadersKb:    60,
						ConnectionBufferLimits: 32768,
						Http2: &configmap.Http2{
							MaxConcurrentStreams:        100,
							InitialStreamWindowSize:     65535,
							InitialConnectionWindowSize: 1048576,
						},
						RouteTimeout: 15,
					},
					Upstream: &configmap.Upstream{
						IdleTimeout:            10,
						ConnectionBufferLimits: 10485760,
					},
				},
				envoyAssertion: []envoy.Assertion{
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config.request_headers_to_add.#.header",
						CheckType:       envoy.CheckTypeNotExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"key":   "x-real-ip",
							"value": "%REQ(X-ENVOY-EXTERNAL-ADDRESS)%",
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
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"http2_protocol_options": map[string]interface{}{
								"max_concurrent_streams":         100,
								"initial_stream_window_size":     65535,
								"initial_connection_window_size": 1048576,
							},
							"stream_idle_timeout":    "180s",
							"max_request_headers_kb": 60,
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "180s",
							},
						},
					},
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config.virtual_hosts.#.routes.#.route",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"timeout": "15s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
							"per_connection_buffer_limit_bytes": 10485760,
						},
					},
				},
			},
			{
				name: "did not set Downstream, will use default value",
				higressConfig: &configmap.HigressConfig{
					Upstream: &configmap.Upstream{
						IdleTimeout: 10,
					},
					DisableXEnvoyHeaders: true,
					AddXRealIpHeader:     true,
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
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"name": "envoy.filters.http.router",
							"typed_config": map[string]interface{}{
								"@type":                  "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
								"suppress_envoy_headers": true,
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"max_concurrent_streams":         100,
							"initial_stream_window_size":     65535,
							"initial_connection_window_size": 1048576,
							"stream_idle_timeout":            "180s",
							"max_request_headers_kb":         60,
							"idle_timeout":                   "180s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
						},
					},
				},
			},
			{
				name: "did not set Upstream, will use default value",
				higressConfig: &configmap.HigressConfig{
					Downstream: &configmap.Downstream{
						IdleTimeout:            180,
						MaxRequestHeadersKb:    60,
						ConnectionBufferLimits: 32768,
						Http2: &configmap.Http2{
							MaxConcurrentStreams:        100,
							InitialStreamWindowSize:     65535,
							InitialConnectionWindowSize: 1048576,
						},
						RouteTimeout: 15,
					},
					DisableXEnvoyHeaders: true,
					AddXRealIpHeader:     true,
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
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"name": "envoy.filters.http.router",
							"typed_config": map[string]interface{}{
								"@type":                  "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
								"suppress_envoy_headers": true,
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"http2_protocol_options": map[string]interface{}{
								"max_concurrent_streams":         100,
								"initial_stream_window_size":     65535,
								"initial_connection_window_size": 1048576,
							},
							"stream_idle_timeout":    "180s",
							"max_request_headers_kb": 60,
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "180s",
							},
						},
					},
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config.virtual_hosts.#.routes.#.route",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"timeout": "15s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
							"per_connection_buffer_limit_bytes": 10485760,
						},
					},
				},
			},
			{
				name: "modify Downstream",
				higressConfig: &configmap.HigressConfig{
					Downstream: &configmap.Downstream{
						IdleTimeout:            200,
						MaxRequestHeadersKb:    60,
						ConnectionBufferLimits: 32768,
						Http2: &configmap.Http2{
							MaxConcurrentStreams:        200,
							InitialStreamWindowSize:     65535,
							InitialConnectionWindowSize: 1048576,
						},
						RouteTimeout: 60,
					},
					DisableXEnvoyHeaders: true,
					AddXRealIpHeader:     true,
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
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"name": "envoy.filters.http.router",
							"typed_config": map[string]interface{}{
								"@type":                  "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
								"suppress_envoy_headers": true,
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"http2_protocol_options": map[string]interface{}{
								"max_concurrent_streams":         200,
								"initial_stream_window_size":     65535,
								"initial_connection_window_size": 1048576,
							},
							"stream_idle_timeout":    "200s",
							"max_request_headers_kb": 60,
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "200s",
							},
						},
					},
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config.virtual_hosts.#.routes.#.route",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"timeout": "60s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
						},
					},
				},
			},
			{
				name:          "did not set global config, downstream and upstream will use default value",
				higressConfig: &configmap.HigressConfig{},
				envoyAssertion: []envoy.Assertion{
					{
						Path:            "configs.#.dynamic_route_configs.#.route_config.request_headers_to_add.#.header",
						CheckType:       envoy.CheckTypeNotExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"key":   "x-real-ip",
							"value": "%REQ(X-ENVOY-EXTERNAL-ADDRESS)%",
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
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"max_concurrent_streams":         100,
							"initial_stream_window_size":     65535,
							"initial_connection_window_size": 1048576,
							"stream_idle_timeout":            "180s",
							"max_request_headers_kb":         60,
							"idle_timeout":                   "180s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
						},
					},
				},
			},
			{
				name: "close the setting of idle timeout in downstream",
				higressConfig: &configmap.HigressConfig{
					Downstream: &configmap.Downstream{
						IdleTimeout:            0,
						MaxRequestHeadersKb:    60,
						ConnectionBufferLimits: 32768,
						Http2: &configmap.Http2{
							MaxConcurrentStreams:        100,
							InitialStreamWindowSize:     65535,
							InitialConnectionWindowSize: 1048576,
						},
					},
					Upstream: &configmap.Upstream{
						IdleTimeout: 10,
					},
					DisableXEnvoyHeaders: true,
					AddXRealIpHeader:     true,
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
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"name": "envoy.filters.http.router",
							"typed_config": map[string]interface{}{
								"@type":                  "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
								"suppress_envoy_headers": true,
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"max_concurrent_streams":         100,
							"initial_stream_window_size":     65535,
							"initial_connection_window_size": 1048576,
							"stream_idle_timeout":            "0s",
							"max_request_headers_kb":         60,
							"idle_timeout":                   "0s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
							"per_connection_buffer_limit_bytes": 10485760,
						},
					},
				},
			},
			{
				name: "close the setting of route timeout in downstream",
				higressConfig: &configmap.HigressConfig{
					Downstream: &configmap.Downstream{
						IdleTimeout:            180,
						MaxRequestHeadersKb:    60,
						ConnectionBufferLimits: 32768,
						Http2: &configmap.Http2{
							MaxConcurrentStreams:        100,
							InitialStreamWindowSize:     65535,
							InitialConnectionWindowSize: 1048576,
						},
						RouteTimeout: 0,
					},
					Upstream: &configmap.Upstream{
						IdleTimeout: 10,
					},
					DisableXEnvoyHeaders: true,
					AddXRealIpHeader:     true,
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
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"name": "envoy.filters.http.router",
							"typed_config": map[string]interface{}{
								"@type":                  "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
								"suppress_envoy_headers": true,
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"max_concurrent_streams":         100,
							"initial_stream_window_size":     65535,
							"initial_connection_window_size": 1048576,
							"stream_idle_timeout":            "180s",
							"max_request_headers_kb":         60,
							"idle_timeout":                   "180s",
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "10s",
							},
							"per_connection_buffer_limit_bytes": 10485760,
						},
					},
				},
			},
			{
				name: "close the setting of idle timeout in upstream",
				higressConfig: &configmap.HigressConfig{
					Downstream: &configmap.Downstream{
						IdleTimeout:            180,
						MaxRequestHeadersKb:    60,
						ConnectionBufferLimits: 32768,
						Http2: &configmap.Http2{
							MaxConcurrentStreams:        100,
							InitialStreamWindowSize:     65535,
							InitialConnectionWindowSize: 1048576,
						},
					},
					Upstream: &configmap.Upstream{
						IdleTimeout:            0,
						ConnectionBufferLimits: 32768,
					},
					DisableXEnvoyHeaders: true,
					AddXRealIpHeader:     true,
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
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"name": "envoy.filters.http.router",
							"typed_config": map[string]interface{}{
								"@type":                  "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
								"suppress_envoy_headers": true,
							},
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
					{
						Path:            "configs.#.dynamic_listeners.#.active_state.listener.filter_chains.#.filters.#.typed_config",
						CheckType:       envoy.CheckTypeExist,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"http2_protocol_options": map[string]interface{}{
								"max_concurrent_streams":         100,
								"initial_stream_window_size":     65535,
								"initial_connection_window_size": 1048576,
							},
							"stream_idle_timeout":    "180s",
							"max_request_headers_kb": 60,
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "180s",
							},
						},
					},
					{
						Path:            "configs.#.dynamic_active_clusters.#.cluster",
						CheckType:       envoy.CheckTypeMatch,
						TargetNamespace: "higress-system",
						ExpectEnvoyConfig: map[string]interface{}{
							"common_http_protocol_options": map[string]interface{}{
								"idle_timeout": "0s",
							},
							"per_connection_buffer_limit_bytes": 32768,
						},
					},
				},
			},
		}

		t.Run("ConfigMap Global Envoy", func(t *testing.T) {
			for _, testcase := range testCases {
				// apply config
				err := kubernetes.ApplyConfigmapDataWithYaml(t, suite.Client, "higress-system", "higress-config", "higress", testcase.higressConfig)
				if err != nil {
					t.Fatalf("can't apply conifgmap %s in namespace %s for data key %s", "higress-config", "higress-system", "higress")
				}
				t.Logf("Test Case %s", testcase.name)
				for _, assertion := range testcase.envoyAssertion {
					envoy.AssertEnvoyConfig(t, suite.TimeoutConfig, assertion)
				}
			}
		})
	},
}
