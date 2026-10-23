// Copyright (c) 2024 Alibaba Group Holding Ltd.
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

package common

import (
	"encoding/hex"
	"net/url"
	"strings"
	"testing"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/stretchr/testify/require"
)

func TestSha256Hex(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := sha256Hex([]byte(""))
		// SHA256 of empty string
		expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		require.Equal(t, expected, result)
	})

	t.Run("simple string", func(t *testing.T) {
		result := sha256Hex([]byte("hello"))
		expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
		require.Equal(t, expected, result)
	})

	t.Run("unicode string", func(t *testing.T) {
		result := sha256Hex([]byte("你好"))
		// Just verify it returns a valid hex string
		require.Len(t, result, 64)
		_, err := hex.DecodeString(result)
		require.NoError(t, err)
	})
}

func TestHmac256(t *testing.T) {
	t.Run("valid hmac", func(t *testing.T) {
		key := []byte("test-key")
		message := "test-message"
		result, err := hmac256(key, message)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result, 32) // SHA256 produces 32 bytes
	})

	t.Run("empty key", func(t *testing.T) {
		key := []byte("")
		message := "test-message"
		result, err := hmac256(key, message)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result, 32)
	})

	t.Run("empty message", func(t *testing.T) {
		key := []byte("test-key")
		message := ""
		result, err := hmac256(key, message)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result, 32)
	})

	t.Run("verify hmac consistency", func(t *testing.T) {
		key := []byte("test-key")
		message := "test-message"
		result1, err1 := hmac256(key, message)
		result2, err2 := hmac256(key, message)
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.Equal(t, result1, result2)
	})
}

func TestPercentCode(t *testing.T) {
	t.Run("replace plus sign", func(t *testing.T) {
		input := "test+value"
		result := percentCode(input)
		require.Equal(t, "test%20value", result)
	})

	t.Run("replace asterisk", func(t *testing.T) {
		input := "test*value"
		result := percentCode(input)
		require.Equal(t, "test%2Avalue", result)
	})

	t.Run("replace tilde encoding", func(t *testing.T) {
		input := "test%7Evalue"
		result := percentCode(input)
		require.Equal(t, "test~value", result)
	})

	t.Run("multiple replacements", func(t *testing.T) {
		input := "test+value*test%7E"
		result := percentCode(input)
		require.Equal(t, "test%20value%2Atest~", result)
	})

	t.Run("no replacements needed", func(t *testing.T) {
		input := "test-value"
		result := percentCode(input)
		require.Equal(t, "test-value", result)
	})
}

func TestProcessObject(t *testing.T) {
	t.Run("simple string value", func(t *testing.T) {
		result := make(map[string]interface{})
		processObject(result, "key", "value")
		require.Equal(t, "value", result["key"])
	})

	t.Run("simple int value", func(t *testing.T) {
		result := make(map[string]interface{})
		processObject(result, "key", 123)
		require.Equal(t, "123", result["key"])
	})

	t.Run("nil value", func(t *testing.T) {
		result := make(map[string]interface{})
		processObject(result, "key", nil)
		require.Empty(t, result)
	})

	t.Run("map value", func(t *testing.T) {
		result := make(map[string]interface{})
		input := map[string]interface{}{
			"subkey1": "value1",
			"subkey2": "value2",
		}
		processObject(result, "key", input)
		require.Equal(t, "value1", result["key.subkey1"])
		require.Equal(t, "value2", result["key.subkey2"])
	})

	t.Run("array value", func(t *testing.T) {
		result := make(map[string]interface{})
		input := []interface{}{"item1", "item2", "item3"}
		processObject(result, "key", input)
		require.Equal(t, "item1", result["key.1"])
		require.Equal(t, "item2", result["key.2"])
		require.Equal(t, "item3", result["key.3"])
	})

	t.Run("nested map", func(t *testing.T) {
		result := make(map[string]interface{})
		input := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": "value",
			},
		}
		processObject(result, "key", input)
		require.Equal(t, "value", result["key.level1.level2"])
	})

	t.Run("nested array", func(t *testing.T) {
		result := make(map[string]interface{})
		input := []interface{}{
			[]interface{}{"nested1", "nested2"},
		}
		processObject(result, "key", input)
		require.Equal(t, "nested1", result["key.1.1"])
		require.Equal(t, "nested2", result["key.1.2"])
	})

	t.Run("key with leading dot", func(t *testing.T) {
		result := make(map[string]interface{})
		processObject(result, ".key", "value")
		require.Equal(t, "value", result["key"])
	})

	t.Run("byte array value", func(t *testing.T) {
		result := make(map[string]interface{})
		input := []byte("test")
		processObject(result, "key", input)
		require.Equal(t, "test", result["key"])
	})

	t.Run("complex nested structure", func(t *testing.T) {
		result := make(map[string]interface{})
		input := map[string]interface{}{
			"array": []interface{}{
				map[string]interface{}{
					"item": "value",
				},
			},
		}
		processObject(result, "key", input)
		require.Equal(t, "value", result["key.array.1.item"])
	})
}

