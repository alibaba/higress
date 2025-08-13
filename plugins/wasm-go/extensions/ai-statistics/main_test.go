// Copyright (c) 2024 Alibaba Group Holding Ltd.
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

package main

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基本统计配置
var basicConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":                   "request_id",
				"value_source":          "request_header",
				"value":                 "x-request-id",
				"apply_to_log":          true,
				"apply_to_span":         false,
				"as_separate_log_field": false,
			},
			{
				"key":                   "api_version",
				"value_source":          "fixed_value",
				"value":                 "v1",
				"apply_to_log":          true,
				"apply_to_span":         true,
				"as_separate_log_field": false,
			},
		},
		"disable_openai_usage": false,
	})
	return data
}()

// 测试配置：流式响应体属性配置
var streamingBodyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":                   "response_content",
				"value_source":          "response_streaming_body",
				"value":                 "choices.0.message.content",
				"rule":                  "first",
				"apply_to_log":          true,
				"apply_to_span":         false,
				"as_separate_log_field": false,
			},
			{
				"key":                   "model_name",
				"value_source":          "response_streaming_body",
				"value":                 "model",
				"rule":                  "replace",
				"apply_to_log":          true,
				"apply_to_span":         true,
				"as_separate_log_field": false,
			},
		},
		"disable_openai_usage": false,
	})
	return data
}()

// 测试配置：请求体属性配置
var requestBodyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":                   "user_message_count",
				"value_source":          "request_body",
				"value":                 "messages.#(role==\"user\")",
				"apply_to_log":          true,
				"apply_to_span":         false,
				"as_separate_log_field": false,
			},
			{
				"key":                   "request_model",
				"value_source":          "request_body",
				"value":                 "model",
				"apply_to_log":          true,
				"apply_to_span":         true,
				"as_separate_log_field": false,
			},
		},
		"disable_openai_usage": false,
	})
	return data
}()

// 测试配置：响应体属性配置
var responseBodyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":                   "response_status",
				"value_source":          "response_body",
				"value":                 "status",
				"apply_to_log":          true,
				"apply_to_span":         false,
				"as_separate_log_field": false,
			},
			{
				"key":                   "response_message",
				"value_source":          "response_body",
				"value":                 "message",
				"apply_to_log":          true,
				"apply_to_span":         true,
				"as_separate_log_field": false,
			},
		},
		"disable_openai_usage": false,
	})
	return data
}()

// 测试配置：禁用 OpenAI 使用统计
var disableOpenaiUsageConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":                   "custom_attribute",
				"value_source":          "fixed_value",
				"value":                 "custom_value",
				"apply_to_log":          true,
				"apply_to_span":         false,
				"as_separate_log_field": false,
			},
		},
		"disable_openai_usage": true,
	})
	return data
}()

