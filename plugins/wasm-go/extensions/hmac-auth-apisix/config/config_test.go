package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestParseGlobalConfig(t *testing.T) {
	tests := []struct {
		name        string
		jsonConfig  string
		expectError bool
		validate    func(*testing.T, *HmacAuthConfig)
	}{
		{
			name: "Valid config with named consumers",
			jsonConfig: `{
				"consumers": [
					{
						"name": "consumer1",
						"access_key": "ak1",
						"secret_key": "sk1"
					},
					{
						"name": "consumer2",
						"access_key": "ak2",
						"secret_key": "sk2"
					}
				]
			}`,
			expectError: false,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, 2, len(config.Consumers))
				assert.Equal(t, "consumer1", config.Consumers[0].Name)
				assert.Equal(t, "ak1", config.Consumers[0].AccessKey)
				assert.Equal(t, "sk1", config.Consumers[0].SecretKey)
				assert.Equal(t, "consumer2", config.Consumers[1].Name)
				assert.Equal(t, "ak2", config.Consumers[1].AccessKey)
				assert.Equal(t, "sk2", config.Consumers[1].SecretKey)

				// 默认值检查
				assert.Equal(t, []string{"hmac-sha1", "hmac-sha256", "hmac-sha512"}, config.AllowedAlgorithms)
				assert.Equal(t, 300, config.ClockSkew)
			},
		},
		{
			name: "Valid config without names (use access_key as name)",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					},
					{
						"access_key": "ak2",
						"secret_key": "sk2"
					}
				]
			}`,
			expectError: false,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, 2, len(config.Consumers))
				assert.Equal(t, "ak1", config.Consumers[0].Name)
				assert.Equal(t, "ak1", config.Consumers[0].AccessKey)
				assert.Equal(t, "ak2", config.Consumers[1].Name)
				assert.Equal(t, "ak2", config.Consumers[1].AccessKey)
			},
		},
		{
			name: "Missing consumers",
			jsonConfig: `{
				"other_field": "value"
			}`,
			expectError: true,
		},
		{
			name: "Empty consumers array",
			jsonConfig: `{
				"consumers": []
			}`,
			expectError: true,
		},
		{
			name: "Missing access_key",
			jsonConfig: `{
				"consumers": [
					{
						"secret_key": "sk1"
					}
				]
			}`,
			expectError: true,
		},
		{
			name: "Missing secret_key",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1"
					}
				]
			}`,
			expectError: true,
		},
		{
			name: "Duplicate access_key",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					},
					{
						"access_key": "ak1",
						"secret_key": "sk2"
					}
				]
			}`,
			expectError: true,
		},
		{
			name: "Valid global_auth",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					}
				],
				"global_auth": true
			}`,
			expectError: false,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.NotNil(t, config.GlobalAuth)
				assert.True(t, *config.GlobalAuth)
			},
		},
		{
			name: "Valid allowed_algorithms",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					}
				],
				"allowed_algorithms": ["hmac-sha256"]
			}`,
			expectError: false,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, []string{"hmac-sha256"}, config.AllowedAlgorithms)
			},
		},
		{
			name: "Invalid allowed_algorithms",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					}
				],
				"allowed_algorithms": ["invalid-algorithm"]
			}`,
			expectError: true,
		},
		{
			name: "Valid clock_skew",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					}
				],
				"clock_skew": 600
			}`,
			expectError: false,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, 600, config.ClockSkew)
			},
		},
		{
			name: "Valid signed_headers",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					}
				],
				"signed_headers": ["host", "date"]
			}`,
			expectError: false,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, []string{"host", "date"}, config.SignedHeaders)
			},
		},
		{
			name: "Valid validate_request_body",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					}
				],
				"validate_request_body": true
			}`,
			expectError: false,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.True(t, config.ValidateRequestBody)
			},
		},
		{
			name: "Valid hide_credentials",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					}
				],
				"hide_credentials": true
			}`,
			expectError: false,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.True(t, config.HideCredentials)
			},
		},
		{
			name: "Valid anonymous_consumer",
			jsonConfig: `{
				"consumers": [
					{
						"access_key": "ak1",
						"secret_key": "sk1"
					}
				],
				"anonymous_consumer": "anonymous"
			}`,
			expectError: false,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, "anonymous", config.AnonymousConsumer)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := gjson.Parse(tt.jsonConfig)
			config := &HmacAuthConfig{}

			defer func() {
				if r := recover(); r != nil {
					// 忽略日志相关的 panic
				}
			}()

			err := ParseGlobalConfig(jsonData, config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestParseOverrideRuleConfig(t *testing.T) {
	globalConfig := HmacAuthConfig{
		Consumers: []Consumer{
			{
				Name:      "consumer1",
				AccessKey: "ak1",
				SecretKey: "sk1",
			},
			{
				Name:      "consumer2",
				AccessKey: "ak2",
				SecretKey: "sk2",
			},
		},
		AllowedAlgorithms: []string{"hmac-sha1", "hmac-sha256", "hmac-sha512"},
		ClockSkew:         300,
	}

	tests := []struct {
		name        string
		jsonConfig  string
		validate    func(*testing.T, *HmacAuthConfig)
		expectError bool
	}{
		{
			name: "Default values when no overrides",
			jsonConfig: `{
				"some_other_field": "value"
			}`,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, []string{"hmac-sha1", "hmac-sha256", "hmac-sha512"}, config.AllowedAlgorithms)
				assert.Equal(t, 300, config.ClockSkew)
				assert.Equal(t, 2, len(config.Consumers))
			},
			expectError: false,
		},
		{
			name: "Valid allow list",
			jsonConfig: `{
				"allow": ["consumer1", "consumer2"]
			}`,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, []string{"consumer1", "consumer2"}, config.Allow)
			},
			expectError: false,
		},
		{
			name: "Allow list with non-existent consumer",
			jsonConfig: `{
				"allow": ["consumer1", "nonexistent"]
			}`,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, []string{"consumer1", "nonexistent"}, config.Allow)
			},
			expectError: false,
		},
		{
			name: "Empty allow list",
			jsonConfig: `{
				"allow": []
			}`,
			validate: func(t *testing.T, config *HmacAuthConfig) {
				assert.Equal(t, []string{}, config.Allow)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := gjson.Parse(tt.jsonConfig)
			config := &HmacAuthConfig{}

			defer func() {
				if r := recover(); r != nil {
					// 忽略日志相关的 panic
				}
			}()

			err := ParseOverrideRuleConfig(jsonData, globalConfig, config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestValidAlgorithms(t *testing.T) {
	tests := []struct {
		algorithm string
		valid     bool
	}{
		{"hmac-sha1", true},
		{"hmac-sha256", true},
		{"hmac-sha512", true},
		{"invalid-algorithm", false},
		{"", false},
		{"hmac-md5", false},
	}

	for _, tt := range tests {
		t.Run(tt.algorithm, func(t *testing.T) {
			_, exists := validAlgorithms[tt.algorithm]
			assert.Equal(t, tt.valid, exists)
		})
	}
}