func TestFormDataToString(t *testing.T) {
	t.Run("simple map", func(t *testing.T) {
		input := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		result := formDataToString(input)
		require.NotNil(t, result)
		require.Contains(t, *result, "key1=value1")
		require.Contains(t, *result, "key2=value2")
	})

	t.Run("map with array", func(t *testing.T) {
		input := map[string]interface{}{
			"key": []interface{}{"item1", "item2"},
		}
		result := formDataToString(input)
		require.NotNil(t, result)
		require.Contains(t, *result, "key.1=item1")
		require.Contains(t, *result, "key.2=item2")
	})

	t.Run("map with nested map", func(t *testing.T) {
		input := map[string]interface{}{
			"key": map[string]interface{}{
				"subkey": "value",
			},
		}
		result := formDataToString(input)
		require.NotNil(t, result)
		require.Contains(t, *result, "key.subkey=value")
	})

	t.Run("empty map", func(t *testing.T) {
		input := map[string]interface{}{}
		result := formDataToString(input)
		require.NotNil(t, result)
		require.Empty(t, *result)
	})

	t.Run("map with nil value", func(t *testing.T) {
		input := map[string]interface{}{
			"key1": "value1",
			"key2": nil,
		}
		result := formDataToString(input)
		require.NotNil(t, result)
		require.Contains(t, *result, "key1=value1")
		require.NotContains(t, *result, "key2")
	})
}

func TestGenerateRequestForText(t *testing.T) {
	config := cfg.AISecurityConfig{
		Host:  "security.example.com",
		AK:    "test-ak",
		SK:    "test-sk",
		Token: "",
	}

	t.Run("basic text request", func(t *testing.T) {
		path, headers, body := GenerateRequestForText(
			config,
			"TextModerationPlus",
			"llm_query_moderation",
			"test content",
			"test-session-id",
		)

		require.NotEmpty(t, path)
		require.True(t, strings.HasPrefix(path, "?"))
		require.Contains(t, path, "Service=llm_query_moderation")

		require.NotEmpty(t, headers)
		headerMap := make(map[string]string)
		for _, h := range headers {
			headerMap[h[0]] = h[1]
		}

		require.Equal(t, "TextModerationPlus", headerMap["x-acs-action"])
		require.Equal(t, "2022-03-02", headerMap["x-acs-version"])
		require.Equal(t, "application/x-www-form-urlencoded", headerMap["content-type"])
		require.Equal(t, cfg.AliyunUserAgent, headerMap["User-Agent"])
		require.Contains(t, headerMap, "Authorization")
		require.Contains(t, headerMap, "x-acs-date")
		require.Contains(t, headerMap, "x-acs-signature-nonce")
		require.Contains(t, headerMap, "x-acs-content-sha256")

		require.NotEmpty(t, body)
		bodyStr := string(body)
		require.Contains(t, bodyStr, "ServiceParameters")
		// Body is URL encoded, so decode it to check content
		decodedBody, err := url.QueryUnescape(bodyStr)
		require.NoError(t, err)
		require.Contains(t, decodedBody, "test content")
		require.Contains(t, decodedBody, "test-session-id")
		require.Contains(t, decodedBody, cfg.AliyunUserAgent)
	})

	t.Run("request with security token", func(t *testing.T) {
		configWithToken := config
		configWithToken.Token = "test-token"
		path, headers, body := GenerateRequestForText(
			configWithToken,
			"TextModerationPlus",
			"llm_query_moderation",
			"test content",
			"test-session-id",
		)

		require.NotEmpty(t, path)
		require.NotEmpty(t, headers)
		headerMap := make(map[string]string)
		for _, h := range headers {
			headerMap[h[0]] = h[1]
		}

		require.Equal(t, "test-token", headerMap["x-acs-security-token"])
		require.NotEmpty(t, body)
	})

	t.Run("empty content", func(t *testing.T) {
		path, headers, body := GenerateRequestForText(
			config,
			"TextModerationPlus",
			"llm_query_moderation",
			"",
			"test-session-id",
		)

		require.NotEmpty(t, path)
		require.NotEmpty(t, headers)
		require.NotEmpty(t, body)
		bodyStr := string(body)
		require.Contains(t, bodyStr, "ServiceParameters")
		decodedBody, err := url.QueryUnescape(bodyStr)
		require.NoError(t, err)
		require.Contains(t, decodedBody, `"content":""`)
	})

	t.Run("different check service", func(t *testing.T) {
		path, headers, body := GenerateRequestForText(
			config,
			"TextModerationPlus",
			"llm_response_moderation",
			"test content",
			"test-session-id",
		)

		require.Contains(t, path, "Service=llm_response_moderation")
		require.NotEmpty(t, headers)
		require.NotEmpty(t, body)
	})
}

