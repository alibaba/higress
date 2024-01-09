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
	Register(WasmPluginsTransformer)
}

// TODO(WeixinX): Request and response body conformance check is not supported now
var WasmPluginsTransformer = suite.ConformanceTest{
	ShortName:   "WasmPluginTransformer",
	Description: "The Ingress in the higress-conformance-infra namespace test the transformer WASM plugin.",
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Manifests:   []string{"tests/go-wasm-transformer.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: request header&query transformer",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo1.com",
						Path: "/get?k1=v11&k1=v12&k2=v2",
						Headers: map[string]string{
							"X-remove":        "exist",
							"X-not-renamed":   "test",
							"X-replace":       "not-replaced",
							"X-dedupe-first":  "1,2,3",
							"X-dedupe-last":   "a,b,c",
							"X-dedupe-unique": "1,2,3,3,2,1",
						},
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Host: "foo1.com",
							Path: "/get?k2-new=v2-new&k3=v31&k3=v32&k4=v31", // url.Value.Encode() is ordered by key
							Headers: map[string]string{
								"X-renamed":       "test",
								"X-replace":       "replaced",
								"X-add-append":    "add,append", // header with same name
								"X-map":           "add,append",
								"X-dedupe-first":  "1",
								"X-dedupe-last":   "c",
								"X-dedupe-unique": "1,2,3",
							},
						},
						AbsentHeaders: []string{"X-remove"},
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
					TestCaseName:    "case 2: response header&query transformer",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo2.com",
						Path: "/get/index.html",
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Host: "foo2.com",
							Path: "/get/index.html",
						},
					},
				},
				Response: http.AssertionResponse{
					AdditionalResponseHeaders: map[string]string{
						"X-remove":      "exist",
						"X-not-renamed": "test",
						"X-replace":     "not-replaced",
					},
					ExpectedResponse: http.Response{
						StatusCode: 200,
						Headers: map[string]string{
							"X-renamed":    "test",
							"X-replace":    "replace-get",           // regexp matches path and replace "replace-$1"
							"X-add-append": "add-foo2,append-index", // regexp matches host and replace "add-$1"
							"X-map":        "add-foo2,append-index",
						},
						AbsentHeaders: []string{"X-remove"},
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 4: request body transformer",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo4.com",
						Path: "/post",
						// TODO(Uncle-Justice) dedupe, replace的body插件逻辑有问题，暂跳过测试
						Method: "POST",
						Body: []byte(`
						{
							"X-removed":["v1", "v2"],
							"X-not-renamed":["v1"]
						}
						`),
						ContentType: http.ContentTypeApplicationJson,
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Host: "foo4.com",
							Path: "/post",
							// TODO(Uncle-Justice) dedupe, replace的body插件逻辑有问题，暂跳过测试
							Method:      "POST",
							ContentType: http.ContentTypeApplicationJson,
							Body: []byte(`
							{
								"X-renamed":["v1"],
								"X-add-append":["add","append"],
								"X-map":["add","append"]
							}
						`),
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
					TestCaseName:    "case 5: response json body transformer",
					TargetBackend:   "infra-backend-echo-body-v1",
					TargetNamespace: "higress-conformance-infra",
					CompareTarget:   http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host: "foo5.com",
						Path: "/post",
						// TODO(Uncle-Justice) dedupe, replace的body插件逻辑有问题，暂跳过测试
						Method: "POST",
						Body: []byte(`
						{
							"X-removed":["v1", "v2"],
							"X-not-renamed":["v1"]
						}
						`),
						ContentType: http.ContentTypeApplicationJson,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						Body: []byte(`
						{
							"X-renamed":["v1"],
							"X-add-append":["add","append"],
							"X-map":["add","append"]
						}
						`),
					},
				},
			},
		}
		t.Run("WasmPlugin transformer", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
