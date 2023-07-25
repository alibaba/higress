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
	HigressConformanceTests = append(HigressConformanceTests, HTTPRouteRewritePath)
}

var HTTPRouteRewritePath = suite.ConformanceTest{
	ShortName:   "HTTPRouteRewritePath",
	Description: "A single Ingress in the higress-conformance-infra namespace uses the rewrite path.",
	Manifests:   []string{"tests/httproute-rewrite-path.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testCases := []http.Assertion{
			{
				Request: http.AssertionRequest{
					ActualRequest: http.Request{Path: "/svc/foo"},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{Path: "/foo"},
					},
				},

				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},

				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
			},
		}

		t.Run("Rewrite HTTPRoute Path", func(t *testing.T) {
			for _, testCase := range testCases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testCase)
			}

		})
	},
}
