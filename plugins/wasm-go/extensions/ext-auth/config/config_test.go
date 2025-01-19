package config

import (
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expected    ExtAuthConfig
		expectedErr string
	}{
		{
			name: "Valid Config with Default Values",
			json: `{
				"http_service": {
					"endpoint_mode": "envoy",
					"endpoint": {
						"service_name": "example.com",
						"service_port": 80,
						"path_prefix": "/auth"
					}
				}
			}`,
			expected: ExtAuthConfig{
				HttpService: HttpService{
					EndpointMode: "envoy",
					Client: wrapper.NewClusterClient(wrapper.FQDNCluster{
						FQDN: "example.com",
						Port: 80,
						Host: "",
					}),
					PathPrefix: "/auth",
					Timeout:    1000,
				},
				SkippedPathPrefixes:       []string{},
				FailureModeAllow:          false,
				FailureModeAllowHeaderAdd: false,
				StatusOnError:             403,
			},
		},
		{
			name: "Valid Config with Custom Values",
			json: `{
				"http_service": {
					"endpoint_mode": "forward_auth",
					"endpoint": {
						"service_name": "auth.example.com",
						"service_port": 8080,
						"service_host": "auth.example.com",
						"request_method": "POST",
						"path": "/auth"
					},
					"timeout": 2000,
					"authorization_request": {
						"headers_to_add": {
							"X-Auth-Source": "wasm"
						},
						"with_request_body": true,
						"max_request_body_bytes": 1048576
					}
				},
				"skipped_path_prefixes": ["/health", "/metrics"],
				"failure_mode_allow": true,
				"failure_mode_allow_header_add": true,
				"status_on_error": 500
			}`,
			expected: ExtAuthConfig{
				HttpService: HttpService{
					EndpointMode: "forward_auth",
					Client: wrapper.NewClusterClient(wrapper.FQDNCluster{
						FQDN: "auth.example.com",
						Port: 8080,
						Host: "auth.example.com",
					}),
					RequestMethod: "POST",
					Path:          "/auth",
					Timeout:       2000,
					AuthorizationRequest: AuthorizationRequest{
						HeadersToAdd: map[string]string{
							"X-Auth-Source": "wasm",
						},
						WithRequestBody:     true,
						MaxRequestBodyBytes: 1048576,
					},
				},
				SkippedPathPrefixes:       []string{"/health", "/metrics"},
				FailureModeAllow:          true,
				FailureModeAllowHeaderAdd: true,
				StatusOnError:             500,
			},
		},
		{
			name:        "Missing HttpService Configuration",
			json:        `{}`,
			expectedErr: "missing http_service in config",
		},
		{
			name: "Invalid Endpoint Mode",
			json: `{
				"http_service": {
					"endpoint_mode": "invalid_mode",
					"endpoint": {
						"service_name": "example.com",
						"service_port": 80
					}
				}
			}`,
			expectedErr: "endpoint_mode invalid_mode is not supported",
		},
		{
			name: "Missing Endpoint Configuration",
			json: `{
				"http_service": {
					"endpoint_mode": "envoy"
				}
			}`,
			expectedErr: "missing endpoint in config",
		},
		{
			name: "Empty Service Name",
			json: `{
				"http_service": {
					"endpoint_mode": "envoy",
					"endpoint": {
						"service_name": "",
						"service_port": 80
					}
				}
			}`,
			expectedErr: "endpoint service name must not be empty",
		},
		{
			name: "Invalid Request Method with Request Body",
			json: `{
				"http_service": {
					"endpoint_mode": "forward_auth",
					"endpoint": {
						"service_name": "auth.example.com",
						"service_port": 8080,
						"request_method": "GET",
						"path": "/auth"
					},
					"authorization_request": {
						"with_request_body": true
					}
				}
			}`,
			expectedErr: "requestMethod GET does not support with_request_body set to true",
		},
		{
			name: "Missing Path for Forward Auth",
			json: `{
				"http_service": {
					"endpoint_mode": "forward_auth",
					"endpoint": {
						"service_name": "auth.example.com",
						"service_port": 8080,
						"service_host": "auth.example.com",
						"request_method": "POST"
					}
				}
			}`,
			expectedErr: "when endpoint_mode is forward_auth, endpoint path must not be empty",
		},
		{
			name: "Missing Path Prefix for Envoy",
			json: `{
				"http_service": {
					"endpoint_mode": "envoy",
					"endpoint": {
						"service_name": "example.com",
						"service_port": 80
					}
				}
			}`,
			expectedErr: "when endpoint_mode is envoy, endpoint path_prefix must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config ExtAuthConfig
			result := gjson.Parse(tt.json)
			err := ParseConfig(result, &config, &wrapper.DefaultLog{})

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, config)
			}
		})
	}
}
