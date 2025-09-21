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

// 测试配置：基本配置
var basicConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"apiKey":      "sk-test123456789",
		"promptParam": "prompt",
		"model":       "text-davinci-003",
	})
	return data
}()

// 测试配置：自定义模型配置
var customModelConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"apiKey":      "sk-test123456789",
		"promptParam": "text",
		"model":       "curie",
	})
	return data
}()

// 测试配置：自定义提示参数配置
var customPromptParamConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"apiKey":      "sk-test123456789",
		"promptParam": "question",
		"model":       "text-davinci-003",
	})
	return data
}()

// 测试配置：自定义 ChatGPT URI 配置
var customUriConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"apiKey":      "sk-test123456789",
		"promptParam": "prompt",
		"model":       "text-davinci-003",
		"chatgptUri":  "https://custom-ai.example.com/v1/chat/completions",
	})
	return data
}()

// 测试配置：自定义 Human ID 和 AI ID 配置
var customIdsConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"apiKey":      "sk-test123456789",
		"promptParam": "prompt",
		"model":       "text-davinci-003",
		"HumainId":    "User:",
		"AIId":        "Assistant:",
	})
	return data
}()

// 测试配置：无效配置（缺少 API Key）
var invalidConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"promptParam": "prompt",
		"model":       "text-davinci-003",
	})
	return data
}()

// 测试配置：无效 URI 配置
var invalidUriConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"apiKey":      "sk-test123456789",
		"promptParam": "prompt",
		"model":       "text-davinci-003",
		"chatgptUri":  "://invalid-uri",
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基本配置解析
		t.Run("basic config", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试自定义模型配置解析
		t.Run("custom model config", func(t *testing.T) {
			host, status := test.NewTestHost(customModelConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试自定义提示参数配置解析
		t.Run("custom prompt param config", func(t *testing.T) {
			host, status := test.NewTestHost(customPromptParamConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试自定义 URI 配置解析
		t.Run("custom uri config", func(t *testing.T) {
			host, status := test.NewTestHost(customUriConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试自定义 ID 配置解析
		t.Run("custom ids config", func(t *testing.T) {
			host, status := test.NewTestHost(customIdsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// 测试无效配置（缺少 API Key）
		t.Run("invalid config - missing api key", func(t *testing.T) {
			host, status := test.NewTestHost(invalidConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试无效 URI 配置
		t.Run("invalid config - invalid uri", func(t *testing.T) {
			host, status := test.NewTestHost(invalidUriConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试基本请求头处理（带查询参数）
		t.Run("basic request headers with query params", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含查询参数
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat?prompt=Hello, how are you?"},
				{":method", "GET"},
			})

			// 由于需要调用外部 AI 服务，应该返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			// 模拟外部 AI 服务响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(`{"choices":[{"text":"I'm doing well, thank you for asking!"}]}`))

			response := host.GetLocalResponse()
			require.Equal(t, uint32(200), response.StatusCode)
			require.Equal(t, `{"choices":[{"text":"I'm doing well, thank you for asking!"}]}`, string(response.Data))

			host.CompleteHttp()
		})

		// 测试自定义提示参数请求头处理
		t.Run("custom prompt param request headers", func(t *testing.T) {
			host, status := test.NewTestHost(customPromptParamConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，使用自定义提示参数
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat?question=What is the weather like?"},
				{":method", "GET"},
			})

			// 由于需要调用外部 AI 服务，应该返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			// 模拟外部 AI 服务响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(`{"choices":[{"text":"I don't have access to real-time weather information."}]}`))

			response := host.GetLocalResponse()
			require.Equal(t, uint32(200), response.StatusCode)
			require.Equal(t, `{"choices":[{"text":"I don't have access to real-time weather information."}]}`, string(response.Data))

			host.CompleteHttp()
		})

		// 测试缺少查询参数的情况
		t.Run("missing query params", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，不包含查询参数
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "GET"},
			})

			// 应该返回 ActionContinue，因为缺少查询参数
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试缺少提示参数的情况
		t.Run("missing prompt param", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含查询参数但不包含提示参数
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat?other=value"},
				{":method", "GET"},
			})

			// 应该返回 ActionContinue，因为缺少提示参数
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试空提示参数的情况
		t.Run("empty prompt param", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头，包含空的提示参数
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat?prompt="},
				{":method", "GET"},
			})

			// 由于需要调用外部 AI 服务，应该返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			// 模拟外部 AI 服务响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(`{"choices":[{"text":"Empty prompt response"}]}`))

			response := host.GetLocalResponse()
			require.Equal(t, uint32(200), response.StatusCode)
			require.Equal(t, `{"choices":[{"text":"Empty prompt response"}]}`, string(response.Data))

			host.CompleteHttp()
		})

		// 测试外部服务调用成功的情况
		t.Run("external service call success", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat?prompt=Tell me a joke"},
				{":method", "GET"},
			})

			// 由于需要调用外部 AI 服务，应该返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			// 模拟外部 AI 服务成功响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(`{"choices":[{"text":"Why don't scientists trust atoms? Because they make up everything!"}]}`))

			response := host.GetLocalResponse()
			require.Equal(t, uint32(200), response.StatusCode)
			require.Equal(t, `{"choices":[{"text":"Why don't scientists trust atoms? Because they make up everything!"}]}`, string(response.Data))

			host.CompleteHttp()
		})

		// 测试外部服务调用失败的情况
		t.Run("external service call failure", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat?prompt=Hello"},
				{":method", "GET"},
			})

			// 由于需要调用外部 AI 服务，应该返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			// 模拟外部 AI 服务失败响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "429"},
			}, []byte(`{"error":"Rate limit exceeded"}`))

			response := host.GetLocalResponse()
			require.Equal(t, uint32(429), response.StatusCode)
			require.Equal(t, `{"error":"Rate limit exceeded"}`, string(response.Data))

			host.CompleteHttp()
		})
	})
}

func TestCompleteFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("complete chatgpt proxy flow", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 1. 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat?prompt=What is artificial intelligence?"},
				{":method", "GET"},
			})

			// 由于需要调用外部 AI 服务，应该返回 ActionPause
			require.Equal(t, types.ActionPause, action)

			// 2. 模拟外部 AI 服务响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(`{"choices":[{"text":"Artificial Intelligence (AI) is a branch of computer science that aims to create systems capable of performing tasks that typically require human intelligence."}]}`))

			response := host.GetLocalResponse()
			require.Equal(t, uint32(200), response.StatusCode)
			require.Equal(t, `{"choices":[{"text":"Artificial Intelligence (AI) is a branch of computer science that aims to create systems capable of performing tasks that typically require human intelligence."}]}`, string(response.Data))

			// 3. 完成请求
			host.CompleteHttp()
		})
	})
}
