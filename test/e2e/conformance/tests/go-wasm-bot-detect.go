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
	Register(WasmPluginsBotDetect)
}

var WasmPluginsBotDetect = suite.ConformanceTest{
	ShortName:   "WasmPluginsBotDetect",
	Description: "The Ingress in the higress-conformance-infra namespace test the bot detect WASM plugin.",
	Manifests:   []string{"tests/go-wasm-bot-detect.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: Test Default Deny",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo.com",
						Path:    "/foo",
						Headers: map[string]string{"User-Agent": "BaiduMobaider/1.1.0"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 401,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 2: Test Default Allow",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo.com",
						Path:    "/foo",
						Headers: map[string]string{"User-Agent": "Mozilla/5.0"},
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
					TestCaseName:    "case 3: Test Rule Allow",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo.com",
						Path:    "/foo",
						Headers: map[string]string{"User-Agent": "Tailsweep"},
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
					TestCaseName:    "case 4: Test Rule Deny",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "foo.com",
						Path:    "/foo",
						Headers: map[string]string{"User-Agent": "Go-Client"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 401,
					},
				},
			},
		}
		t.Run("WasmPlugins bot detect", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
