package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestParseContent(t *testing.T) {
	tests := []struct {
		name           string
		json           string
		expectedText   string
		expectedImages []ImageItem
	}{
		{
			name:           "plain string content",
			json:           `{"content":"Hello world"}`,
			expectedText:   "Hello world",
			expectedImages: []ImageItem{},
		},
		{
			name:           "empty string content",
			json:           `{"content":""}`,
			expectedText:   "",
			expectedImages: []ImageItem{},
		},
		{
			name: "array with text only",
			json: `{"content":[{"type":"text","text":"Hello"},{"type":"text","text":" World"}]}`,
			expectedText:   "Hello World",
			expectedImages: []ImageItem{},
		},
		{
			name: "array with image URL",
			json: `{"content":[
				{"type":"text","text":"Describe this"},
				{"type":"image_url","image_url":{"url":"https://example.com/img.png"}}
			]}`,
			expectedText: "Describe this",
			expectedImages: []ImageItem{
				{Content: "https://example.com/img.png", Type: "URL"},
			},
		},
		{
			name: "array with base64 image",
			json: `{"content":[
				{"type":"text","text":"What is this?"},
				{"type":"image_url","image_url":{"url":"data:image/png;base64,iVBORw0KGgo="}}
			]}`,
			expectedText: "What is this?",
			expectedImages: []ImageItem{
				{Content: "data:image/png;base64,iVBORw0KGgo=", Type: "BASE64"},
			},
		},
		{
			name: "array with multiple images",
			json: `{"content":[
				{"type":"text","text":"Compare these"},
				{"type":"image_url","image_url":{"url":"https://example.com/a.png"}},
				{"type":"image_url","image_url":{"url":"https://example.com/b.png"}}
			]}`,
			expectedText: "Compare these",
			expectedImages: []ImageItem{
				{Content: "https://example.com/a.png", Type: "URL"},
				{Content: "https://example.com/b.png", Type: "URL"},
			},
		},
		{
			name: "array with mixed base64 and URL images",
			json: `{"content":[
				{"type":"image_url","image_url":{"url":"data:image/jpeg;base64,/9j/4AAQ="}},
				{"type":"image_url","image_url":{"url":"https://example.com/photo.jpg"}}
			]}`,
			expectedText: "",
			expectedImages: []ImageItem{
				{Content: "data:image/jpeg;base64,/9j/4AAQ=", Type: "BASE64"},
				{Content: "https://example.com/photo.jpg", Type: "URL"},
			},
		},
		{
			name: "array with only images, no text",
			json: `{"content":[
				{"type":"image_url","image_url":{"url":"https://example.com/img.png"}}
			]}`,
			expectedText: "",
			expectedImages: []ImageItem{
				{Content: "https://example.com/img.png", Type: "URL"},
			},
		},
		{
			name: "array with unknown type is ignored",
			json: `{"content":[
				{"type":"text","text":"Hello"},
				{"type":"video","video":{"url":"https://example.com/v.mp4"}},
				{"type":"image_url","image_url":{"url":"https://example.com/img.png"}}
			]}`,
			expectedText: "Hello",
			expectedImages: []ImageItem{
				{Content: "https://example.com/img.png", Type: "URL"},
			},
		},
		{
			name:           "empty array",
			json:           `{"content":[]}`,
			expectedText:   "",
			expectedImages: []ImageItem{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentResult := gjson.Get(tt.json, "content")
			text, images := parseContent(contentResult)
			assert.Equal(t, tt.expectedText, text)
			assert.Equal(t, tt.expectedImages, images)
		})
	}
}
