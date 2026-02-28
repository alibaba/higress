package provider

import (
	"fmt"
	"testing"
)

func TestResolvePublisher(t *testing.T) {
	tests := []struct {
		name            string
		model           string
		vertexPublisher string // explicit override
		isExpressMode   bool
		wantPublisherID string
		wantRouting     publisherRouting
	}{
		// Auto-detection by model prefix
		{
			name:            "claude model routes to anthropic rawPredict",
			model:           "claude-3-opus@20240229",
			wantPublisherID: "anthropic",
			wantRouting:     routeRawPredict,
		},
		{
			name:            "claude sonnet model routes to anthropic rawPredict",
			model:           "claude-sonnet-4-5-20250929",
			wantPublisherID: "anthropic",
			wantRouting:     routeRawPredict,
		},
		{
			name:            "mistral model routes to mistralai rawPredict",
			model:           "mistral-large@2407",
			wantPublisherID: "mistralai",
			wantRouting:     routeRawPredict,
		},
		{
			name:            "mistral medium model routes to mistralai rawPredict",
			model:           "mistral-medium-3",
			wantPublisherID: "mistralai",
			wantRouting:     routeRawPredict,
		},
		{
			name:            "codestral model routes to mistralai rawPredict",
			model:           "codestral-2",
			wantPublisherID: "mistralai",
			wantRouting:     routeRawPredict,
		},
		{
			name:            "gemini model routes to OpenAI-compatible",
			model:           "gemini-2.5-pro",
			wantPublisherID: "",
			wantRouting:     routeOpenAICompatible,
		},
		{
			name:            "meta llama model routes to OpenAI-compatible",
			model:           "meta/llama-4-scout-17b-16e-instruct-maas",
			wantPublisherID: "",
			wantRouting:     routeOpenAICompatible,
		},
		{
			name:            "deepseek model routes to OpenAI-compatible",
			model:           "deepseek/deepseek-v3-2-0324",
			wantPublisherID: "",
			wantRouting:     routeOpenAICompatible,
		},
		{
			name:            "qwen model routes to OpenAI-compatible",
			model:           "qwen/qwen-3-235b-a22b",
			wantPublisherID: "",
			wantRouting:     routeOpenAICompatible,
		},
		{
			name:            "unknown model routes to OpenAI-compatible",
			model:           "some-custom-model",
			wantPublisherID: "",
			wantRouting:     routeOpenAICompatible,
		},

		// Express mode: non-claude/non-mistral defaults to native vertex
		{
			name:            "express mode gemini defaults to native vertex",
			model:           "gemini-2.5-flash",
			isExpressMode:   true,
			wantPublisherID: "google",
			wantRouting:     routeNativeVertex,
		},
		{
			name:            "express mode claude still routes to rawPredict",
			model:           "claude-3-opus",
			isExpressMode:   true,
			wantPublisherID: "anthropic",
			wantRouting:     routeRawPredict,
		},
		{
			name:            "express mode mistral still routes to rawPredict",
			model:           "mistral-large@2407",
			isExpressMode:   true,
			wantPublisherID: "mistralai",
			wantRouting:     routeRawPredict,
		},

		// Explicit publisher override
		{
			name:            "explicit publisher override forces rawPredict",
			model:           "custom-model-name",
			vertexPublisher: "mistralai",
			wantPublisherID: "mistralai",
			wantRouting:     routeRawPredict,
		},
		{
			name:            "explicit publisher override overrides prefix detection",
			model:           "gemini-2.5-pro",
			vertexPublisher: "anthropic",
			wantPublisherID: "anthropic",
			wantRouting:     routeRawPredict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProviderConfig{
				vertexPublisher: tt.vertexPublisher,
			}
			if tt.isExpressMode {
				config.apiTokens = []string{"test-key"}
			}

			v := &vertexProvider{config: config}
			got := v.resolvePublisher(tt.model)

			if got.publisherID != tt.wantPublisherID {
				t.Errorf("resolvePublisher(%q).publisherID = %q, want %q", tt.model, got.publisherID, tt.wantPublisherID)
			}
			if got.routing != tt.wantRouting {
				t.Errorf("resolvePublisher(%q).routing = %v, want %v", tt.model, got.routing, tt.wantRouting)
			}
		})
	}
}

