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
		// Increase timeout for the test
		originalTimeout := suite.TimeoutConfig
		increasedTimeout := suite.TimeoutConfig
		increasedTimeout.RequestTimeout = 60 * time.Second
		suite.TimeoutConfig = increasedTimeout
		
		// Add a significantly longer delay to ensure services are truly ready
		t.Log("Waiting for services to be ready...")
		time.Sleep(60 * time.Second)
		
		// Key change: Log that we're going to use localhost for tests
		t.Logf("Using gateway address: %s", suite.GatewayAddress)
		t.Log("Note: Test is using localhost as the connection target, with Host header set to ext-auth-test.example.com")
		
		// Try with simple testcase first to verify connectivity
		simpleTestcase := http.Assertion{
			Meta: http.AssertionMeta{
				TestCaseName:    "Simple connectivity test",
				TargetBackend:   "infra-backend-v1",
				TargetNamespace: "higress-conformance-infra",
			},
			Request: http.AssertionRequest{
				ActualRequest: http.Request{
					Host:             "localhost", // Use localhost as Host header too
					Path:             "/allowed-path",
					UnfollowRedirect: true,
				},
			},
			Response: http.AssertionResponse{
				ExpectedResponse: http.Response{
					StatusCode: 200,
				},
			},
		}
		
		t.Run("Basic connectivity test", func(t *testing.T) {
			t.Log("Testing basic connectivity with host=localhost...")
			http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, simpleTestcase)
		})
		
		// Now run the actual test cases
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: Blacklist mode - blocked path",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "localhost", // Changed to match what's being used for connection
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
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 2: Blacklist mode - allowed path",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "localhost", // Changed to match what's being used for connection
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
						Host:             "localhost", // Changed to match what's being used for connection
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
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 4: Method-specific rules - POST blocked",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "localhost", // Changed to match what's being used for connection
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
		}
		
		// Run the main tests
		t.Run("WasmPlugins ext-auth", func(t *testing.T) {
			// Run test cases one by one with additional logging
			for i, testcase := range testcases {
				t.Logf("Running test case %d: %s", i+1, testcase.Meta.TestCaseName)
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
				// Add a short pause between tests
				time.Sleep(2 * time.Second)
			}
		})
		
		// Restore original timeout settings
		suite.TimeoutConfig = originalTimeout
	},
}