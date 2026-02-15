package provider

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Trace(msg string)                             {}
func (m *mockLogger) Tracef(format string, args ...interface{})    {}
func (m *mockLogger) Debug(msg string)                             {}
func (m *mockLogger) Debugf(format string, args ...interface{})    {}
func (m *mockLogger) Info(msg string)                              {}
func (m *mockLogger) Infof(format string, args ...interface{})     {}
func (m *mockLogger) Warn(msg string)                              {}
func (m *mockLogger) Warnf(format string, args ...interface{})     {}
func (m *mockLogger) Error(msg string)                             {}
func (m *mockLogger) Errorf(format string, args ...interface{})    {}
func (m *mockLogger) Critical(msg string)                          {}
func (m *mockLogger) Criticalf(format string, args ...interface{}) {}
func (m *mockLogger) ResetID(pluginID string)                      {}

func init() {
	// Initialize mock logger for testing
	log.SetPluginLog(&mockLogger{})
}

func TestClaudeToOpenAIConverter_ConvertClaudeRequestToOpenAI(t *testing.T) {
	converter := &ClaudeToOpenAIConverter{}

	t.Run("convert_multiple_text_content_blocks", func(t *testing.T) {
		// Test case: multiple text content blocks should remain as separate array elements with cache control support
		// Both system and user messages should handle array content format
		claudeRequest := `{
			"max_tokens": 32000,
			"messages": [{
				"content": [{
					"text": "<system-reminder>\nThis is a reminder that your todo list is currently empty. DO NOT mention this to the user explicitly because they are already aware. If you are working on tasks that would benefit from a todo list please use the TodoWrite tool to create one. If not, please feel free to ignore. Again do not mention this message to the user.</system-reminder>",
					"type": "text"
				}, {
					"text": "<system-reminder>\nyyy</system-reminder>",
					"type": "text"
				}, {
					"cache_control": {
						"type": "ephemeral"
					},
					"text": "你是谁",
					"type": "text"
				}],
				"role": "user"
			}],
			"metadata": {
				"user_id": "user_dd3c52c1d698a4486bdef490197846b7c1f7e553202dae5763f330c35aeb9823_account__session_b2e14122-0ac6-4959-9c5d-b49ae01ccb7c"
			},
			"model": "anthropic/claude-sonnet-4",
			"stream": true,
			"system": [{
				"cache_control": {
					"type": "ephemeral"
				},
				"text": "xxx",
				"type": "text"
			}, {
				"cache_control": {
					"type": "ephemeral"
				},
				"text": "yyy",
				"type": "text"
			}],
			"temperature": 1,
			"stream_options": {
				"include_usage": true
			}
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		// Parse the result to verify the conversion
		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		// Verify basic fields are converted correctly
		assert.Equal(t, "anthropic/claude-sonnet-4", openaiRequest.Model)
		assert.Equal(t, true, openaiRequest.Stream)
		assert.Equal(t, 1.0, openaiRequest.Temperature)
		assert.Equal(t, 32000, openaiRequest.MaxTokens)

		// Verify messages structure
		require.Len(t, openaiRequest.Messages, 2)

		// First message should be system message (converted from Claude's system field)
		systemMsg := openaiRequest.Messages[0]
		assert.Equal(t, roleSystem, systemMsg.Role)

		// System content should now also be an array for multiple text blocks
		systemContentArray, ok := systemMsg.Content.([]interface{})
		require.True(t, ok, "System content should be an array for multiple text blocks")
		require.Len(t, systemContentArray, 2)

		// First system text block
		firstSystemElement, ok := systemContentArray[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, contentTypeText, firstSystemElement["type"])
		assert.Equal(t, "xxx", firstSystemElement["text"])
		assert.NotNil(t, firstSystemElement["cache_control"]) // Has cache control
		systemCacheControl1, ok := firstSystemElement["cache_control"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "ephemeral", systemCacheControl1["type"])

		// Second system text block
		secondSystemElement, ok := systemContentArray[1].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, contentTypeText, secondSystemElement["type"])
		assert.Equal(t, "yyy", secondSystemElement["text"])
		assert.NotNil(t, secondSystemElement["cache_control"]) // Has cache control
		systemCacheControl2, ok := secondSystemElement["cache_control"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "ephemeral", systemCacheControl2["type"])

		// Second message should be user message with text content as array
		userMsg := openaiRequest.Messages[1]
		assert.Equal(t, "user", userMsg.Role)

		// The content should now be an array of separate text blocks, not merged
		contentArray, ok := userMsg.Content.([]interface{})
		require.True(t, ok, "Content should be an array for multiple text blocks")
		require.Len(t, contentArray, 3)

		// First text block
		firstElement, ok := contentArray[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, contentTypeText, firstElement["type"])
		assert.Equal(t, "<system-reminder>\nThis is a reminder that your todo list is currently empty. DO NOT mention this to the user explicitly because they are already aware. If you are working on tasks that would benefit from a todo list please use the TodoWrite tool to create one. If not, please feel free to ignore. Again do not mention this message to the user.</system-reminder>", firstElement["text"])
		assert.Nil(t, firstElement["cache_control"]) // No cache control for first block

		// Second text block
		secondElement, ok := contentArray[1].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, contentTypeText, secondElement["type"])
		assert.Equal(t, "<system-reminder>\nyyy</system-reminder>", secondElement["text"])
		assert.Nil(t, secondElement["cache_control"]) // No cache control for second block

		// Third text block with cache control
		thirdElement, ok := contentArray[2].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, contentTypeText, thirdElement["type"])
		assert.Equal(t, "你是谁", thirdElement["text"])
		assert.NotNil(t, thirdElement["cache_control"]) // Has cache control
		cacheControl, ok := thirdElement["cache_control"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "ephemeral", cacheControl["type"])
	})

	t.Run("convert_mixed_content_with_image", func(t *testing.T) {
		// Test case with mixed text and image content (should remain as array)
		claudeRequest := `{
			"model": "claude-3-sonnet-20240229",
			"messages": [{
				"role": "user",
				"content": [{
					"type": "text",
					"text": "What's in this image?"
				}, {
					"type": "image",
					"source": {
						"type": "base64",
						"media_type": "image/jpeg",
						"data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
					}
				}]
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		// Should have one user message
		require.Len(t, openaiRequest.Messages, 1)
		userMsg := openaiRequest.Messages[0]
		assert.Equal(t, "user", userMsg.Role)

		// Content should be an array (mixed content) - after JSON marshaling/unmarshaling it becomes []interface{}
		contentArray, ok := userMsg.Content.([]interface{})
		require.True(t, ok, "Content should be an array for mixed content")
		require.Len(t, contentArray, 2)

		// First element should be text
		firstElement, ok := contentArray[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, contentTypeText, firstElement["type"])
		assert.Equal(t, "What's in this image?", firstElement["text"])

		// Second element should be image
		secondElement, ok := contentArray[1].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, contentTypeImageUrl, secondElement["type"])
		assert.NotNil(t, secondElement["image_url"])
		imageUrl, ok := secondElement["image_url"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, imageUrl["url"], "data:image/jpeg;base64,")
	})

	t.Run("convert_simple_string_content", func(t *testing.T) {
		// Test case with simple string content
		claudeRequest := `{
			"model": "claude-3-sonnet-20240229",
			"messages": [{
				"role": "user",
				"content": "Hello, how are you?"
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		require.Len(t, openaiRequest.Messages, 1)
		userMsg := openaiRequest.Messages[0]
		assert.Equal(t, "user", userMsg.Role)
		assert.Equal(t, "Hello, how are you?", userMsg.Content)
	})

	t.Run("convert_empty_content_array", func(t *testing.T) {
		// Test case with empty content array
		claudeRequest := `{
			"model": "claude-3-sonnet-20240229",
			"messages": [{
				"role": "user",
				"content": []
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		require.Len(t, openaiRequest.Messages, 1)
		userMsg := openaiRequest.Messages[0]
		assert.Equal(t, "user", userMsg.Role)

		// Empty array should result in empty array, not string - after JSON marshaling/unmarshaling becomes []interface{}
		if userMsg.Content != nil {
			contentArray, ok := userMsg.Content.([]interface{})
			require.True(t, ok, "Empty content should be an array")
			assert.Empty(t, contentArray)
		} else {
			// null is also acceptable for empty content
			assert.Nil(t, userMsg.Content)
		}
	})

	t.Run("convert_tool_use_to_tool_calls", func(t *testing.T) {
		// Test Claude tool_use conversion to OpenAI tool_calls format
		claudeRequest := `{
			"model": "anthropic/claude-sonnet-4",
			"messages": [{
				"role": "assistant",
				"content": [{
					"type": "text",
					"text": "I'll help you search for information."
				}, {
					"type": "tool_use",
					"id": "toolu_01D7FLrfh4GYq7yT1ULFeyMV",
					"name": "web_search",
					"input": {
						"query": "Claude AI capabilities",
						"max_results": 5
					}
				}]
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		// Should have one assistant message with tool_calls
		require.Len(t, openaiRequest.Messages, 1)
		assistantMsg := openaiRequest.Messages[0]
		assert.Equal(t, "assistant", assistantMsg.Role)
		assert.Equal(t, "I'll help you search for information.", assistantMsg.Content)

		// Verify tool_calls format
		require.NotNil(t, assistantMsg.ToolCalls)
		require.Len(t, assistantMsg.ToolCalls, 1)

		toolCall := assistantMsg.ToolCalls[0]
		assert.Equal(t, "toolu_01D7FLrfh4GYq7yT1ULFeyMV", toolCall.Id)
		assert.Equal(t, "function", toolCall.Type)
		assert.Equal(t, "web_search", toolCall.Function.Name)

		// Verify arguments are properly JSON encoded
		var args map[string]interface{}
		err = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		require.NoError(t, err)
		assert.Equal(t, "Claude AI capabilities", args["query"])
		assert.Equal(t, float64(5), args["max_results"])
	})

	t.Run("convert_tool_result_to_tool_message", func(t *testing.T) {
		// Test Claude tool_result conversion to OpenAI tool message format
		claudeRequest := `{
			"model": "anthropic/claude-sonnet-4",
			"messages": [{
				"role": "user",
				"content": [{
					"type": "tool_result",
					"tool_use_id": "toolu_01D7FLrfh4GYq7yT1ULFeyMV",
					"content": "Search results: Claude is an AI assistant created by Anthropic."
				}]
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		// Should have one tool message
		require.Len(t, openaiRequest.Messages, 1)
		toolMsg := openaiRequest.Messages[0]
		assert.Equal(t, "tool", toolMsg.Role)
		assert.Equal(t, "Search results: Claude is an AI assistant created by Anthropic.", toolMsg.Content)
		assert.Equal(t, "toolu_01D7FLrfh4GYq7yT1ULFeyMV", toolMsg.ToolCallId)
	})

	t.Run("convert_tool_result_with_array_content", func(t *testing.T) {
		// Test Claude tool_result with array content format (new format that was causing the error)
		claudeRequest := `{
			"model": "anthropic/claude-sonnet-4",
			"messages": [{
				"role": "user",
				"content": [{
					"type": "tool_result",
					"tool_use_id": "toolu_vrtx_01UbCfwoTgoDBqbYEwkVaxd5",
					"content": [{
						"text": "Search results for three.js libraries and frameworks",
						"type": "text"
					}]
				}]
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		// Should have one tool message
		require.Len(t, openaiRequest.Messages, 1)
		toolMsg := openaiRequest.Messages[0]

		assert.Equal(t, "tool", toolMsg.Role)
		assert.Equal(t, "Search results for three.js libraries and frameworks", toolMsg.Content)
		assert.Equal(t, "toolu_vrtx_01UbCfwoTgoDBqbYEwkVaxd5", toolMsg.ToolCallId)
	})

	t.Run("convert_tool_result_with_actual_error_data", func(t *testing.T) {
		// Test using the actual JSON data from the error log to ensure our fix works
		// This tests the fix for issue #3344 - text content alongside tool_result should be preserved
		claudeRequest := `{
			"model": "anthropic/claude-sonnet-4", 
			"messages": [{
				"role": "user",
				"content": [{
					"content": [{
						"text": "\n  ## 结果 1\n  - **id**: /websites/threejs\n  - **title**: three.js\n  - **description**: three.js is a JavaScript 3D library that makes it easy to create and display animated 3D computer graphics in a web browser. It provides a powerful and flexible way to build interactive 3D experiences.\n",
						"type": "text"
					}],
					"tool_use_id": "toolu_vrtx_01UbCfwoTgoDBqbYEwkVaxd5",
					"type": "tool_result"
				}, {
					"cache_control": {"type": "ephemeral"},
					"text": "继续",
					"type": "text"
				}]
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		// Should have two messages: tool message + user message with text content
		// This is the fix for issue #3344 - text content alongside tool_result is preserved
		require.Len(t, openaiRequest.Messages, 2)

		// First should be tool message
		toolMsg := openaiRequest.Messages[0]
		assert.Equal(t, "tool", toolMsg.Role)
		assert.Contains(t, toolMsg.Content, "three.js")
		assert.Equal(t, "toolu_vrtx_01UbCfwoTgoDBqbYEwkVaxd5", toolMsg.ToolCallId)

		// Second should be user message with text content
		userMsg := openaiRequest.Messages[1]
		assert.Equal(t, "user", userMsg.Role)
		assert.Equal(t, "继续", userMsg.Content)
	})

	t.Run("convert_multiple_tool_calls", func(t *testing.T) {
		// Test multiple tool_use in single message
		claudeRequest := `{
			"model": "anthropic/claude-sonnet-4", 
			"messages": [{
				"role": "assistant",
				"content": [{
					"type": "tool_use",
					"id": "toolu_search",
					"name": "web_search",
					"input": {"query": "weather"}
				}, {
					"type": "tool_use", 
					"id": "toolu_calc",
					"name": "calculate",
					"input": {"expression": "2+2"}
				}]
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		// Should have one assistant message with multiple tool_calls
		require.Len(t, openaiRequest.Messages, 1)
		assistantMsg := openaiRequest.Messages[0]
		assert.Equal(t, "assistant", assistantMsg.Role)
		assert.Nil(t, assistantMsg.Content) // No text content, so should be null

		// Verify multiple tool_calls
		require.NotNil(t, assistantMsg.ToolCalls)
		require.Len(t, assistantMsg.ToolCalls, 2)

		// First tool call
		assert.Equal(t, "toolu_search", assistantMsg.ToolCalls[0].Id)
		assert.Equal(t, "web_search", assistantMsg.ToolCalls[0].Function.Name)

		// Second tool call
		assert.Equal(t, "toolu_calc", assistantMsg.ToolCalls[1].Id)
		assert.Equal(t, "calculate", assistantMsg.ToolCalls[1].Function.Name)
	})

	t.Run("convert_multiple_tool_results", func(t *testing.T) {
		// Test multiple tool_result messages
		claudeRequest := `{
			"model": "anthropic/claude-sonnet-4",
			"messages": [{
				"role": "user",
				"content": [{
					"type": "tool_result",
					"tool_use_id": "toolu_search",
					"content": "Weather: 25°C sunny"
				}, {
					"type": "tool_result",
					"tool_use_id": "toolu_calc", 
					"content": "Result: 4"
				}]
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		// Should have two tool messages
		require.Len(t, openaiRequest.Messages, 2)

		// First tool result
		toolMsg1 := openaiRequest.Messages[0]
		assert.Equal(t, "tool", toolMsg1.Role)
		assert.Equal(t, "Weather: 25°C sunny", toolMsg1.Content)
		assert.Equal(t, "toolu_search", toolMsg1.ToolCallId)

		// Second tool result
		toolMsg2 := openaiRequest.Messages[1]
		assert.Equal(t, "tool", toolMsg2.Role)
		assert.Equal(t, "Result: 4", toolMsg2.Content)
		assert.Equal(t, "toolu_calc", toolMsg2.ToolCallId)
	})

	t.Run("convert_mixed_text_and_tool_use", func(t *testing.T) {
		// Test message with both text and tool_use
		claudeRequest := `{
			"model": "anthropic/claude-sonnet-4",
			"messages": [{
				"role": "assistant",
				"content": [{
					"type": "text",
					"text": "Let me search for that information and do a calculation."
				}, {
					"type": "tool_use",
					"id": "toolu_search123",
					"name": "search_database",
					"input": {"table": "users", "limit": 10}
				}]
			}],
			"max_tokens": 1000
		}`

		result, err := converter.ConvertClaudeRequestToOpenAI([]byte(claudeRequest))
		require.NoError(t, err)

		var openaiRequest chatCompletionRequest
		err = json.Unmarshal(result, &openaiRequest)
		require.NoError(t, err)

		// Should have one assistant message with both content and tool_calls
		require.Len(t, openaiRequest.Messages, 1)
		assistantMsg := openaiRequest.Messages[0]
		assert.Equal(t, "assistant", assistantMsg.Role)
		assert.Equal(t, "Let me search for that information and do a calculation.", assistantMsg.Content)

		// Should have tool_calls
		require.NotNil(t, assistantMsg.ToolCalls)
		require.Len(t, assistantMsg.ToolCalls, 1)
		assert.Equal(t, "toolu_search123", assistantMsg.ToolCalls[0].Id)
		assert.Equal(t, "search_database", assistantMsg.ToolCalls[0].Function.Name)
	})
}

func TestClaudeToOpenAIConverter_ConvertOpenAIResponseToClaude(t *testing.T) {
	converter := &ClaudeToOpenAIConverter{}

	t.Run("convert_tool_calls_response", func(t *testing.T) {
		// Test OpenAI response with tool calls conversion to Claude format
		openaiResponse := `{
			"id": "gen-1756214072-tVFkPBV6lxee00IqNAC5",
			"provider": "Google",
			"model": "anthropic/claude-sonnet-4", 
			"object": "chat.completion",
			"created": 1756214072,
			"choices": [{
				"logprobs": null,
				"finish_reason": "tool_calls",
				"native_finish_reason": "tool_calls",
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "I'll analyze the README file to understand this project's purpose.",
					"refusal": null,
					"reasoning": null,
					"tool_calls": [{
						"id": "toolu_vrtx_017ijjgx8hpigatPzzPW59Wq",
						"index": 0,
						"type": "function",
						"function": {
							"name": "Read",
							"arguments": "{\"file_path\": \"/Users/zhangty/git/higress/README.md\"}"
						}
					}]
				}
			}],
			"usage": {
				"prompt_tokens": 14923,
				"completion_tokens": 81,
				"total_tokens": 15004
			}
		}`

		result, err := converter.ConvertOpenAIResponseToClaude(nil, []byte(openaiResponse))
		require.NoError(t, err)

		var claudeResponse claudeTextGenResponse
		err = json.Unmarshal(result, &claudeResponse)
		require.NoError(t, err)

		// Verify basic response fields
		assert.Equal(t, "gen-1756214072-tVFkPBV6lxee00IqNAC5", claudeResponse.Id)
		assert.Equal(t, "message", claudeResponse.Type)
		assert.Equal(t, "assistant", claudeResponse.Role)
		assert.Equal(t, "anthropic/claude-sonnet-4", claudeResponse.Model)
		assert.Equal(t, "tool_use", *claudeResponse.StopReason)

		// Verify usage
		assert.Equal(t, 14923, claudeResponse.Usage.InputTokens)
		assert.Equal(t, 81, claudeResponse.Usage.OutputTokens)

		// Verify content array has both text and tool_use
		require.Len(t, claudeResponse.Content, 2)

		// First content should be text
		textContent := claudeResponse.Content[0]
		assert.Equal(t, "text", textContent.Type)
		assert.Equal(t, "I'll analyze the README file to understand this project's purpose.", textContent.Text)

		// Second content should be tool_use
		toolContent := claudeResponse.Content[1]
		assert.Equal(t, "tool_use", toolContent.Type)
		assert.Equal(t, "toolu_vrtx_017ijjgx8hpigatPzzPW59Wq", toolContent.Id)
		assert.Equal(t, "Read", toolContent.Name)

		// Verify tool arguments
		require.NotNil(t, toolContent.Input)
		assert.Equal(t, "/Users/zhangty/git/higress/README.md", (*toolContent.Input)["file_path"])
	})
}

func TestClaudeToOpenAIConverter_ConvertThinkingConfig(t *testing.T) {
	converter := &ClaudeToOpenAIConverter{}

	tests := []struct {
		name                 string
		claudeRequest        string
		expectedMaxTokens    int
		expectedEffort       string
		expectThinkingConfig bool
	}{
		{
			name: "thinking_enabled_low",
			claudeRequest: `{
				"model": "claude-sonnet-4",
				"max_tokens": 1000,
				"messages": [{"role": "user", "content": "Hello"}],
				"thinking": {"type": "enabled", "budget_tokens": 2048}
			}`,
			expectedMaxTokens:    2048,
			expectedEffort:       "low",
			expectThinkingConfig: true,
		},
		{
			name: "thinking_enabled_medium",
			claudeRequest: `{
				"model": "claude-sonnet-4",
				"max_tokens": 1000,
				"messages": [{"role": "user", "content": "Hello"}],
				"thinking": {"type": "enabled", "budget_tokens": 8192}
			}`,
			expectedMaxTokens:    8192,
			expectedEffort:       "medium",
			expectThinkingConfig: true,
		},
		{
			name: "thinking_enabled_high",
			claudeRequest: `{
				"model": "claude-sonnet-4",
				"max_tokens": 1000,
				"messages": [{"role": "user", "content": "Hello"}],
				"thinking": {"type": "enabled", "budget_tokens": 20480}
			}`,
			expectedMaxTokens:    20480,
			expectedEffort:       "high",
			expectThinkingConfig: true,
		},
		{
			name: "thinking_disabled",
			claudeRequest: `{
				"model": "claude-sonnet-4",
				"max_tokens": 1000,
				"messages": [{"role": "user", "content": "Hello"}],
				"thinking": {"type": "disabled"}
			}`,
			expectedMaxTokens:    0,
			expectedEffort:       "",
			expectThinkingConfig: false,
		},
		{
			name: "no_thinking",
			claudeRequest: `{
				"model": "claude-sonnet-4",
				"max_tokens": 1000,
				"messages": [{"role": "user", "content": "Hello"}]
			}`,
			expectedMaxTokens:    0,
			expectedEffort:       "",
			expectThinkingConfig: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ConvertClaudeRequestToOpenAI([]byte(tt.claudeRequest))
			assert.NoError(t, err)
			assert.NotNil(t, result)

			var openaiRequest chatCompletionRequest
			err = json.Unmarshal(result, &openaiRequest)
			assert.NoError(t, err)

			if tt.expectThinkingConfig {
				assert.Equal(t, tt.expectedMaxTokens, openaiRequest.ReasoningMaxTokens)
				assert.Equal(t, tt.expectedEffort, openaiRequest.ReasoningEffort)
			} else {
				assert.Equal(t, 0, openaiRequest.ReasoningMaxTokens)
				assert.Equal(t, "", openaiRequest.ReasoningEffort)
			}
		})
	}
}

func TestClaudeToOpenAIConverter_ConvertReasoningResponseToClaude(t *testing.T) {
	converter := &ClaudeToOpenAIConverter{}

	tests := []struct {
		name           string
		openaiResponse string
		expectThinking bool
		expectedText   string
	}{
		{
			name: "response_with_reasoning_content",
			openaiResponse: `{
				"id": "chatcmpl-test123",
				"object": "chat.completion",
				"created": 1699999999,
				"model": "gpt-4o",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Based on my analysis, the answer is 42.",
						"reasoning_content": "Let me think about this step by step:\n1. The question asks about the meaning of life\n2. According to Douglas Adams, the answer is 42\n3. Therefore, 42 is the correct answer"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`,
			expectThinking: true,
			expectedText:   "Based on my analysis, the answer is 42.",
		},
		{
			name: "response_with_reasoning_field",
			openaiResponse: `{
				"id": "chatcmpl-test789",
				"object": "chat.completion",
				"created": 1699999999,
				"model": "gpt-4o",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Based on my analysis, the answer is 42.",
						"reasoning": "Let me think about this step by step:\n1. The question asks about the meaning of life\n2. According to Douglas Adams, the answer is 42\n3. Therefore, 42 is the correct answer"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`,
			expectThinking: true,
			expectedText:   "Based on my analysis, the answer is 42.",
		},
		{
			name: "response_without_reasoning_content",
			openaiResponse: `{
				"id": "chatcmpl-test456",
				"object": "chat.completion",
				"created": 1699999999,
				"model": "gpt-4o",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "The answer is 42."
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 5,
					"completion_tokens": 10,
					"total_tokens": 15
				}
			}`,
			expectThinking: false,
			expectedText:   "The answer is 42.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ConvertOpenAIResponseToClaude(nil, []byte(tt.openaiResponse))
			assert.NoError(t, err)
			assert.NotNil(t, result)

			var claudeResponse claudeTextGenResponse
			err = json.Unmarshal(result, &claudeResponse)
			assert.NoError(t, err)

			// Verify response structure
			assert.Equal(t, "message", claudeResponse.Type)
			assert.Equal(t, "assistant", claudeResponse.Role)
			assert.NotEmpty(t, claudeResponse.Id) // ID should be present

			if tt.expectThinking {
				// Should have both thinking and text content
				assert.Len(t, claudeResponse.Content, 2)

				// First should be thinking
				thinkingContent := claudeResponse.Content[0]
				assert.Equal(t, "thinking", thinkingContent.Type)
				assert.Equal(t, "", thinkingContent.Signature) // OpenAI doesn't provide signature
				assert.Contains(t, thinkingContent.Thinking, "Let me think about this step by step")

				// Second should be text
				textContent := claudeResponse.Content[1]
				assert.Equal(t, "text", textContent.Type)
				assert.Equal(t, tt.expectedText, textContent.Text)
			} else {
				// Should only have text content
				assert.Len(t, claudeResponse.Content, 1)

				textContent := claudeResponse.Content[0]
				assert.Equal(t, "text", textContent.Type)
				assert.Equal(t, tt.expectedText, textContent.Text)
			}
		})
	}
}
