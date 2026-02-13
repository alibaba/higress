package provider

import (
	"testing"
)

func TestIsStatefulAPI(t *testing.T) {
	tests := []struct {
		name    string
		apiName string
		want    bool
	}{
		// Stateful APIs
		{"responses api", string(ApiNameResponses), true},
		{"files api", string(ApiNameFiles), true},
		{"retrieve file api", string(ApiNameRetrieveFile), true},
		{"retrieve file content api", string(ApiNameRetrieveFileContent), true},
		{"batches api", string(ApiNameBatches), true},
		{"retrieve batch api", string(ApiNameRetrieveBatch), true},
		{"cancel batch api", string(ApiNameCancelBatch), true},
		{"fine tuning jobs api", string(ApiNameFineTuningJobs), true},
		{"retrieve fine tuning job api", string(ApiNameRetrieveFineTuningJob), true},
		{"fine tuning job events api", string(ApiNameFineTuningJobEvents), true},
		{"fine tuning job checkpoints api", string(ApiNameFineTuningJobCheckpoints), true},
		// Stateless APIs
		{"chat completion api", string(ApiNameChatCompletion), false},
		{"completion api", string(ApiNameCompletion), false},
		{"embeddings api", string(ApiNameEmbeddings), false},
		{"image generation api", string(ApiNameImageGeneration), false},
		{"audio speech api", string(ApiNameAudioSpeech), false},
		{"models api", string(ApiNameModels), false},
		{"unknown api", "unknown", false},
		{"empty api", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStatefulAPI(tt.apiName); got != tt.want {
				t.Errorf("isStatefulAPI(%q) = %v, want %v", tt.apiName, got, tt.want)
			}
		})
	}
}

func TestGetTokenWithConsumerAffinity(t *testing.T) {
	// Test with multiple tokens
	tokens := []string{"token1", "token2", "token3", "token4"}

	tests := []struct {
		name            string
		consumer        string
		tokens          []string
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
			config := &ProviderConfig{
				apiTokens: tt.tokens,
			}

			// For consistency tests, call multiple times with same consumer
			if tt.wantConsistency && len(tt.tokens) > 1 {
				token1 := config.GetTokenWithConsumerAffinity(nil, tt.consumer)
				token2 := config.GetTokenWithConsumerAffinity(nil, tt.consumer)
				token3 := config.GetTokenWithConsumerAffinity(nil, tt.consumer)

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
				_ = config.GetTokenWithConsumerAffinity(nil, tt.consumer)
			}
		})
	}
}

func TestConsumerAffinityDistribution(t *testing.T) {
	// Test that different consumers get different tokens (when enough consumers)
	tokens := []string{"token1", "token2", "token3", "token4", "token5"}
	consumers := []string{
		"consumer-a", "consumer-b", "consumer-c", "consumer-c", "consumer-d",
		"consumer-e", "consumer-f", "consumer-g", "consumer-h", "consumer-i", "consumer-j",
	}

	config := &ProviderConfig{
		apiTokens: tokens,
	}

	tokenDistribution := make(map[string][]string)

	for _, consumer := range consumers {
		token := config.GetTokenWithConsumerAffinity(nil, consumer)
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

func TestSelectApiToken(t *testing.T) {
	tokens := []string{"token1", "token2", "token3"}

	config := &ProviderConfig{
		apiTokens: tokens,
	}

	// Test with stateful API and consumer
	t.Run("stateful API with consumer", func(t *testing.T) {
		// This would require mocking the HttpContext, which is complex
		// For now, we'll test the logic through GetTokenWithConsumerAffinity
		token := config.GetTokenWithConsumerAffinity(nil, "test-consumer")
		if token == "" {
			t.Error("Expected non-empty token")
		}
		// Verify consistency
		token2 := config.GetTokenWithConsumerAffinity(nil, "test-consumer")
		if token != token2 {
			t.Errorf("Inconsistent token selection: %s vs %s", token, token2)
		}
	})

	// Test with no consumer (should fall back to random)
	t.Run("no consumer", func(t *testing.T) {
		token := config.GetTokenWithConsumerAffinity(nil, "")
		if token == "" {
			t.Error("Expected non-empty token")
		}
	})
}
