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
	Register(WasmPluginsRequestValidation)
}

var WasmPluginsRequestValidation = suite.ConformanceTest{
	ShortName:   "WasmPluginsRequestValidation",
	Description: "The Ingress in the higress-conformance-infra namespace test the request-validation wasmplugins.",
	Manifests:   []string{"tests/go-wasm-request-validation.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testCases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "request validation pass",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"enum_payload": "enum_string_1",
								"bool_payload": true,
								"integer_payload": 100,
								"string_payload": "abc",
								"regex_payload": "abc123",
								"array_payload": [200, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
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
					TestCaseName:    "header lack of require parameter",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Body: []byte(`
							{
								"enum_payload": "enum_string_1",
								"bool_payload": true,
								"integer_payload": 100,
								"string_payload": "abc",
								"regex_payload": "abc123",
								"array_payload": [200, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "body lack of require parameter",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"bool_payload": true,
								"integer_payload": 100,
								"string_payload": "abc",
								"regex_payload": "abc123",
								"array_payload": [200, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "body enum payload not in enum list",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"enum_payload": "enum_string_3",
								"bool_payload": true,
								"integer_payload": 100,
								"string_payload": "abc",
								"regex_payload": "abc123",
								"array_payload": [200, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "body bool payload not bool type",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"enum_payload": "enum_string_1",
								"bool_payload": "string",
								"integer_payload": 100,
								"string_payload": "abc",
								"regex_payload": "abc123",
								"array_payload": [200, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "body integer payload not in range",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"enum_payload": "enum_string_1",
								"bool_payload": true,
								"integer_payload": 70000,
								"string_payload": "abc",
								"regex_payload": "abc123",
								"array_payload": [200, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "body string payload length not in range",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"enum_payload": "enum_string_1",
								"bool_payload": true,
								"integer_payload": 100,
								"string_payload": "a",
								"regex_payload": "abc123",
								"array_payload": [200, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "body regex payload not match regex pattern",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"enum_payload": "enum_string_1",
								"bool_payload": true,
								"integer_payload": 100,
								"string_payload": "abc",
								"regex_payload": "abc@123",
								"array_payload": [200, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "body array payload not in array range",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"enum_payload": "enum_string_1",
								"bool_payload": true,
								"integer_payload": 100,
								"string_payload": "abc",
								"regex_payload": "abc123",
								"array_payload": [150, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "body array payload not unique array items",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"enum_payload": "enum_string_1",
								"bool_payload": true,
								"integer_payload": 100,
								"string_payload": "abc",
								"regex_payload": "abc123",
								"array_payload": [302, 302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "body array payload length not in range",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo.com",
						Path: "/foo",
						Headers: map[string]string{
							"enum_payload": "enum_string_1",
						},
						Body: []byte(`
							{
								"enum_payload": "enum_string_1",
								"bool_payload": true,
								"integer_payload": 100,
								"string_payload": "abc",
								"regex_payload": "abc123",
								"array_payload": [302]
							}
 						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
						Body:       []byte(`customize reject message`),
					},
				},
			},
		}

		t.Run("WasmPlugins request-validation", func(t *testing.T) {
			for _, testcase := range testCases {
				t.Logf("Running test case: %s", testcase.Meta.TestCaseName)
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
