package test

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// Test config: Basic Bedrock config with AWS Access Key/Secret Key (AWS Signature V4)
var basicBedrockConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":         "bedrock",
			"awsAccessKey": "test-ak-for-unit-test",
			"awsSecretKey": "test-sk-for-unit-test",
			"awsRegion":    "us-east-1",
			"modelMapping": map[string]string{
				"*": "anthropic.claude-3-5-haiku-20241022-v1:0",
			},
		},
	})
	return data
}()

// Test config: Bedrock config with Bearer Token authentication
var bedrockApiTokenConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "bedrock",
			"apiTokens": []string{
				"test-token-for-unit-test",
			},
			"awsRegion": "us-east-1",
			"modelMapping": map[string]string{
				"*": "anthropic.claude-3-5-haiku-20241022-v1:0",
			},
		},
	})
	return data
}()

// Test config: Bedrock config with multiple Bearer Tokens
var bedrockMultiTokenConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "bedrock",
			"apiTokens": []string{
				"test-token-1-for-unit-test",
				"test-token-2-for-unit-test",
			},
			"awsRegion": "us-west-2",
			"modelMapping": map[string]string{
				"gpt-4": "anthropic.claude-3-opus-20240229-v1:0",
				"*":     "anthropic.claude-3-haiku-20240307-v1:0",
			},
		},
	})
	return data
}()

// Test config: Bedrock config with additional fields
var bedrockWithAdditionalFieldsConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":         "bedrock",
			"awsAccessKey": "test-ak-for-unit-test",
			"awsSecretKey": "test-sk-for-unit-test",
			"awsRegion":    "us-east-1",
			"bedrockAdditionalFields": map[string]interface{}{
				"top_k": 200,
			},
			"modelMapping": map[string]string{
				"*": "anthropic.claude-3-5-haiku-20241022-v1:0",
			},
		},
	})
	return data
}()

// Test config: Invalid config - missing both apiTokens and ak/sk
var bedrockInvalidConfigMissingAuth = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":      "bedrock",
			"awsRegion": "us-east-1",
			"modelMapping": map[string]string{
				"*": "anthropic.claude-3-5-haiku-20241022-v1:0",
			},
		},
	})
	return data
}()

// Test config: Invalid config - missing region
var bedrockInvalidConfigMissingRegion = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type": "bedrock",
			"apiTokens": []string{
				"test-token-for-unit-test",
			},
			"modelMapping": map[string]string{
				"*": "anthropic.claude-3-5-haiku-20241022-v1:0",
			},
		},
	})
	return data
}()

// Test config: Invalid config - only has access key without secret key
var bedrockInvalidConfigPartialAkSk = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"provider": map[string]interface{}{
			"type":         "bedrock",
			"awsAccessKey": "test-ak-for-unit-test",
			"awsRegion":    "us-east-1",
			"modelMapping": map[string]string{
				"*": "anthropic.claude-3-5-haiku-20241022-v1:0",
			},
		},
	})
	return data
}()

func RunBedrockParseConfigTests(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// Test basic Bedrock config with AWS Signature V4 authentication
		t.Run("basic bedrock config with ak/sk", func(t *testing.T) {
			host, status := test.NewTestHost(basicBedrockConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// Test Bedrock config with Bearer Token authentication
		t.Run("bedrock config with api token", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockApiTokenConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// Test Bedrock config with multiple tokens
		t.Run("bedrock config with multiple tokens", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockMultiTokenConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// Test Bedrock config with additional fields
		t.Run("bedrock config with additional fields", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockWithAdditionalFieldsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})

		// Test invalid config - missing authentication
		t.Run("bedrock invalid config missing auth", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockInvalidConfigMissingAuth)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// Test invalid config - missing region
		t.Run("bedrock invalid config missing region", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockInvalidConfigMissingRegion)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})

		// Test invalid config - partial ak/sk (only access key, no secret key)
		t.Run("bedrock invalid config partial ak/sk", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockInvalidConfigPartialAkSk)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusFailed, status)
		})
	})
}

func RunBedrockOnHttpRequestHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// Test Bedrock request headers processing with AWS Signature V4
		t.Run("bedrock chat completion request headers with ak/sk", func(t *testing.T) {
			host, status := test.NewTestHost(basicBedrockConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// Set request headers
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.HeaderStopIteration, action)

			// Verify request headers
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// Verify Host is changed to Bedrock service domain
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Contains(t, hostValue, "bedrock-runtime.us-east-1.amazonaws.com", "Host should be changed to Bedrock service domain")
		})

		// Test Bedrock request headers processing with Bearer Token
		t.Run("bedrock chat completion request headers with api token", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockApiTokenConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// Set request headers
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})

			require.Equal(t, types.HeaderStopIteration, action)

			// Verify request headers
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// Verify Host is changed to Bedrock service domain
			hostValue, hasHost := test.GetHeaderValue(requestHeaders, ":authority")
			require.True(t, hasHost, "Host header should exist")
			require.Contains(t, hostValue, "bedrock-runtime.us-east-1.amazonaws.com", "Host should be changed to Bedrock service domain")
		})
	})
}

func RunBedrockOnHttpRequestBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// Test Bedrock request body processing with Bearer Token authentication
		t.Run("bedrock chat completion request body with api token", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockApiTokenConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// Set request headers
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// Set request body
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{
						"role": "user",
						"content": "Hello, how are you?"
					}
				],
				"temperature": 0.7
			}`

			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// Verify request headers for Bearer Token authentication
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// Verify Authorization header uses Bearer token
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist")
			require.Contains(t, authValue, "Bearer ", "Authorization should use Bearer token")
			require.Contains(t, authValue, "test-token-for-unit-test", "Authorization should contain the configured token")

			// Verify path is transformed to Bedrock format
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/model/", "Path should contain Bedrock model path")
			require.Contains(t, pathValue, "/converse", "Path should contain converse endpoint")
		})

		// Test Bedrock request body processing with AWS Signature V4 authentication
		t.Run("bedrock chat completion request body with ak/sk", func(t *testing.T) {
			host, status := test.NewTestHost(basicBedrockConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// Set request headers
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// Set request body
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{
						"role": "user",
						"content": "Hello, how are you?"
					}
				],
				"temperature": 0.7
			}`

			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// Verify request headers for AWS Signature V4 authentication
			requestHeaders := host.GetRequestHeaders()
			require.NotNil(t, requestHeaders)

			// Verify Authorization header uses AWS Signature
			authValue, hasAuth := test.GetHeaderValue(requestHeaders, "Authorization")
			require.True(t, hasAuth, "Authorization header should exist")
			require.Contains(t, authValue, "AWS4-HMAC-SHA256", "Authorization should use AWS4-HMAC-SHA256 signature")
			require.Contains(t, authValue, "Credential=", "Authorization should contain Credential")
			require.Contains(t, authValue, "Signature=", "Authorization should contain Signature")

			// Verify X-Amz-Date header exists
			dateValue, hasDate := test.GetHeaderValue(requestHeaders, "X-Amz-Date")
			require.True(t, hasDate, "X-Amz-Date header should exist for AWS Signature V4")
			require.NotEmpty(t, dateValue, "X-Amz-Date should not be empty")

			// Verify path is transformed to Bedrock format
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/model/", "Path should contain Bedrock model path")
			require.Contains(t, pathValue, "/converse", "Path should contain converse endpoint")
		})

		// Test Bedrock streaming request
		t.Run("bedrock streaming request", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockApiTokenConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// Set request headers
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// Set streaming request body
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{
						"role": "user",
						"content": "Hello"
					}
				],
				"stream": true
			}`

			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// Verify path is transformed to Bedrock streaming format
			requestHeaders := host.GetRequestHeaders()
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/model/", "Path should contain Bedrock model path")
			require.Contains(t, pathValue, "/converse-stream", "Path should contain converse-stream endpoint for streaming")
		})
	})
}

func RunBedrockOnHttpResponseHeadersTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// Test Bedrock response headers processing
		t.Run("bedrock response headers", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockApiTokenConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// Set request headers
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// Set request body
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{
						"role": "user",
						"content": "Hello"
					}
				]
			}`
			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// Process response headers
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
				{"X-Amzn-Requestid", "test-request-id-12345"},
			})
			require.Equal(t, types.ActionContinue, action)

			// Verify response headers
			responseHeaders := host.GetResponseHeaders()
			require.NotNil(t, responseHeaders)

			// Verify status code
			statusValue, hasStatus := test.GetHeaderValue(responseHeaders, ":status")
			require.True(t, hasStatus, "Status header should exist")
			require.Equal(t, "200", statusValue, "Status should be 200")
		})
	})
}

func RunBedrockOnHttpResponseBodyTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// Test Bedrock response body processing
		t.Run("bedrock response body", func(t *testing.T) {
			host, status := test.NewTestHost(bedrockApiTokenConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// Set request headers
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// Set request body
			requestBody := `{
				"model": "gpt-4",
				"messages": [
					{
						"role": "user",
						"content": "Hello"
					}
				]
			}`
			action = host.CallOnHttpRequestBody([]byte(requestBody))
			require.Equal(t, types.ActionContinue, action)

			// Set response property to ensure IsResponseFromUpstream() returns true
			host.SetProperty([]string{"response", "code_details"}, []byte("via_upstream"))

			// Process response headers (must include :status 200 for body processing)
			action = host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"Content-Type", "application/json"},
			})
			require.Equal(t, types.ActionContinue, action)

			// Process response body (Bedrock format)
			responseBody := `{
				"output": {
					"message": {
						"role": "assistant",
						"content": [
							{
								"text": "Hello! How can I help you today?"
							}
						]
					}
				},
				"stopReason": "end_turn",
				"usage": {
					"inputTokens": 10,
					"outputTokens": 15,
					"totalTokens": 25
				}
			}`

			action = host.CallOnHttpResponseBody([]byte(responseBody))
			require.Equal(t, types.ActionContinue, action)

			// Verify response body is transformed to OpenAI format
			transformedResponseBody := host.GetResponseBody()
			require.NotNil(t, transformedResponseBody)

			var responseMap map[string]interface{}
			err := json.Unmarshal(transformedResponseBody, &responseMap)
			require.NoError(t, err)

			// Verify choices exist in transformed response
			choices, exists := responseMap["choices"]
			require.True(t, exists, "Choices should exist in response body")
			require.NotNil(t, choices, "Choices should not be nil")

			// Verify usage exists
			usage, exists := responseMap["usage"]
			require.True(t, exists, "Usage should exist in response body")
			require.NotNil(t, usage, "Usage should not be nil")
		})
	})
}
