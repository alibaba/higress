package main

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：完整配置
var completeConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"returnResponseTemplate": `{"id":"error","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`,
		"llm": map[string]interface{}{
			"apiKey":           "test-api-key",
			"serviceName":      "llm-service",
			"servicePort":      8080,
			"domain":           "llm.example.com",
			"path":             "/v1/chat/completions",
			"model":            "qwen-turbo",
			"maxIterations":    20,
			"maxExecutionTime": 60000,
			"maxTokens":        2000,
		},
		"apis": []map[string]interface{}{
			{
				"apiProvider": map[string]interface{}{
					"serviceName":      "api-service",
					"servicePort":      9090,
					"domain":           "api.example.com",
					"maxExecutionTime": 30000,
					"apiKey": map[string]interface{}{
						"in":    "header",
						"name":  "Authorization",
						"value": "Bearer test-token",
					},
				},
				"api": `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /weather:
    get:
      operationId: getWeather
      summary: Get weather information
      description: Retrieve current weather data
      parameters:
        - name: city
          in: query
          required: true
          schema:
            type: string
        - name: date
          in: query
          required: false
          schema:
            type: string
  /translate:
    post:
      operationId: translateText
      summary: Translate text
      description: Translate text to target language
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required:
                - text
                - targetLang
              properties:
                text:
                  type: string
                sourceLang:
                  type: string
                targetLang:
                  type: string`,
			},
		},
		"promptTemplate": map[string]interface{}{
			"language": "EN",
			"enTemplate": map[string]interface{}{
				"question":    "What is your question?",
				"thought1":    "Let me think about this",
				"observation": "Based on the observation",
				"thought2":    "Now I understand",
			},
			"chTemplate": map[string]interface{}{
				"question":    "你的问题是什么？",
				"thought1":    "让我思考一下",
				"observation": "基于观察结果",
				"thought2":    "现在我明白了",
			},
		},
		"jsonResp": map[string]interface{}{
			"enable": true,
			"jsonSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"answer": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	})
	return data
}()

// 测试配置：最小配置（使用默认值）
var minimalConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"llm": map[string]interface{}{
			"apiKey":      "test-api-key",
			"serviceName": "llm-service",
			"servicePort": 8080,
			"domain":      "llm.example.com",
			"path":        "/v1/chat/completions",
			"model":       "qwen-turbo",
		},
		"apis": []map[string]interface{}{
			{
				"apiProvider": map[string]interface{}{
					"serviceName": "api-service",
					"servicePort": 9090,
					"domain":      "api.example.com",
					"apiKey": map[string]interface{}{
						"in":    "query",
						"name":  "api_key",
						"value": "test-token",
					},
				},
				"api": `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /simple:
    get:
      operationId: simpleGet
      summary: Simple GET endpoint
      parameters:
        - name: id
          in: query
          required: true
          schema:
            type: string`,
			},
		},
	})
	return data
}()

// 测试配置：中文提示模板
var chinesePromptConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"llm": map[string]interface{}{
			"apiKey":      "test-api-key",
			"serviceName": "llm-service",
			"servicePort": 8080,
			"domain":      "llm.example.com",
			"path":        "/v1/chat/completions",
			"model":       "qwen-turbo",
		},
		"apis": []map[string]interface{}{
			{
				"apiProvider": map[string]interface{}{
					"serviceName": "api-service",
					"servicePort": 9090,
					"domain":      "api.example.com",
					"apiKey": map[string]interface{}{
						"in":    "header",
						"name":  "X-API-Key",
						"value": "test-token",
					},
				},
				"api": `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /test:
    post:
      operationId: testPost
      summary: Test POST endpoint
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required:
                - data
              properties:
                data:
                  type: string`,
			},
		},
		"promptTemplate": map[string]interface{}{
			"language": "CH",
		},
	})
	return data
}()

// 测试配置：缺少必需字段
var missingRequiredFieldsConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"llm": map[string]interface{}{
			"apiKey": "test-api-key",
			// 缺少 serviceName, servicePort, domain, path, model
		},
		// 缺少 apis
	})
	return data
}()

