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
	Register(HTTPRouteMatchPath)
}

var HTTPRouteMatchPath = suite.ConformanceTest{
	ShortName:   "HTTPRouteMatchPath",
	Description: "A Ingress in the higress-conformance-infra namespace that match different path.",
	Manifests:   []string{"tests/httproute-match-path.yaml"},
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case1: normal request",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo",
						Host: "foo.com",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			}, {
				Meta: http.AssertionMeta{
					TestCaseName:    "case2: path is '/bar' and match prefix path successfully",
					TargetBackend:   "infra-backend-v2",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/bar",
						Host: "foo.com",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			}, {
				Meta: http.AssertionMeta{
					TestCaseName: "case2: path is '/bar' and match prefix path failed",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/bard",
						Host: "foo.com",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 404,
					},
				},
			}, {
				Meta: http.AssertionMeta{
					TestCaseName:    "case3: path is '/cat/' and match prefix path successfully",
					TargetBackend:   "infra-backend-v3",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/cat",
						Host: "foo.com",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			}, {
				Meta: http.AssertionMeta{
					TestCaseName:    "case4: path is '/cat/' and match prefix path successfully",
					TargetBackend:   "infra-backend-v3",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/cat/ok",
						Host: "foo.com",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			}, {
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v2",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo/",
						Host: "foo.com",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}

		t.Run("HTTPRoute Match different path Cases", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})

	},
}
