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
	Register(WasmPluginsGeoIPPlugin)
}

var WasmPluginsGeoIPPlugin = suite.ConformanceTest{
	ShortName:   "WasmPluginsGeoIPPlugin",
	Description: "The geo-ip wasm pluin finds the client's geographic information according to the client's ip address.",
	Manifests:   []string{"tests/go-wasm-geo-ip.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/info",
						UnfollowRedirect: true,
						Headers: map[string]string{
							"X-Forwarded-For": "70.155.208.224,10.1.1.1",
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/info",
							Host: "foo.com",
							Headers: map[string]string{
								"X-Higress-Geo-Isp":      "美国电话电报",
								"X-Higress-Geo-City":     "0",
								"X-Higress-Geo-Province": "密西西比",
								"X-Higress-Geo-Country":  "美国",
							},
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
					ExpectedResponseNoRequest: true,
				},
			},
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "foo.com",
						Path:             "/info",
						UnfollowRedirect: true,
						Headers: map[string]string{
							"X-Forwarded-For": "2.2.128.100,10.1.1.2",
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Path: "/info",
							Host: "foo.com",
							Headers: map[string]string{
								"X-Higress-Geo-Isp":      "橘子电信",
								"X-Higress-Geo-City":     "0",
								"X-Higress-Geo-Province": "Var",
								"X-Higress-Geo-Country":  "法国",
							},
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
					ExpectedResponseNoRequest: true,
				},
			},
		}
		t.Run("WasmPlugins geo-ip", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}