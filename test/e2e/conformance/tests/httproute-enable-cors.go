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
	HigressConformanceTests = append(HigressConformanceTests, HTTPRouteEnableCors)
}

var HTTPRouteEnableCors = suite.ConformanceTest{
	ShortName:   "HTTPRouteEnableCors",
	Description: "A single Ingress in the higress-conformance-infra namespace demonstrates enable cors ability.",
	Manifests:   []string{"tests/httproute-enable-cors.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case1: unable cors",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path:    "/foo",
						Host:    "foo1.com",
						Method:  "OPTIONS",
						Headers: map[string]string{"Origin": "http://bar.com"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:    200,
						AbsentHeaders: []string{"Access-Control-Allow-Credentials", "Access-Control-Allow-Origin"},
					},
				},
			}, {
				Meta: http.AssertionMeta{
					TestCaseName:    "case2: enable cors",
					TargetBackend:   "infra-backend-v2",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path:    "/foo",
						Host:    "foo2.com",
						Method:  "OPTIONS",
						Headers: map[string]string{"Origin": "http://bar.com"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers:    map[string]string{"Access-Control-Allow-Credentials": "true", "Access-Control-Allow-Origin": "http://bar.com"},
					},
				},
			}, {
				Meta: http.AssertionMeta{
					TestCaseName:    "case3: enable cors and allow origin headers",
					TargetBackend:   "infra-backend-v3",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path:    "/foo",
						Host:    "foo3.com",
						Method:  "OPTIONS",
						Headers: map[string]string{"Origin": "http://bar.com"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers:    map[string]string{"Access-Control-Allow-Credentials": "true", "Access-Control-Allow-Origin": "http://bar.com", "Access-Control-Expose-Headers": "*"},
					},
				},
			}, {
				Meta: http.AssertionMeta{
					TestCaseName:    "case4: enable cors and use forbidden Origin",
					TargetBackend:   "infra-backend-v3",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path:    "/foo",
						Host:    "foo3.com",
						Method:  "OPTIONS",
						Headers: map[string]string{"Origin": "http://foo.com"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:    200,
						AbsentHeaders: []string{"Access-Control-Allow-Credentials", "Access-Control-Allow-Origin", "Access-Control-Expose-Headers"},
					},
				},
			},
		}

		t.Run("Enable Cors Cases Split", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})

	},
}
