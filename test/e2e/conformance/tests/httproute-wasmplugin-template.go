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
	Register(HTTPRouteWasmpluginTemplate)
}

var HTTPRouteWasmpluginTemplate = suite.ConformanceTest{
	ShortName:   "HTTPRouteWasmpluginTemplate",
	Description: "A single Ingress in the higress-conformance-infra namespace test for wasmplugin template.",
	Manifests:   []string{"tests/httproute-wasmplugin-template.yaml"},
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo",
						Host: "foo.com",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 401,
					},
				},
			},
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo",
						Host: "foo.com",
						Headers: map[string]string{
							"authorization": "Bearer credential1",
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
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo",
						Host: "foo.com",
						Headers: map[string]string{
							"authorization": "Bearer credential2",
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
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo",
						Host: "foo.com",
						Headers: map[string]string{
							"authorization": "Bearer credential3",
						},
					},
				},

				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}

		t.Run("Match HTTPRoute by wasm plugin template", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
