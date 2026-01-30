package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupContextMessages(t *testing.T) {
	t.Run("empty_cleanup_commands", func(t *testing.T) {
		body := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)
		result, err := cleanupContextMessages(body, []string{})
		assert.NoError(t, err)
		assert.Equal(t, body, result)
	})

	t.Run("no_matching_command", func(t *testing.T) {
		body := []byte(`{"messages":[{"role":"system","content":"你是助手"},{"role":"user","content":"hello"}]}`)
		result, err := cleanupContextMessages(body, []string{"清理上下文", "/clear"})
		assert.NoError(t, err)
		assert.Equal(t, body, result)
	})

	t.Run("cleanup_with_single_command", func(t *testing.T) {
		input := chatCompletionRequest{
			Messages: []chatMessage{
				{Role: "system", Content: "你是一个助手"},
				{Role: "user", Content: "你好"},
				{Role: "assistant", Content: "你好！"},
				{Role: "user", Content: "清理上下文"},
				{Role: "user", Content: "新问题"},
			},
		}
		body, err := json.Marshal(input)
		require.NoError(t, err)

		result, err := cleanupContextMessages(body, []string{"清理上下文"})
		assert.NoError(t, err)

		var output chatCompletionRequest
		err = json.Unmarshal(result, &output)
		require.NoError(t, err)

		assert.Len(t, output.Messages, 2)
		assert.Equal(t, "system", output.Messages[0].Role)
		assert.Equal(t, "你是一个助手", output.Messages[0].Content)
		assert.Equal(t, "user", output.Messages[1].Role)
		assert.Equal(t, "新问题", output.Messages[1].Content)
	})

	t.Run("cleanup_with_multiple_commands_match_first", func(t *testing.T) {
		input := chatCompletionRequest{
			Messages: []chatMessage{
				{Role: "system", Content: "你是一个助手"},
				{Role: "user", Content: "你好"},
				{Role: "assistant", Content: "你好！"},
				{Role: "user", Content: "/clear"},
				{Role: "user", Content: "新问题"},
			},
		}
		body, err := json.Marshal(input)
		require.NoError(t, err)

		result, err := cleanupContextMessages(body, []string{"清理上下文", "/clear", "重新开始"})
		assert.NoError(t, err)

		var output chatCompletionRequest
		err = json.Unmarshal(result, &output)
		require.NoError(t, err)

		assert.Len(t, output.Messages, 2)
		assert.Equal(t, "system", output.Messages[0].Role)
		assert.Equal(t, "user", output.Messages[1].Role)
		assert.Equal(t, "新问题", output.Messages[1].Content)
	})

	t.Run("cleanup_removes_tool_messages", func(t *testing.T) {
		input := chatCompletionRequest{
			Messages: []chatMessage{
				{Role: "system", Content: "你是一个助手"},
				{Role: "user", Content: "查天气"},
				{Role: "assistant", Content: ""},
				{Role: "tool", Content: "北京 25°C"},
				{Role: "assistant", Content: "北京今天25度"},
				{Role: "user", Content: "清理上下文"},
				{Role: "user", Content: "新问题"},
			},
		}
		body, err := json.Marshal(input)
		require.NoError(t, err)

		result, err := cleanupContextMessages(body, []string{"清理上下文"})
		assert.NoError(t, err)

		var output chatCompletionRequest
		err = json.Unmarshal(result, &output)
		require.NoError(t, err)

		assert.Len(t, output.Messages, 2)
		assert.Equal(t, "system", output.Messages[0].Role)
		assert.Equal(t, "user", output.Messages[1].Role)
	})

	t.Run("cleanup_keeps_multiple_system_messages", func(t *testing.T) {
		input := chatCompletionRequest{
			Messages: []chatMessage{
				{Role: "system", Content: "系统提示1"},
				{Role: "system", Content: "系统提示2"},
				{Role: "user", Content: "你好"},
				{Role: "assistant", Content: "你好！"},
				{Role: "user", Content: "清理上下文"},
				{Role: "user", Content: "新问题"},
			},
		}
		body, err := json.Marshal(input)
		require.NoError(t, err)

		result, err := cleanupContextMessages(body, []string{"清理上下文"})
		assert.NoError(t, err)

		var output chatCompletionRequest
		err = json.Unmarshal(result, &output)
		require.NoError(t, err)

		assert.Len(t, output.Messages, 3)
		assert.Equal(t, "system", output.Messages[0].Role)
		assert.Equal(t, "系统提示1", output.Messages[0].Content)
		assert.Equal(t, "system", output.Messages[1].Role)
		assert.Equal(t, "系统提示2", output.Messages[1].Content)
		assert.Equal(t, "user", output.Messages[2].Role)
	})

	t.Run("cleanup_finds_last_matching_command", func(t *testing.T) {
		input := chatCompletionRequest{
			Messages: []chatMessage{
				{Role: "system", Content: "你是一个助手"},
				{Role: "user", Content: "清理上下文"},
				{Role: "user", Content: "中间问题"},
				{Role: "assistant", Content: "中间回答"},
				{Role: "user", Content: "清理上下文"},
				{Role: "user", Content: "最后问题"},
			},
		}
		body, err := json.Marshal(input)
		require.NoError(t, err)

		result, err := cleanupContextMessages(body, []string{"清理上下文"})
		assert.NoError(t, err)

		var output chatCompletionRequest
		err = json.Unmarshal(result, &output)
		require.NoError(t, err)

		// 应该匹配最后一个清理命令，保留 system 和 "最后问题"
		assert.Len(t, output.Messages, 2)
		assert.Equal(t, "system", output.Messages[0].Role)
		assert.Equal(t, "user", output.Messages[1].Role)
		assert.Equal(t, "最后问题", output.Messages[1].Content)
	})

	t.Run("cleanup_at_end_of_messages", func(t *testing.T) {
		input := chatCompletionRequest{
			Messages: []chatMessage{
				{Role: "system", Content: "你是一个助手"},
				{Role: "user", Content: "你好"},
				{Role: "assistant", Content: "你好！"},
				{Role: "user", Content: "清理上下文"},
			},
		}
		body, err := json.Marshal(input)
		require.NoError(t, err)

		result, err := cleanupContextMessages(body, []string{"清理上下文"})
		assert.NoError(t, err)

		var output chatCompletionRequest
		err = json.Unmarshal(result, &output)
		require.NoError(t, err)

		// 清理命令在最后，只保留 system
		assert.Len(t, output.Messages, 1)
		assert.Equal(t, "system", output.Messages[0].Role)
	})

	t.Run("cleanup_without_system_message", func(t *testing.T) {
		input := chatCompletionRequest{
			Messages: []chatMessage{
				{Role: "user", Content: "你好"},
				{Role: "assistant", Content: "你好！"},
				{Role: "user", Content: "清理上下文"},
				{Role: "user", Content: "新问题"},
			},
		}
		body, err := json.Marshal(input)
		require.NoError(t, err)

		result, err := cleanupContextMessages(body, []string{"清理上下文"})
		assert.NoError(t, err)

		var output chatCompletionRequest
		err = json.Unmarshal(result, &output)
		require.NoError(t, err)

		// 没有 system 消息，只保留清理命令之后的消息
		assert.Len(t, output.Messages, 1)
		assert.Equal(t, "user", output.Messages[0].Role)
		assert.Equal(t, "新问题", output.Messages[0].Content)
	})

	t.Run("cleanup_with_empty_messages", func(t *testing.T) {
		input := chatCompletionRequest{
			Messages: []chatMessage{},
		}
		body, err := json.Marshal(input)
		require.NoError(t, err)

		result, err := cleanupContextMessages(body, []string{"清理上下文"})
		assert.NoError(t, err)

		var output chatCompletionRequest
		err = json.Unmarshal(result, &output)
		require.NoError(t, err)

		assert.Len(t, output.Messages, 0)
	})

	t.Run("cleanup_command_partial_match_not_triggered", func(t *testing.T) {
		input := chatCompletionRequest{
			Messages: []chatMessage{
				{Role: "system", Content: "你是一个助手"},
				{Role: "user", Content: "请清理上下文吧"},
				{Role: "assistant", Content: "好的"},
			},
		}
		body, err := json.Marshal(input)
		require.NoError(t, err)

		result, err := cleanupContextMessages(body, []string{"清理上下文"})
		assert.NoError(t, err)

		// 部分匹配不应触发清理
		assert.Equal(t, body, result)
	})

	t.Run("invalid_json_body", func(t *testing.T) {
		body := []byte(`invalid json`)
		result, err := cleanupContextMessages(body, []string{"清理上下文"})
		assert.Error(t, err)
		assert.Equal(t, body, result)
	})
}
