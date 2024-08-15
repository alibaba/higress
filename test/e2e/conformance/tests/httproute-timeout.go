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
	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
	"testing"
)

func init() {
	Register(HttpTimeout)
}

var HttpTimeout = suite.ConformanceTest{
	ShortName:   "HttpTimeout",
	Description: "The Ingress in the higress-conformance-infra namespace uses timeout annotation",
	Manifests:   []string{"tests/httproute-timeout.yaml"},
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "case 1: backend response is delayed for 5s",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "timeout.higress.io",
						Path:    "/timeout",
						Headers: map[string]string{"X-Delay": "6000", "content-type": "application/json"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 504,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "case 2: backend response is delayed for 1s",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "timeout.higress.io",
						Path:    "/timeout",
						Headers: map[string]string{"X-Delay": "1000", "content-type": "application/json"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}

		t.Run("HttpRedirectAsHttps", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})

	},
}
