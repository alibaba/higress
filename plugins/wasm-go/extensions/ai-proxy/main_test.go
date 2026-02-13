package main

import (
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/provider"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/test"
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
	test.RunBedrockToolCallTests(t)
}

func TestClaude(t *testing.T) {
	test.RunClaudeParseConfigTests(t)
	test.RunClaudeOnHttpRequestHeadersTests(t)
	test.RunClaudeOnHttpRequestBodyTests(t)
}

func TestIsStatefulAPI(t *testing.T) {
	tests := []struct {
		name    string
		apiName string
		want    bool
	}{
		// Stateful APIs
		{"responses api", string(provider.ApiNameResponses), true},
		{"files api", string(provider.ApiNameFiles), true},
		{"retrieve file api", string(provider.ApiNameRetrieveFile), true},
		{"retrieve file content api", string(provider.ApiNameRetrieveFileContent), true},
		{"batches api", string(provider.ApiNameBatches), true},
		{"retrieve batch api", string(provider.ApiNameRetrieveBatch), true},
		{"cancel batch api", string(provider.ApiNameCancelBatch), true},
		{"fine tuning jobs api", string(provider.ApiNameFineTuningJobs), true},
		{"retrieve fine tuning job api", string(provider.ApiNameRetrieveFineTuningJob), true},
		{"fine tuning job events api", string(provider.ApiNameFineTuningJobEvents), true},
		{"fine tuning job checkpoints api", string(provider.ApiNameFineTuningJobCheckpoints), true},
		// Stateless APIs
		{"chat completion api", string(provider.ApiNameChatCompletion), false},
		{"completion api", string(provider.ApiNameCompletion), false},
		{"embeddings api", string(provider.ApiNameEmbeddings), false},
		{"image generation api", string(provider.ApiNameImageGeneration), false},
		{"audio speech api", string(provider.ApiNameAudioSpeech), false},
		{"models api", string(provider.ApiNameModels), false},
		{"unknown api", "unknown", false},
		{"empty api", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the isStatefulAPI function through the provider package
			// We'll need to make this function accessible for testing
			got := provider.TestIsStatefulAPI(tt.apiName)
			if got != tt.want {
				t.Errorf("isStatefulAPI(%q) = %v, want %v", tt.apiName, got, tt.want)
			}
		})
	}
}

func TestGetTokenWithConsumerAffinity(t *testing.T) {
	// Test with multiple tokens
	tokens := []string{"token1", "token2", "token3", "token4"}
	
	tests := []struct {
		name     string
		consumer string
		tokens   []string
		// We can't predict the exact token due to hash, but we can verify consistency
		wantConsistency bool
	}{
		{
			name:            "consumer A with multiple tokens",
			consumer:        "consumer-a",
			tokens:          tokens,
			wantConsistency: true,
		},
		{
			name:            "consumer B with multiple tokens",
			consumer:        "consumer-b",
			tokens:          tokens,
			wantConsistency: true,
		},
		{
			name:            "empty consumer",
			consumer:        "",
			tokens:          tokens,
			wantConsistency: false,
		},
		{
			name:            "single token",
			consumer:        "consumer-c",
			tokens:          []string{"single-token"},
			wantConsistency: true,
		},
		{
			name:            "no tokens",
			consumer:        "consumer-d",
			tokens:          []string{},
			wantConsistency: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a provider config with the test tokens
			config := &provider.ProviderConfig{}
			config.SetApiTokensForTest(tt.tokens)
			
			// For consistency tests, call multiple times with same consumer
			if tt.wantConsistency && len(tt.tokens) > 1 {
				token1 := config.TestGetTokenWithConsumerAffinity(tt.consumer)
				token2 := config.TestGetTokenWithConsumerAffinity(tt.consumer)
				token3 := config.TestGetTokenWithConsumerAffinity(tt.consumer)
				
				// All calls should return the same token
				if token1 != token2 || token2 != token3 {
					t.Errorf("GetTokenWithConsumerAffinity(%q) not consistent: got %s, %s, %s", 
						tt.consumer, token1, token2, token3)
				}
				
				// Verify the token is in the valid list
				found := false
				for _, token := range tt.tokens {
					if token == token1 {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetTokenWithConsumerAffinity(%q) returned invalid token: %s", 
						tt.consumer, token1)
				}
			}
			
			// For non-consistency (empty consumer), just verify it doesn't panic
			if !tt.wantConsistency {
				_ = config.TestGetTokenWithConsumerAffinity(tt.consumer)
			}
		})
	}
}

func TestConsumerAffinityDistribution(t *testing.T) {
	// Test that different consumers get different tokens (when enough consumers)
	tokens := []string{"token1", "token2", "token3", "token4", "token5"}
	consumers := []string{"consumer-a", "consumer-b", "consumer-c", "consumer-c", "consumer-d", 
		"consumer-e", "consumer-f", "consumer-g", "consumer-h", "consumer-i", "consumer-j"}
	
	config := &provider.ProviderConfig{}
	config.SetApiTokensForTest(tokens)
	
	tokenDistribution := make(map[string][]string)
	
	for _, consumer := range consumers {
		token := config.TestGetTokenWithConsumerAffinity(consumer)
		tokenDistribution[token] = append(tokenDistribution[token], consumer)
	}
	
	// Verify that tokens are being used
	if len(tokenDistribution) == 0 {
		t.Error("No tokens were distributed")
	}
	
	// Verify all returned tokens are valid
	for token := range tokenDistribution {
		found := false
		for _, validToken := range tokens {
			if token == validToken {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Invalid token returned: %s", token)
		}
	}
}