// 测试配置：空APIs数组
var emptyAPIsConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"llm": map[string]interface{}{
			"apiKey":      "test-api-key",
			"serviceName": "llm-service",
			"servicePort": 8080,
			"domain":      "llm.example.com",
			"path":        "/v1/chat/completions",
			"model":       "qwen-turbo",
		},
		"apis": []map[string]interface{}{},
	})
	return data
}()

// 测试配置：缺少API提供者信息
var missingAPIProviderConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"llm": map[string]interface{}{
			"apiKey":      "test-api-key",
			"serviceName": "llm-service",
			"servicePort": 8080,
			"domain":      "llm.example.com",
			"path":        "/v1/chat/completions",
			"model":       "qwen-turbo",
		},
		"apis": []map[string]interface{}{
			{
				"api": `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /test:
    get:
      operationId: testGet
      summary: Test endpoint`,
			},
		},
	})
	return data
}()

// 测试配置：用于HTTP请求测试的简化配置
var httpTestConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"returnResponseTemplate": `{"id":"error","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`,
		"llm": map[string]interface{}{
			"apiKey":      "test-api-key",
			"serviceName": "llm-service",
			"servicePort": 8080,
			"domain":      "llm.example.com",
			"path":        "/v1/chat/completions",
			"model":       "qwen-turbo",
		},
		"apis": []map[string]interface{}{
			{
				"apiProvider": map[string]interface{}{
					"serviceName": "api-service",
					"servicePort": 9090,
					"domain":      "api.example.com",
					"apiKey": map[string]interface{}{
						"in":    "header",
						"name":  "Authorization",
						"value": "Bearer test-token",
					},
				},
				"api": `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /weather:
    get:
      operationId: getWeather
      summary: Get weather information
      parameters:
        - name: city
          in: query
          required: true
          schema:
            type: string`,
			},
		},
		"promptTemplate": map[string]interface{}{
			"language": "EN",
		},
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试完整配置解析
		t.Run("complete config", func(t *testing.T) {
			host, status := test.NewTestHost(completeConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			configRaw, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, configRaw)

			config, ok := configRaw.(*PluginConfig)
			require.True(t, ok, "config should be of type *PluginConfig")

			// 验证响应模板
			require.Contains(t, config.ReturnResponseTemplate, "gpt-4o")

			// 验证LLM配置
			require.Equal(t, "test-api-key", config.LLMInfo.APIKey)
			require.Equal(t, "llm-service", config.LLMInfo.ServiceName)
			require.Equal(t, int64(8080), config.LLMInfo.ServicePort)
			require.Equal(t, "llm.example.com", config.LLMInfo.Domain)
			require.Equal(t, "/v1/chat/completions", config.LLMInfo.Path)
			require.Equal(t, "qwen-turbo", config.LLMInfo.Model)
			require.Equal(t, int64(20), config.LLMInfo.MaxIterations)
			require.Equal(t, int64(60000), config.LLMInfo.MaxExecutionTime)
			require.Equal(t, int64(2000), config.LLMInfo.MaxTokens)

			// 验证API配置
			require.Len(t, config.APIsParam, 1)
			require.Len(t, config.APIsParam[0].ToolsParam, 2)

			// 验证GET工具
			getTool := config.APIsParam[0].ToolsParam[0]
			require.Equal(t, "getWeather", getTool.ToolName)
			require.Equal(t, "GET", getTool.Method)
			require.Equal(t, "/weather", getTool.Path)
			require.Contains(t, getTool.ParamName, "city")
			require.Contains(t, getTool.ParamName, "date")

			// 验证POST工具
			postTool := config.APIsParam[0].ToolsParam[1]
			require.Equal(t, "translateText", postTool.ToolName)
			require.Equal(t, "POST", postTool.Method)
			require.Equal(t, "/translate", postTool.Path)
			require.Contains(t, postTool.ParamName, "text")
			require.Contains(t, postTool.ParamName, "targetLang")

			// 验证提示模板
			require.Equal(t, "EN", config.PromptTemplate.Language)
			require.Equal(t, "What is your question?", config.PromptTemplate.ENTemplate.Question)
			require.Equal(t, "Let me think about this", config.PromptTemplate.ENTemplate.Thought1)
			require.Equal(t, "Based on the observation", config.PromptTemplate.ENTemplate.Observation)
			require.Equal(t, "Now I understand", config.PromptTemplate.ENTemplate.Thought2)

			// 验证JSON响应配置
			require.True(t, config.JsonResp.Enable)
			require.NotNil(t, config.JsonResp.JsonSchema)
		})

		// 测试最小配置解析（使用默认值）
		t.Run("minimal config with defaults", func(t *testing.T) {
			host, status := test.NewTestHost(minimalConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			configRaw, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, configRaw)

			config, ok := configRaw.(*PluginConfig)
			require.True(t, ok, "config should be of type *PluginConfig")

			// 验证默认响应模板
			require.Contains(t, config.ReturnResponseTemplate, "gpt-4o")

			// 验证LLM默认值
			require.Equal(t, int64(15), config.LLMInfo.MaxIterations)
			require.Equal(t, int64(50000), config.LLMInfo.MaxExecutionTime)
			require.Equal(t, int64(1000), config.LLMInfo.MaxTokens)

			// 验证API默认值
			require.Equal(t, int64(50000), config.APIsParam[0].MaxExecutionTime)

			// 验证提示模板默认值
			require.Equal(t, "EN", config.PromptTemplate.Language)
			require.Equal(t, "input question to answer", config.PromptTemplate.ENTemplate.Question)
			require.Equal(t, "consider previous and subsequent steps", config.PromptTemplate.ENTemplate.Thought1)
			require.Equal(t, "action result", config.PromptTemplate.ENTemplate.Observation)
			require.Equal(t, "I know what to respond", config.PromptTemplate.ENTemplate.Thought2)

			// 验证JSON响应默认值
			require.False(t, config.JsonResp.Enable)
		})

		// 测试中文提示模板配置
		t.Run("chinese prompt template config", func(t *testing.T) {
			host, status := test.NewTestHost(chinesePromptConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			configRaw, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, configRaw)

			config, ok := configRaw.(*PluginConfig)
			require.True(t, ok, "config should be of type *PluginConfig")

			// 验证中文提示模板
			require.Equal(t, "CH", config.PromptTemplate.Language)
			require.Equal(t, "输入要回答的问题", config.PromptTemplate.CHTemplate.Question)
			require.Equal(t, "考虑之前和之后的步骤", config.PromptTemplate.CHTemplate.Thought1)
			require.Equal(t, "行动结果", config.PromptTemplate.CHTemplate.Observation)
			require.Equal(t, "我知道该回应什么", config.PromptTemplate.CHTemplate.Thought2)
		})

		// 测试缺少必需字段的配置
		t.Run("missing required fields config", func(t *testing.T) {
			host, status := test.NewTestHost(missingRequiredFieldsConfig)
			defer host.Reset()
			// 由于缺少必需字段（apis），配置应该失败
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试空APIs数组配置
		t.Run("empty APIs config", func(t *testing.T) {
			host, status := test.NewTestHost(emptyAPIsConfig)
			defer host.Reset()
			// 空APIs数组应该导致配置解析失败
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// 测试缺少API提供者信息的配置
		t.Run("missing API provider config", func(t *testing.T) {
			host, status := test.NewTestHost(missingAPIProviderConfig)
			defer host.Reset()
			// 缺少API提供者信息应该导致配置解析失败
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})
	})
}

func TestOnHttpRequestHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("basic request headers", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// onHttpRequestHeaders应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestOnHttpRequestBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("valid request body with single message", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 构造有效的请求体
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`

			// 调用请求体处理
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 应该返回ActionContinue，因为需要等待LLM响应
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确修改
			modifiedBody := host.GetRequestBody()
			require.NotNil(t, modifiedBody)

			// 解析修改后的请求体
			var modifiedRequest Request
			err := json.Unmarshal(modifiedBody, &modifiedRequest)
			require.NoError(t, err)

			// 验证消息是否被正确设置
			require.Len(t, modifiedRequest.Messages, 1)
			require.Equal(t, "user", modifiedRequest.Messages[0].Role)
			require.Contains(t, modifiedRequest.Messages[0].Content, "今天天气怎么样？")

			// 验证stream是否被设置为false
			require.False(t, modifiedRequest.Stream)
		})

		t.Run("request body with conversation history", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 构造包含对话历史的请求体
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "你好"
					},
					{
						"role": "assistant",
						"content": "你好！有什么可以帮助你的吗？"
					},
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`

			// 调用请求体处理
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确修改
			modifiedBody := host.GetRequestBody()
			require.NotNil(t, modifiedBody)

			// 解析修改后的请求体
			var modifiedRequest Request
			err := json.Unmarshal(modifiedBody, &modifiedRequest)
			require.NoError(t, err)

			// 验证消息是否被正确设置
			require.Len(t, modifiedRequest.Messages, 1)
			require.Equal(t, "user", modifiedRequest.Messages[0].Role)
			require.Contains(t, modifiedRequest.Messages[0].Content, "今天天气怎么样？")
		})

		t.Run("stream request body", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 构造流式请求体
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "流式响应测试"
					}
				],
				"stream": true
			}`

			// 调用请求体处理
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确修改
			modifiedBody := host.GetRequestBody()
			require.NotNil(t, modifiedBody)

			// 解析修改后的请求体
			var modifiedRequest Request
			err := json.Unmarshal(modifiedBody, &modifiedRequest)
			require.NoError(t, err)

			// 验证stream是否被设置为false
			require.False(t, modifiedRequest.Stream)
		})

		t.Run("empty messages array", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 构造空消息数组的请求体
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [],
				"stream": false
			}`

			// 调用请求体处理
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 应该返回ActionContinue，因为没有消息需要处理
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("invalid JSON request body", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 构造无效JSON的请求体
			invalidJSON := []byte(`{"model": "qwen-turbo", "messages": [{"role": "user", "content": "test"}`)

			// 调用请求体处理
			action := host.CallOnHttpRequestBody(invalidJSON)

			// 应该返回ActionContinue，因为JSON解析失败
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("empty content in message", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 构造空内容的请求体
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": ""
					}
				],
				"stream": false
			}`

			// 调用请求体处理
			action := host.CallOnHttpRequestBody([]byte(requestBody))

			// 应该返回ActionContinue，因为内容为空
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func TestOnHttpResponseBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("valid LLM response with content", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 构造有效的LLM响应体
			responseBody := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": \"{\\\"city\\\": \\\"北京\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			// 应该返回ActionPause，因为需要等待工具调用结果
			require.Equal(t, types.ActionPause, action)

			// 模拟API工具调用的响应
			apiResponse := `{"temperature": 25, "condition": "晴朗", "humidity": 60}`
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(apiResponse))

			// 模拟LLM对工具调用结果的响应（Final Answer）
			llmFinalResponse := `{
				"id": "chatcmpl-124",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Final Answer: 今天北京天气晴朗，温度25度，湿度60%"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652289,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 15,
					"completion_tokens": 25,
					"total_tokens": 40
				}
			}`

			// 模拟LLM客户端的响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(llmFinalResponse))

			// 完成HTTP请求
			host.CompleteHttp()
		})

		t.Run("LLM response with Final Answer", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 构造包含Final Answer的LLM响应体
			responseBody := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Final Answer: 今天北京天气晴朗，温度25度"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			// 应该返回ActionContinue，因为得到了Final Answer
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("LLM response with empty content", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 构造空内容的LLM响应体
			responseBody := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": ""
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 0,
					"total_tokens": 10
				}
			}`

			// 调用响应体处理
			action := host.CallOnHttpResponseBody([]byte(responseBody))

			// 应该返回ActionContinue，因为内容为空
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("invalid LLM response JSON", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 构造无效JSON的响应体
			invalidJSON := []byte(`{"id": "chatcmpl-123", "choices": [{"index": 0, "message": {"role": "assistant", "content": "test"}`)

			// 调用响应体处理
			action := host.CallOnHttpResponseBody(invalidJSON)

			// 应该返回ActionContinue，因为JSON解析失败
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("complete ReAct loop flow", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "查询北京和上海的天气"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 第一次LLM响应，要求调用工具查询北京天气
			llmResponse1 := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": \"{\\\"city\\\": \\\"北京\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理，这会触发toolsCall
			action := host.CallOnHttpResponseBody([]byte(llmResponse1))

			// 应该返回ActionPause，因为需要等待工具调用结果
			require.Equal(t, types.ActionPause, action)

			// 模拟API工具调用的响应（北京天气）
			apiResponse1 := `{"temperature": 25, "condition": "晴朗", "humidity": 60}`
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(apiResponse1))

			// 第二次LLM响应，要求调用工具查询上海天气
			llmResponse2 := `{
				"id": "chatcmpl-124",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": \"{\\\"city\\\": \\\"上海\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652289,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 15,
					"completion_tokens": 25,
					"total_tokens": 40
				}
			}`

			// 模拟LLM客户端的响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(llmResponse2))

			// 模拟API工具调用的响应（上海天气）
			apiResponse2 := `{"temperature": 28, "condition": "多云", "humidity": 70}`
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(apiResponse2))

			// 第三次LLM响应，给出Final Answer
			llmResponse3 := `{
				"id": "chatcmpl-125",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Final Answer: 北京今天天气晴朗，温度25度，湿度60%；上海今天天气多云，温度28度，湿度70%"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652290,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 20,
					"completion_tokens": 30,
					"total_tokens": 50
				}
			}`

			// 模拟LLM客户端的响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(llmResponse3))

			// 完成HTTP请求
			host.CompleteHttp()
		})
	})
}

