// Copyright (c) 2025 Alibaba Group Holding Ltd.
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
	Register(WasmPluginsAiStatistics)
}

var WasmPluginsAiStatistics = suite.ConformanceTest{
	ShortName:   "WasmPluginAiStatistics",
	Description: "The Ingress in the higress-conformance-ai-backend namespace test the ai-statistics WASM plugin with OpenAI responses format support.",
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Manifests:   []string{"tests/go-wasm-ai-statistics.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "ai-statistics case 1: non-streaming responses format",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "api.openai.com",
						Path:        "/v1/responses",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}],"stream":false}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"id":"resp_test","object":"response","model":"gpt-4o-2024-08-06","usage":{"input_tokens":328,"output_tokens":52,"total_tokens":380}}`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "ai-statistics case 2: streaming responses format",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "api.openai.com",
						Path:        "/v1/responses",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}],"stream":true}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeTextEventStream,
						Body: []byte(`data: {"type":"response.created","response":{"id":"resp_test","object":"response","model":"gpt-4o-2024-08-06","usage":null}}  
  
data: {"type":"response.completed","response":{"id":"resp_test","object":"response","model":"gpt-4o-2024-08-06","usage":{"input_tokens":328,"output_tokens":52,"total_tokens":380}}}  
  
data: [DONE]  
  
`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "ai-statistics case 3: chat completions format (backward compatibility)",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "api.openai.com",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}],"stream":false}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"id":"chatcmpl-mock","choices":[{"index":0,"message":{"role":"assistant","content":"Hello! How can I help you?"},"finish_reason":"stop"}],"created":10,"model":"gpt-3.5-turbo","object":"chat.completion","usage":{"prompt_tokens":9,"completion_tokens":8,"total_tokens":17}}`),
					},
				},
			},
		}

		t.Run("WasmPlugins ai-statistics", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
