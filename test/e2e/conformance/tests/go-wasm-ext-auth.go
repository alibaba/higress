
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
	"fmt"
	"testing"

	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
	Register(WasmPluginsExtAuth)
}

var WasmPluginsExtAuth = suite.ConformanceTest{
	ShortName:   "WasmPluginsExtAuth",
	Description: "E2E tests for the ext-auth WASM plugin (envoy & forward_auth modes, whitelist/blacklist, failure_mode_allow, header propagation).",
	Manifests:   []string{"tests/go-wasm-ext-auth.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			// Envoy mode: successful auth
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Envoy Mode - Successful Authentication",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo-envoy.com", 
						Path: "/allowed", 
						Method: "GET", 
						Headers: map[string]string{"Authorization": "Bearer valid-token"}, 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 200},
				},
			},
			// Envoy mode: invalid token
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Envoy Mode - Invalid Token",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo-envoy.com", 
						Path: "/allowed", 
						Method: "GET", 
						Headers: map[string]string{"Authorization": "Bearer invalid-token"}, 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 403},
				},
			},
			// Envoy mode: no auth header
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Envoy Mode - Missing Auth Header",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo-envoy.com", 
						Path: "/allowed", 
						Method: "GET", 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 401},
				},
			},
			// Envoy mode: whitelist bypass
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Envoy Mode - Whitelist Bypass",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo-envoy.com", 
						Path: "/whitelisted", 
						Method: "GET", 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 200},
				},
			},
			// Forward_auth mode: successful auth
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Forward_Auth Mode - Successful Authentication",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo-forward.com", 
						Path: "/allowed", 
						Method: "GET", 
						Headers: map[string]string{"Authorization": "Bearer valid-token"}, 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 200},
				},
			},
			// Forward_auth mode: missing auth header
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Forward_Auth Mode - Missing Auth Header",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo-forward.com", 
						Path: "/allowed", 
						Method: "GET", 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 401},
				},
			},
			// Forward_auth mode: whitelist bypass
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Forward_Auth Mode - Whitelist Bypass",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo-forward.com", 
						Path: "/whitelisted", 
						Method: "GET", 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 200},
				},
			},
			// Blacklist envoy: blocked
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Blacklist - Path Blocked",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "bar-envoy.com", 
						Path: "/blocked", 
						Method: "GET", 
						Headers: map[string]string{"Authorization": "Bearer valid-token"}, 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 403},
				},
			},
			// Blacklist envoy: allowed
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Blacklist - Path Allowed",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "bar-envoy.com", 
						Path: "/allowed", 
						Method: "GET", 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 200},
				},
			},
			// Blacklist POST blocked
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Blacklist - Method Restricted (POST Blocked)",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "bar-envoy.com", 
						Path: "/method-restricted", 
						Method: "POST", 
						Headers: map[string]string{"Authorization": "Bearer valid-token"}, 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 403},
				},
			},
			// Blacklist GET allowed
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Blacklist - Method Restricted (GET Allowed)",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "bar-envoy.com", 
						Path: "/method-restricted", 
						Method: "GET", 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 200},
				},
			},
			// Failure mode allow
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Failure Mode Allow - Service Unavailable",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "failover.com", 
						Path: "/test", 
						Method: "GET", 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200, 
						Headers: map[string]string{
							"x-envoy-auth-failure-mode-allowed": "true",
						},
					},
				},
			},
			// Header propagation
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Forward_Auth Mode - Header Propagation",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo-forward.com", 
						Path: "/inspect", 
						Method: "POST", 
						Headers: map[string]string{"Authorization": "Bearer valid-token"}, 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200, 
						Headers: map[string]string{
							"X-Forwarded-Proto": "HTTP", 
							"X-Forwarded-Host": "foo-forward.com", 
							"X-Forwarded-Uri": "/inspect", 
							"X-Forwarded-Method": "POST",
						},
					},
				},
			},
			// Request body testing
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Request Body Testing - JSON",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "body-test.com", 
						Path: "/inspect-body", 
						Method: "POST", 
						Headers: map[string]string{
							"Authorization": "Bearer valid-token",
							"Content-Type": "application/json",
						}, 
						Body: []byte(`{"test":"data","key":"value"}`),
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 200},
				},
			},
			// Custom status error code test
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Custom Status on Error",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "custom-error.com", 
						Path: "/test", 
						Method: "GET", 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 503},
				},
			},
			// Header transformation test
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Header Transformation",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "header-test.com", 
						Path: "/test-headers", 
						Method: "GET", 
						Headers: map[string]string{
							"Authorization": "Bearer valid-token",
							"X-Custom-Header": "should-be-forwarded",
							"X-Auth-Version": "1.0",
						}, 
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200, 
						Headers: map[string]string{
							"X-User-Id": "test-user",
							"X-Auth-Version": "1.0",
						},
					},
				},
			},
			// Large request body test (testing max_request_body_bytes)
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Large Request Body Test",
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "body-test.com", 
						Path: "/large-body", 
						Method: "POST", 
						Headers: map[string]string{
							"Authorization": "Bearer valid-token",
							"Content-Type": "application/json",
						}, 
						// Generate a large JSON body
						Body: []byte(generateLargeBody(1024 * 1024)), // 1MB body
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 200},
				},
			},
			// Timeout test (new)
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Timeout Test",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "timeout-test.com",
						Path: "/test-timeout",
						Method: "GET",
						Headers: map[string]string{
							"X-Sleep-Time": "2000", // 2 seconds, longer than the 1s timeout
						},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 504}, // Gateway Timeout
				},
			},
			// Body size limit exceeded test (new)
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Body Size Limit Exceeded",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "body-limit-test.com",
						Path: "/test-body-limit",
						Method: "POST",
						Headers: map[string]string{
							"Authorization": "Bearer valid-token",
							"Content-Type": "application/json",
						},
						// Generate a body larger than the defined limit (100KB)
						Body: []byte(generateLargeBody(150 * 1024)), // 150KB, exceeds 100KB limit
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 413}, // Payload Too Large
				},
			},
		}

		t.Run("ext-auth plugin", func(t *testing.T) {
			for _, tc := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(
					t, s.RoundTripper, s.TimeoutConfig, s.GatewayAddress, tc,
				)
			}
		})
	},
}

// Helper function to generate a large JSON body for testing
func generateLargeBody(sizeInBytes int) string {
	// Create a simple repeating pattern to reach desired size
	const basePattern = `{"data":"0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"}`
	baseLen := len(basePattern)
	
	// Calculate how many times to repeat the pattern
	repetitions := sizeInBytes / baseLen
	if repetitions < 1 {
		repetitions = 1
	}
	
	// Build the large body
	body := "{"
	for i := 0; i < repetitions; i++ {
		if i > 0 {
			body += ","
		}
		body += fmt.Sprintf(`"field%d":%s`, i, basePattern)
	}
	body += "}"
	
	return body
}