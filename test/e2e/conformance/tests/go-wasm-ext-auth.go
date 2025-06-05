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
			// Basic Envoy mode - successful authentication
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Envoy Mode - Successful Authentication",
					TargetBackend:   "echo-server", 
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
			// Envoy mode - invalid token should return 403
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Envoy Mode - Invalid Token",
					TargetBackend:   "echo-server", 
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
			// Envoy mode - missing auth header should return 401
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Envoy Mode - Missing Auth Header",
					TargetBackend:   "echo-server", 
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
			// Envoy mode - whitelist bypass (should pass without auth)
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Envoy Mode - Whitelist Bypass",
					TargetBackend:   "echo-server", 
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
			// Forward_auth mode - successful authentication
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Forward_Auth Mode - Successful Authentication",
					TargetBackend:   "echo-server", 
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
			// Forward_auth mode - missing auth header
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Forward_Auth Mode - Missing Auth Header",
					TargetBackend:   "echo-server", 
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
			// Forward_auth mode - whitelist bypass
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Forward_Auth Mode - Whitelist Bypass",
					TargetBackend:   "echo-server", 
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
			// Blacklist mode - blocked path should return 403
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Blacklist - Path Blocked",
					TargetBackend:   "echo-server", 
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
			// Blacklist mode - allowed path should return 200
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Blacklist - Path Allowed",
					TargetBackend:   "echo-server", 
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
			// Blacklist mode - POST method blocked
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Blacklist - Method Restricted (POST Blocked)",
					TargetBackend:   "echo-server", 
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
			// Blacklist mode - GET method allowed
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Blacklist - Method Restricted (GET Allowed)",
					TargetBackend:   "echo-server", 
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
			// Failure mode allow - when auth service is unavailable
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Failure Mode Allow - Service Unavailable",
					TargetBackend:   "echo-server", 
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
		}

		t.Run("ext-auth plugin comprehensive tests", func(t *testing.T) {
			for _, tc := range testcases {
				tc := tc // capture loop variable
				t.Run(tc.Meta.TestCaseName, func(t *testing.T) {
					http.MakeRequestAndExpectEventuallyConsistentResponse(
						t, s.RoundTripper, s.TimeoutConfig, s.GatewayAddress, tc,
					)
				})
			}
		})
	},
}