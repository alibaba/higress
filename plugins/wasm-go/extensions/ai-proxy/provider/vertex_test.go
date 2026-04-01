package provider

import (
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVertexProviderBuildChatRequestStructuredOutputMapping(t *testing.T) {
	t.Run("json_object response format", func(t *testing.T) {
		v := &vertexProvider{}
		req := &chatCompletionRequest{
			Model: "gemini-2.5-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: map[string]interface{}{
				"type": "json_object",
			},
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.NoError(t, err)
		require.NotNil(t, vertexReq)

		assert.Equal(t, util.MimeTypeApplicationJson, vertexReq.GenerationConfig.ResponseMimeType)
		assert.Nil(t, vertexReq.GenerationConfig.ResponseSchema)
	})

	t.Run("json_schema response format with nested schema", func(t *testing.T) {
		v := &vertexProvider{}
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"answer": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{"answer"},
		}
		req := &chatCompletionRequest{
			Model: "gemini-2.5-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: map[string]interface{}{
				"type": "json_schema",
				"json_schema": map[string]interface{}{
					"name":   "response",
					"strict": true,
					"schema": schema,
				},
			},
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.NoError(t, err)
		require.NotNil(t, vertexReq)

		assert.Equal(t, util.MimeTypeApplicationJson, vertexReq.GenerationConfig.ResponseMimeType)
		assert.Equal(t, schema, vertexReq.GenerationConfig.ResponseSchema)
	})

	t.Run("json_schema response format with direct schema object", func(t *testing.T) {
		v := &vertexProvider{}
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"city": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{"city"},
		}
		req := &chatCompletionRequest{
			Model: "gemini-2.5-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: map[string]interface{}{
				"type":        "json_schema",
				"json_schema": schema,
			},
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.NoError(t, err)
		require.NotNil(t, vertexReq)

		assert.Equal(t, util.MimeTypeApplicationJson, vertexReq.GenerationConfig.ResponseMimeType)
		assert.Equal(t, schema, vertexReq.GenerationConfig.ResponseSchema)
	})

	t.Run("json_schema response format without valid schema should return error", func(t *testing.T) {
		v := &vertexProvider{}
		req := &chatCompletionRequest{
			Model: "gemini-2.5-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: map[string]interface{}{
				"type":        "json_schema",
				"json_schema": "invalid",
			},
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.Error(t, err)
		assert.Nil(t, vertexReq)
		assert.Contains(t, err.Error(), "invalid response_format.json_schema")
	})

	t.Run("direct schema in response_format for compatibility", func(t *testing.T) {
		v := &vertexProvider{}
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"result": map[string]interface{}{
					"type": "string",
				},
			},
		}
		req := &chatCompletionRequest{
			Model: "gemini-2.5-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: schema,
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.NoError(t, err)
		require.NotNil(t, vertexReq)

		assert.Equal(t, util.MimeTypeApplicationJson, vertexReq.GenerationConfig.ResponseMimeType)
		assert.Equal(t, schema, vertexReq.GenerationConfig.ResponseSchema)
	})

	t.Run("text response format keeps default text output", func(t *testing.T) {
		v := &vertexProvider{}
		req := &chatCompletionRequest{
			Model: "gemini-2.5-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: map[string]interface{}{
				"type": "text",
			},
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.NoError(t, err)
		require.NotNil(t, vertexReq)

		assert.Empty(t, vertexReq.GenerationConfig.ResponseMimeType)
		assert.Nil(t, vertexReq.GenerationConfig.ResponseSchema)
	})

	t.Run("unknown response format does not inject schema config", func(t *testing.T) {
		v := &vertexProvider{}
		req := &chatCompletionRequest{
			Model: "gemini-2.5-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: map[string]interface{}{
				"type": "xml",
			},
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.NoError(t, err)
		require.NotNil(t, vertexReq)

		assert.Empty(t, vertexReq.GenerationConfig.ResponseMimeType)
		assert.Nil(t, vertexReq.GenerationConfig.ResponseSchema)
	})

	t.Run("gemini 2.0 json_schema is ignored for stability", func(t *testing.T) {
		v := &vertexProvider{}
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"answer": map[string]interface{}{
					"type": "string",
				},
			},
		}
		req := &chatCompletionRequest{
			Model: "gemini-2.0-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: map[string]interface{}{
				"type": "json_schema",
				"json_schema": map[string]interface{}{
					"name":   "response",
					"strict": true,
					"schema": schema,
				},
			},
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.NoError(t, err)
		require.NotNil(t, vertexReq)
		assert.Empty(t, vertexReq.GenerationConfig.ResponseMimeType)
		assert.Nil(t, vertexReq.GenerationConfig.ResponseSchema)
	})

	t.Run("gemini 2.0 malformed json_schema is also ignored", func(t *testing.T) {
		v := &vertexProvider{}
		req := &chatCompletionRequest{
			Model: "gemini-2.0-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: map[string]interface{}{
				"type":        "json_schema",
				"json_schema": "invalid",
			},
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.NoError(t, err)
		require.NotNil(t, vertexReq)
		assert.Empty(t, vertexReq.GenerationConfig.ResponseMimeType)
		assert.Nil(t, vertexReq.GenerationConfig.ResponseSchema)
	})

	t.Run("gemini 2.0 json_object is ignored", func(t *testing.T) {
		v := &vertexProvider{}
		req := &chatCompletionRequest{
			Model: "gemini-2.0-flash",
			Messages: []chatMessage{
				{Role: roleUser, Content: "hello"},
			},
			ResponseFormat: map[string]interface{}{
				"type": "json_object",
			},
		}

		vertexReq, err := v.buildVertexChatRequest(req)
		require.NoError(t, err)
		require.NotNil(t, vertexReq)
		assert.Empty(t, vertexReq.GenerationConfig.ResponseMimeType)
		assert.Nil(t, vertexReq.GenerationConfig.ResponseSchema)
	})
}

func TestVertexProviderApplyResponseFormatNilSafety(t *testing.T) {
	v := &vertexProvider{}
	require.NoError(t, v.applyResponseFormatToGenerationConfig(map[string]interface{}{"type": "json_object"}, nil, "gemini-2.5-flash"))
	require.NoError(t, v.applyResponseFormatToGenerationConfig(nil, &vertexChatGenerationConfig{}, "gemini-2.5-flash"))
	require.NoError(t, v.applyResponseFormatToGenerationConfig(map[string]interface{}{}, &vertexChatGenerationConfig{}, "gemini-2.5-flash"))
}
