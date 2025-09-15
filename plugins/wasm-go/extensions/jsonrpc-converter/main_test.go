package main

import (
	"testing"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"Short String", "Higress Is an AI-Native API Gateway", 1000, "Higress Is an AI-Native API Gateway"},
		{"Exact Length", "Higress Is an AI-Native API Gateway", 35, "Higress Is an AI-Native API Gateway"},
		{"Truncated String", "Higress Is an AI-Native API Gateway", 20, "Higress Is...(truncated)...PI Gateway"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := McpConverterConfig{MaxHeaderLength: tt.maxLen}
			result := truncateString(tt.input, config)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q; want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}
