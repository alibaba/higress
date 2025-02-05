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

// The llm-mock service response has a fixed id of `chatcmpl-llm-mock`.
// The created field is fixed to 10.
// The response content is echoed back as the request content.
// The usage field is fixed to `{"prompt_tokens":9,"completion_tokens":1,"total_tokens":10}` (specific values may vary based on the corresponding response fields).

func init() {
	Register(WasmPluginsAiProxy)
}

var WasmPluginsAiProxy = suite.ConformanceTest{
	ShortName:   "WasmPluginAiProxy",
	Description: "The Ingress in the higress-conformance-ai-backend namespace test the ai-proxy WASM plugin.",
	Features:    []suite.SupportedFeature{suite.WASMGoConformanceFeature},
	Manifests:   []string{"tests/go-wasm-ai-proxy.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "baidu case 1: non-streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "qianfan.baidubce.com",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":false}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"id":"chatcmpl-llm-mock","choices":[{"index":0,"message":{"role":"assistant","content":"你好，你是谁？"},"finish_reason":"stop"}],"created":10,"model":"ernie-3.5-8k","object":"chat.completion","usage":{"prompt_tokens":9,"completion_tokens":1,"total_tokens":10}}`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "baidu case 2: streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "qianfan.baidubce.com",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":true}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeTextEventStream,
						Body: []byte(`data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"你"}}],"created":10,"model":"ernie-3.5-8k","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"好"}}],"created":10,"model":"ernie-3.5-8k","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"，"}}],"created":10,"model":"ernie-3.5-8k","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"你"}}],"created":10,"model":"ernie-3.5-8k","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"是"}}],"created":10,"model":"ernie-3.5-8k","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"谁"}}],"created":10,"model":"ernie-3.5-8k","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"？"}}],"created":10,"model":"ernie-3.5-8k","object":"chat.completion.chunk","usage":{}}

data: [DONE]

`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "doubao case 1: non-streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "ark.cn-beijing.volces.com",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":false}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"id":"chatcmpl-llm-mock","choices":[{"index":0,"message":{"role":"assistant","content":"你好，你是谁？"},"finish_reason":"stop"}],"created":10,"model":"fake_doubao_endpoint","object":"chat.completion","usage":{"prompt_tokens":9,"completion_tokens":1,"total_tokens":10}}`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "doubao case 2: streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "ark.cn-beijing.volces.com",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":true}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeTextEventStream,
						Body: []byte(`data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"你"}}],"created":10,"model":"fake_doubao_endpoint","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"好"}}],"created":10,"model":"fake_doubao_endpoint","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"，"}}],"created":10,"model":"fake_doubao_endpoint","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"你"}}],"created":10,"model":"fake_doubao_endpoint","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"是"}}],"created":10,"model":"fake_doubao_endpoint","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"谁"}}],"created":10,"model":"fake_doubao_endpoint","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"？"}}],"created":10,"model":"fake_doubao_endpoint","object":"chat.completion.chunk","usage":{}}

data: [DONE]

`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "minimax case 1: proxy completion V2 API, non-streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "api.minimax.chat-v2-api",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":false}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"id":"chatcmpl-llm-mock","choices":[{"index":0,"message":{"role":"assistant","content":"你好，你是谁？"},"finish_reason":"stop"}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{"prompt_tokens":9,"completion_tokens":1,"total_tokens":10}}`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "minimax case 2: proxy completion V2 API, streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "api.minimax.chat-v2-api",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":true}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeTextEventStream,
						Body: []byte(`data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"你"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"好"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"，"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"你"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"是"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"谁"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"？"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion.chunk","usage":{}}

data: [DONE]

`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "minimax case 3: proxy completion Pro API, non-streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "api.minimax.chat-pro-api",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":false}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"id":"chatcmpl-llm-mock","choices":[{"index":0,"message":{"name":"MM智能助理","role":"assistant","content":"你好，你是谁？"},"finish_reason":"stop"}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{"prompt_tokens":9,"completion_tokens":1,"total_tokens":10}}`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "minimax case 4: proxy completion Pro API, streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "api.minimax.chat-pro-api",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":true}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeTextEventStream,
						Body: []byte(`data: {"choices":[{"index":0,"message":{"name":"MM智能助理","role":"assistant","content":"你"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{}}

data: {"choices":[{"index":0,"message":{"name":"MM智能助理","role":"assistant","content":"好"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{}}

data: {"choices":[{"index":0,"message":{"name":"MM智能助理","role":"assistant","content":"，"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{}}

data: {"choices":[{"index":0,"message":{"name":"MM智能助理","role":"assistant","content":"你"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{}}

data: {"choices":[{"index":0,"message":{"name":"MM智能助理","role":"assistant","content":"是"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{}}

data: {"choices":[{"index":0,"message":{"name":"MM智能助理","role":"assistant","content":"谁"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{}}

data: {"choices":[{"index":0,"message":{"name":"MM智能助理","role":"assistant","content":"？"}}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"message":{"name":"MM智能助理","role":"assistant","content":"你好，你是谁？"},"finish_reason":"stop"}],"created":10,"model":"abab6.5s-chat","object":"chat.completion","usage":{"prompt_tokens":9,"completion_tokens":1,"total_tokens":10}}

`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "qwen case 1: compatible mode, non-streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "dashscope.aliyuncs.com-compatible-mode",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":false}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"id":"chatcmpl-llm-mock","choices":[{"index":0,"message":{"role":"assistant","content":"你好，你是谁？"},"finish_reason":"stop"}],"created":10,"model":"qwen-turbo","object":"chat.completion","usage":{"prompt_tokens":9,"completion_tokens":1,"total_tokens":10}}`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "qwen case 2: compatible mode, streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "dashscope.aliyuncs.com-compatible-mode",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":true}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeTextEventStream,
						Body: []byte(`data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"你"}}],"created":10,"model":"qwen-turbo","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"好"}}],"created":10,"model":"qwen-turbo","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"，"}}],"created":10,"model":"qwen-turbo","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"你"}}],"created":10,"model":"qwen-turbo","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"是"}}],"created":10,"model":"qwen-turbo","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"谁"}}],"created":10,"model":"qwen-turbo","object":"chat.completion.chunk","usage":{}}

data: {"id":"chatcmpl-llm-mock","choices":[{"index":0,"delta":{"content":"？"}}],"created":10,"model":"qwen-turbo","object":"chat.completion.chunk","usage":{}}

data: [DONE]

`),
					},
				},
			},
			{
				Meta: http.AssertionMeta{
					TestCaseName:  "qwen case 3: non-streaming request",
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:        "dashscope.aliyuncs.com",
						Path:        "/v1/chat/completions",
						Method:      "POST",
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte(`{"model":"gpt-3","messages":[{"role":"user","content":"你好，你是谁？"}],"stream":false}`),
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						// Since the "created" field is generated by the ai-proxy plugin based on the current timestamp, it is ignored during comparison
						JsonBodyIgnoreFields: []string{"created"},
						Body:                 []byte(`{"id":"chatcmpl-llm-mock","choices":[{"index":0,"message":{"role":"assistant","content":"你好，你是谁？"},"finish_reason":"stop"}],"created":1738218357,"model":"qwen-turbo","object":"chat.completion","usage":{"prompt_tokens":9,"completion_tokens":1,"total_tokens":10}}`),
					},
				},
			},
		}
		t.Run("WasmPlugins ai-proxy", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}
