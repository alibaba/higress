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
	Register(WasmPluginsAiCache)
}

var WasmPluginsAiCache = suite.ConformanceTest{
	ShortName:   "WasmPluginAiCache",
	Description: "The Ingress in the higress-conformance-infra namespace test the ai-cache WASM plugin.",
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Manifests:   []string{"tests/go-wasm-ai-cache.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: openai",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "openai.ai.com",
						Path:             "/v1/chat/completions",
						Method:"POST",
						ContentType:      http.ContentTypeApplicationJson,
						Body: []byte(`{
							"model": "gpt-3",
                            "messages": [{"role":"user","content":"hi"}]}`),
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Host:        "api.openai.com",
							Path:        "/v1/chat/completions",
							Method:      "POST",
							ContentType: http.ContentTypeApplicationJson,
							Body: []byte(`{
								"model": "gpt-3",
                                "messages": [{"role":"user","content":"hi"}],
                                "max_tokens": 123,
								"temperature": 0.66}`),
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
					TestCaseName:    "case 2: qwen",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "qwen.ai.com",
						Path:             "/v1/chat/completions",
						Method:"POST",
						ContentType:      http.ContentTypeApplicationJson,
						Body: []byte(`{
							"model": "qwen-long",
							"input": {"messages": [{"role":"user","content":"hi"}]},
							"parameters": {"max_tokens": 321, "temperature": 0.7}}`),
					},
					ExpectedRequest: &http.ExpectedRequest{
						Request: http.Request{
							Host:        "dashscope.aliyuncs.com",
							Path:        "/api/v1/services/aigc/text-generation/generation",
							Method:      "POST",
							ContentType: http.ContentTypeApplicationJson,
							Body: []byte(`{
							"model": "qwen-long",
							"input": {"messages": [{"role":"user","content":"hi"}]},
							"parameters": {"max_tokens": 321, "temperature": 0.66}}`),
						},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 500,
					},
				},
			},
			
		}
		t.Run("WasmPlugins ai-cache", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
