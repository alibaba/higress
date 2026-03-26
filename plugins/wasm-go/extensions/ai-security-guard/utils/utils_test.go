package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/resp"
)

func TestParseAllMessages(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected []MessageInfo
	}{
		{
			name: "standard multi-turn conversation",
			body: `{"messages":[
				{"role":"system","content":"You are a helpful assistant."},
				{"role":"user","content":"Hello"},
				{"role":"assistant","content":"Hi there!"},
				{"role":"user","content":"How are you?"}
			]}`,
			expected: []MessageInfo{
				{Index: 0, Role: "system", Content: "You are a helpful assistant."},
				{Index: 1, Role: "user", Content: "Hello"},
				{Index: 2, Role: "assistant", Content: "Hi there!"},
				{Index: 3, Role: "user", Content: "How are you?"},
			},
		},
		{
			name: "multi-modal content array with text and image",
			body: `{"messages":[
				{"role":"user","content":[
					{"type":"text","text":"Describe this image"},
					{"type":"image_url","image_url":{"url":"https://example.com/img.png"}}
				]}
			]}`,
			expected: []MessageInfo{
				{Index: 0, Role: "user", Content: "Describe this image", ImageFingerprint: func() string {
					h := sha256.Sum256([]byte("https://example.com/img.png"))
					return hex.EncodeToString(h[:16])
				}()},
			},
		},
		{
			name: "multi-modal content array with multiple text parts",
			body: `{"messages":[
				{"role":"user","content":[
					{"type":"text","text":"Part 1. "},
					{"type":"text","text":"Part 2."}
				]}
			]}`,
			expected: []MessageInfo{
				{Index: 0, Role: "user", Content: "Part 1. Part 2."},
			},
		},
		{
			name: "empty messages array",
			body: `{"messages":[]}`,
			expected: nil,
		},
		{
			name: "no messages field",
			body: `{"prompt":"hello"}`,
			expected: nil,
		},
		{
			name: "invalid JSON",
			body: `not json`,
			expected: nil,
		},
		{
			name: "single message",
			body: `{"messages":[{"role":"user","content":"test"}]}`,
			expected: []MessageInfo{
				{Index: 0, Role: "user", Content: "test"},
			},
		},
		{
			name: "empty content string",
			body: `{"messages":[{"role":"user","content":""}]}`,
			expected: []MessageInfo{
				{Index: 0, Role: "user", Content: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseAllMessages([]byte(tt.body))
			assert.Equal(t, tt.expected, result)
		})
	}

	// Additional: same text with different images should produce different ImageFingerprint
	t.Run("different images produce different fingerprints", func(t *testing.T) {
		body1 := `{"messages":[{"role":"user","content":[
			{"type":"text","text":"describe"},
			{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}
		]}]}`
		body2 := `{"messages":[{"role":"user","content":[
			{"type":"text","text":"describe"},
			{"type":"image_url","image_url":{"url":"https://example.com/b.png"}}
		]}]}`
		msgs1 := ParseAllMessages([]byte(body1))
		msgs2 := ParseAllMessages([]byte(body2))
		assert.Equal(t, msgs1[0].Content, msgs2[0].Content)
		assert.NotEqual(t, msgs1[0].ImageFingerprint, msgs2[0].ImageFingerprint)
	})

	// Pure image message (no text) should still have a fingerprint
	t.Run("pure image message has fingerprint", func(t *testing.T) {
		body := `{"messages":[{"role":"user","content":[
			{"type":"image_url","image_url":{"url":"https://example.com/img.png"}}
		]}]}`
		msgs := ParseAllMessages([]byte(body))
		assert.Equal(t, "", msgs[0].Content)
		assert.NotEmpty(t, msgs[0].ImageFingerprint)
	})
}

