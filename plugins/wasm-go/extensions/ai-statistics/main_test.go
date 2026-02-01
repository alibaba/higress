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
	"time"

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
			{
				"key":                   "model",
				"value_source":          "request_body",
				"value":                 "model",
				"apply_to_log":          true,
				"apply_to_span":         true,
				"as_separate_log_field": false,
			},
			{
				"key":                   "input_token",
				"value_source":          "response_body",
				"value":                 "usage.prompt_tokens",
				"apply_to_log":          true,
				"apply_to_span":         true,
				"as_separate_log_field": false,
			},
			{
				"key":                   "output_token",
				"value_source":          "response_body",
				"value":                 "usage.completion_tokens",
				"apply_to_log":          true,
				"apply_to_span":         true,
				"as_separate_log_field": false,
			},
			{
				"key":                   "total_token",
				"value_source":          "response_body",
				"value":                 "usage.total_tokens",
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
			action := host.CallOnHttpStreamingResponseBody(firstChunk, false)

			result := host.GetResponseBody()
			require.Equal(t, firstChunk, result)

			// 应该返回原始数据
			require.Equal(t, types.ActionContinue, action)

			// 处理最后一个流式块
			lastChunk := []byte(`data: {"choices":[{"message":{"content":"How can I help you?"}}],"model":"gpt-3.5-turbo"}`)
			action = host.CallOnHttpStreamingResponseBody(lastChunk, true)

			// 应该返回原始数据
			require.Equal(t, types.ActionContinue, action)

			result = host.GetResponseBody()
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
			action := host.CallOnHttpStreamingResponseBody(chunk, true)

			// 应该返回原始数据
			require.Equal(t, types.ActionContinue, action)

			result := host.GetResponseBody()
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
				"usage": {"prompt_tokens": 10, "completion_tokens": 15, "total_tokens": 25},
				"model": "gpt-3.5-turbo"
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

func TestMetrics(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试指标收集
		t.Run("test token usage metrics", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置路由和集群名称
			host.SetRouteName("api-v1")
			host.SetClusterName("cluster-1")

			// 1. 处理请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"x-mse-consumer", "user1"},
			})

			// 2. 处理请求体
			requestBody := []byte(`{
				"model": "gpt-3.5-turbo",
				"messages": [{"role": "user", "content": "Hello"}]
			}`)
			host.CallOnHttpRequestBody(requestBody)

			// 添加延迟，确保有足够的时间间隔来计算 llm_service_duration
			time.Sleep(10 * time.Millisecond)

			// 3. 处理响应体
			responseBody := []byte(`{
				"choices": [{"message": {"content": "Hello, how can I help you?"}}],
				"usage": {"prompt_tokens": 5, "completion_tokens": 8, "total_tokens": 13},
				"model": "gpt-3.5-turbo"
			}`)
			host.CallOnHttpResponseBody(responseBody)

			// 4. 完成请求
			host.CompleteHttp()

			// 5. 验证指标值
			// 检查输入 token 指标
			inputTokenMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.user1.metric.input_token"
			inputTokenValue, err := host.GetCounterMetric(inputTokenMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(5), inputTokenValue)

			// 检查输出 token 指标
			outputTokenMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.user1.metric.output_token"
			outputTokenValue, err := host.GetCounterMetric(outputTokenMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(8), outputTokenValue)

			// 检查总 token 指标
			totalTokenMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.user1.metric.total_token"
			totalTokenValue, err := host.GetCounterMetric(totalTokenMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(13), totalTokenValue)

			// 检查服务时长指标
			serviceDurationMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.user1.metric.llm_service_duration"
			serviceDurationValue, err := host.GetCounterMetric(serviceDurationMetric)
			require.NoError(t, err)
			require.Greater(t, serviceDurationValue, uint64(0))

			// 检查请求计数指标
			durationCountMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.user1.metric.llm_duration_count"
			durationCountValue, err := host.GetCounterMetric(durationCountMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(1), durationCountValue)
		})

		// 测试流式响应指标
		t.Run("test streaming metrics", func(t *testing.T) {
			host, status := test.NewTestHost(streamingBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置路由和集群名称
			host.SetRouteName("api-v1")
			host.SetClusterName("cluster-1")

			// 1. 处理请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"x-mse-consumer", "user2"},
			})

			// 2. 处理请求体
			requestBody := []byte(`{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "Hello"}
				]
			}`)
			action := host.CallOnHttpRequestBody(requestBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 添加延迟，确保有足够的时间间隔来计算 llm_service_duration
			time.Sleep(10 * time.Millisecond)

			// 3. 处理流式响应头
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/event-stream"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 4. 处理流式响应体 - 添加 usage 信息
			firstChunk := []byte(`data: {"choices":[{"message":{"content":"Hello"}}],"model":"gpt-4","usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`)
			action = host.CallOnHttpStreamingResponseBody(firstChunk, false)

			// 应该返回原始数据
			require.Equal(t, types.ActionContinue, action)

			result := host.GetResponseBody()
			require.Equal(t, firstChunk, result)

			// 5. 处理最后一个流式块 - 添加 usage 信息
			lastChunk := []byte(`data: {"choices":[{"message":{"content":"How can I help you?"}}],"model":"gpt-4","usage":{"prompt_tokens":5,"completion_tokens":8,"total_tokens":13}}`)
			action = host.CallOnHttpStreamingResponseBody(lastChunk, true)

			// 应该返回原始数据
			require.Equal(t, types.ActionContinue, action)

			result = host.GetResponseBody()
			require.Equal(t, lastChunk, result)

			// 添加延迟，确保有足够的时间间隔来计算 llm_service_duration
			time.Sleep(10 * time.Millisecond)

			// 6. 完成请求
			host.CompleteHttp()

			// 7. 验证流式响应指标
			// 检查首 token 延迟指标
			firstTokenDurationMetric := "route.api-v1.upstream.cluster-1.model.gpt-4.consumer.user2.metric.llm_first_token_duration"
			firstTokenDurationValue, err := host.GetCounterMetric(firstTokenDurationMetric)
			require.NoError(t, err)
			require.Greater(t, firstTokenDurationValue, uint64(0))

			// 检查流式请求计数指标
			streamDurationCountMetric := "route.api-v1.upstream.cluster-1.model.gpt-4.consumer.user2.metric.llm_stream_duration_count"
			streamDurationCountValue, err := host.GetCounterMetric(streamDurationCountMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(1), streamDurationCountValue)

			// 检查服务时长指标
			serviceDurationMetric := "route.api-v1.upstream.cluster-1.model.gpt-4.consumer.user2.metric.llm_service_duration"
			serviceDurationValue, err := host.GetCounterMetric(serviceDurationMetric)
			require.NoError(t, err)
			require.Greater(t, serviceDurationValue, uint64(0))

			// 检查 token 指标
			inputTokenMetric := "route.api-v1.upstream.cluster-1.model.gpt-4.consumer.user2.metric.input_token"
			inputTokenValue, err := host.GetCounterMetric(inputTokenMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(5), inputTokenValue)

			outputTokenMetric := "route.api-v1.upstream.cluster-1.model.gpt-4.consumer.user2.metric.output_token"
			outputTokenValue, err := host.GetCounterMetric(outputTokenMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(8), outputTokenValue)

			totalTokenMetric := "route.api-v1.upstream.cluster-1.model.gpt-4.consumer.user2.metric.total_token"
			totalTokenValue, err := host.GetCounterMetric(totalTokenMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(13), totalTokenValue)
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

			// 设置路由和集群名称
			host.SetRouteName("api-v1")
			host.SetClusterName("cluster-1")

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

			// 添加延迟，确保有足够的时间间隔来计算 llm_service_duration
			time.Sleep(10 * time.Millisecond)

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
				"usage": {"prompt_tokens": 5, "completion_tokens": 8, "total_tokens": 13},
				"model": "gpt-3.5-turbo"
			}`)
			action = host.CallOnHttpResponseBody(responseBody)

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 5. 完成请求
			host.CompleteHttp()

			// 6. 验证指标值
			// 检查输入 token 指标
			inputTokenMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.consumer1.metric.input_token"
			inputTokenValue, err := host.GetCounterMetric(inputTokenMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(5), inputTokenValue)

			// 检查输出 token 指标
			outputTokenMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.consumer1.metric.output_token"
			outputTokenValue, err := host.GetCounterMetric(outputTokenMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(8), outputTokenValue)

			// 检查总 token 指标
			totalTokenMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.consumer1.metric.total_token"
			totalTokenValue, err := host.GetCounterMetric(totalTokenMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(13), totalTokenValue)

			// 检查服务时长指标
			serviceDurationMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.consumer1.metric.llm_service_duration"
			serviceDurationValue, err := host.GetCounterMetric(serviceDurationMetric)
			require.NoError(t, err)
			require.Greater(t, serviceDurationValue, uint64(0))

			// 检查请求计数指标
			durationCountMetric := "route.api-v1.upstream.cluster-1.model.gpt-3.5-turbo.consumer.consumer1.metric.llm_duration_count"
			durationCountValue, err := host.GetCounterMetric(durationCountMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(1), durationCountValue)
		})

		// 测试流式响应的完整流程
		t.Run("complete streaming flow", func(t *testing.T) {
			host, status := test.NewTestHost(streamingBodyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置路由和集群名称
			host.SetRouteName("api-v1")
			host.SetClusterName("cluster-1")

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

			// 添加延迟，确保有足够的时间间隔来计算 llm_service_duration
			time.Sleep(10 * time.Millisecond)

			// 3. 处理流式响应头
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "text/event-stream"},
			})

			// 应该返回 ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 4. 处理流式响应体 - 添加 usage 信息
			firstChunk := []byte(`data: {"choices":[{"message":{"content":"Hello"}}],"model":"gpt-4","usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`)
			action = host.CallOnHttpStreamingResponseBody(firstChunk, false)

			// 应该返回原始数据
			require.Equal(t, types.ActionContinue, action)

			result := host.GetResponseBody()
			require.Equal(t, firstChunk, result)

			// 5. 处理最后一个流式块 - 添加 usage 信息
			lastChunk := []byte(`data: {"choices":[{"message":{"content":"How can I help you?"}}],"model":"gpt-4","usage":{"prompt_tokens":5,"completion_tokens":8,"total_tokens":13}}`)
			action = host.CallOnHttpStreamingResponseBody(lastChunk, true)

			// 应该返回原始数据
			require.Equal(t, types.ActionContinue, action)

			result = host.GetResponseBody()
			require.Equal(t, lastChunk, result)

			// 添加延迟，确保有足够的时间间隔来计算 llm_service_duration
			time.Sleep(10 * time.Millisecond)

			// 6. 完成请求
			host.CompleteHttp()

			// 7. 验证流式响应指标
			// 检查首 token 延迟指标
			firstTokenDurationMetric := "route.api-v1.upstream.cluster-1.model.gpt-4.consumer.consumer2.metric.llm_first_token_duration"
			firstTokenDurationValue, err := host.GetCounterMetric(firstTokenDurationMetric)
			require.NoError(t, err)
			require.Greater(t, firstTokenDurationValue, uint64(0))

			// 检查流式请求计数指标
			streamDurationCountMetric := "route.api-v1.upstream.cluster-1.model.gpt-4.consumer.consumer2.metric.llm_stream_duration_count"
			streamDurationCountValue, err := host.GetCounterMetric(streamDurationCountMetric)
			require.NoError(t, err)
			require.Equal(t, uint64(1), streamDurationCountValue)

			// 检查服务时长指标
			serviceDurationMetric := "route.api-v1.upstream.cluster-1.model.gpt-4.consumer.consumer2.metric.llm_service_duration"
			serviceDurationValue, err := host.GetCounterMetric(serviceDurationMetric)
			require.NoError(t, err)
			require.Greater(t, serviceDurationValue, uint64(0))
		})
	})
}