func TestGetRawPredictRequestPath(t *testing.T) {
	tests := []struct {
		name          string
		publisherID   string
		modelId       string
		stream        bool
		isExpressMode bool
		wantContains  []string
	}{
		{
			name:        "anthropic non-streaming standard mode",
			publisherID: "anthropic",
			modelId:     "claude-3-opus",
			stream:      false,
			wantContains: []string{
				"/v1/projects/test-project/locations/us-central1/publishers/anthropic/models/claude-3-opus:rawPredict",
			},
		},
		{
			name:        "anthropic streaming standard mode",
			publisherID: "anthropic",
			modelId:     "claude-3-opus",
			stream:      true,
			wantContains: []string{
				"/v1/projects/test-project/locations/us-central1/publishers/anthropic/models/claude-3-opus:streamRawPredict",
			},
		},
		{
			name:        "mistralai non-streaming standard mode",
			publisherID: "mistralai",
			modelId:     "mistral-large",
			stream:      false,
			wantContains: []string{
				"/v1/projects/test-project/locations/us-central1/publishers/mistralai/models/mistral-large:rawPredict",
			},
		},
		{
			name:        "mistralai streaming standard mode",
			publisherID: "mistralai",
			modelId:     "mistral-large",
			stream:      true,
			wantContains: []string{
				"/v1/projects/test-project/locations/us-central1/publishers/mistralai/models/mistral-large:streamRawPredict",
			},
		},
		{
			name:          "anthropic express mode",
			publisherID:   "anthropic",
			modelId:       "claude-3-opus",
			stream:        false,
			isExpressMode: true,
			wantContains: []string{
				"/v1/publishers/anthropic/models/claude-3-opus:rawPredict",
				"key=",
			},
		},
		{
			name:          "mistralai express mode streaming",
			publisherID:   "mistralai",
			modelId:       "mistral-large",
			stream:        true,
			isExpressMode: true,
			wantContains: []string{
				"/v1/publishers/mistralai/models/mistral-large:streamRawPredict",
				"key=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProviderConfig{
				vertexProjectId: "test-project",
				vertexRegion:    "us-central1",
			}
			if tt.isExpressMode {
				config.apiTokens = []string{"test-api-key"}
			}

			v := &vertexProvider{config: config}
			got := v.getRawPredictRequestPath(tt.publisherID, tt.modelId, tt.stream)

			for _, want := range tt.wantContains {
				if !contains(got, want) {
					t.Errorf("getRawPredictRequestPath() = %q, want to contain %q", got, want)
				}
			}
		})
	}
}

func TestGetOpenAICompatibleRequestPath(t *testing.T) {
	v := &vertexProvider{
		config: ProviderConfig{
			vertexProjectId: "my-project",
			vertexRegion:    "us-central1",
		},
	}

	got := v.getOpenAICompatibleRequestPath()
	want := "/v1beta1/projects/my-project/locations/us-central1/endpoints/openapi/chat/completions"

	if got != want {
		t.Errorf("getOpenAICompatibleRequestPath() = %q, want %q", got, want)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPublisherRoutingString(t *testing.T) {
	// Ensure all routing types have meaningful values
	routings := []publisherRouting{routeOpenAICompatible, routeRawPredict, routeNativeVertex}
	for i, r := range routings {
		if int(r) != i {
			t.Errorf("expected routing %d to equal %d", r, i)
		}
	}
}

func TestVertexPublisherPrefixMapCompleteness(t *testing.T) {
	// Verify known publishers are in the prefix map
	expectedPrefixes := map[string]string{
		"claude":    "anthropic",
		"mistral":   "mistralai",
		"codestral": "mistralai",
	}

	for prefix, expectedPublisher := range expectedPrefixes {
		found := false
		for _, entry := range vertexPublisherPrefixMap {
			if entry.prefix == prefix {
				found = true
				if entry.publisher.publisherID != expectedPublisher {
					t.Errorf("prefix %q maps to publisher %q, want %q", prefix, entry.publisher.publisherID, expectedPublisher)
				}
				if entry.publisher.routing != routeRawPredict {
					t.Errorf("prefix %q should use routeRawPredict, got %v", prefix, entry.publisher.routing)
				}
				break
			}
		}
		if !found {
			t.Errorf("prefix %q not found in vertexPublisherPrefixMap", prefix)
		}
	}
}

// TestResolvePublisherOpenModels verifies that open/MaaS models from various publishers
// correctly route to the OpenAI-compatible endpoint
func TestResolvePublisherOpenModels(t *testing.T) {
	v := &vertexProvider{config: ProviderConfig{}}

	openModels := []string{
		"meta/llama-4-scout-17b-16e-instruct-maas",
		"meta/llama-4-maverick-17b-128e-instruct-maas",
		"deepseek/deepseek-v3-2-0324",
		"deepseek/deepseek-r1-0528",
		"qwen/qwen-3-235b-a22b",
		"qwen/qwen-3-coder",
		"glm/glm-4-7",
		"gemini-2.5-pro",
		"gemini-2.5-flash",
	}

	for _, model := range openModels {
		t.Run(fmt.Sprintf("model_%s", model), func(t *testing.T) {
			got := v.resolvePublisher(model)
			if got.routing != routeOpenAICompatible {
				t.Errorf("resolvePublisher(%q).routing = %v, want routeOpenAICompatible", model, got.routing)
			}
		})
	}
}
