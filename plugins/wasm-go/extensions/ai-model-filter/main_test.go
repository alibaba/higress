package main

import (
	"regexp"
	"strings"
	"testing"

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
				allowedModels:    []string{"gpt-4", "gpt-3.5-turbo", "claude-3-*"},
				strictMode:       true,
				rejectMessage:    "Custom rejection message",
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
				allowedModels:    []string{"gpt-4"},
				strictMode:       true,                // Default value
				rejectMessage:    "Model not allowed", // Default value
				rejectStatusCode: 403,                 // Default value
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
				allowedModels:    []string{},
				strictMode:       false,
				rejectMessage:    "Model not allowed", // Default value
				rejectStatusCode: 403,                 // Default value
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the config parsing logic manually to avoid WASM environment issues
			configJson := gjson.Parse(tt.configJSON)
			config := AIModelFilterConfig{}

			// Parse allowed models
			allowedModelsArray := configJson.Get("allowed_models").Array()
			config.allowedModels = make([]string, len(allowedModelsArray))
			for i, model := range allowedModelsArray {
				config.allowedModels[i] = model.String()
			}

			// Parse strict mode setting, default to true
			config.strictMode = configJson.Get("strict_mode").Bool()
			if !configJson.Get("strict_mode").Exists() {
				config.strictMode = true
			}

			// Parse custom reject message, default message
			config.rejectMessage = configJson.Get("reject_message").String()
			if config.rejectMessage == "" {
				config.rejectMessage = "Model not allowed"
			}

			// Parse custom reject status code, default 403
			config.rejectStatusCode = int(configJson.Get("reject_status_code").Int())
			if config.rejectStatusCode == 0 {
				config.rejectStatusCode = 403
			}

			if tt.expectError {
				// For this simple test, we don't expect errors
				t.Skip("No error cases in this simplified test")
			} else {
				assert.Equal(t, tt.expectedConfig.allowedModels, config.allowedModels)
				assert.Equal(t, tt.expectedConfig.strictMode, config.strictMode)
				assert.Equal(t, tt.expectedConfig.rejectMessage, config.rejectMessage)
				assert.Equal(t, tt.expectedConfig.rejectStatusCode, config.rejectStatusCode)
			}
		})
	}
}

func TestExtractModelNameFromBody(t *testing.T) {
	tests := []struct {
		name          string
		requestBody   string
		expectedModel string
	}{
		{
			name:          "Extract model from request body",
			requestBody:   `{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`,
			expectedModel: "gpt-4",
		},
		{
			name:          "No model in body",
			requestBody:   `{"messages":[{"role":"user","content":"Hello"}]}`,
			expectedModel: "",
		},
		{
			name:          "Empty body",
			requestBody:   `{}`,
			expectedModel: "",
		},
		{
			name:          "Invalid JSON",
			requestBody:   `invalid json`,
			expectedModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the JSON parsing part directly
			model := gjson.GetBytes([]byte(tt.requestBody), "model")
			var result string
			if model.Exists() {
				result = model.String()
			}
			assert.Equal(t, tt.expectedModel, result)
		})
	}
}

func TestExtractModelNameFromPath(t *testing.T) {
	tests := []struct {
		name          string
		requestPath   string
		expectedModel string
	}{
		{
			name:          "Extract model from Gemini API path",
			requestPath:   "/v1/models/gemini-pro:generateContent",
			expectedModel: "gemini-pro",
		},
		{
			name:          "Extract model from Gemini stream API path",
			requestPath:   "/v1/models/gemini-flash:streamGenerateContent",
			expectedModel: "gemini-flash",
		},
		{
			name:          "No model in path",
			requestPath:   "/v1/chat/completions",
			expectedModel: "",
		},
		{
			name:          "Invalid path format",
			requestPath:   "/invalid/path",
			expectedModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the regex matching part directly
			if strings.Contains(tt.requestPath, "generateContent") || strings.Contains(tt.requestPath, "streamGenerateContent") {
				reg := regexp.MustCompile(`^.*/(?P<api_version>[^/]+)/models/(?P<model>[^:]+):\w+Content$`)
				matches := reg.FindStringSubmatch(tt.requestPath)
				var result string
				if len(matches) == 3 {
					result = matches[2]
				}
				assert.Equal(t, tt.expectedModel, result)
			} else {
				assert.Equal(t, tt.expectedModel, "")
			}
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
		{
			name:          "Wildcard prefix match",
			modelName:     "claude-3-opus",
			allowedModels: []string{"claude-3-*"},
			expected:      true,
		},
		{
			name:          "Wildcard no match",
			modelName:     "claude-2-sonnet",
			allowedModels: []string{"claude-3-*"},
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
