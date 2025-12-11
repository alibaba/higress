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

	"github.com/alibaba/higress/v2/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/v2/test/e2e/conformance/utils/suite"
)

func init() {
	Register(WasmPluginsAIModelFilter)
}

var WasmPluginsAIModelFilter = suite.ConformanceTest{
	ShortName:   "WasmPluginsAIModelFilter",
	Description: "The Ingress in the higress-conformance-infra namespace test the ai-model-filter WASM plugin.",
	Manifests:   []string{"tests/go-wasm-ai-model-filter.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 1: Allowed model in request body",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "ai-api.com",
						Path:    "/v1/chat/completions",
						Method:  "POST",
						Body:    []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`),
						Headers: map[string]string{"Content-Type": "application/json"},
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
					TestCaseName:    "case 2: Allowed model with wildcard match",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "ai-api.com",
						Path:    "/v1/chat/completions",
						Method:  "POST",
						Body:    []byte(`{"model":"claude-3-sonnet","messages":[{"role":"user","content":"Hello"}]}`),
						Headers: map[string]string{"Content-Type": "application/json"},
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
					TestCaseName:    "case 3: Disallowed model",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "ai-api.com",
						Path:    "/v1/chat/completions",
						Method:  "POST",
						Body:    []byte(`{"model":"llama-3","messages":[{"role":"user","content":"Hello"}]}`),
						Headers: map[string]string{"Content-Type": "application/json"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:    "case 4: Model in URL path (Gemini API style)",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "ai-api.com",
						Path:    "/v1/models/gemini-pro:generateContent",
						Method:  "POST",
						Body:    []byte(`{"contents":[{"parts":[{"text":"Hello"}]}]}`),
						Headers: map[string]string{"Content-Type": "application/json"},
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
					TestCaseName:    "case 5: Disallowed model in URL path",
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:    "ai-api.com",
						Path:    "/v1/models/gemini-flash:generateContent",
						Method:  "POST",
						Body:    []byte(`{"contents":[{"parts":[{"text":"Hello"}]}]}`),
						Headers: map[string]string{"Content-Type": "application/json"},
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 403,
					},
				},
			},
		}
		t.Run("WasmPlugins ai-model-filter", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
