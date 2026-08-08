package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatMessageParseContentSkipsMalformedMultimodalParts(t *testing.T) {
	msg := &chatMessage{
		Content: []any{
			map[string]any{
				"type": "text",
				"text": "keep text",
			},
			map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{},
			},
			map[string]any{
				"type": "image_url",
				"image_url": map[string]any{
					"url": 123,
				},
			},
			map[string]any{
				"type": "image_url",
				"image_url": map[string]any{
					"url":    "https://example.com/image.png",
					"detail": "high",
				},
			},
			map[string]any{
				"type": "input_audio",
				"input_audio": map[string]any{
					"data": "abc",
				},
			},
			map[string]any{
				"type": "input_audio",
				"input_audio": map[string]any{
					"data":   123,
					"format": "wav",
				},
			},
			map[string]any{
				"type": "input_audio",
				"input_audio": map[string]any{
					"data":   "abc",
					"format": "wav",
				},
			},
			map[string]any{
				"type": "file",
				"file": map[string]any{
					"file_id": 123,
				},
			},
			map[string]any{
				"type": "file",
				"file": map[string]any{
					"file_id": "file-123",
				},
			},
		},
	}

	contents := msg.ParseContent()

	require.Len(t, contents, 4)
	assert.Equal(t, chatMessageContent{
		Type: contentTypeText,
		Text: "keep text",
	}, contents[0])
	require.NotNil(t, contents[1].ImageUrl)
	assert.Equal(t, "https://example.com/image.png", contents[1].ImageUrl.Url)
	assert.Equal(t, "high", contents[1].ImageUrl.Detail)
	require.NotNil(t, contents[2].InputAudio)
	assert.Equal(t, "abc", contents[2].InputAudio.Data)
	assert.Equal(t, "wav", contents[2].InputAudio.Format)
	require.NotNil(t, contents[3].File)
	assert.Equal(t, "file-123", contents[3].File.FileId)
}
