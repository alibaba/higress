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

package wrapper

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPasswordURLEncoding verifies that special characters in Redis credentials
// are properly URL-encoded to prevent authentication failures.
// This addresses https://github.com/alibaba/higress/issues/2267
func TestPasswordURLEncoding(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal password without special chars",
			input:    "password123",
			expected: "password123",
		},
		{
			name:     "password with question mark",
			input:    "higress123E?",
			expected: "higress123E%3F",
		},
		{
			name:     "password with at sign",
			input:    "pass@word",
			expected: "pass%40word",
		},
		{
			name:     "password with hash",
			input:    "pass#word",
			expected: "pass%23word",
		},
		{
			name:     "password with ampersand",
			input:    "pass&word",
			expected: "pass%26word",
		},
		{
			name:     "password with multiple special chars",
			input:    "p@ss?w#rd&123",
			expected: "p%40ss%3Fw%23rd%26123",
		},
		{
			name:     "password with space",
			input:    "pass word",
			expected: "pass+word",
		},
		{
			name:     "password with plus sign",
			input:    "pass+word",
			expected: "pass%2Bword",
		},
		{
			name:     "password with equals sign",
			input:    "pass=word",
			expected: "pass%3Dword",
		},
		{
			name:     "empty password",
			input:    "",
			expected: "",
		},
		{
			name:     "password with Chinese characters",
			input:    "密码123",
			expected: "%E5%AF%86%E7%A0%81123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := url.QueryEscape(tc.input)
			assert.Equal(t, tc.expected, result, "URL encoding mismatch for input: %s", tc.input)
		})
	}
}

// TestUsernameURLEncoding verifies URL encoding for username as well
func TestUsernameURLEncoding(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal username",
			input:    "admin",
			expected: "admin",
		},
		{
			name:     "username with special char",
			input:    "admin@domain",
			expected: "admin%40domain",
		},
		{
			name:     "empty username",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := url.QueryEscape(tc.input)
			assert.Equal(t, tc.expected, result, "URL encoding mismatch for username: %s", tc.input)
		})
	}
}