func TestBuildRedisKeys(t *testing.T) {
	messages := []MessageInfo{
		{Index: 0, Role: "user", Content: "Hello"},
		{Index: 1, Role: "assistant", Content: "Hi"},
	}

	consumer := "consumerA"
	policy := "text_moderation_plus:llm_query_moderation:high"
	keys := BuildRedisKeys(messages, consumer, policy)
	assert.Len(t, keys, 2)

	for _, key := range keys {
		assert.True(t, strings.HasPrefix(key, "{ai_sec}:"), "key should start with {ai_sec}: prefix")
	}

	// Verify deterministic: same input produces same key
	keys2 := BuildRedisKeys(messages, consumer, policy)
	assert.Equal(t, keys, keys2)

	// Verify hash correctness (now includes policyFingerprint prefix)
	hash := sha256.Sum256([]byte("text_moderation_plus:llm_query_moderation:high:consumerA:user:Hello:"))
	expectedKey := fmt.Sprintf("{ai_sec}:%s", hex.EncodeToString(hash[:16]))
	assert.Equal(t, expectedKey, keys[0])

	// Different role same content should produce different keys
	msgs := []MessageInfo{
		{Index: 0, Role: "user", Content: "Hello"},
		{Index: 1, Role: "system", Content: "Hello"},
	}
	diffKeys := BuildRedisKeys(msgs, consumer, policy)
	assert.NotEqual(t, diffKeys[0], diffKeys[1])

	// Different consumer same message should produce different keys
	keysA := BuildRedisKeys(messages, "consumerA", policy)
	keysB := BuildRedisKeys(messages, "consumerB", policy)
	assert.NotEqual(t, keysA[0], keysB[0], "same message with different consumers must produce different keys")

	// Empty consumer should still work (backward compatible for no-consumer scenarios)
	keysEmpty := BuildRedisKeys(messages, "", policy)
	assert.Len(t, keysEmpty, 2)
	assert.NotEqual(t, keysEmpty[0], keysA[0], "empty consumer should differ from named consumer")

	// Same text, different images should produce different keys
	imgHash1 := sha256.Sum256([]byte("https://example.com/safe.jpg"))
	imgHash2 := sha256.Sum256([]byte("https://example.com/malicious.jpg"))
	msgsWithImages := []MessageInfo{
		{Index: 0, Role: "user", Content: "Describe this", ImageFingerprint: hex.EncodeToString(imgHash1[:16])},
		{Index: 1, Role: "user", Content: "Describe this", ImageFingerprint: hex.EncodeToString(imgHash2[:16])},
	}
	imageKeys := BuildRedisKeys(msgsWithImages, consumer, policy)
	assert.NotEqual(t, imageKeys[0], imageKeys[1], "same text with different images must produce different keys")

	// Pure image messages (empty text) with different images should produce different keys
	msgsImageOnly := []MessageInfo{
		{Index: 0, Role: "user", Content: "", ImageFingerprint: hex.EncodeToString(imgHash1[:16])},
		{Index: 1, Role: "user", Content: "", ImageFingerprint: hex.EncodeToString(imgHash2[:16])},
	}
	imageOnlyKeys := BuildRedisKeys(msgsImageOnly, consumer, policy)
	assert.NotEqual(t, imageOnlyKeys[0], imageOnlyKeys[1], "pure image messages with different images must produce different keys")

	// Different policy same consumer/message should produce different keys (core fix validation)
	policyStrict := "multi_modal_guard:llm_query_moderation_strict:medium"
	keysLax := BuildRedisKeys(messages, consumer, policy)
	keysStrict := BuildRedisKeys(messages, consumer, policyStrict)
	assert.NotEqual(t, keysLax[0], keysStrict[0], "same message with different policies must produce different keys")
}