func TestGenerateRequestForImage(t *testing.T) {
	config := cfg.AISecurityConfig{
		Host:  "security.example.com",
		AK:    "test-ak",
		SK:    "test-sk",
		Token: "",
	}

	t.Run("image request with URL", func(t *testing.T) {
		path, headers, body := GenerateRequestForImage(
			config,
			"MultiModalGuard",
			"llm_image_moderation",
			"https://example.com/image.jpg",
			"",
		)

		require.NotEmpty(t, path)
		require.True(t, strings.HasPrefix(path, "?"))
		require.Contains(t, path, "Service=llm_image_moderation")

		require.NotEmpty(t, headers)
		headerMap := make(map[string]string)
		for _, h := range headers {
			headerMap[h[0]] = h[1]
		}

		require.Equal(t, "MultiModalGuard", headerMap["x-acs-action"])
		require.Equal(t, "2022-03-02", headerMap["x-acs-version"])
		require.Equal(t, "application/x-www-form-urlencoded", headerMap["content-type"])
		require.Equal(t, cfg.AliyunUserAgent, headerMap["User-Agent"])
		require.Contains(t, headerMap, "Authorization")
		require.Contains(t, headerMap, "x-acs-date")
		require.Contains(t, headerMap, "x-acs-signature-nonce")
		require.Contains(t, headerMap, "x-acs-content-sha256")

		require.NotEmpty(t, body)
		bodyStr := string(body)
		require.Contains(t, bodyStr, "ServiceParameters")
		decodedBody, err := url.QueryUnescape(bodyStr)
		require.NoError(t, err)
		require.Contains(t, decodedBody, "https://example.com/image.jpg")
		require.Contains(t, decodedBody, cfg.AliyunUserAgent)
	})

	t.Run("image request with base64", func(t *testing.T) {
		base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
		path, headers, body := GenerateRequestForImage(
			config,
			"MultiModalGuard",
			"llm_image_moderation",
			"",
			base64Data,
		)

		require.NotEmpty(t, path)
		require.NotEmpty(t, headers)
		require.NotEmpty(t, body)
		bodyStr := string(body)
		require.Contains(t, bodyStr, "ImageBase64Str")
		// Base64 data is URL encoded, decode to check
		decodedBody, err := url.QueryUnescape(bodyStr)
		require.NoError(t, err)
		require.Contains(t, decodedBody, base64Data)
	})

	t.Run("image request with both URL and base64", func(t *testing.T) {
		path, headers, body := GenerateRequestForImage(
			config,
			"MultiModalGuard",
			"llm_image_moderation",
			"https://example.com/image.jpg",
			"base64data",
		)

		require.NotEmpty(t, path)
		require.NotEmpty(t, headers)
		require.NotEmpty(t, body)
		bodyStr := string(body)
		require.Contains(t, bodyStr, "ImageBase64Str")
		decodedBody, err := url.QueryUnescape(bodyStr)
		require.NoError(t, err)
		require.Contains(t, decodedBody, "https://example.com/image.jpg")
		require.Contains(t, decodedBody, "base64data")
	})

	t.Run("image request with security token", func(t *testing.T) {
		configWithToken := config
		configWithToken.Token = "test-token"
		path, headers, body := GenerateRequestForImage(
			configWithToken,
			"MultiModalGuard",
			"llm_image_moderation",
			"https://example.com/image.jpg",
			"",
		)

		require.NotEmpty(t, path)
		require.NotEmpty(t, headers)
		headerMap := make(map[string]string)
		for _, h := range headers {
			headerMap[h[0]] = h[1]
		}

		require.Equal(t, "test-token", headerMap["x-acs-security-token"])
		require.NotEmpty(t, body)
	})

	t.Run("empty image URL and base64", func(t *testing.T) {
		path, headers, body := GenerateRequestForImage(
			config,
			"MultiModalGuard",
			"llm_image_moderation",
			"",
			"",
		)

		require.NotEmpty(t, path)
		require.NotEmpty(t, headers)
		require.NotEmpty(t, body)
		bodyStr := string(body)
		require.Contains(t, bodyStr, "ServiceParameters")
		decodedBody, err := url.QueryUnescape(bodyStr)
		require.NoError(t, err)
		require.Contains(t, decodedBody, cfg.AliyunUserAgent)
		require.NotContains(t, decodedBody, "imageUrls")
		require.NotContains(t, decodedBody, "ImageBase64Str")
	})
}