// 测试配置：空属性配置
var emptyAttributesConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"attributes":           []map[string]interface{}{},
		"disable_openai_usage": false,
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本统计配置解析
		t.Run("basic config", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试流式响应体属性配置解析
		t.Run("streaming body config", func(t *testing.T) {
			host, status := test.NewTestHost(streamingBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试请求体属性配置解析
		t.Run("request body config", func(t *testing.T) {
			host, status := test.NewTestHost(requestBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试响应体属性配置解析
		t.Run("response body config", func(t *testing.T) {
			host, status := test.NewTestHost(responseBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试禁用 OpenAI 使用统计配置解析
		t.Run("disable openai usage config", func(t *testing.T) {
			host, status := test.NewTestHost(disableOpenaiUsageConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试空属性配置解析
		t.Run("empty attributes config", func(t *testing.T) {
			host, status := test.NewTestHost(emptyAttributesConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试基本请求头处理
		t.Run("basic request headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"x-request-id", "req-123"},
				{"x-mse-consumer", "consumer1"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试包含 consumer 的请求头处理
		t.Run("request headers with consumer", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"x-request-id", "req-456"},
				{"x-mse-consumer", "consumer2"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试不包含 consumer 的请求头处理
		t.Run("request headers without consumer", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"x-request-id", "req-789"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})
	})
}

func TestOnHttpRequestBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试基本请求体处理
		t.Run("basic request body", func(t *testing.T) {
			host, status := test.NewTestHost(requestBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
			})

			// 设置请求体
			requestBody := []byte(`{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "user", "content": "Hello"},
					{"role": "assistant", "content": "Hi there"},
					{"role": "user", "content": "How are you?"}
				]
			}`)
			action := host.CallOnHttpRequestBody(requestBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试 Google Gemini 格式的请求体处理
		t.Run("gemini request body", func(t *testing.T) {
			host, status := test.NewTestHost(requestBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/models/gemini-pro:generateContent"},
				{":method", "POST"},
			})

			// 设置请求体
			requestBody := []byte(`{
				"contents": [
					{"role": "user", "parts": [{"text": "Hello"}]},
					{"parts": [{"text": "Hi there"}]}
				]
			}`)
			action := host.CallOnHttpRequestBody(requestBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试不包含消息的请求体处理
		t.Run("request body without messages", func(t *testing.T) {
			host, status := test.NewTestHost(requestBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
			})

			// 设置请求体
			requestBody := []byte(`{
				"model": "gpt-3.5-turbo",
				"temperature": 0.7
			}`)
			action := host.CallOnHttpRequestBody(requestBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})
	})
}

func TestOnHttpResponseHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试基本响应头处理
		t.Run("basic response headers", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
			})

			// 设置响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试流式响应头处理
		t.Run("streaming response headers", func(t *testing.T) {
			host, status := test.NewTestHost(streamingBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
			})

			// 设置流式响应头
			action := host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/event-stream"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})
	})
}

func TestOnHttpStreamingBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试流式响应体处理
		t.Run("streaming response body", func(t *testing.T) {
			host, status := test.NewTestHost(streamingBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
			})

			// 设置流式响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/event-stream"},
			})

			// 处理第一个流式块
			firstChunk := []byte(`data: {"choices":[{"message":{"content":"Hello"}}],"model":"gpt-3.5-turbo"}`)
			result := host.CallOnHttpStreamingRequestBody(firstChunk, false)

			// 应该返回原始数据
			require.Equal(t, firstChunk, result)

			// 处理最后一个流式块
			lastChunk := []byte(`data: {"choices":[{"message":{"content":"How can I help you?"}}],"model":"gpt-3.5-turbo"}`)
			result = host.CallOnHttpStreamingRequestBody(lastChunk, true)

			// 应该返回原始数据
			require.Equal(t, lastChunk, result)

			host.CompleteHttp()
		})

		// 测试不包含 token 统计的流式响应体处理
		t.Run("streaming body without token usage", func(t *testing.T) {
			host, status := test.NewTestHost(streamingBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
			})

			// 设置流式响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/event-stream"},
			})

			// 处理流式响应体
			chunk := []byte(`data: {"message": "Hello world"}`)
			result := host.CallOnHttpStreamingRequestBody(chunk, true)

			// 应该返回原始数据
			require.Equal(t, chunk, result)

			host.CompleteHttp()
		})
	})
}

func TestOnHttpResponseBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试基本响应体处理
		t.Run("basic response body", func(t *testing.T) {
			host, status := test.NewTestHost(responseBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
			})

			// 设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 设置响应体
			responseBody := []byte(`{
				"status": "success",
				"message": "Hello, how can I help you?",
				"choices": [{"message": {"content": "Hello"}}],
				"usage": {"prompt_tokens": 10, "completion_tokens": 15, "total_tokens": 25}
			}`)
			action := host.CallOnHttpResponseBody(responseBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试不包含 token 统计的响应体处理
		t.Run("response body without token usage", func(t *testing.T) {
			host, status := test.NewTestHost(responseBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
			})

			// 设置响应头
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 设置响应体
			responseBody := []byte(`{
				"status": "success",
				"message": "Hello world"
			}`)
			action := host.CallOnHttpResponseBody(responseBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})
	})
}

func TestCompleteFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试完整的统计流程
		t.Run("complete statistics flow", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 1. 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"x-request-id", "req-123"},
				{"x-mse-consumer", "consumer1"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 2. 处理请求体
			requestBody := []byte(`{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "user", "content": "Hello"}
				]
			}`)
			action = host.CallOnHttpRequestBody(requestBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 3. 处理响应头
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 4. 处理响应体
			responseBody := []byte(`{
				"choices": [{"message": {"content": "Hello, how can I help you?"}}],
				"usage": {"prompt_tokens": 5, "completion_tokens": 8, "total_tokens": 13}
			}`)
			action = host.CallOnHttpResponseBody(responseBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 5. 完成请求
			host.CompleteHttp()
		})

		// 测试流式响应的完整流程
		t.Run("complete streaming flow", func(t *testing.T) {
			host, status := test.NewTestHost(streamingBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 1. 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"x-mse-consumer", "consumer2"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 2. 处理请求体
			requestBody := []byte(`{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "Hello"}
				]
			}`)
			action = host.CallOnHttpRequestBody(requestBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 3. 处理流式响应头
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/event-stream"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 4. 处理流式响应体
			firstChunk := []byte(`data: {"choices":[{"message":{"content":"Hello"}}],"model":"gpt-4"}`)
			result := host.CallOnHttpStreamingRequestBody(firstChunk, false)

			// 应该返回原始数据
			require.Equal(t, firstChunk, result)

			// 5. 处理最后一个流式块
			lastChunk := []byte(`data: {"choices":[{"message":{"content":"How can I help you?"}}],"model":"gpt-4"}`)
			result = host.CallOnHttpStreamingRequestBody(lastChunk, true)

			// 应该返回原始数据
			require.Equal(t, lastChunk, result)

			// 6. 完成请求
			host.CompleteHttp()
		})
	})
}
