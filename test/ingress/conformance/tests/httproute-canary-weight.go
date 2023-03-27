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

	"github.com/alibaba/higress/test/ingress/conformance/utils/http"
	"github.com/alibaba/higress/test/ingress/conformance/utils/suite"
)

func init() {
	HigressConformanceTests = append(HigressConformanceTests, HTTPRouteCanaryWeight)
}

var HTTPRouteCanaryWeight = suite.ConformanceTest{
	ShortName:   "HTTPRouteCanaryWeight",
	Description: "The Ingress in the higress-conformance-infra namespace uses the canary weight traffic split.",
	Manifests:   []string{"tests/httproute-canary-weight.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		tt := []struct {
			assertion   http.Assertion
			minSuccRate float32
			maxSuccRate float32
		}{
			{
				minSuccRate: 0.9,
				maxSuccRate: 1.0,
				assertion: http.Assertion{
					// test if the weight is 0
					Meta: http.AssertionMeta{
						TargetBackend:   "infra-backend-v1",
						TargetNamespace: "higress-conformance-infra",
					},
					Request: http.AssertionRequest{
						ActualRequest: http.Request{
							Path: "/weight-0",
							Host: "canary.higress.io",
						},
					},
					Response: http.AssertionResponse{
						ExpectedResponse: http.Response{
							StatusCode: 200,
						},
					},
				},
			}, { // test if the weight is 100
				minSuccRate: 0.9,
				maxSuccRate: 1.0,
				assertion: http.Assertion{
					Meta: http.AssertionMeta{
						TargetBackend:   "infra-backend-v2",
						TargetNamespace: "higress-conformance-infra",
					},
					Request: http.AssertionRequest{
						ActualRequest: http.Request{
							Path: "/weight-100",
							Host: "canary.higress.io",
						},
					},
					Response: http.AssertionResponse{
						ExpectedResponse: http.Response{
							StatusCode: 200,
						},
					},
				},
			}, {
				minSuccRate: 0.4,
				maxSuccRate: 0.6,
				assertion: http.Assertion{
					Meta: http.AssertionMeta{
						TargetBackend:   "infra-backend-v2",
						TargetNamespace: "higress-conformance-infra",
					},
					Request: http.AssertionRequest{
						ActualRequest: http.Request{
							Path: "/weight-50",
							Host: "canary.higress.io",
						},
					},
					Response: http.AssertionResponse{
						ExpectedResponse: http.Response{
							StatusCode: 200,
						},
					},
				},
			},
		}

		t.Run("Canary HTTPRoute Traffic Split", func(t *testing.T) {
			for _, testcase := range tt {
				http.MakeRequestAndCountExpectedResponse(t, suite.RoundTripper, suite.GatewayAddress, testcase.assertion, testcase.minSuccRate, testcase.maxSuccRate)
			}
		})
	},
}
