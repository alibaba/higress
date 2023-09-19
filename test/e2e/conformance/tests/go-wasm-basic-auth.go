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
	Register(WasmPluginsBasicAuth)
}

var WasmPluginsBasicAuth = suite.ConformanceTest{
	ShortName:   "WasmPluginsBasicAuth",
	Description: "The Ingress in the higress-conformance-infra namespace test the basic-auth WASM plugin.",
	Manifests:   []string{"tests/go-wasm-basic-auth.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: Successful authentication",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo.com",
						Path:    "/foo",
						Headers: map[string]string{"Authorization": "Basic YWRtaW46MTIzNDU2"}, // base64("admin:123456")
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Host:    "foo.com",
							Path:    "/foo",
							Headers: map[string]string{"X-Mse-Consumer": "consumer1"},
						},
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
					TestCaseName:    "case 2: No Basic Authentication information found",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 401,
					},
					AdditionalResponseHeaders: map[string]string{
						"WWW-Authenticate": "Basic realm=MSE Gateway",
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 3: Invalid username and/or password",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo.com",
						Path:    "/foo",
						Headers: map[string]string{"Authorization": "Basic YWRtaW46cXdlcg=="}, // base64("admin:qwer")
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 401,
					},
					AdditionalResponseHeaders: map[string]string{
						"WWW-Authenticate": "Basic realm=MSE Gateway",
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 4: Unauthorized consumer",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo.com",
						Path:    "/foo",
						Headers: map[string]string{"Authorization": "Basic Z3Vlc3Q6YWJj"}, // base64("guest:abc")
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
					},
					AdditionalResponseHeaders: map[string]string{
						"WWW-Authenticate": "Basic realm=MSE Gateway",
					},
				},
			},
		}
		t.Run("WasmPlugins basic-auth", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