func TestNewRequest(t *testing.T) {
	// Test newRequest indirectly through GenerateRequestForText
	// Since it's a private function, we test it through public API
	t.Run("request structure", func(t *testing.T) {
		config := cfg.AISecurityConfig{
			Host:  "security.example.com",
			AK:    "test-ak",
			SK:    "test-sk",
			Token: "",
		}

		path, headers, _ := GenerateRequestForText(
			config,
			"TextModerationPlus",
			"llm_query_moderation",
			"test",
			"session-id",
		)

		// Verify that newRequest was called correctly by checking headers
		headerMap := make(map[string]string)
		for _, h := range headers {
			headerMap[h[0]] = h[1]
		}

		// Verify headers set by newRequest
		require.Equal(t, "TextModerationPlus", headerMap["x-acs-action"])
		require.Equal(t, "2022-03-02", headerMap["x-acs-version"])
		require.Contains(t, headerMap, "x-acs-date")
		require.Contains(t, headerMap, "x-acs-signature-nonce")
		require.NotEmpty(t, path)
	})
}

func TestGetAuthorization(t *testing.T) {
	// Test getAuthorization indirectly through GenerateRequestForText
	// Since it's a private function, we test it through public API
	t.Run("authorization header format", func(t *testing.T) {
		config := cfg.AISecurityConfig{
			Host:  "security.example.com",
			AK:    "test-ak",
			SK:    "test-sk",
			Token: "",
		}

		_, headers, _ := GenerateRequestForText(
			config,
			"TextModerationPlus",
			"llm_query_moderation",
			"test content",
			"test-session-id",
		)

		headerMap := make(map[string]string)
		for _, h := range headers {
			headerMap[h[0]] = h[1]
		}

		authHeader := headerMap["Authorization"]
		require.NotEmpty(t, authHeader)
		require.Contains(t, authHeader, "ACS3-HMAC-SHA256")
		require.Contains(t, authHeader, "Credential=test-ak")
		require.Contains(t, authHeader, "SignedHeaders=")
		require.Contains(t, authHeader, "Signature=")

		// Verify content SHA256 is set
		require.Contains(t, headerMap, "x-acs-content-sha256")
		require.Len(t, headerMap["x-acs-content-sha256"], 64) // SHA256 hex string length
	})

	t.Run("authorization with security token", func(t *testing.T) {
		config := cfg.AISecurityConfig{
			Host:  "security.example.com",
			AK:    "test-ak",
			SK:    "test-sk",
			Token: "test-token",
		}

		_, headers, _ := GenerateRequestForText(
			config,
			"TextModerationPlus",
			"llm_query_moderation",
			"test content",
			"test-session-id",
		)

		headerMap := make(map[string]string)
		for _, h := range headers {
			headerMap[h[0]] = h[1]
		}

		require.Equal(t, "test-token", headerMap["x-acs-security-token"])
		require.Contains(t, headerMap, "Authorization")
	})

	t.Run("authorization signature consistency", func(t *testing.T) {
		config := cfg.AISecurityConfig{
			Host:  "security.example.com",
			AK:    "test-ak",
			SK:    "test-sk",
			Token: "",
		}

		// Generate two requests with same content
		_, headers1, body1 := GenerateRequestForText(
			config,
			"TextModerationPlus",
			"llm_query_moderation",
			"test content",
			"test-session-id",
		)

		_, headers2, body2 := GenerateRequestForText(
			config,
			"TextModerationPlus",
			"llm_query_moderation",
			"test content",
			"test-session-id",
		)

		// Bodies should be the same (except for sessionId which is random)
		require.NotEmpty(t, body1)
		require.NotEmpty(t, body2)

		// Headers should have authorization
		headerMap1 := make(map[string]string)
		for _, h := range headers1 {
			headerMap1[h[0]] = h[1]
		}

		headerMap2 := make(map[string]string)
		for _, h := range headers2 {
			headerMap2[h[0]] = h[1]
		}

		require.Contains(t, headerMap1, "Authorization")
		require.Contains(t, headerMap2, "Authorization")
		// Signatures will be different due to nonce and timestamp, but format should be same
		require.Contains(t, headerMap1["Authorization"], "ACS3-HMAC-SHA256")
		require.Contains(t, headerMap2["Authorization"], "ACS3-HMAC-SHA256")
	})
}
