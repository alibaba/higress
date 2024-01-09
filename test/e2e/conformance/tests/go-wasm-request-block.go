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
	Register(WasmPluginsRequestBlock)
}

var WasmPluginsRequestBlock = suite.ConformanceTest{
	ShortName:   "WasmPluginsRequestBlock",
	Description: "The Ingress in the higress-conformance-infra namespace test the request-block wasmplugins.",
	Manifests:   []string{"tests/go-wasm-request-block.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/swagger.html",
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
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/env/info",
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
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/web/info",
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
				// post blocked body
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/foo",
						Method:           "POST",
						ContentType:      http.ContentTypeTextPlain,
						Body:             []byte(`hello world`),
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
				// check body echoed back in expected request(same as ActualRequest if not set)
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetRequest,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/foo",
						Method:           "POST",
						ContentType:      http.ContentTypeTextPlain,
						Body:             []byte(`hello higress`),
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
				// check body echoed back in expected response
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-echo-body-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo2.com",
						Path:             "/foo",
						Method:           "POST",
						ContentType:      http.ContentTypeTextPlain,
						Body:             []byte(`hello higress`),
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeTextPlain,
						Body:        []byte(`hello higress`),
					},
				},
			},
		}
		t.Run("WasmPlugins request-block", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
