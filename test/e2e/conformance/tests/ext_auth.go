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

	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
	Register(WasmPluginsExtAuth)
}

var WasmPluginsExtAuth = suite.ConformanceTest{
	ShortName:   "WasmPluginsExtAuth",
	Description: "The Ingress in the higress-conformance-infra namespace test the ext-auth wasmplugin.",
	Manifests:   []string{"tests/ext_auth.yaml", "tests/ext_auth_plugin.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		// First test the allowed paths to verify basic connectivity
		allowedPathTests := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 2: Blacklist mode - allowed path",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "localhost",
						Path:             "/allowed-path",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 3: Method-specific rules - GET allowed",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "localhost",
						Path:             "/api",
						Method:           "GET",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}
		
		// Add a small delay to let services fully initialize
		time.Sleep(10 * time.Second)
		
		// Run allowed path tests first
		t.Run("WasmPlugins ext-auth allowed paths", func(t *testing.T) {
			for _, testcase := range allowedPathTests {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
				// Add a small delay between tests
				time.Sleep(2 * time.Second)
			}
		})
		
		// Now test the blocked paths
		blockedPathTests := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 4: Method-specific rules - POST blocked",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "localhost",
						Path:             "/api",
						Method:           "POST",
						ContentType:      http.ContentTypeTextPlain,
						Body:             []byte(`test body`),
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: Blacklist mode - blocked path",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "localhost",
						Path:             "/blocked-path",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
					},
				},
			},
		}
		
		// Run blocked path tests after allowed path tests succeed
		t.Run("WasmPlugins ext-auth blocked paths", func(t *testing.T) {
			for _, testcase := range blockedPathTests {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
				// Add a small delay between tests
				time.Sleep(2 * time.Second)
			}
		})
	},
}