// 测试配置：Token Details 配置
var tokenDetailsConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":          "reasoning_tokens",
				"apply_to_log": true,
			},
			{
				"key":          "cached_tokens",
				"apply_to_log": true,
			},
			{
				"key":          "input_token_details",
				"apply_to_log": true,
			},
			{
				"key":          "output_token_details",
				"apply_to_log": true,
			},
		},
		"disable_openai_usage": false,
	})
	return data
}()

// TestTokenDetails 测试 token details 功能
func TestTokenDetails(t *testing.T) {
	t.Run("test builtin token details attributes", func(t *testing.T) {
		host, status := test.NewTestHost(tokenDetailsConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		// 设置路由和集群名称
		host.SetRouteName("api-v1")
		host.SetClusterName("cluster-1")

		// 1. 处理请求头
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"},
			{":path", "/v1/chat/completions"},
			{":method", "POST"},
		})
		require.Equal(t, types.ActionContinue, action)

		// 2. 处理请求体
		requestBody := []byte(`{
			"model": "gpt-4o",
			"messages": [
				{"role": "user", "content": "Test question"}
			]
		}`)
		action = host.CallOnHttpRequestBody(requestBody)
		require.Equal(t, types.ActionContinue, action)

		// 3. 处理响应头
		action = host.CallOnHttpResponseHeaders([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
		})
		require.Equal(t, types.ActionContinue, action)

		// 4. 处理响应体（包含 token details）
		responseBody := []byte(`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-4o",
			"usage": {
				"prompt_tokens": 100,
				"completion_tokens": 50,
				"total_tokens": 150,
				"completion_tokens_details": {
					"reasoning_tokens": 25
				},
				"prompt_tokens_details": {
					"cached_tokens": 80
				}
			},
			"choices": [{
				"message": {
					"role": "assistant",
					"content": "Test answer"
				},
				"finish_reason": "stop"
			}]
		}`)
		action = host.CallOnHttpResponseBody(responseBody)
		require.Equal(t, types.ActionContinue, action)

		// 5. 完成请求
		host.CompleteHttp()

		// 6. 验证 AI 日志字段包含 token details
		aiLogField := host.GetFilterState("wasm.ai_log")
		require.NotEmpty(t, aiLogField)
		
		// 验证日志中包含 reasoning_tokens 和 cached_tokens
		require.Contains(t, string(aiLogField), "reasoning_tokens")
		require.Contains(t, string(aiLogField), "cached_tokens")
		require.Contains(t, string(aiLogField), "input_token_details")
		require.Contains(t, string(aiLogField), "output_token_details")
	})
}
