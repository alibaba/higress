package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestParseClusterKeyRateLimitConfig(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		expected    ClusterKeyRateLimitConfig
		expectedErr error
	}{
		{
			name:        "MissingRuleName",
			json:        `{}`,
			expectedErr: errors.New("missing rule_name in config"),
		},
		{
			name: "GlobalThreshold_InvalidThreshold",
			json: `{
				"rule_name": "invalid-threshold",
				"global_threshold": {
					"query_per_minute": -100
				}
			}`,
			expectedErr: errors.New("failed to parse global_threshold: 'query_per_minute' must be a positive integer, got -100"),
		},
		{
			name: "GlobalThreshold_QueryPerSecond",
			json: `{
				"rule_name": "global-route-limit",
				"global_threshold": {
					"query_per_second": 100
				}
			}`,
			expected: ClusterKeyRateLimitConfig{
				RuleName: "global-route-limit",
				GlobalThreshold: &GlobalThreshold{
					Count:      100,
					TimeWindow: Second,
				},
				RejectedCode: DefaultRejectedCode,
				RejectedMsg:  DefaultRejectedMsg,
			},
		},
		{
			name: "GlobalThreshold_QueryPerMinute",
			json: `{
				"rule_name": "global-route-limit",
				"global_threshold": {
					"query_per_minute": 1000
				}
			}`,
			expected: ClusterKeyRateLimitConfig{
				RuleName: "global-route-limit",
				GlobalThreshold: &GlobalThreshold{
					Count:      1000,
					TimeWindow: SecondsPerMinute,
				},
				RejectedCode: DefaultRejectedCode,
				RejectedMsg:  DefaultRejectedMsg,
			},
		},
		{
			name: "RuleItems_InvalidThreshold",
			json: `{
				"rule_name": "invalid-threshold",
				"rule_items": [
					{
						"limit_by_header": "x-test",
						"limit_keys": [
							{"key": "key1", "query_per_minute": -100}
						]
					}
				]
			}`,
			expectedErr: errors.New("failed to parse rule_item in rule_items: 'query_per_minute' must be a positive integer for key 'key1', got -100"),
		},
		{
			name: "RuleItems_SingleRule",
			json: `{
				"rule_name": "rule-based-limit",
				"rule_items": [
					{
						"limit_by_header": "x-test",
						"limit_keys": [
							{"key": "key1", "query_per_second": 10}
						]
					}
				]
			}`,
			expected: ClusterKeyRateLimitConfig{
				RuleName: "rule-based-limit",
				RuleItems: []LimitRuleItem{
					{
						LimitType: LimitByHeaderType,
						Key:       "x-test",
						ConfigItems: []LimitConfigItem{
							{
								ConfigType: ExactType,
								Key:        "key1",
								Count:      10,
								TimeWindow: Second,
							},
						},
					},
				},
				RejectedCode: DefaultRejectedCode,
				RejectedMsg:  DefaultRejectedMsg,
			},
		},
		{
			name: "RuleItems_MultipleRules",
			json: `{
				"rule_name": "multi-rule-limit",
				"rule_items": [
					{
						"limit_by_param": "user_id",
						"limit_keys": [
							{"key": "123", "query_per_hour": 50}
						]
					},
					{
						"limit_by_per_cookie": "session_id",
						"limit_keys": [
							{"key": "*", "query_per_day": 100}
						]
					}
				]
			}`,
			expected: ClusterKeyRateLimitConfig{
				RuleName: "multi-rule-limit",
				RuleItems: []LimitRuleItem{
					{
						LimitType: LimitByParamType,
						Key:       "user_id",
						ConfigItems: []LimitConfigItem{
							{
								ConfigType: ExactType,
								Key:        "123",
								Count:      50,
								TimeWindow: SecondsPerHour,
							},
						},
					},
					{
						LimitType: LimitByPerCookieType,
						Key:       "session_id",
						ConfigItems: []LimitConfigItem{
							{
								ConfigType: AllType,
								Key:        "*",
								Count:      100,
								TimeWindow: SecondsPerDay,
							},
						},
					},
				},
				RejectedCode: DefaultRejectedCode,
				RejectedMsg:  DefaultRejectedMsg,
			},
		},
		{
			name: "Conflict_GlobalThresholdAndRuleItems",
			json: `{
				"rule_name": "test-conflict",
				"global_threshold": {"query_per_second": 100},
				"rule_items": [{"limit_by_header": "x-test"}]
			}`,
			expectedErr: errors.New("'global_threshold' and 'rule_items' cannot be set at the same time"),
		},
		{
			name: "Missing_GlobalThresholdAndRuleItems",
			json: `{
				"rule_name": "test-missing"
			}`,
			expectedErr: errors.New("at least one of 'global_threshold' or 'rule_items' must be set"),
		},
		{
			name: "Custom_RejectedCodeAndMessage",
			json: `{
				"rule_name": "custom-reject",
				"rejected_code": 403,
				"rejected_msg": "Forbidden",
				"global_threshold": {"query_per_second": 100}
			}`,
			expected: ClusterKeyRateLimitConfig{
				RuleName: "custom-reject",
				GlobalThreshold: &GlobalThreshold{
					Count:      100,
					TimeWindow: Second,
				},
				RejectedCode: 403,
				RejectedMsg:  "Forbidden",
			},
		},
		{
			name: "ShowLimitQuotaHeader_Enabled",
			json: `{
				"rule_name": "show-header",
				"show_limit_quota_header": true,
				"global_threshold": {"query_per_second": 100}
			}`,
			expected: ClusterKeyRateLimitConfig{
				RuleName: "show-header",
				GlobalThreshold: &GlobalThreshold{
					Count:      100,
					TimeWindow: Second,
				},
				ShowLimitQuotaHeader: true,
				RejectedCode:         DefaultRejectedCode,
				RejectedMsg:          DefaultRejectedMsg,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config ClusterKeyRateLimitConfig
			result := gjson.Parse(tt.json)
			err := ParseClusterKeyRateLimitConfig(result, &config)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, config)
			}
		})
	}
}
