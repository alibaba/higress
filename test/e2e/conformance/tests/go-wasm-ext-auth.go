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
	Description: "E2E tests for the ext-auth WASM plugin in both envoy and forward_auth modes, covering whitelist/blacklist, header propagation, and failure_mode_allow.",
	Manifests:   []string{"tests/go-wasm-ext-auth.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			// Envoy mode: successful auth
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo-envoy.com",
						Path:             "/allowed",
						Method:           "GET",
						Headers:          map[string]string{"Authorization": "Bearer valid-token"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 200}},
			},
			// Envoy mode: invalid token
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo-envoy.com",
						Path:             "/allowed",
						Method:           "GET",
						Headers:          map[string]string{"Authorization": "Bearer invalid-token"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 403}},
			},
			// Envoy mode: no auth header
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo-envoy.com",
						Path:             "/allowed",
						Method:           "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 401}},
			},
			// Envoy mode: whitelist path bypass
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backbone-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo-envoy.com",
						Path:             "/whitelisted",
						Method:           "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 200}},
			},
			// Forward_auth mode: successful auth
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backbone-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo-forward.com",
						Path:             "/allowed",
						Method:           "GET",
						Headers:          map[string]string{"Authorization": "Bearer valid-token"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 200}},
			},
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo-forward.com",
						Path:             "/allowed",
						Method:           "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 401}},
			},
			// Forward_auth mode: whitelist bypass
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo-forward.com",
						Path:             "/whitelisted",
						Method:           "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 200}},
			},
			// Blacklist in envoy mode: blocked path
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "bar-envoy.com",
						Path:             "/blocked",
						Method:           "GET",
						Headers:          map[string]string{"Authorization": "Bearer valid-token"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 403}},
			},
			// Blacklist in envoy mode: allowed path
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "bar-envoy.com",
						Path:             "/allowed",
						Method:           "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 200}},
			},
			// Blacklist method filter: POST blocked
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "bar-envoy.com",
						Path:             "/method-restricted",
						Method:           "POST",
						Headers:          map[string]string{"Authorization": "Bearer valid-token"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 403}},
			},
			// Blacklist method filter: GET allowed
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "bar-envoy.com",
						Path:             "/method-restricted",
						Method:           "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{ExpectedResponse: http.Response{StatusCode: 200}},
			},
			// Failure_mode_allow: auth service down
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "failover.com",
						Path:             "/test",
						Method:           "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers:    map[string]string{"x-envoy-auth-failure-mode-allowed": "true"},
					},
				},
			},
			// Forward_auth header propagation check (requires mock-inspect endpoint)
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1", 
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo-forward.com",
						Path:             "/inspect",
						Method:           "POST",
						Headers:          map[string]string{"Authorization": "Bearer valid-token"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"X-Forwarded-Proto":  "HTTP",
							"X-Forwarded-Host":   "foo-forward.com",
							"X-Forwarded-Uri":    "/inspect",
							"X-Forwarded-Method": "POST",
						},
					},
				},
			},
		}

		t.Run("ext-auth plugin", func(t *testing.T) {
			for _, tc := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(
					t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, tc,
				)
			}
		})
	},
}