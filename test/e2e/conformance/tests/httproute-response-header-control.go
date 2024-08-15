// Copyright (c) 2023 Alibaba Group Holding Ltd.
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
	Register(HTTPRouteResponseHeaderControl)
}

var HTTPRouteResponseHeaderControl = suite.ConformanceTest{
	ShortName:   "HTTPRouteResponseHeaderControl",
	Description: "A single Ingress in the higress-conformance-infra namespace controls the response header.",
	Manifests:   []string{"tests/httproute-response-header-control.yaml"},
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: add one",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},

				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo1",
						Host: "foo.com",
					},
				},

				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"stage": "test",
						},
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 2: add more",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},

				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo2",
						Host: "foo.com",
					},
				},

				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"stage":   "test",
							"canary":  "true",
							"x-test":  "higress; test=true",
							"x-test2": "higress; test=false",
						},
					},
				},
			},
		}
		t.Run("Response header control", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
