package main

import (
	"strings"
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/provider"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/test"
	"github.com/higress-group/wasm-go/pkg/iface"
)

func Test_getApiName(t *testing.T) {
	tests := []struct {
		name string
		path string
		want provider.ApiName
	}{
		// OpenAI style
		{"openai chat completions", "/v1/chat/completions", provider.ApiNameChatCompletion},
		{"openai completions", "/v1/completions", provider.ApiNameCompletion},
		{"openai embeddings", "/v1/embeddings", provider.ApiNameEmbeddings},
		{"openai audio speech", "/v1/audio/speech", provider.ApiNameAudioSpeech},
		{"openai audio transcriptions", "/v1/audio/transcriptions", provider.ApiNameAudioTranscription},
		{"openai audio transcriptions with prefix", "/proxy/v1/audio/transcriptions", provider.ApiNameAudioTranscription},
		{"openai audio translations", "/v1/audio/translations", provider.ApiNameAudioTranslation},
		{"openai realtime", "/v1/realtime", provider.ApiNameRealtime},
		{"openai realtime with prefix", "/proxy/v1/realtime", provider.ApiNameRealtime},
		{"openai realtime with trailing slash", "/v1/realtime/", ""},
		{"openai image generation", "/v1/images/generations", provider.ApiNameImageGeneration},
		{"openai image variation", "/v1/images/variations", provider.ApiNameImageVariation},
		{"openai image edit", "/v1/images/edits", provider.ApiNameImageEdit},
		{"openai batches", "/v1/batches", provider.ApiNameBatches},
		{"openai retrieve batch", "/v1/batches/batchid", provider.ApiNameRetrieveBatch},
		{"openai cancel batch", "/v1/batches/batchid/cancel", provider.ApiNameCancelBatch},
		{"openai files", "/v1/files", provider.ApiNameFiles},
		{"openai retrieve file", "/v1/files/fileid", provider.ApiNameRetrieveFile},
		{"openai retrieve file content", "/v1/files/fileid/content", provider.ApiNameRetrieveFileContent},
		{"openai videos", "/v1/videos", provider.ApiNameVideos},
		{"openai retrieve video", "/v1/videos/videoid", provider.ApiNameRetrieveVideo},
		{"openai retrieve video content", "/v1/videos/videoid/content", provider.ApiNameRetrieveVideoContent},
		{"openai video remix", "/v1/videos/videoid/remix", provider.ApiNameVideoRemix},
		{"openai models", "/v1/models", provider.ApiNameModels},
		{"openai fine tuning jobs", "/v1/fine_tuning/jobs", provider.ApiNameFineTuningJobs},
		{"openai retrieve fine tuning job", "/v1/fine_tuning/jobs/jobid", provider.ApiNameRetrieveFineTuningJob},
		{"openai fine tuning job events", "/v1/fine_tuning/jobs/jobid/events", provider.ApiNameFineTuningJobEvents},
		{"openai fine tuning job checkpoints", "/v1/fine_tuning/jobs/jobid/checkpoints", provider.ApiNameFineTuningJobCheckpoints},
		{"openai cancel fine tuning job", "/v1/fine_tuning/jobs/jobid/cancel", provider.ApiNameCancelFineTuningJob},
		{"openai resume fine tuning job", "/v1/fine_tuning/jobs/jobid/resume", provider.ApiNameResumeFineTuningJob},
		{"openai pause fine tuning job", "/v1/fine_tuning/jobs/jobid/pause", provider.ApiNamePauseFineTuningJob},
		{"openai fine tuning checkpoint permissions", "/v1/fine_tuning/checkpoints/checkpointid/permissions", provider.ApiNameFineTuningCheckpointPermissions},
		{"openai delete fine tuning checkpoint permission", "/v1/fine_tuning/checkpoints/checkpointid/permissions/permissionid", provider.ApiNameDeleteFineTuningCheckpointPermission},
		{"openai responses", "/v1/responses", provider.ApiNameResponses},
		// Anthropic
		{"anthropic messages", "/v1/messages", provider.ApiNameAnthropicMessages},
		{"anthropic complete", "/v1/complete", provider.ApiNameAnthropicComplete},
		// Gemini
		{"gemini generate content", "/v1beta/models/gemini-1.0-pro:generateContent", provider.ApiNameGeminiGenerateContent},
		{"gemini stream generate content", "/v1beta/models/gemini-1.0-pro:streamGenerateContent", provider.ApiNameGeminiStreamGenerateContent},
		// Cohere
		{"cohere rerank", "/v1/rerank", provider.ApiNameCohereV1Rerank},
		// Unknown
		{"unknown", "/v1/unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getApiName(tt.path)
			if got != tt.want {
				t.Errorf("getApiName(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func Test_isSupportedRequestContentType(t *testing.T) {
	tests := []struct {
		name        string
		apiName     provider.ApiName
		contentType string
		want        bool
	}{
		{
			name:        "json chat completion",
			apiName:     provider.ApiNameChatCompletion,
			contentType: "application/json",
			want:        true,
		},
		{
			name:        "multipart image edit",
			apiName:     provider.ApiNameImageEdit,
			contentType: "multipart/form-data; boundary=----boundary",
			want:        true,
		},
		{
			name:        "multipart image variation",
			apiName:     provider.ApiNameImageVariation,
			contentType: "multipart/form-data; boundary=----boundary",
			want:        true,
		},
		{
			name:        "multipart chat completion",
			apiName:     provider.ApiNameChatCompletion,
			contentType: "multipart/form-data; boundary=----boundary",
			want:        false,
		},
		{
			name:        "text plain image edit",
			apiName:     provider.ApiNameImageEdit,
			contentType: "text/plain",
			want:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSupportedRequestContentType(tt.apiName, tt.contentType)
			if got != tt.want {
				t.Errorf("isSupportedRequestContentType(%v, %q) = %v, want %v", tt.apiName, tt.contentType, got, tt.want)
			}
		})
	}
}

func TestAi360(t *testing.T) {
	test.RunAi360ParseConfigTests(t)
	test.RunAi360OnHttpRequestHeadersTests(t)
	test.RunAi360OnHttpRequestBodyTests(t)
	test.RunAi360OnHttpResponseHeadersTests(t)
	test.RunAi360OnHttpResponseBodyTests(t)
	test.RunAi360OnStreamingResponseBodyTests(t)
}

func TestOpenAI(t *testing.T) {
	test.RunOpenAIParseConfigTests(t)
	test.RunOpenAIOnHttpRequestHeadersTests(t)
	test.RunOpenAIOnHttpRequestBodyTests(t)
	test.RunOpenAIOnHttpResponseHeadersTests(t)
	test.RunOpenAIOnHttpResponseBodyTests(t)
	test.RunOpenAIOnStreamingResponseBodyTests(t)
}

func TestQwen(t *testing.T) {
	test.RunQwenParseConfigTests(t)
	test.RunQwenOnHttpRequestHeadersTests(t)
	test.RunQwenOnHttpRequestBodyTests(t)
	test.RunQwenOnHttpResponseHeadersTests(t)
	test.RunQwenOnHttpResponseBodyTests(t)
	test.RunQwenOnStreamingResponseBodyTests(t)
}

func TestGemini(t *testing.T) {
	test.RunGeminiParseConfigTests(t)
	test.RunGeminiOnHttpRequestHeadersTests(t)
	test.RunGeminiOnHttpRequestBodyTests(t)
	test.RunGeminiOnHttpResponseHeadersTests(t)
	test.RunGeminiOnHttpResponseBodyTests(t)
	test.RunGeminiOnStreamingResponseBodyTests(t)
	test.RunGeminiGetImageURLTests(t)
}

func TestAzure(t *testing.T) {
	test.RunAzureParseConfigTests(t)
	test.RunAzureOnHttpRequestHeadersTests(t)
	test.RunAzureOnHttpRequestBodyTests(t)
	test.RunAzureOnHttpResponseHeadersTests(t)
	test.RunAzureOnHttpResponseBodyTests(t)
	test.RunAzureBasePathHandlingTests(t)
}

func TestFireworks(t *testing.T) {
	test.RunFireworksParseConfigTests(t)
	test.RunFireworksOnHttpRequestHeadersTests(t)
	test.RunFireworksOnHttpRequestBodyTests(t)
}

func TestMinimax(t *testing.T) {
	test.RunMinimaxBasePathHandlingTests(t)
}

func TestUtil(t *testing.T) {
	test.RunMapRequestPathByCapabilityTests(t)
}

func TestApiPathRegression(t *testing.T) {
	test.RunApiPathRegressionTests(t)
}

func TestGeneric(t *testing.T) {
	test.RunGenericParseConfigTests(t)
	test.RunGenericOnHttpRequestHeadersTests(t)
	test.RunGenericOnHttpRequestBodyTests(t)
}

func TestVertex(t *testing.T) {
	test.RunVertexParseConfigTests(t)
	test.RunVertexExpressModeOnHttpRequestHeadersTests(t)
	test.RunVertexExpressModeOnHttpRequestBodyTests(t)
	test.RunVertexExpressModeOnHttpResponseBodyTests(t)
	test.RunVertexExpressModeOnStreamingResponseBodyTests(t)
	test.RunVertexExpressModeImageGenerationRequestBodyTests(t)
	test.RunVertexExpressModeImageGenerationResponseBodyTests(t)
	test.RunVertexExpressModeImageEditVariationRequestBodyTests(t)
	test.RunVertexExpressModeImageEditVariationResponseBodyTests(t)
	// Vertex Raw 模式测试
	test.RunVertexRawModeOnHttpRequestHeadersTests(t)
	test.RunVertexRawModeOnHttpRequestBodyTests(t)
	test.RunVertexRawModeOnHttpResponseBodyTests(t)
}

func TestBedrock(t *testing.T) {
	test.RunBedrockParseConfigTests(t)
	test.RunBedrockOnHttpRequestHeadersTests(t)
	test.RunBedrockOnHttpRequestBodyTests(t)
	test.RunBedrockOnHttpResponseHeadersTests(t)
	test.RunBedrockOnHttpResponseBodyTests(t)
	test.RunBedrockOnStreamingResponseBodyTests(t)
	test.RunBedrockToolCallTests(t)
}

func TestClaude(t *testing.T) {
	test.RunClaudeParseConfigTests(t)
	test.RunClaudeOnHttpRequestHeadersTests(t)
	test.RunClaudeOnHttpRequestBodyTests(t)
}

func TestConsumerAffinity(t *testing.T) {
	test.RunConsumerAffinityParseConfigTests(t)
	test.RunConsumerAffinityOnHttpRequestHeadersTests(t)
}

// mockHttpContext is a minimal mock for wrapper.HttpContext used in streaming tests.
type mockHttpContext struct {
	contextMap map[string]interface{}
}

func newMockHttpContext() *mockHttpContext {
	return &mockHttpContext{contextMap: make(map[string]interface{})}
}

func (m *mockHttpContext) SetContext(key string, value interface{})          { m.contextMap[key] = value }
func (m *mockHttpContext) GetContext(key string) interface{}                 { return m.contextMap[key] }
func (m *mockHttpContext) GetBoolContext(key string, def bool) bool          { return def }
func (m *mockHttpContext) GetStringContext(key, def string) string           { return def }
func (m *mockHttpContext) GetByteSliceContext(key string, def []byte) []byte { return def }
func (m *mockHttpContext) Scheme() string                                    { return "" }
func (m *mockHttpContext) Host() string                                      { return "" }
func (m *mockHttpContext) Path() string                                      { return "" }
func (m *mockHttpContext) Method() string                                    { return "" }
func (m *mockHttpContext) GetUserAttribute(key string) interface{}           { return nil }
func (m *mockHttpContext) SetUserAttribute(key string, value interface{})    {}
func (m *mockHttpContext) SetUserAttributeMap(kvmap map[string]interface{})  {}
func (m *mockHttpContext) GetUserAttributeMap() map[string]interface{}       { return nil }
func (m *mockHttpContext) WriteUserAttributeToLog() error                    { return nil }
func (m *mockHttpContext) WriteUserAttributeToLogWithKey(key string) error   { return nil }
func (m *mockHttpContext) WriteUserAttributeToTrace() error                  { return nil }
func (m *mockHttpContext) DontReadRequestBody()                              {}
func (m *mockHttpContext) DontReadResponseBody()                             {}
func (m *mockHttpContext) BufferRequestBody()                                {}
func (m *mockHttpContext) BufferResponseBody()                               {}
func (m *mockHttpContext) NeedPauseStreamingResponse()                       {}
func (m *mockHttpContext) PushBuffer(buffer []byte)                          {}
func (m *mockHttpContext) PopBuffer() []byte                                 { return nil }
func (m *mockHttpContext) BufferQueueSize() int                              { return 0 }
func (m *mockHttpContext) DisableReroute()                                   {}
func (m *mockHttpContext) SetRequestBodyBufferLimit(byteSize uint32)         {}
func (m *mockHttpContext) SetResponseBodyBufferLimit(byteSize uint32)        {}
func (m *mockHttpContext) RouteCall(method, url string, headers [][2]string, body []byte, callback iface.RouteResponseCallback) error {
	return nil
}
func (m *mockHttpContext) GetExecutionPhase() iface.HTTPExecutionPhase { return 0 }
func (m *mockHttpContext) HasRequestBody() bool      { return false }
func (m *mockHttpContext) HasResponseBody() bool     { return false }
func (m *mockHttpContext) IsWebsocket() bool         { return false }
func (m *mockHttpContext) IsBinaryRequestBody() bool { return false }
func (m *mockHttpContext) IsBinaryResponseBody() bool { return false }

func TestPromoteThinkingOnEmptyResponse(t *testing.T) {
	t.Run("promotes_reasoning_when_content_empty", func(t *testing.T) {
		body := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"","reasoning_content":"这是思考内容"},"finish_reason":"stop"}]}`)
		result, err := provider.PromoteThinkingOnEmptyResponse(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// content should now contain the reasoning text
		if !contains(result, `"content":"这是思考内容"`) {
			t.Errorf("expected reasoning promoted to content, got: %s", result)
		}
		// reasoning_content should be cleared
		if contains(result, `"reasoning_content":"这是思考内容"`) {
			t.Errorf("expected reasoning_content to be cleared, got: %s", result)
		}
	})

	t.Run("promotes_reasoning_when_content_nil", func(t *testing.T) {
		body := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","reasoning_content":"思考结果"},"finish_reason":"stop"}]}`)
		result, err := provider.PromoteThinkingOnEmptyResponse(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !contains(result, `"content":"思考结果"`) {
			t.Errorf("expected reasoning promoted to content, got: %s", result)
		}
	})

	t.Run("no_change_when_content_present", func(t *testing.T) {
		body := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"正常回复","reasoning_content":"思考过程"},"finish_reason":"stop"}]}`)
		result, err := provider.PromoteThinkingOnEmptyResponse(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should return original body unchanged
		if string(result) != string(body) {
			t.Errorf("expected body unchanged, got: %s", result)
		}
	})

	t.Run("no_change_when_no_reasoning", func(t *testing.T) {
		body := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"正常回复"},"finish_reason":"stop"}]}`)
		result, err := provider.PromoteThinkingOnEmptyResponse(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != string(body) {
			t.Errorf("expected body unchanged, got: %s", result)
		}
	})

	t.Run("no_change_when_both_empty", func(t *testing.T) {
		body := []byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"stop"}]}`)
		result, err := provider.PromoteThinkingOnEmptyResponse(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != string(body) {
			t.Errorf("expected body unchanged, got: %s", result)
		}
	})

	t.Run("invalid_json_returns_original", func(t *testing.T) {
		body := []byte(`not json`)
		result, err := provider.PromoteThinkingOnEmptyResponse(body)
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
		if string(result) != string(body) {
			t.Errorf("expected original body returned on error")
		}
	})
}

func TestPromoteStreamingThinkingOnEmptyChunk(t *testing.T) {
	t.Run("promotes_reasoning_delta_when_no_content", func(t *testing.T) {
		ctx := newMockHttpContext()
		data := []byte(`{"choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"思考中"}}]}`)
		result, err := provider.PromoteStreamingThinkingOnEmptyChunk(ctx, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !contains(result, `"content":"思考中"`) {
			t.Errorf("expected reasoning promoted to content delta, got: %s", result)
		}
	})

	t.Run("no_promote_after_content_seen", func(t *testing.T) {
		ctx := newMockHttpContext()
		// First chunk: content delta
		data1 := []byte(`{"choices":[{"index":0,"delta":{"content":"正文"}}]}`)
		_, _ = provider.PromoteStreamingThinkingOnEmptyChunk(ctx, data1)

		// Second chunk: reasoning only — should NOT be promoted
		data2 := []byte(`{"choices":[{"index":0,"delta":{"reasoning_content":"后续思考"}}]}`)
		result, err := provider.PromoteStreamingThinkingOnEmptyChunk(ctx, data2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should return unchanged since content was already seen
		if string(result) != string(data2) {
			t.Errorf("expected no promotion after content seen, got: %s", result)
		}
	})

	t.Run("promotes_reasoning_field_when_no_content", func(t *testing.T) {
		ctx := newMockHttpContext()
		data := []byte(`{"choices":[{"index":0,"delta":{"reasoning":"流式思考"}}]}`)
		result, err := provider.PromoteStreamingThinkingOnEmptyChunk(ctx, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !contains(result, `"content":"流式思考"`) {
			t.Errorf("expected reasoning promoted to content delta, got: %s", result)
		}
	})

	t.Run("no_change_when_content_present_in_delta", func(t *testing.T) {
		ctx := newMockHttpContext()
		data := []byte(`{"choices":[{"index":0,"delta":{"content":"有内容","reasoning_content":"也有思考"}}]}`)
		result, err := provider.PromoteStreamingThinkingOnEmptyChunk(ctx, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != string(data) {
			t.Errorf("expected no change when content present, got: %s", result)
		}
	})

	t.Run("invalid_json_returns_original", func(t *testing.T) {
		ctx := newMockHttpContext()
		data := []byte(`not json`)
		result, err := provider.PromoteStreamingThinkingOnEmptyChunk(ctx, data)
		if err != nil {
			t.Fatalf("unexpected error for invalid json: %v", err)
		}
		if string(result) != string(data) {
			t.Errorf("expected original data returned")
		}
	})
}

func TestPromoteThinkingInStreamingChunk(t *testing.T) {
	t.Run("promotes_in_sse_format", func(t *testing.T) {
		ctx := newMockHttpContext()
		chunk := []byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"思考\"}}]}\n\n")
		result := promoteThinkingInStreamingChunk(ctx, chunk)
		if !contains(result, `"content":"思考"`) {
			t.Errorf("expected reasoning promoted in SSE chunk, got: %s", result)
		}
	})

	t.Run("skips_done_marker", func(t *testing.T) {
		ctx := newMockHttpContext()
		chunk := []byte("data: [DONE]\n\n")
		result := promoteThinkingInStreamingChunk(ctx, chunk)
		if string(result) != string(chunk) {
			t.Errorf("expected [DONE] unchanged, got: %s", result)
		}
	})

	t.Run("handles_multiple_events", func(t *testing.T) {
		ctx := newMockHttpContext()
		chunk := []byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"第一段\"}}]}\n\ndata: {\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"第二段\"}}]}\n\n")
		result := promoteThinkingInStreamingChunk(ctx, chunk)
		if !contains(result, `"content":"第一段"`) {
			t.Errorf("expected first reasoning promoted, got: %s", result)
		}
		if !contains(result, `"content":"第二段"`) {
			t.Errorf("expected second reasoning promoted, got: %s", result)
		}
	})
}

// contains checks if s contains substr
func contains(b []byte, substr string) bool {
	return strings.Contains(string(b), substr)
}
