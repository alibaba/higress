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
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-a2as/test"
	"github.com/tidwall/gjson"
)

// 测试 Authenticated Prompts 功能
func TestAuthenticatedPrompts(t *testing.T) {
	test.RunAuthenticatedPromptsParseConfigTests(t)
	test.RunAuthenticatedPromptsOnHttpRequestBodyTests(t)
	test.RunAuthenticatedPromptsConfigValidationTests(t)
}

// 测试 Behavior Certificates 功能
func TestBehaviorCertificates(t *testing.T) {
	test.RunBehaviorCertificatesParseConfigTests(t)
	test.RunBehaviorCertificatesOnHttpRequestBodyTests(t)
}

// 测试 In-Context Defenses 和 Codified Policies 功能
func TestDefensesAndPolicies(t *testing.T) {
	test.RunDefensesAndPoliciesParseConfigTests(t)
	test.RunDefensesAndPoliciesOnHttpRequestBodyTests(t)
}

// 测试 Per-Consumer 配置功能
func TestPerConsumer(t *testing.T) {
	test.RunPerConsumerParseConfigTests(t)
	test.RunPerConsumerOnHttpRequestHeadersTests(t)
	test.RunPerConsumerOnHttpRequestBodyTests(t)
}

// 测试基础配置解析
func TestParseConfigBasic(t *testing.T) {
	tests := []struct {
		name       string
		jsonConfig string
		wantErr    bool
		validate   func(*A2ASConfig) bool
	}{
		{
			name: "behavior certificates enabled",
			jsonConfig: `{
				"behaviorCertificates": {
					"enabled": true,
					"allowedTools": ["tool1", "tool2"]
				}
			}`,
			wantErr: false,
			validate: func(config *A2ASConfig) bool {
				return config.BehaviorCertificates.Enabled && len(config.BehaviorCertificates.AllowedTools) == 2
			},
		},
		{
			name: "in-context defenses with default template",
			jsonConfig: `{
				"inContextDefenses": {
					"enabled": true
				}
			}`,
			wantErr: false,
			validate: func(config *A2ASConfig) bool {
				return config.InContextDefenses.Enabled && config.InContextDefenses.Template == "default"
			},
		},
		{
			name: "codified policies with medium severity default",
			jsonConfig: `{
				"codifiedPolicies": {
					"enabled": true,
					"policies": [{
						"name": "test-policy",
						"content": "test content"
					}]
				}
			}`,
			wantErr: false,
			validate: func(config *A2ASConfig) bool {
				return config.CodifiedPolicies.Enabled &&
					len(config.CodifiedPolicies.Policies) == 1 &&
					config.CodifiedPolicies.Policies[0].Severity == "medium"
			},
		},
		{
			name: "invalid defense position",
			jsonConfig: `{
				"inContextDefenses": {
					"enabled": true,
					"position": "invalid"
				}
			}`,
			wantErr: true,
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

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil && !tt.validate(config) {
				t.Errorf("Config validation failed for test %s", tt.name)
			}
		})
	}
}

// 测试是否为聊天完成请求
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

// 测试 BuildDefenseBlock 功能
func TestBuildDefenseBlock(t *testing.T) {
	// 测试默认模板
	defaultBlock := BuildDefenseBlock("default")
	if defaultBlock == "" {
		t.Error("BuildDefenseBlock('default') returned empty string")
	}

	// 测试自定义模板
	customBlock := BuildDefenseBlock("custom")
	if customBlock != "" {
		t.Error("BuildDefenseBlock('custom') should return empty string")
	}
}

// 测试 BuildPolicyBlock 功能
func TestBuildPolicyBlock(t *testing.T) {
	tests := []struct {
		name     string
		policies []Policy
		isEmpty  bool
	}{
		{
			name: "single policy with high severity",
			policies: []Policy{
				{
					Name:     "no-pii",
					Content:  "Do not process PII",
					Severity: "high",
				},
			},
			isEmpty: false,
		},
		{
			name:     "empty policies",
			policies: []Policy{},
			isEmpty:  true,
		},
		{
			name: "multiple policies",
			policies: []Policy{
				{Name: "policy1", Content: "content1", Severity: "high"},
				{Name: "policy2", Content: "content2", Severity: "medium"},
			},
			isEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPolicyBlock(tt.policies)
			if tt.isEmpty && result != "" {
				t.Errorf("Expected empty string, got: %s", result)
			}
			if !tt.isEmpty && result == "" {
				t.Error("Expected non-empty string, got empty")
			}
		})
	}
}
