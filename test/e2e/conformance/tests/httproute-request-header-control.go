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
	Register(HTTPRouteRequestHeaderControl)
}

var HTTPRouteRequestHeaderControl = suite.ConformanceTest{
	ShortName:   "HTTPRouteRequestHeaderControl",
	Description: "A single Ingress in the higress-conformance-infra namespace controls the request header.",
	Manifests:   []string{"tests/httproute-request-header-control.yaml"},
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: add one",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},

				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo1",
						Host: "foo.com",
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/foo1",
							Host: "foo.com",
							Headers: map[string]string{
								"stage": "test",
							},
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
					TestCaseName:    "case 2: add more",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},

				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo2",
						Host: "foo.com",
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/foo2",
							Host: "foo.com",
							Headers: map[string]string{
								"stage":   "test",
								"canary":  "true",
								"x-test":  "higress; test=true",
								"x-test2": "higress; test=false",
							},
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
					TestCaseName:    "case 3: update one",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},

				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo3",
						Host: "foo.com",
						Headers: map[string]string{
							"stage": "test",
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/foo3",
							Host: "foo.com",
							Headers: map[string]string{
								"stage": "pro",
							},
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
					TestCaseName:    "case 4: update more",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},

				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo4",
						Host: "foo.com",
						Headers: map[string]string{
							"stage":  "test",
							"canary": "true",
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/foo4",
							Host: "foo.com",
							Headers: map[string]string{
								"stage":  "pro",
								"canary": "false",
							},
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
					TestCaseName:    "case 5: remove one",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},

				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo5",
						Host: "foo.com",
						Headers: map[string]string{
							"stage": "test",
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/foo5",
							Host: "foo.com",
						},
						AbsentHeaders: []string{"stage"},
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
					TestCaseName:    "case 6: remove more",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},

				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/foo6",
						Host: "foo.com",
						Headers: map[string]string{
							"stage":  "test",
							"canary": "true",
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/foo6",
							Host: "foo.com",
						},
						AbsentHeaders: []string{"stage", "canary"},
					},
				},

				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}

		t.Run("Request header control", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
