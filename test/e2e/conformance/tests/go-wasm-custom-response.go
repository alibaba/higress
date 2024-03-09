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
	Register(WasmPluginsCustomResponse)
}

var WasmPluginsCustomResponse = suite.ConformanceTest{
	ShortName:   "WasmPluginsCustomResponse",
	Description: "The Ingress in the higress-conformance-infra namespace test the custom-response WASM plugin.",
	Manifests:   []string{"tests/go-wasm-custom-response.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "case 1: Match global config",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"key1": "value1",
						},
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte("{\"hello\":\"foo\"}"),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "case 2: Match rule config",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "bar.com",
						Path: "/",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"key2": "value2",
						},
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte("{\"hello\":\"bar\"}"),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "case 3: Match enable_on_status",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "baz.com",
						Path: "/",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"key3": "value3",
						},
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte("{\"hello\":\"baz\"}"),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 4: Not match enable_on_status",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetRequest,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "baz.com",
						Path:    "/",
						Headers: map[string]string{"Authorization": "Basic YWRtaW46MTIzNDU2"}, // base64("admin:123456")
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Host: "baz.com",
							Path: "/",
							Headers: map[string]string{
								"X-Mse-Consumer": "consumer1",
							},
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
					TestCaseName:  "case 5: Change status code",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "qux.com",
						Path: "/",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 201,
						Headers: map[string]string{
							"key5": "value5",
						},
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte("{\"hello\":\"qux\"}"),
					},
				},
			},
		}
		t.Run("WasmPlugins custom-response", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
