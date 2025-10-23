// Copyright (c) 2025 Alibaba Group Holding Ltd.
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

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestComputeContentDigest(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple content",
			content:  "Hello, World!",
			expected: "dffd6021",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "e3b0c442",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeContentDigest(tt.content)
			if result != tt.expected {
				t.Errorf("ComputeContentDigest() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWrapWithSecurityTag(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		tagType        string
		includeDigest  bool
		expectedOutput string
	}{
		{
			name:           "wrap user message without digest",
			content:        "Hello",
			tagType:        "user",
			includeDigest:  false,
			expectedOutput: "<a2as:user>Hello</a2as:user>",
		},
		{
			name:           "wrap user message with digest",
			content:        "Hello",
			tagType:        "user",
			includeDigest:  true,
			expectedOutput: "<a2as:user:185f8db3>Hello</a2as:user:185f8db3>",
		},
		{
			name:           "empty content",
			content:        "",
			tagType:        "user",
			includeDigest:  false,
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithSecurityTag(tt.content, tt.tagType, tt.includeDigest)
			if result != tt.expectedOutput {
				t.Errorf("WrapWithSecurityTag() = %v, want %v", result, tt.expectedOutput)
			}
		})
	}
}

func TestBuildDefenseBlock(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "with template",
			template: "Test defense",
			expected: "<a2as:defense>\nTest defense\n</a2as:defense>",
		},
		{
			name:     "empty template",
			template: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildDefenseBlock(tt.template)
			if result != tt.expected {
				t.Errorf("BuildDefenseBlock() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildPolicyBlock(t *testing.T) {
	tests := []struct {
		name     string
		policies []PolicyRule
		expected string
	}{
		{
			name: "single policy",
			policies: []PolicyRule{
				{
					Name:     "TEST_POLICY",
					Content:  "Test content",
					Severity: "critical",
				},
			},
			expected: "<a2as:policy>\nPOLICIES:\n1. TEST_POLICY [CRITICAL]: Test content\n</a2as:policy>",
		},
		{
			name:     "empty policies",
			policies: []PolicyRule{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPolicyBlock(tt.policies)
			if result != tt.expected {
				t.Errorf("BuildPolicyBlock() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name       string
		jsonConfig string
		wantErr    bool
		validate   func(*A2ASConfig) bool
	}{
		{
			name: "basic config",
			jsonConfig: `{
				"securityBoundaries": {
					"enabled": true,
					"wrapUserMessages": true
				}
			}`,
			wantErr: false,
			validate: func(config *A2ASConfig) bool {
				return config.SecurityBoundaries.Enabled == true &&
					config.SecurityBoundaries.WrapUserMessages == true
			},
		},
		{
			name: "default values",
			jsonConfig: `{
				"securityBoundaries": {
					"enabled": true
				}
			}`,
			wantErr: false,
			validate: func(config *A2ASConfig) bool {
				return config.Protocol == "openai" &&
					config.InContextDefenses.Position == "as_system"
			},
		},
		{
			name: "invalid protocol",
			jsonConfig: `{
				"protocol": "invalid"
			}`,
			wantErr: true,
			validate: func(config *A2ASConfig) bool {
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &A2ASConfig{}
			jsonResult := gjson.Parse(tt.jsonConfig)

			err := ParseConfig(jsonResult, config)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			err = config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !tt.validate(config) {
				t.Errorf("Config validation failed for test %s", tt.name)
			}
		})
	}
}

func TestIsToolAllowed(t *testing.T) {
	tests := []struct {
		name        string
		permissions AgentPermissions
		toolName    string
		expected    bool
	}{
		{
			name: "explicitly allowed",
			permissions: AgentPermissions{
				AllowedTools: []string{"tool1", "tool2"},
				DeniedTools:  []string{},
			},
			toolName: "tool1",
			expected: true,
		},
		{
			name: "explicitly denied",
			permissions: AgentPermissions{
				AllowedTools: []string{"tool1", "tool2"},
				DeniedTools:  []string{"tool3"},
			},
			toolName: "tool3",
			expected: false,
		},
		{
			name: "no allow list - default allow",
			permissions: AgentPermissions{
				AllowedTools: []string{},
				DeniedTools:  []string{"tool3"},
			},
			toolName: "tool1",
			expected: true,
		},
		{
			name: "wildcard allow",
			permissions: AgentPermissions{
				AllowedTools: []string{"*"},
				DeniedTools:  []string{},
			},
			toolName: "any_tool",
			expected: true,
		},
		{
			name: "wildcard deny",
			permissions: AgentPermissions{
				AllowedTools: []string{"tool1"},
				DeniedTools:  []string{"*"},
			},
			toolName: "tool1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isToolAllowed(tt.permissions, tt.toolName)
			if result != tt.expected {
				t.Errorf("isToolAllowed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsChatCompletionRequest(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name: "valid chat completion",
			body: `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "Hello"}
				]
			}`,
			expected: true,
		},
		{
			name: "not a chat completion",
			body: `{
				"prompt": "Hello"
			}`,
			expected: false,
		},
		{
			name:     "empty body",
			body:     `{}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isChatCompletionRequest([]byte(tt.body))
			if result != tt.expected {
				t.Errorf("isChatCompletionRequest() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMergeConsumerConfig 测试消费者配置合并
func TestMergeConsumerConfig(t *testing.T) {
	// 创建全局配置
	globalConfig := A2ASConfig{
		SecurityBoundaries: SecurityBoundariesConfig{
			Enabled:          true,
			WrapUserMessages: true,
		},
		BehaviorCertificates: BehaviorCertificatesConfig{
			Enabled: true,
			Permissions: AgentPermissions{
				AllowedTools: []string{"global_tool"},
			},
		},
		ConsumerConfigs: map[string]*ConsumerA2ASConfig{
			"consumer_strict": {
				SecurityBoundaries: &SecurityBoundariesConfig{
					Enabled:              true,
					WrapUserMessages:     true,
					IncludeContentDigest: true, // 消费者特定配置
				},
				BehaviorCertificates: &BehaviorCertificatesConfig{
					Enabled: true,
					Permissions: AgentPermissions{
						AllowedTools: []string{"restricted_tool"}, // 更严格的工具列表
					},
				},
			},
		},
	}

	tests := []struct {
		name         string
		consumerName string
		checkFunc    func(merged A2ASConfig) bool
	}{
		{
			name:         "unknown consumer uses global config",
			consumerName: "unknown_consumer",
			checkFunc: func(merged A2ASConfig) bool {
				return len(merged.BehaviorCertificates.Permissions.AllowedTools) == 1 &&
					merged.BehaviorCertificates.Permissions.AllowedTools[0] == "global_tool"
			},
		},
		{
			name:         "empty consumer name uses global config",
			consumerName: "",
			checkFunc: func(merged A2ASConfig) bool {
				return !merged.SecurityBoundaries.IncludeContentDigest
			},
		},
		{
			name:         "known consumer uses merged config",
			consumerName: "consumer_strict",
			checkFunc: func(merged A2ASConfig) bool {
				return merged.SecurityBoundaries.IncludeContentDigest &&
					len(merged.BehaviorCertificates.Permissions.AllowedTools) == 1 &&
					merged.BehaviorCertificates.Permissions.AllowedTools[0] == "restricted_tool"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := globalConfig.MergeConsumerConfig(tt.consumerName)
			if !tt.checkFunc(merged) {
				t.Errorf("MergeConsumerConfig() produced unexpected result")
			}
		})
	}
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    A2ASConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: A2ASConfig{
				Protocol: "openai",
				InContextDefenses: InContextDefensesConfig{
					Enabled:  true,
					Position: "as_system",
					Template: "test template",
				},
			},
			expectErr: false,
		},
		{
			name: "invalid protocol",
			config: A2ASConfig{
				Protocol: "invalid",
			},
			expectErr: true,
			errMsg:    "protocol must be",
		},
		{
			name: "invalid defense position",
			config: A2ASConfig{
				Protocol: "openai",
				InContextDefenses: InContextDefensesConfig{
					Enabled:  true,
					Position: "invalid_position",
				},
			},
			expectErr: true,
			errMsg:    "position must be",
		},
		{
			name: "template too long",
			config: A2ASConfig{
				Protocol: "openai",
				InContextDefenses: InContextDefensesConfig{
					Enabled:  true,
					Position: "as_system",
					Template: string(make([]byte, 10001)),
				},
			},
			expectErr: true,
			errMsg:    "too long",
		},
		{
			name: "too many policies",
			config: A2ASConfig{
				Protocol: "openai",
				CodifiedPolicies: CodifiedPoliciesConfig{
					Enabled:  true,
					Position: "as_system",
					Policies: make([]PolicyRule, 51),
				},
			},
			expectErr: true,
			errMsg:    "too many policies",
		},
		{
			name: "auth enabled without secret and unsigned not allowed",
			config: A2ASConfig{
				Protocol: "openai",
			AuthenticatedPrompts: AuthenticatedPromptsConfig{
				Enabled:       true,
				Mode:          "simple",
				AllowUnsigned: false,
				SharedSecret:  "",
				Algorithm:     "hmac-sha256",
			},
			},
			expectErr: true,
			errMsg:    "sharedSecret is required",
		},
		{
			name: "auth enabled with unsigned allowed",
		config: A2ASConfig{
			Protocol: "openai",
			AuthenticatedPrompts: AuthenticatedPromptsConfig{
				Enabled:       true,
				Mode:          "simple",
				AllowUnsigned: true,
				SharedSecret:  "",
				Algorithm:     "hmac-sha256",
			},
		},
		expectErr: false,
	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr && err == nil {
				t.Errorf("Validate() expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
			if tt.expectErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestHMACSignatureGeneration 测试 HMAC 签名生成（辅助测试）
// 这个测试帮助理解如何生成正确的签名
func TestHMACSignatureGeneration(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
	secret := "test-secret-key"

	// 生成 HMAC-SHA256 签名
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	
	hexSignature := hex.EncodeToString(mac.Sum(nil))
	base64Signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// 验证签名格式
	if len(hexSignature) != 64 { // SHA256 产生 32 字节 = 64 hex 字符
		t.Errorf("Hex signature length = %d, want 64", len(hexSignature))
	}

	if len(base64Signature) != 44 { // 32 字节 base64 编码 = 44 字符
		t.Errorf("Base64 signature length = %d, want 44", len(base64Signature))
	}

	t.Logf("Example HMAC-SHA256 signatures for body: %s", body)
	t.Logf("  Hex:    %s", hexSignature)
	t.Logf("  Base64: %s", base64Signature)
}