func TestFilterUnchecked(t *testing.T) {
	messages := []MessageInfo{
		{Index: 0, Role: "system", Content: "sys prompt"},
		{Index: 1, Role: "user", Content: "msg1"},
		{Index: 2, Role: "assistant", Content: "resp1"},
		{Index: 3, Role: "user", Content: "msg2"},
	}

	tests := []struct {
		name     string
		response resp.Value
		expected []MessageInfo
	}{
		{
			name: "all null - all unchecked",
			response: resp.ArrayValue([]resp.Value{
				resp.NullValue(),
				resp.NullValue(),
				resp.NullValue(),
				resp.NullValue(),
			}),
			expected: messages,
		},
		{
			name: "all checked",
			response: resp.ArrayValue([]resp.Value{
				resp.StringValue("1"),
				resp.StringValue("1"),
				resp.StringValue("1"),
				resp.StringValue("1"),
			}),
			expected: nil,
		},
		{
			name: "mixed - first and third checked",
			response: resp.ArrayValue([]resp.Value{
				resp.StringValue("1"),
				resp.NullValue(),
				resp.StringValue("1"),
				resp.NullValue(),
			}),
			expected: []MessageInfo{
				{Index: 1, Role: "user", Content: "msg1"},
				{Index: 3, Role: "user", Content: "msg2"},
			},
		},
		{
			name: "response shorter than messages",
			response: resp.ArrayValue([]resp.Value{
				resp.StringValue("1"),
			}),
			expected: []MessageInfo{
				{Index: 1, Role: "user", Content: "msg1"},
				{Index: 2, Role: "assistant", Content: "resp1"},
				{Index: 3, Role: "user", Content: "msg2"},
			},
		},
		{
			name:     "empty response array",
			response: resp.ArrayValue(nil),
			expected: messages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterUnchecked(messages, tt.response)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConcatTextContent(t *testing.T) {
	tests := []struct {
		name     string
		messages []MessageInfo
		expected string
	}{
		{
			name: "multiple messages",
			messages: []MessageInfo{
				{Content: "Hello"},
				{Content: "World"},
			},
			expected: "Hello\nWorld",
		},
		{
			name: "skip empty content",
			messages: []MessageInfo{
				{Content: "Hello"},
				{Content: ""},
				{Content: "World"},
			},
			expected: "Hello\nWorld",
		},
		{
			name:     "all empty",
			messages: []MessageInfo{{Content: ""}, {Content: ""}},
			expected: "",
		},
		{
			name:     "single message",
			messages: []MessageInfo{{Content: "Only one"}},
			expected: "Only one",
		},
		{
			name:     "nil input",
			messages: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConcatTextContent(tt.messages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterByRole(t *testing.T) {
	messages := []MessageInfo{
		{Index: 0, Role: "system", Content: "sys prompt"},
		{Index: 1, Role: "user", Content: "hello"},
		{Index: 2, Role: "assistant", Content: "hi there"},
		{Index: 3, Role: "user", Content: "how are you"},
		{Index: 4, Role: "tool", Content: "tool result"},
	}

	tests := []struct {
		name     string
		roles    []string
		expected []MessageInfo
	}{
		{
			name:  "system and user only",
			roles: []string{"system", "user"},
			expected: []MessageInfo{
				{Index: 0, Role: "system", Content: "sys prompt"},
				{Index: 1, Role: "user", Content: "hello"},
				{Index: 3, Role: "user", Content: "how are you"},
			},
		},
		{
			name:  "assistant only",
			roles: []string{"assistant"},
			expected: []MessageInfo{
				{Index: 2, Role: "assistant", Content: "hi there"},
			},
		},
		{
			name:     "no roles specified returns all",
			roles:    nil,
			expected: messages,
		},
		{
			name:     "non-existent role returns empty",
			roles:    []string{"admin"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterByRole(messages, tt.roles...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
	}{
		{
			name:     "plain string content",
			json:     `"Hello world"`,
			expected: "Hello world",
		},
		{
			name:     "array with text type",
			json:     `[{"type":"text","text":"Hello"},{"type":"text","text":" World"}]`,
			expected: "Hello World",
		},
		{
			name:     "array with mixed types",
			json:     `[{"type":"text","text":"desc"},{"type":"image_url","image_url":{"url":"http://img.png"}}]`,
			expected: "desc",
		},
		{
			name:     "array with no text type",
			json:     `[{"type":"image_url","image_url":{"url":"http://img.png"}}]`,
			expected: "",
		},
		{
			name:     "empty string",
			json:     `""`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to parse via gjson to get the result
			body := fmt.Sprintf(`{"content":%s}`, tt.json)
			result := ParseAllMessages([]byte(fmt.Sprintf(`{"messages":[{"role":"user","content":%s}]}`, tt.json)))
			_ = body
			if len(result) > 0 {
				assert.Equal(t, tt.expected, result[0].Content)
			} else if tt.expected != "" {
				t.Errorf("expected content %q but got no messages", tt.expected)
			}
		})
	}
}
