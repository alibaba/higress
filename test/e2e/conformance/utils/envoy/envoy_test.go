/*
Copyright 2022 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package envoy

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

var testActualJSON = `
{
"dynamic_listeners": [
    {
     "name": "0.0.0.0_80",
     "active_state": {
      "version_info": "2023-12-21T05:56:53Z/50",
      "listener": {
       "@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
       "name": "0.0.0.0_80",
       "address": {
        "socket_address": {
         "address": "0.0.0.0",
         "port_value": 80
        }
       },
       "filter_chains": [
        {
         "filters": [
          {
           "name": "envoy.filters.network.http_connection_manager",
           "typed_config": {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "stat_prefix": "outbound_0.0.0.0_80",
            "rds": {
             "config_source": {
              "ads": {},
              "initial_fetch_timeout": "0s",
              "resource_api_version": "V3"
             },
             "route_config_name": "http.80"
            },
            "http_filters": [
             {
              "name": "envoy.filters.http.cors",
              "typed_config": {
               "@type": "type.googleapis.com/envoy.extensions.filters.http.cors.v3.Cors"
              }
             },
             {
              "name": "envoy.filters.http.rbac",
              "typed_config": {
               "@type": "type.googleapis.com/envoy.extensions.filters.http.rbac.v3.RBAC"
              }
             },
             {
              "name": "envoy.filters.http.local_ratelimit",
              "typed_config": {
               "@type": "type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit",
               "stat_prefix": "http_local_rate_limiter"
              }
             },
             {
              "name": "envoy.filters.http.fault",
              "typed_config": {
               "@type": "type.googleapis.com/envoy.extensions.filters.http.fault.v3.HTTPFault"
              }
             },
             {
              "name": "envoy.filters.http.compressor",
              "typed_config": {
               "@type": "type.googleapis.com/envoy.extensions.filters.http.compressor.v3.Compressor",
               "compressor_library": {
                "name": "text_optimized",
                "typed_config": {
                 "@type": "type.googleapis.com/envoy.extensions.compression.gzip.compressor.v3.Gzip",
                 "memory_level": 5,
                 "compression_level": "COMPRESSION_LEVEL_9",
                 "window_bits": 12
                }
               },
               "request_direction_config": {
                "common_config": {
                 "enabled": {
                  "default_value": false,
                  "runtime_key": "request_compressor_enabled"
                 }
                }
               },
               "response_direction_config": {
                "common_config": {
                 "min_content_length": 100,
                 "content_type": [
                  "text/html",
                  "text/css",
                  "text/plain",
                  "text/xml",
                  "application/json",
                  "application/javascript",
                  "application/xhtml+xml",
                  "image/svg+xml"
                 ]
                }
               }
              }
             },
             {
              "name": "envoy.filters.http.router",
              "typed_config": {
               "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
              }
             }
            ],
            "tracing": {
             "client_sampling": {
              "value": 100
             },
             "random_sampling": {
              "value": 1
             },
             "overall_sampling": {
              "value": 100
             },
             "custom_tags": [
              {
               "tag": "istio.authorization.dry_run.allow_policy.name",
               "metadata": {
                "kind": {
                 "request": {}
                },
                "metadata_key": {
                 "key": "envoy.filters.http.rbac",
                 "path": [
                  {
                   "key": "istio_dry_run_allow_shadow_effective_policy_id"
                  }
                 ]
                }
               }
              },
              {
               "tag": "istio.authorization.dry_run.allow_policy.result",
               "metadata": {
                "kind": {
                 "request": {}
                },
                "metadata_key": {
                 "key": "envoy.filters.http.rbac",
                 "path": [
                  {
                   "key": "istio_dry_run_allow_shadow_engine_result"
                  }
                 ]
                }
               }
              },
              {
               "tag": "istio.authorization.dry_run.deny_policy.name",
               "metadata": {
                "kind": {
                 "request": {}
                },
                "metadata_key": {
                 "key": "envoy.filters.http.rbac",
                 "path": [
                  {
                   "key": "istio_dry_run_deny_shadow_effective_policy_id"
                  }
                 ]
                }
               }
              },
              {
               "tag": "istio.authorization.dry_run.deny_policy.result",
               "metadata": {
                "kind": {
                 "request": {}
                },
                "metadata_key": {
                 "key": "envoy.filters.http.rbac",
                 "path": [
                  {
                   "key": "istio_dry_run_deny_shadow_engine_result"
                  }
                 ]
                }
               }
              },
              {
               "tag": "istio.canonical_revision",
               "literal": {
                "value": "latest"
               }
              },
              {
               "tag": "istio.canonical_service",
               "literal": {
                "value": "unknown"
               }
              },
              {
               "tag": "istio.mesh_id",
               "literal": {
                "value": "unknown"
               }
              },
              {
               "tag": "istio.namespace",
               "literal": {
                "value": "higress-system"
               }
              }
             ]
            },
            "http_protocol_options": {
             "accept_http_10": true
            },
            "server_name": "istio-envoy",
            "access_log": [
             {
              "name": "envoy.access_loggers.file",
              "filter": {
               "not_health_check_filter": {}
              },
              "typed_config": {
               "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
               "path": "/dev/stdout",
               "log_format": {
                "text_format_source": {
                 "inline_string": "{\"authority\":\"%REQ(:AUTHORITY)%\",\"bytes_received\":\"%BYTES_RECEIVED%\",\"bytes_sent\":\"%BYTES_SENT%\",\"downstream_local_address\":\"%DOWNSTREAM_LOCAL_ADDRESS%\",\"downstream_remote_address\":\"%DOWNSTREAM_REMOTE_ADDRESS%\",\"duration\":\"%DURATION%\",\"istio_policy_status\":\"%DYNAMIC_METADATA(istio.mixer:status)%\",\"method\":\"%REQ(:METHOD)%\",\"path\":\"%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%\",\"protocol\":\"%PROTOCOL%\",\"request_id\":\"%REQ(X-REQUEST-ID)%\",\"requested_server_name\":\"%REQUESTED_SERVER_NAME%\",\"response_code\":\"%RESPONSE_CODE%\",\"response_flags\":\"%RESPONSE_FLAGS%\",\"route_name\":\"%ROUTE_NAME%\",\"start_time\":\"%START_TIME%\",\"trace_id\":\"%REQ(X-B3-TRACEID)%\",\"upstream_cluster\":\"%UPSTREAM_CLUSTER%\",\"upstream_host\":\"%UPSTREAM_HOST%\",\"upstream_local_address\":\"%UPSTREAM_LOCAL_ADDRESS%\",\"upstream_service_time\":\"%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%\",\"upstream_transport_failure_reason\":\"%UPSTREAM_TRANSPORT_FAILURE_REASON%\",\"user_agent\":\"%REQ(USER-AGENT)%\",\"x_forwarded_for\":\"%REQ(X-FORWARDED-FOR)%\"}\n"
                }
               }
              }
             }
            ],
            "use_remote_address": true,
            "forward_client_cert_details": "SANITIZE_SET",
            "set_current_client_cert_details": {
             "subject": true,
             "cert": true,
             "dns": true,
             "uri": true
            },
            "upgrade_configs": [
             {
              "upgrade_type": "websocket"
             }
            ],
            "stream_idle_timeout": "0s",
            "normalize_path": true,
            "path_with_escaped_slashes_action": "KEEP_UNCHANGED"
           }
          }
         ]
        }
       ],
       "traffic_direction": "OUTBOUND",
       "access_log": [
        {
         "name": "envoy.access_loggers.file",
         "filter": {
          "response_flag_filter": {
           "flags": [
            "NR"
           ]
          }
         },
         "typed_config": {
          "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
          "path": "/dev/stdout",
          "log_format": {
           "text_format_source": {
            "inline_string": "{\"authority\":\"%REQ(:AUTHORITY)%\",\"bytes_received\":\"%BYTES_RECEIVED%\",\"bytes_sent\":\"%BYTES_SENT%\",\"downstream_local_address\":\"%DOWNSTREAM_LOCAL_ADDRESS%\",\"downstream_remote_address\":\"%DOWNSTREAM_REMOTE_ADDRESS%\",\"duration\":\"%DURATION%\",\"istio_policy_status\":\"%DYNAMIC_METADATA(istio.mixer:status)%\",\"method\":\"%REQ(:METHOD)%\",\"path\":\"%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%\",\"protocol\":\"%PROTOCOL%\",\"request_id\":\"%REQ(X-REQUEST-ID)%\",\"requested_server_name\":\"%REQUESTED_SERVER_NAME%\",\"response_code\":\"%RESPONSE_CODE%\",\"response_flags\":\"%RESPONSE_FLAGS%\",\"route_name\":\"%ROUTE_NAME%\",\"start_time\":\"%START_TIME%\",\"trace_id\":\"%REQ(X-B3-TRACEID)%\",\"upstream_cluster\":\"%UPSTREAM_CLUSTER%\",\"upstream_host\":\"%UPSTREAM_HOST%\",\"upstream_local_address\":\"%UPSTREAM_LOCAL_ADDRESS%\",\"upstream_service_time\":\"%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%\",\"upstream_transport_failure_reason\":\"%UPSTREAM_TRANSPORT_FAILURE_REASON%\",\"user_agent\":\"%REQ(USER-AGENT)%\",\"x_forwarded_for\":\"%REQ(X-FORWARDED-FOR)%\"}\n"
           }
          }
         }
        }
       ]
      },
      "last_updated": "2023-12-21T05:57:47.497Z"
     }
    }
   ]
}
`

func Test_match(t *testing.T) {
	testCases := []struct {
		name         string
		actual       interface{}
		expected     map[string]interface{}
		expectResult bool
	}{
		{
			name: "case 1",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				map[string]interface{}{
					"foo": "baz",
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 2",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				map[string]interface{}{
					"foo": "baz",
				},
			},
			expected: map[string]interface{}{
				"foo": "bay",
			},
			expectResult: false,
		},
		{
			name: "case 3",
			actual: map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
				"baz": "bay",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 4",
			actual: map[string]interface{}{
				"foo": "bar",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 5",
			actual: map[string]interface{}{
				"foo": "bar",
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 6",
			actual: []interface{}{
				[]interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 7",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := match(t, testCase.actual, testCase.expected)
			if result != testCase.expectResult {
				t.Errorf("expected %v, got %v", testCase.expectResult, result)
			}
		})
	}
}

func Test_findMustExist(t *testing.T) {
	testCases := []struct {
		name         string
		actual       interface{}
		expected     map[string]interface{}
		expectResult bool
	}{
		{
			name: "case 1",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 2",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 3",
			actual: map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
				"baz": "bay",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 4",
			actual: map[string]interface{}{
				"foo": "bar",
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 5",
			actual: []interface{}{
				[]interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 6",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 7",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"test": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo":  "bar",
				"test": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 8",
			actual: []interface{}{
				map[string]interface{}{
					"foo":  "bar",
					"test": "baz",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo":  "baz",
				"test": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 9",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					[]interface{}{
						map[string]interface{}{
							"foo": "baz",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 9",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					[]interface{}{
						map[string]interface{}{
							"foo": "baz",
						},
					},
				},
				map[string]interface{}{
					"content": []interface{}{
						"one",
						"two",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
				"content": []interface{}{
					"one",
					"two",
				},
			},
			expectResult: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := findMustExist(t, testCase.actual, testCase.expected)
			if result != testCase.expectResult {
				t.Errorf("expected %v, got %v", testCase.expectResult, result)
			}
		})
	}
}

func Test_findMustNotExist(t *testing.T) {
	testCases := []struct {
		name         string
		actual       interface{}
		expected     map[string]interface{}
		expectResult bool
	}{
		{
			name: "case 1",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: false,
		},
		{
			name: "case 2",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 3",
			actual: map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
				"baz": "bay",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: false,
		},
		{
			name: "case 4",
			actual: map[string]interface{}{
				"foo": "bar",
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 5",
			actual: []interface{}{
				[]interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: false,
		},
		{
			name: "case 6",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 7",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"test": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: false,
		},
		{
			name: "case 8",
			actual: []interface{}{
				map[string]interface{}{
					"foo":  "bar",
					"test": "baz",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo":  "baz",
				"test": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 9",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					[]interface{}{
						map[string]interface{}{
							"foo": "baz",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := findMustNotExist(t, testCase.actual, testCase.expected)
			if result != testCase.expectResult {
				t.Errorf("expected %v, got %v", testCase.expectResult, result)
			}
		})
	}
}

func Test_findFromEnvoyFilter(t *testing.T) {
	actual := make(map[string]interface{})
	err := json.Unmarshal([]byte(testActualJSON), &actual)
	require.NoError(t, err)
	testCase := []struct {
		name         string
		actual       interface{}
		expected     map[string]interface{}
		expectResult bool
	}{
		{
			name:   "case 1",
			actual: actual,
			expected: map[string]interface{}{
				"memory_level":      5,
				"compression_level": "COMPRESSION_LEVEL_9",
				"window_bits":       12,
			},
			expectResult: true,
		},
	}

	for _, test := range testCase {
		t.Run(test.name, func(t *testing.T) {
			result := findMustExist(t, test.actual, test.expected)
			if result != test.expectResult {
				t.Errorf("expected %v, got %v", test.expectResult, result)
			}
		})
	}
}
