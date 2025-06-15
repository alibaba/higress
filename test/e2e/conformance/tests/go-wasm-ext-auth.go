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
	Description: "E2E tests for the ext-auth WASM plugin using mock server endpoints (/always-200, /always-500, etc).",
	Manifests:   []string{"tests/go-wasm-ext-auth.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, s *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			// Always-200 with valid token
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Always 200 - Valid Token",
					TargetBackend:   "echo-server",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo-envoy.com",
						Path:    "/always-200",
						Method:  "GET",
						Headers: map[string]string{"Authorization": "Bearer valid-token"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"X-User-ID": "123456",
						},
					},
				},
			},
			// Always-200 with missing token
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Always 200 - Missing Token (Should Fail)",
					TargetBackend:   "echo-server",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo-envoy.com",
						Path:    "/always-200",
						Method:  "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 401},
				},
			},
			// Always-500 with valid token
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Always 500 - Valid Token",
					TargetBackend:   "echo-server",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo-envoy.com",
						Path:    "/always-500",
						Method:  "GET",
						Headers: map[string]string{"Authorization": "Bearer valid-token"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 500},
				},
			},
			// Require body with POST and body present
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Require Body - Valid Token and Body",
					TargetBackend:   "echo-server",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo-envoy.com",
						Path:    "/require-request-body-200",
						Method:  "POST",
						Body:    []byte(`{"key":"value"}`),
						Headers: map[string]string{"Authorization": "Bearer valid-token", "Content-Type": "application/json"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"X-User-ID": "123456",
						},
					},
				},
			},
			// Require body with POST but missing body
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "Require Body - Missing Body",
					TargetBackend:   "echo-server",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo-envoy.com",
						Path:    "/require-request-body-200",
						Method:  "POST",
						Headers: map[string]string{"Authorization": "Bearer valid-token"},
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{StatusCode: 400},
				},
			},
		}

		t.Run("ext-auth plugin mock server tests", func(t *testing.T) {
			for _, tc := range testcases {
				tc := tc // capture variable
				t.Run(tc.Meta.TestCaseName, func(t *testing.T) {
					http.MakeRequestAndExpectEventuallyConsistentResponse(
						t, s.RoundTripper, s.TimeoutConfig, s.GatewayAddress, tc,
					)
				})
			}
		})
	},
}
