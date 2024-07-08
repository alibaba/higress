// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

func TestParseTLSSecret(t *testing.T) {
	tests := []struct {
		tlsSecret          string
		expectedNamespace  string
		expectedSecretName string
	}{
		{
			tlsSecret:          "example-com-tls",
			expectedNamespace:  "",
			expectedSecretName: "example-com-tls",
		},

		{
			tlsSecret:          "kube-system/example-com-tls",
			expectedNamespace:  "kube-system",
			expectedSecretName: "example-com-tls",
		},
		{
			tlsSecret:          "kube-system/example-com/wildcard",
			expectedNamespace:  "",
			expectedSecretName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.tlsSecret, func(t *testing.T) {
			resultNamespace, resultSecretName := ParseTLSSecret(tt.tlsSecret)
			assert.Equal(t, tt.expectedNamespace, resultNamespace)
			assert.Equal(t, tt.expectedSecretName, resultSecretName)
		})
	}
}
