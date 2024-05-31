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
	Register(ConfigmapHttps)
}

var ConfigmapHttps = suite.ConformanceTest{
	ShortName:   "ConfigmapHttps",
	Description: "The Ingress in the higress-conformance-infra namespace uses the configmap https.",
	Manifests:   []string{"tests/configmap-https.yaml"},
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {

		testCases := []struct {
			httpAssert http.Assertion
		}{
			{
				httpAssert: http.Assertion{
					Meta: http.AssertionMeta{
						TestCaseName:    "test configmap bar-com https",
						TargetBackend:   "infra-backend-v2",
						TargetNamespace: "higress-conformance-infra",
					},
					Request: http.AssertionRequest{
						ActualRequest: http.Request{
							Path: "/barhttps",
							Host: "bar.com",
							TLSConfig: &http.TLSConfig{
								SNI: "bar.com",
							},
						},
						ExpectedRequest: &http.ExpectedRequest{
							Request: http.Request{
								Path: "/barhttps",
								Host: "bar.com",
							},
						},
					},
					Response: http.AssertionResponse{
						ExpectedResponse: http.Response{
							StatusCode: 200,
						},
					},
				},
			},
			{
				httpAssert: http.Assertion{
					Meta: http.AssertionMeta{
						TestCaseName:    "test configmap a-foo-com https",
						TargetBackend:   "infra-backend-v2",
						TargetNamespace: "higress-conformance-infra",
					},
					Request: http.AssertionRequest{
						ActualRequest: http.Request{
							Path: "/afoohttps",
							Host: "a.foo.com",
							TLSConfig: &http.TLSConfig{
								SNI: "a.foo.com",
							},
						},
						ExpectedRequest: &http.ExpectedRequest{
							Request: http.Request{
								Path: "/afoohttps",
								Host: "a.foo.com",
							},
						},
					},
					Response: http.AssertionResponse{
						ExpectedResponse: http.Response{
							StatusCode: 200,
						},
					},
				},
			},
			{
				httpAssert: http.Assertion{
					Meta: http.AssertionMeta{
						TestCaseName:    "test configmap b-foo-com https",
						TargetBackend:   "infra-backend-v2",
						TargetNamespace: "higress-conformance-infra",
					},
					Request: http.AssertionRequest{
						ActualRequest: http.Request{
							Path: "/bfoohttps",
							Host: "b.foo.com",
							TLSConfig: &http.TLSConfig{
								SNI: "b.foo.com",
							},
						},
						ExpectedRequest: &http.ExpectedRequest{
							Request: http.Request{
								Path: "/bfoohttps",
								Host: "b.foo.com",
							},
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
		t.Run("Configmap Https", func(t *testing.T) {
			for _, testcase := range testCases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase.httpAssert)
			}
		})
	},
}