func TestFirstReq(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("successful request body replacement", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 构造原始请求
			originalRequest := Request{
				Model: "qwen-turbo",
				Messages: []Message{
					{
						Role:    "user",
						Content: "原始消息",
					},
				},
				Stream: true,
			}

			// 调用firstReq（通过onHttpRequestBody间接调用）
			requestBody, _ := json.Marshal(originalRequest)
			action := host.CallOnHttpRequestBody(requestBody)

			// 应该返回ActionContinue
			require.Equal(t, types.ActionContinue, action)

			// 验证请求体是否被正确修改
			modifiedBody := host.GetRequestBody()
			require.NotNil(t, modifiedBody)

			// 解析修改后的请求体
			var modifiedRequest Request
			err := json.Unmarshal(modifiedBody, &modifiedRequest)
			require.NoError(t, err)

			// 验证stream是否被设置为false
			require.False(t, modifiedRequest.Stream)

			// 验证消息是否被正确设置
			require.Len(t, modifiedRequest.Messages, 1)
			require.Equal(t, "user", modifiedRequest.Messages[0].Role)
			require.Contains(t, modifiedRequest.Messages[0].Content, "原始消息")
		})
	})
}

func TestToolsCall(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("GET tool call with complete flow", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 模拟LLM响应，要求调用GET工具
			llmResponse := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": \"{\\\"city\\\": \\\"北京\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理，这会触发toolsCall
			action := host.CallOnHttpResponseBody([]byte(llmResponse))

			// 应该返回ActionPause，因为需要等待工具调用结果
			require.Equal(t, types.ActionPause, action)

			// 模拟API工具调用的响应
			apiResponse := `{"temperature": 25, "condition": "晴朗"}`
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(apiResponse))

			// 模拟LLM对工具调用结果的响应（Final Answer）
			llmFinalResponse := `{
				"id": "chatcmpl-124",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Final Answer: 今天北京天气晴朗，温度25度，湿度60%"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652289,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 15,
					"completion_tokens": 25,
					"total_tokens": 40
				}
			}`

			// 模拟LLM客户端的响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(llmFinalResponse))

			// 完成HTTP请求
			host.CompleteHttp()
		})

		t.Run("POST tool call with complete flow", func(t *testing.T) {
			// 创建一个支持POST工具的配置
			postToolConfig := func() json.RawMessage {
				data, _ := json.Marshal(map[string]interface{}{
					"returnResponseTemplate": `{"id":"error","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`,
					"llm": map[string]interface{}{
						"apiKey":      "test-api-key",
						"serviceName": "llm-service",
						"servicePort": 8080,
						"domain":      "llm.example.com",
						"path":        "/v1/chat/completions",
						"model":       "qwen-turbo",
					},
					"apis": []map[string]interface{}{
						{
							"apiProvider": map[string]interface{}{
								"serviceName": "api-service",
								"servicePort": 9090,
								"domain":      "api.example.com",
								"apiKey": map[string]interface{}{
									"in":    "header",
									"name":  "Authorization",
									"value": "Bearer test-token",
								},
							},
							"api": `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /translate:
    post:
      operationId: translateText
      summary: Translate text
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required:
                - text
                - targetLang
              properties:
                text:
                  type: string
                targetLang:
                  type: string`,
						},
					},
					"promptTemplate": map[string]interface{}{
						"language": "EN",
					},
				})
				return data
			}()

			host, status := test.NewTestHost(postToolConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "翻译这段文字"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 模拟LLM响应，要求调用POST工具
			llmResponse := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"translateText\", \"action_input\": \"{\\\"text\\\": \\\"Hello\\\", \\\"targetLang\\\": \\\"zh\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理，这会触发toolsCall
			action := host.CallOnHttpResponseBody([]byte(llmResponse))

			// 应该返回ActionPause，因为需要等待工具调用结果
			require.Equal(t, types.ActionPause, action)

			// 模拟API工具调用的响应
			apiResponse := `{"translatedText": "你好", "sourceLang": "en", "targetLang": "zh"}`
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(apiResponse))

			// 模拟LLM对工具调用结果的响应（Final Answer）
			llmFinalResponse := `{
				"id": "chatcmpl-124",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Final Answer: Hello翻译成中文是：你好"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652289,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 15,
					"completion_tokens": 25,
					"total_tokens": 40
				}
			}`

			// 模拟LLM客户端的响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(llmFinalResponse))

			// 完成HTTP请求
			host.CompleteHttp()
		})

		t.Run("Final Answer response", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 模拟LLM响应，直接给出Final Answer
			llmResponse := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Final Answer: 今天北京天气晴朗，温度25度"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理，这会触发toolsCall
			action := host.CallOnHttpResponseBody([]byte(llmResponse))

			// 应该返回ActionContinue，因为得到了Final Answer
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("unknown tool name", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "调用一个工具"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 模拟LLM响应，要求调用未知工具
			llmResponse := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"unknownTool\", \"action_input\": \"{\\\"param\\\": \\\"value\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理，这会触发toolsCall
			action := host.CallOnHttpResponseBody([]byte(llmResponse))

			// 应该返回ActionContinue，因为工具名称未知
			require.Equal(t, types.ActionContinue, action)
		})

		t.Run("tool call with max iterations", func(t *testing.T) {
			// 创建一个设置最大迭代次数为2的配置
			maxIterConfig := func() json.RawMessage {
				data, _ := json.Marshal(map[string]interface{}{
					"returnResponseTemplate": `{"id":"error","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`,
					"llm": map[string]interface{}{
						"apiKey":        "test-api-key",
						"serviceName":   "llm-service",
						"servicePort":   8080,
						"domain":        "llm.example.com",
						"path":          "/v1/chat/completions",
						"model":         "qwen-turbo",
						"maxIterations": 2,
					},
					"apis": []map[string]interface{}{
						{
							"apiProvider": map[string]interface{}{
								"serviceName": "api-service",
								"servicePort": 9090,
								"domain":      "api.example.com",
								"apiKey": map[string]interface{}{
									"in":    "header",
									"name":  "Authorization",
									"value": "Bearer test-token",
								},
							},
							"api": `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /weather:
    get:
      operationId: getWeather
      summary: Get weather information
      parameters:
        - name: city
          in: query
          required: true
          schema:
            type: string`,
						},
					},
					"promptTemplate": map[string]interface{}{
						"language": "EN",
					},
				})
				return data
			}()

			host, status := test.NewTestHost(maxIterConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 第一次LLM响应，要求调用工具
			llmResponse1 := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": \"{\\\"city\\\": \\\"北京\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理，这会触发toolsCall
			action := host.CallOnHttpResponseBody([]byte(llmResponse1))

			// 应该返回ActionPause，因为需要等待工具调用结果
			require.Equal(t, types.ActionPause, action)

			// 模拟API工具调用的响应
			apiResponse := `{"temperature": 25, "condition": "晴朗"}`
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(apiResponse))

			// 第二次LLM响应，再次要求调用工具
			llmResponse2 := `{
				"id": "chatcmpl-124",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": \"{\\\"city\\\": \\\"上海\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652289,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 15,
					"completion_tokens": 25,
					"total_tokens": 40
				}
			}`

			// 模拟LLM客户端的响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(llmResponse2))

			// 第三次LLM响应，应该因为达到最大迭代次数而返回ActionContinue
			llmResponse3 := `{
				"id": "chatcmpl-125",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": \"{\\\"city\\\": \\\"广州\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652290,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 20,
					"completion_tokens": 30,
					"total_tokens": 50
				}
			}`

			// 模拟LLM客户端的响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(llmResponse3))

			// 完成HTTP请求
			host.CompleteHttp()
		})
	})
}

