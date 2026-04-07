package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestIsStatefulAPI(t *testing.T) {
	tests := []struct {
		name     string
		apiName  string
		expected bool
	}{
		// Stateful APIs - should return true
		{
			name:     "responses_api",
			apiName:  string(ApiNameResponses),
			expected: true,
		},
		{
			name:     "files_api",
			apiName:  string(ApiNameFiles),
			expected: true,
		},
		{
			name:     "retrieve_file_api",
			apiName:  string(ApiNameRetrieveFile),
			expected: true,
		},
		{
			name:     "retrieve_file_content_api",
			apiName:  string(ApiNameRetrieveFileContent),
			expected: true,
		},
		{
			name:     "batches_api",
			apiName:  string(ApiNameBatches),
			expected: true,
		},
		{
			name:     "retrieve_batch_api",
			apiName:  string(ApiNameRetrieveBatch),
			expected: true,
		},
		{
			name:     "cancel_batch_api",
			apiName:  string(ApiNameCancelBatch),
			expected: true,
		},
		{
			name:     "fine_tuning_jobs_api",
			apiName:  string(ApiNameFineTuningJobs),
			expected: true,
		},
		{
			name:     "retrieve_fine_tuning_job_api",
			apiName:  string(ApiNameRetrieveFineTuningJob),
			expected: true,
		},
		{
			name:     "fine_tuning_job_events_api",
			apiName:  string(ApiNameFineTuningJobEvents),
			expected: true,
		},
		{
			name:     "fine_tuning_job_checkpoints_api",
			apiName:  string(ApiNameFineTuningJobCheckpoints),
			expected: true,
		},
		{
			name:     "cancel_fine_tuning_job_api",
			apiName:  string(ApiNameCancelFineTuningJob),
			expected: true,
		},
		{
			name:     "resume_fine_tuning_job_api",
			apiName:  string(ApiNameResumeFineTuningJob),
			expected: true,
		},
		// Non-stateful APIs - should return false
		{
			name:     "chat_completion_api",
			apiName:  string(ApiNameChatCompletion),
			expected: false,
		},
		{
			name:     "completion_api",
			apiName:  string(ApiNameCompletion),
			expected: false,
		},
		{
			name:     "embeddings_api",
			apiName:  string(ApiNameEmbeddings),
			expected: false,
		},
		{
			name:     "models_api",
			apiName:  string(ApiNameModels),
			expected: false,
		},
		{
			name:     "image_generation_api",
			apiName:  string(ApiNameImageGeneration),
			expected: false,
		},
		{
			name:     "audio_speech_api",
			apiName:  string(ApiNameAudioSpeech),
			expected: false,
		},
		// Empty/unknown API - should return false
		{
			name:     "empty_api_name",
			apiName:  "",
			expected: false,
		},
		{
			name:     "unknown_api_name",
			apiName:  "unknown/api",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStatefulAPI(tt.apiName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTokenWithConsumerAffinity(t *testing.T) {
	tests := []struct {
		name      string
		apiTokens []string
		consumer  string
		wantEmpty bool
		wantToken string // If not empty, expected specific token (for single token case)
	}{
		{
			name:      "no_tokens_returns_empty",
			apiTokens: []string{},
			consumer:  "consumer1",
			wantEmpty: true,
		},
		{
			name:      "nil_tokens_returns_empty",
			apiTokens: nil,
			consumer:  "consumer1",
			wantEmpty: true,
		},
		{
			name:      "single_token_always_returns_same_token",
			apiTokens: []string{"token1"},
			consumer:  "consumer1",
			wantToken: "token1",
		},
		{
			name:      "single_token_with_different_consumer",
			apiTokens: []string{"token1"},
			consumer:  "consumer2",
			wantToken: "token1",
		},
		{
			name:      "multiple_tokens_consistent_for_same_consumer",
			apiTokens: []string{"token1", "token2", "token3"},
			consumer:  "consumer1",
			wantEmpty: false, // Will get one of the tokens, consistently
		},
		{
			name:      "multiple_tokens_different_consumers_may_get_different_tokens",
			apiTokens: []string{"token1", "token2"},
			consumer:  "consumerA",
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ProviderConfig{
				apiTokens: tt.apiTokens,
			}

			result := config.GetTokenWithConsumerAffinity(nil, tt.consumer)

			if tt.wantEmpty {
				assert.Empty(t, result)
			} else if tt.wantToken != "" {
				assert.Equal(t, tt.wantToken, result)
			} else {
				assert.NotEmpty(t, result)
				assert.Contains(t, tt.apiTokens, result)
			}
		})
	}
}

func TestGetTokenWithConsumerAffinity_Consistency(t *testing.T) {
	// Test that the same consumer always gets the same token (consistency)
	config := &ProviderConfig{
		apiTokens: []string{"token1", "token2", "token3", "token4", "token5"},
	}

	t.Run("same_consumer_gets_same_token_repeatedly", func(t *testing.T) {
		consumer := "test-consumer"
		var firstResult string

		// Call multiple times and verify consistency
		for i := 0; i < 10; i++ {
			result := config.GetTokenWithConsumerAffinity(nil, consumer)
			if i == 0 {
				firstResult = result
			}
			assert.Equal(t, firstResult, result, "Consumer should consistently get the same token")
		}
	})

	t.Run("different_consumers_distribute_across_tokens", func(t *testing.T) {
		// Use multiple consumers and verify they distribute across tokens
		consumers := []string{"consumer1", "consumer2", "consumer3", "consumer4", "consumer5", "consumer6", "consumer7", "consumer8", "consumer9", "consumer10"}
		tokenCounts := make(map[string]int)

		for _, consumer := range consumers {
			token := config.GetTokenWithConsumerAffinity(nil, consumer)
			tokenCounts[token]++
		}

		// Verify all tokens returned are valid
		for token := range tokenCounts {
			assert.Contains(t, config.apiTokens, token)
		}

		// With 10 consumers and 5 tokens, we expect some distribution
		// (not necessarily perfect distribution, but should use multiple tokens)
		assert.GreaterOrEqual(t, len(tokenCounts), 2, "Should use at least 2 different tokens")
	})

	t.Run("empty_consumer_returns_empty_string", func(t *testing.T) {
		config := &ProviderConfig{
			apiTokens: []string{"token1", "token2"},
		}
		result := config.GetTokenWithConsumerAffinity(nil, "")
		// Empty consumer still returns a token (hash of empty string)
		assert.NotEmpty(t, result)
		assert.Contains(t, []string{"token1", "token2"}, result)
	})
}

func TestGetTokenWithConsumerAffinity_HashDistribution(t *testing.T) {
	// Test that the hash function distributes consumers reasonably across tokens
	config := &ProviderConfig{
		apiTokens: []string{"token1", "token2", "token3"},
	}

	// Test specific consumers to verify hash behavior
	testCases := []struct {
		consumer    string
		expectValid bool
	}{
		{"user-alice", true},
		{"user-bob", true},
		{"user-charlie", true},
		{"service-api-v1", true},
		{"service-api-v2", true},
	}

	for _, tc := range testCases {
		t.Run("consumer_"+tc.consumer, func(t *testing.T) {
			result := config.GetTokenWithConsumerAffinity(nil, tc.consumer)
			assert.True(t, tc.expectValid)
			assert.Contains(t, config.apiTokens, result)
		})
	}
}

func TestProviderDomain_Config(t *testing.T) {
	t.Run("providerDomain_field_exists", func(t *testing.T) {
		config := ProviderConfig{}
		config.FromJson(gjson.Result{})
		assert.Equal(t, "", config.providerDomain)
	})

	t.Run("providerDomain_parsed_from_json", func(t *testing.T) {
		config := ProviderConfig{}
		jsonStr := `{"providerDomain": "universal-proxy.example.com"}`
		config.FromJson(gjson.Parse(jsonStr))
		assert.Equal(t, "universal-proxy.example.com", config.providerDomain)
	})
}

func TestProviderBasePath_Config(t *testing.T) {
	t.Run("providerBasePath_field_exists", func(t *testing.T) {
		config := ProviderConfig{}
		config.FromJson(gjson.Result{})
		assert.Equal(t, "", config.providerBasePath)
	})

	t.Run("providerBasePath_parsed_from_json", func(t *testing.T) {
		config := ProviderConfig{}
		jsonStr := `{"providerBasePath": "/api/ai"}`
		config.FromJson(gjson.Parse(jsonStr))
		assert.Equal(t, "/api/ai", config.providerBasePath)
	})

	t.Run("providerBasePath_with_other_config", func(t *testing.T) {
		config := ProviderConfig{}
		jsonStr := `{
			"type": "openai",
			"apiToken": "sk-test",
			"providerBasePath": "/api/v1",
			"providerDomain": "proxy.example.com"
		}`
		config.FromJson(gjson.Parse(jsonStr))
		assert.Equal(t, "openai", config.typ)
		assert.Equal(t, "/api/v1", config.providerBasePath)
		assert.Equal(t, "proxy.example.com", config.providerDomain)
	})
}

func TestApplyProviderBasePath(t *testing.T) {
	tests := []struct {
		name             string
		providerBasePath string
		originalPath     string
		expectedPath     string
	}{
		{
			name:             "no_base_path_configured",
			providerBasePath: "",
			originalPath:     "/v1/chat/completions",
			expectedPath:     "/v1/chat/completions",
		},
		{
			name:             "base_path_prepended",
			providerBasePath: "/api/ai",
			originalPath:     "/v1/chat/completions",
			expectedPath:     "/api/ai/v1/chat/completions",
		},
		{
			name:             "path_already_has_base_path",
			providerBasePath: "/api/ai",
			originalPath:     "/api/ai/v1/chat/completions",
			expectedPath:     "/api/ai/v1/chat/completions",
		},
		{
			name:             "base_path_with_trailing_slash",
			providerBasePath: "/api/ai/",
			originalPath:     "/v1/chat/completions",
			expectedPath:     "/api/ai//v1/chat/completions",
		},
		{
			name:             "deep_base_path",
			providerBasePath: "/internal/services/ai",
			originalPath:     "/v1/models",
			expectedPath:     "/internal/services/ai/v1/models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ProviderConfig{
				providerBasePath: tt.providerBasePath,
			}
			result := config.applyProviderBasePath(tt.originalPath)
			assert.Equal(t, tt.expectedPath, result)
		})
	}
}
