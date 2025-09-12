package main

import (
	"testing"

	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name           string
		configJSON     string
		expectedConfig AIModelFilterConfig
		expectError    bool
	}{
		{
			name: "Valid config with all fields",
			configJSON: `{
				"allowed_models": ["gpt-4", "gpt-3.5-turbo", "claude-3-*"],
				"strict_mode": true,
				"reject_message": "Custom rejection message",
				"reject_status_code": 403
			}`,
			expectedConfig: AIModelFilterConfig{
				allowedModels:   []string{"gpt-4", "gpt-3.5-turbo", "claude-3-*"},
				strictMode:      true,
				rejectMessage:   "Custom rejection message",
				rejectStatusCode: 403,
			},
			expectError: false,
		},
		{
			name: "Config with default values",
			configJSON: `{
				"allowed_models": ["gpt-4"]
			}`,
			expectedConfig: AIModelFilterConfig{
				allowedModels:   []string{"gpt-4"},
				strictMode:      true, // Default value
				rejectMessage:   "Model not allowed", // Default value
				rejectStatusCode: 403, // Default value
			},
			expectError: false,
		},
		{
			name: "Empty allowed models",
			configJSON: `{
				"allowed_models": [],
				"strict_mode": false
			}`,
			expectedConfig: AIModelFilterConfig{
				allowedModels:   []string{},
				strictMode:      false,
				rejectMessage:   "Model not allowed", // Default value
				rejectStatusCode: 403, // Default value
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := AIModelFilterConfig{}
			err := parseConfig(gjson.Parse(tt.configJSON), &config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedConfig.allowedModels, config.allowedModels)
				assert.Equal(t, tt.expectedConfig.strictMode, config.strictMode)
				assert.Equal(t, tt.expectedConfig.rejectMessage, config.rejectMessage)
				assert.Equal(t, tt.expectedConfig.rejectStatusCode, config.rejectStatusCode)
			}
		})
	}
}

func TestExtractModelName(t *testing.T) {
	tests := []struct {
		name         string
		requestPath  string
		requestBody  string
		expectedModel string
	}{
		{
			name:         "Extract model from request body",
			requestPath:  "/v1/chat/completions",
			requestBody:  `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`,
			expectedModel: "gpt-4",
		},
		{
			name:         "Extract model from Gemini API path",
			requestPath:  "/v1/models/gemini-pro:generateContent",
			requestBody:  `{"contents":[{"parts":[{"text":"Hello"}]}]}`,
			expectedModel: "gemini-pro",
		},
		{
			name:         "No model in body or path",
			requestPath:  "/v1/completions",
			requestBody:  `{"messages":[{"role":"user","content":"Hello"}]}`,
			expectedModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock context
			host := test.NewTestHost()
			ctx := host.NewContext()
			ctx.SetContext(RequestPath, tt.requestPath)

			// Extract model name
			modelName := extractModelName(ctx, []byte(tt.requestBody))
			assert.Equal(t, tt.expectedModel, modelName)
		})
	}
}

func TestIsModelAllowed(t *testing.T) {
	tests := []struct {
		name          string
		modelName     string
		allowedModels []string
		expected      bool
	}{
		{
			name:          "Exact match",
			modelName:     "gpt-4",
			allowedModels: []string{"gpt-4", "gpt-3.5-turbo"},
			expected:      true,
		},
		{
			name:          "Wildcard match",
			modelName:     "claude-3-sonnet",
			allowedModels: []string{"gpt-4", "claude-3-*"},
			expected:      true,
		},
		{
			name:          "No match",
			modelName:     "llama-3",
			allowedModels: []string{"gpt-4", "claude-3-*"},
			expected:      false,
		},
		{
			name:          "Empty allowed models",
			modelName:     "gpt-4",
			allowedModels: []string{},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isModelAllowed(tt.modelName, tt.allowedModels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAIModelFilter(t *testing.T) {
	tests := []struct {
		name           string
		config         AIModelFilterConfig
		requestPath    string
		requestBody    string
		expectedAction bool // true for continue, false for reject
	}{
		{
			name: "Allow listed model",
			config: AIModelFilterConfig{
				allowedModels:   []string{"gpt-4", "claude-3-*"},
				strictMode:      true,
				rejectMessage:   "Model not allowed",
				rejectStatusCode: 403,
			},
			requestPath:    "/v1/chat/completions",
			requestBody:    `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`,
			expectedAction: true, // continue
		},
		{
			name: "Allow model with wildcard",
			config: AIModelFilterConfig{
				allowedModels:   []string{"gpt-4", "claude-3-*"},
				strictMode:      true,
				rejectMessage:   "Model not allowed",
				rejectStatusCode: 403,
			},
			requestPath:    "/v1/chat/completions",
			requestBody:    `{"model":"claude-3-sonnet","messages":[{"role":"user","content":"Hello"}]}`,
			expectedAction: true, // continue
		},
		{
			name: "Reject unlisted model",
			config: AIModelFilterConfig{
				allowedModels:   []string{"gpt-4", "claude-3-*"},
				strictMode:      true,
				rejectMessage:   "Model not allowed",
				rejectStatusCode: 403,
			},
			requestPath:    "/v1/chat/completions",
			requestBody:    `{"model":"llama-3","messages":[{"role":"user","content":"Hello"}]}`,
			expectedAction: false, // reject
		},
		{
			name: "Allow model from URL path",
			config: AIModelFilterConfig{
				allowedModels:   []string{"gemini-pro", "gpt-4"},
				strictMode:      true,
				rejectMessage:   "Model not allowed",
				rejectStatusCode: 403,
			},
			requestPath:    "/v1/models/gemini-pro:generateContent",
			requestBody:    `{"contents":[{"parts":[{"text":"Hello"}]}]}`,
			expectedAction: true, // continue
		},
		{
			name: "No model in strict mode",
			config: AIModelFilterConfig{
				allowedModels:   []string{"gpt-4", "claude-3-*"},
				strictMode:      true,
				rejectMessage:   "Model not allowed",
				rejectStatusCode: 403,
			},
			requestPath:    "/v1/completions",
			requestBody:    `{"messages":[{"role":"user","content":"Hello"}]}`,
			expectedAction: false, // reject in strict mode
		},
		{
			name: "No model in non-strict mode",
			config: AIModelFilterConfig{
				allowedModels:   []string{"gpt-4", "claude-3-*"},
				strictMode:      false,
				rejectMessage:   "Model not allowed",
				rejectStatusCode: 403,
			},
			requestPath:    "/v1/completions",
			requestBody:    `{"messages":[{"role":"user","content":"Hello"}]}`,
			expectedAction: true, // continue in non-strict mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test environment
			host := test.NewTestHost()
			ctx := host.NewContext()
			ctx.SetContext(RequestPath, tt.requestPath)

			// Run the request body handler
			result := onHttpRequestBody(ctx, tt.config, []byte(tt.requestBody))

			if tt.expectedAction {
				// Should continue
				assert.Equal(t, uint32(0), result)
			} else {
				// Should reject
				assert.Equal(t, uint32(2), result)
			}
		})
	}
}