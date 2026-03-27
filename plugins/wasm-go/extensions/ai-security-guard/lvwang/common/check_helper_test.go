package common

import (
	"encoding/json"
	"strings"
	"testing"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectDenyMessage(t *testing.T) {
	tests := []struct {
		name           string
		configDenyMsg  string
		response       cfg.Response
		expectedResult string
	}{
		{
			name:           "default message when config and advice are empty",
			configDenyMsg:  "",
			response:       cfg.Response{},
			expectedResult: cfg.DefaultDenyMessage,
		},
		{
			name:          "config deny message takes priority",
			configDenyMsg: "custom deny message",
			response: cfg.Response{
				Data: cfg.Data{
					Advice: []cfg.Advice{{Answer: "advice answer"}},
				},
			},
			expectedResult: "custom deny message",
		},
		{
			name:          "advice answer used when config is empty",
			configDenyMsg: "",
			response: cfg.Response{
				Data: cfg.Data{
					Advice: []cfg.Advice{{Answer: "from advice"}},
				},
			},
			expectedResult: "from advice",
		},
		{
			name:          "default when advice exists but answer is empty",
			configDenyMsg: "",
			response: cfg.Response{
				Data: cfg.Data{
					Advice: []cfg.Advice{{Answer: "", HitLabel: "spam"}},
				},
			},
			expectedResult: cfg.DefaultDenyMessage,
		},
		{
			name:          "default when advice array is nil",
			configDenyMsg: "",
			response: cfg.Response{
				Data: cfg.Data{
					Advice: nil,
				},
			},
			expectedResult: cfg.DefaultDenyMessage,
		},
		{
			name:          "default when advice array is empty",
			configDenyMsg: "",
			response: cfg.Response{
				Data: cfg.Data{
					Advice: []cfg.Advice{},
				},
			},
			expectedResult: cfg.DefaultDenyMessage,
		},
		{
			name:          "first advice answer used when multiple advice entries",
			configDenyMsg: "",
			response: cfg.Response{
				Data: cfg.Data{
					Advice: []cfg.Advice{
						{Answer: "first advice"},
						{Answer: "second advice"},
					},
				},
			},
			expectedResult: "first advice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectDenyMessage(tt.configDenyMsg, tt.response)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestBuildDenyResponseBody(t *testing.T) {
	denyMessage := "test deny message"

	t.Run("protocol original returns raw message", func(t *testing.T) {
		result := BuildDenyResponseBody(true, false, 200, denyMessage)
		assert.Equal(t, "application/json", result.ContentType)
		assert.Contains(t, string(result.Body), denyMessage)
	})

	t.Run("non-stream returns OpenAI chat completion format", func(t *testing.T) {
		result := BuildDenyResponseBody(false, false, 200, denyMessage)
		assert.Equal(t, "application/json", result.ContentType)
		body := string(result.Body)
		assert.Contains(t, body, "chat.completion")
		assert.Contains(t, body, "chatcmpl-")
		assert.Contains(t, body, denyMessage)

		var parsed map[string]interface{}
		err := json.Unmarshal(result.Body, &parsed)
		require.NoError(t, err)
		assert.Equal(t, "chat.completion", parsed["object"])
		choices := parsed["choices"].([]interface{})
		assert.Len(t, choices, 1)
		choice := choices[0].(map[string]interface{})
		msg := choice["message"].(map[string]interface{})
		assert.Equal(t, "assistant", msg["role"])
		assert.Equal(t, "stop", choice["finish_reason"])
	})

	t.Run("stream returns SSE format with data: prefix", func(t *testing.T) {
		result := BuildDenyResponseBody(false, true, 200, denyMessage)
		assert.Equal(t, "text/event-stream;charset=UTF-8", result.ContentType)
		body := string(result.Body)
		assert.Contains(t, body, "chat.completion.chunk")
		assert.Contains(t, body, denyMessage)
		assert.Contains(t, body, "data:")
		assert.Contains(t, body, "[DONE]")
	})

	t.Run("stream response contains two data blocks plus done", func(t *testing.T) {
		result := BuildDenyResponseBody(false, true, 200, denyMessage)
		body := string(result.Body)
		dataCount := strings.Count(body, "data:")
		assert.Equal(t, 3, dataCount, "should have 2 data chunks + 1 [DONE]")
	})

	t.Run("non-stream response IDs are consistent within response", func(t *testing.T) {
		result := BuildDenyResponseBody(false, false, 200, denyMessage)
		var parsed map[string]interface{}
		err := json.Unmarshal(result.Body, &parsed)
		require.NoError(t, err)
		id := parsed["id"].(string)
		assert.True(t, strings.HasPrefix(id, "chatcmpl-"))
	})

	t.Run("protocol original takes priority over stream", func(t *testing.T) {
		result := BuildDenyResponseBody(true, true, 200, denyMessage)
		assert.Equal(t, "application/json", result.ContentType)
		assert.NotContains(t, string(result.Body), "chat.completion")
	})

	t.Run("different calls produce different IDs", func(t *testing.T) {
		result1 := BuildDenyResponseBody(false, false, 200, denyMessage)
		result2 := BuildDenyResponseBody(false, false, 200, denyMessage)
		var p1, p2 map[string]interface{}
		_ = json.Unmarshal(result1.Body, &p1)
		_ = json.Unmarshal(result2.Body, &p2)
		assert.NotEqual(t, p1["id"], p2["id"], "each response should have a unique ID")
	})

	t.Run("special characters in deny message are preserved", func(t *testing.T) {
		specialMsg := `message with "quotes" and 中文 and <tags>`
		result := BuildDenyResponseBody(false, false, 200, specialMsg)
		body := string(result.Body)
		assert.Contains(t, body, "中文")
	})
}
