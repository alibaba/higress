package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
)

// TestBuildSignatureParamsLine tests signature params line construction
func TestBuildSignatureParamsLine(t *testing.T) {
	tests := []struct {
		name    string
		params  *RFC9421SignatureParams
		wantLen int
	}{
		{
			name: "basic params",
			params: &RFC9421SignatureParams{
				Components: []string{"@method", "@path"},
				Created:    1234567890,
				Expires:    0,
			},
			wantLen: 10, // Should produce non-empty string
		},
		{
			name: "with expires",
			params: &RFC9421SignatureParams{
				Components: []string{"@method", "@path", "content-digest"},
				Created:    1234567890,
				Expires:    1234567900,
			},
			wantLen: 10,
		},
		{
			name: "empty components",
			params: &RFC9421SignatureParams{
				Components: []string{},
				Created:    1234567890,
				Expires:    0,
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSignatureParamsLine(tt.params)
			if len(result) < tt.wantLen {
				t.Errorf("buildSignatureParamsLine() length = %d, want >= %d", len(result), tt.wantLen)
			}
			// Verify format contains created timestamp
			if tt.params.Created > 0 {
				expectedCreated := fmt.Sprintf("created=%d", tt.params.Created)
				if !contains(result, expectedCreated) {
					t.Errorf("Result should contain created timestamp: %s", result)
				}
			}
		})
	}
}

// TestValidateSignature tests HMAC signature validation
func TestValidateSignature(t *testing.T) {
	secret := "test-secret-key"
	signatureBase := `"@method": POST
"@path": /v1/chat/completions
"content-digest": sha-256=:abcd1234:
"@signature-params": ("@method" "@path" "content-digest");created=1234567890`

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signatureBase))
	validSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name          string
		signatureBase string
		signature     string
		secret        string
		wantError     bool
	}{
		{
			name:          "valid signature",
			signatureBase: signatureBase,
			signature:     validSignature,
			secret:        secret,
			wantError:     false,
		},
		{
			name:          "invalid signature",
			signatureBase: signatureBase,
			signature:     "invalid-signature",
			secret:        secret,
			wantError:     true,
		},
		{
			name:          "wrong secret",
			signatureBase: signatureBase,
			signature:     validSignature,
			secret:        "wrong-secret",
			wantError:     true,
		},
		{
			name:          "empty signature",
			signatureBase: signatureBase,
			signature:     "",
			secret:        secret,
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSignature(tt.signatureBase, tt.signature, tt.secret)
			if (err != nil) != tt.wantError {
				t.Errorf("validateSignature() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestMatchesPatternAdditional adds more test cases for matchesPattern
func TestMatchesPatternAdditional(t *testing.T) {
	tests := []struct {
		pattern  string
		toolName string
		expected bool
	}{
		// Edge cases
		{"", "anything", false},
		{"test", "", false},
		{"*", "", true},
		
		// Multiple wildcards (should not match - only single wildcard supported)
		{"*_test_*", "prefix_test_suffix", false},
		
		// Case sensitivity
		{"Read_*", "read_email", false},
		{"read_*", "Read_Email", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s vs %s", tt.pattern, tt.toolName), func(t *testing.T) {
			result := matchesPattern(tt.pattern, tt.toolName)
			if result != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.toolName, result, tt.expected)
			}
		})
	}
}
