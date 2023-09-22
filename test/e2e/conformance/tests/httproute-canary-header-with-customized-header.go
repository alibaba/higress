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
	Register(HTTPRouteCanaryHeaderWithCustomizedHeader)
}

var HTTPRouteCanaryHeaderWithCustomizedHeader = suite.ConformanceTest{
	ShortName:   "HTTPRouteCanaryHeaderWithCustomizedHeader",
	Description: "The Ingress in the higress-conformance-infra namespace uses the canary header traffic split when same host and path but different header",
	Manifests:   []string{"tests/httproute-canary-header-with-customized-header.yaml"},
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				// test canary ingress with different customized header
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/echo",
						Host: "canary.higress.io",
						Headers: map[string]string{
							"traffic-split-higress": "true",
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 404,
					},
				},
			}, {
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/echo",
						Host: "canary.higress.io",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 404,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v2",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/echo",
						Host: "canary.higress.io",
						Headers: map[string]string{
							"abc": "123",
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
			// test canary ingress with same customized header
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/echo",
						Host: "same.canary.higress.io",
						Headers: map[string]string{
							"with-same-customized-header": "true",
							"user":                        "higress",
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
					TargetBackend:   "infra-backend-v2",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/echo",
						Host: "same.canary.higress.io",
						Headers: map[string]string{
							"user": "higress",
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

		t.Run("Canary HTTPRoute Traffic Split With customized header", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