func TestEdgeCases(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("max iterations exceeded", func(t *testing.T) {
			// 创建一个设置最大迭代次数为1的配置
			maxIterConfig := func() json.RawMessage {
				data, _ := json.Marshal(map[string]interface{}{
					"returnResponseTemplate": `{"id":"error","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"finish_reason":"stop"}],"model":"gpt-4o","object":"chat.completion","usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`,
					"llm": map[string]interface{}{
						"apiKey":        "test-api-key",
						"serviceName":   "llm-service",
						"servicePort":   8080,
						"domain":        "llm.example.com",
						"path":          "/v1/chat/completions",
						"model":         "qwen-turbo",
						"maxIterations": 1,
					},
					"apis": []map[string]interface{}{
						{
							"apiProvider": map[string]interface{}{
								"serviceName": "api-service",
								"servicePort": 9090,
								"domain":      "api.example.com",
								"apiKey": map[string]interface{}{
									"in":    "header",
									"name":  "Authorization",
									"value": "Bearer test-token",
								},
							},
							"api": `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /weather:
    get:
      operationId: getWeather
      summary: Get weather information
      parameters:
        - name: city
          in: query
          required: true
          schema:
            type: string`,
						},
					},
					"promptTemplate": map[string]interface{}{
						"language": "EN",
					},
				})
				return data
			}()

			host, status := test.NewTestHost(maxIterConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 模拟LLM响应，要求调用工具
			llmResponse := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": \"{\\\"city\\\": \\\"北京\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理，这会触发toolsCall
			action := host.CallOnHttpResponseBody([]byte(llmResponse))

			// 应该返回ActionPause，因为需要等待工具调用结果
			require.Equal(t, types.ActionPause, action)

			// 模拟API工具调用的响应
			apiResponse := `{"temperature": 25, "condition": "晴朗"}`
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(apiResponse))

			// 模拟LLM对工具调用结果的响应，再次要求调用工具
			llmResponse2 := `{
				"id": "chatcmpl-124",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": \"{\\\"city\\\": \\\"上海\\\"}\"}"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652289,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 15,
					"completion_tokens": 25,
					"total_tokens": 40
				}
			}`

			// 模拟LLM客户端的响应
			host.CallOnHttpCall([][2]string{
				{"Content-Type", "application/json"},
				{":status", "200"},
			}, []byte(llmResponse2))

			// 完成HTTP请求
			host.CompleteHttp()
		})

		t.Run("invalid action input JSON", func(t *testing.T) {
			host, status := test.NewTestHost(httpTestConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 设置请求头
			host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/api/chat"},
				{":method", "POST"},
				{"content-type", "application/json"},
			})

			// 先调用请求体处理来初始化上下文
			requestBody := `{
				"model": "qwen-turbo",
				"messages": [
					{
						"role": "user",
						"content": "今天天气怎么样？"
					}
				],
				"stream": false
			}`
			host.CallOnHttpRequestBody([]byte(requestBody))

			// 模拟LLM响应，包含无效的Action Input JSON
			llmResponse := `{
				"id": "chatcmpl-123",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "{\"action\": \"getWeather\", \"action_input\": {invalid json"
						},
						"finish_reason": "stop"
					}
				],
				"created": 1677652288,
				"model": "qwen-turbo",
				"object": "chat.completion",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`

			// 调用响应体处理，这会触发toolsCall
			action := host.CallOnHttpResponseBody([]byte(llmResponse))

			// 应该返回ActionContinue，因为Action Input JSON无效
			require.Equal(t, types.ActionContinue, action)
		})
	})
}
