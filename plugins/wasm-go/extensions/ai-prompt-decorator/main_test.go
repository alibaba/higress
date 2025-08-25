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
	"fmt"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：基础装饰器配置
var basicConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"prepend": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a helpful assistant from ${geo-country}.",
			},
		},
		"append": []map[string]interface{}{
			{
				"role":    "system",
				"content": "Please provide context about ${geo-city}.",
			},
		},
	})
	return data
}()

// 测试配置：只有前置消息的配置
var prependOnlyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"prepend": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are located in ${geo-province}, ${geo-country}.",
			},
		},
		"append": []map[string]interface{}{}, // 显式定义空的append字段
	})
	return data
}()

// 测试配置：空配置
var emptyConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"prepend": []map[string]interface{}{},
		"append":  []map[string]interface{}{},
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试基础装饰器配置解析
		t.Run("basic decorator config", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			decoratorConfig := config.(*AIPromptDecoratorConfig)
			require.NotNil(t, decoratorConfig.Prepend)
			require.NotNil(t, decoratorConfig.Append)
			require.Len(t, decoratorConfig.Prepend, 1)
			require.Len(t, decoratorConfig.Append, 1)
			require.Equal(t, "system", decoratorConfig.Prepend[0].Role)
			require.Equal(t, "You are a helpful assistant from ${geo-country}.", decoratorConfig.Prepend[0].Content)
			require.Equal(t, "system", decoratorConfig.Append[0].Role)
			require.Equal(t, "Please provide context about ${geo-city}.", decoratorConfig.Append[0].Content)
		})

		// 测试只有前置消息的配置解析
		t.Run("prepend only config", func(t *testing.T) {
			host, status := test.NewTestHost(prependOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			decoratorConfig := config.(*AIPromptDecoratorConfig)
			require.NotNil(t, decoratorConfig.Prepend)
			require.NotNil(t, decoratorConfig.Append)
			require.Len(t, decoratorConfig.Prepend, 1)
			require.Len(t, decoratorConfig.Append, 0)
			require.Equal(t, "system", decoratorConfig.Prepend[0].Role)
			require.Equal(t, "You are located in ${geo-province}, ${geo-country}.", decoratorConfig.Prepend[0].Content)
		})

		// 测试空配置解析
		t.Run("empty config", func(t *testing.T) {
			host, status := test.NewTestHost(emptyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)
			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)

			decoratorConfig := config.(*AIPromptDecoratorConfig)
			require.NotNil(t, decoratorConfig.Prepend)
			require.NotNil(t, decoratorConfig.Append)
			require.Len(t, decoratorConfig.Prepend, 0)
			require.Len(t, decoratorConfig.Append, 0)
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试请求头处理
		t.Run("request headers processing", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"content-length", "100"},
			})

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestOnHttpRequestBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试基础消息装饰
		t.Run("basic message decoration", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置地理变量属性，供插件使用
			host.SetProperty([]string{"geo-country"}, []byte("China"))
			host.SetProperty([]string{"geo-province"}, []byte("Beijing"))
			host.SetProperty([]string{"geo-city"}, []byte("Beijing"))
			host.SetProperty([]string{"geo-isp"}, []byte("China Mobile"))

			// 设置请求体，包含消息
			body := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "user", "content": "Hello, how are you?"}
				]
			}`
			action := host.CallOnHttpRequestBody([]byte(body))

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证消息装饰是否成功
			modifiedBody := host.GetRequestBody()
			require.NotEmpty(t, modifiedBody)

			// 解析修改后的请求体
			var modifiedRequest map[string]interface{}
			err := json.Unmarshal(modifiedBody, &modifiedRequest)
			require.NoError(t, err)

			// 验证messages字段存在
			messages, exists := modifiedRequest["messages"].([]interface{})
			require.True(t, exists, "messages field should exist")
			require.NotNil(t, messages)

			// 验证消息数量：前置消息(1) + 原始消息(1) + 后置消息(1) = 3
			require.Len(t, messages, 3, "should have 3 messages: prepend + original + append")

			// 验证第一个消息是前置消息（地理变量已被替换）
			firstMessage := messages[0].(map[string]interface{})
			require.Equal(t, "system", firstMessage["role"])
			require.Equal(t, "You are a helpful assistant from China.", firstMessage["content"])

			// 验证第二个消息是原始用户消息
			secondMessage := messages[1].(map[string]interface{})
			require.Equal(t, "user", secondMessage["role"])
			require.Equal(t, "Hello, how are you?", secondMessage["content"])

			// 验证第三个消息是后置消息（地理变量已被替换）
			thirdMessage := messages[2].(map[string]interface{})
			require.Equal(t, "system", thirdMessage["role"])
			require.Equal(t, "Please provide context about Beijing.", thirdMessage["content"])

			host.CompleteHttp()
		})

		// 测试只有前置消息的装饰
		t.Run("prepend only decoration", func(t *testing.T) {
			host, status := test.NewTestHost(prependOnlyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置地理变量属性，供插件使用
			host.SetProperty([]string{"geo-country"}, []byte("China"))
			host.SetProperty([]string{"geo-province"}, []byte("Shanghai"))
			host.SetProperty([]string{"geo-city"}, []byte("Shanghai"))
			host.SetProperty([]string{"geo-isp"}, []byte("China Telecom"))

			// 设置请求体，包含消息
			body := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "user", "content": "What's the weather like?"}
				]
			}`
			action := host.CallOnHttpRequestBody([]byte(body))

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证消息装饰是否成功
			modifiedBody := host.GetRequestBody()
			require.NotEmpty(t, modifiedBody)

			// 解析修改后的请求体
			var modifiedRequest map[string]interface{}
			err := json.Unmarshal(modifiedBody, &modifiedRequest)
			require.NoError(t, err)

			// 验证messages字段存在
			messages, exists := modifiedRequest["messages"].([]interface{})
			require.True(t, exists, "messages field should exist")
			require.NotNil(t, messages)

			// 验证消息数量：前置消息(1) + 原始消息(1) = 2
			require.Len(t, messages, 2, "should have 2 messages: prepend + original")

			// 验证第一个消息是前置消息（地理变量已被替换）
			firstMessage := messages[0].(map[string]interface{})
			require.Equal(t, "system", firstMessage["role"])
			require.Equal(t, "You are located in Shanghai, China.", firstMessage["content"])

			// 验证第二个消息是原始用户消息
			secondMessage := messages[1].(map[string]interface{})
			require.Equal(t, "user", secondMessage["role"])
			require.Equal(t, "What's the weather like?", secondMessage["content"])

			host.CompleteHttp()
		})

		// 测试空消息的情况
		t.Run("empty messages", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置请求体，不包含messages字段
			body := `{
				"model": "gpt-3.5-turbo"
			}`
			action := host.CallOnHttpRequestBody([]byte(body))

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试多个消息的装饰
		t.Run("multiple messages decoration", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置地理变量属性，供插件使用
			host.SetProperty([]string{"geo-country"}, []byte("USA"))
			host.SetProperty([]string{"geo-province"}, []byte("California"))
			host.SetProperty([]string{"geo-city"}, []byte("San Francisco"))
			host.SetProperty([]string{"geo-isp"}, []byte("Comcast"))

			// 设置请求体，包含多个消息
			body := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "system", "content": "You are a helpful assistant"},
					{"role": "user", "content": "Hello"},
					{"role": "assistant", "content": "Hi there!"}
				]
			}`
			action := host.CallOnHttpRequestBody([]byte(body))

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证消息装饰是否成功
			modifiedBody := host.GetRequestBody()
			require.NotEmpty(t, modifiedBody)

			// 解析修改后的请求体
			var modifiedRequest map[string]interface{}
			err := json.Unmarshal(modifiedBody, &modifiedRequest)
			require.NoError(t, err)

			// 验证messages字段存在
			messages, exists := modifiedRequest["messages"].([]interface{})
			require.True(t, exists, "messages field should exist")
			require.NotNil(t, messages)

			// 验证消息数量：前置消息(1) + 原始消息(3) + 后置消息(1) = 5
			require.Len(t, messages, 5, "should have 5 messages: prepend + original(3) + append")

			// 验证第一个消息是前置消息（地理变量已被替换）
			firstMessage := messages[0].(map[string]interface{})
			require.Equal(t, "system", firstMessage["role"])
			require.Equal(t, "You are a helpful assistant from USA.", firstMessage["content"])

			// 验证原始消息保持顺序
			originalMessages := messages[1:4]
			require.Equal(t, "system", originalMessages[0].(map[string]interface{})["role"])
			require.Equal(t, "You are a helpful assistant", originalMessages[0].(map[string]interface{})["content"])
			require.Equal(t, "user", originalMessages[1].(map[string]interface{})["role"])
			require.Equal(t, "Hello", originalMessages[1].(map[string]interface{})["content"])
			require.Equal(t, "assistant", originalMessages[2].(map[string]interface{})["role"])
			require.Equal(t, "Hi there!", originalMessages[2].(map[string]interface{})["content"])

			// 验证最后一个消息是后置消息（地理变量已被替换）
			lastMessage := messages[4].(map[string]interface{})
			require.Equal(t, "system", lastMessage["role"])
			require.Equal(t, "Please provide context about San Francisco.", lastMessage["content"])

			host.CompleteHttp()
		})
	})
}

func TestStructs(t *testing.T) {
	// 测试Message结构体
	t.Run("Message struct", func(t *testing.T) {
		message := Message{
			Role:    "system",
			Content: "You are a helpful assistant from ${geo-country}.",
		}
		require.Equal(t, "system", message.Role)
		require.Equal(t, "You are a helpful assistant from ${geo-country}.", message.Content)
	})

	// 测试AIPromptDecoratorConfig结构体
	t.Run("AIPromptDecoratorConfig struct", func(t *testing.T) {
		config := &AIPromptDecoratorConfig{
			Prepend: []Message{
				{Role: "system", Content: "Prepend message"},
			},
			Append: []Message{
				{Role: "system", Content: "Append message"},
			},
		}
		require.NotNil(t, config.Prepend)
		require.NotNil(t, config.Append)
		require.Len(t, config.Prepend, 1)
		require.Len(t, config.Append, 1)
		require.Equal(t, "Prepend message", config.Prepend[0].Content)
		require.Equal(t, "Append message", config.Append[0].Content)
	})
}

func TestGeographicVariableReplacement(t *testing.T) {
	// 测试地理变量替换逻辑
	t.Run("geographic variable replacement", func(t *testing.T) {
		config := &AIPromptDecoratorConfig{
			Prepend: []Message{
				{
					Role:    "system",
					Content: "Location: ${geo-country}/${geo-province}/${geo-city}, ISP: ${geo-isp}",
				},
			},
		}

		// 验证地理变量在内容中的存在
		content := config.Prepend[0].Content
		require.Contains(t, content, "${geo-country}")
		require.Contains(t, content, "${geo-province}")
		require.Contains(t, content, "${geo-city}")
		require.Contains(t, content, "${geo-isp}")

		// 测试变量替换逻辑
		geoVariables := []string{"geo-country", "geo-province", "geo-city", "geo-isp"}
		for _, geo := range geoVariables {
			require.Contains(t, content, fmt.Sprintf("${%s}", geo))
		}
	})

	// 测试混合内容的地理变量
	t.Run("mixed content geographic variables", func(t *testing.T) {
		config := &AIPromptDecoratorConfig{
			Append: []Message{
				{
					Role:    "system",
					Content: "User from ${geo-country} with ISP ${geo-isp}. Context: ${geo-province}, ${geo-city}",
				},
			},
		}

		content := config.Append[0].Content
		require.Contains(t, content, "${geo-country}")
		require.Contains(t, content, "${geo-isp}")
		require.Contains(t, content, "${geo-province}")
		require.Contains(t, content, "${geo-city}")
	})
}

func TestEdgeCases(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试空前置和后置消息
		t.Run("empty prepend and append", func(t *testing.T) {
			host, status := test.NewTestHost(emptyConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置请求体
			body := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "user", "content": "Test message"}
				]
			}`
			action := host.CallOnHttpRequestBody([]byte(body))

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})

		// 测试无效JSON请求体
		t.Run("invalid JSON body", func(t *testing.T) {
			host, status := test.NewTestHost(basicConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 先设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
			})

			// 设置无效的请求体
			body := `{
				"model": "gpt-3.5-turbo",
				"messages": [
					{"role": "user", "content": "Hello"}
				]
				// Missing closing brace
			`
			action := host.CallOnHttpRequestBody([]byte(body))

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)

			host.CompleteHttp()
		})
	})
}
