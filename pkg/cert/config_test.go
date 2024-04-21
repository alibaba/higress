package cert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchSecretNameByDomain(t *testing.T) {
	tests := []struct {
		name          string
		domain        string
		credentialCfg []CredentialEntry
		expected      string
	}{
		{
			name:   "Exact match",
			domain: "example.com",
			credentialCfg: []CredentialEntry{
				{
					Domains:   []string{"example.com"},
					TLSSecret: "example-com-tls",
				},
			},
			expected: "example-com-tls",
		},

		{
			name:   "Exact match ignore case ",
			domain: "eXample.com",
			credentialCfg: []CredentialEntry{
				{
					Domains:   []string{"example.com"},
					TLSSecret: "example-com-tls",
				},
			},
			expected: "example-com-tls",
		},
		{
			name:   "Wildcard match",
			domain: "sub.example.com",
			credentialCfg: []CredentialEntry{
				{
					Domains:   []string{"*.example.com"},
					TLSSecret: "wildcard-example-com-tls",
				},
			},
			expected: "wildcard-example-com-tls",
		},

		{
			name:   "Wildcard match ignore case",
			domain: "sub.Example.com",
			credentialCfg: []CredentialEntry{
				{
					Domains:   []string{"*.example.com"},
					TLSSecret: "wildcard-example-com-tls",
				},
			},
			expected: "wildcard-example-com-tls",
		},
		{
			name:   "* match",
			domain: "blog.example.co.uk",
			credentialCfg: []CredentialEntry{
				{
					Domains:   []string{"*"},
					TLSSecret: "blog-co-uk-tls",
				},
			},
			expected: "blog-co-uk-tls",
		},
		{
			name:   "No match",
			domain: "unknown.com",
			credentialCfg: []CredentialEntry{
				{
					Domains:   []string{"example.com"},
					TLSSecret: "example-com-tls",
				},
			},
			expected: "",
		},
		{
			name:   "Multiple matches - first match wins",
			domain: "example.com",
			credentialCfg: []CredentialEntry{
				{
					Domains:   []string{"example.com"},
					TLSSecret: "example-com-tls",
				},
				{
					Domains:   []string{"*.example.com"},
					TLSSecret: "wildcard-example-com-tls",
				},
			},
			expected: "example-com-tls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{CredentialConfig: tt.credentialCfg}
			result := cfg.MatchSecretNameByDomain(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}
