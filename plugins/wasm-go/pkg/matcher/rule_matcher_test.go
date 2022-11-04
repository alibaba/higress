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

package matcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

type customConfig struct {
	name string
	age  int64
}

func parseConfig(json gjson.Result, config *customConfig) error {
	config.name = json.Get("name").String()
	config.age = json.Get("age").Int()
	return nil
}

func TestHostMatch(t *testing.T) {
	cases := []struct {
		name   string
		config RuleConfig[customConfig]
		host   string
		result bool
	}{
		{
			name: "prefix",
			config: RuleConfig[customConfig]{
				hosts: []HostMatcher{
					{
						matchType: Prefix,
						host:      "www.",
					},
				},
			},
			host:   "www.test.com",
			result: true,
		},
		{
			name: "prefix failed",
			config: RuleConfig[customConfig]{
				hosts: []HostMatcher{
					{
						matchType: Prefix,
						host:      "www.",
					},
				},
			},
			host:   "test.com",
			result: false,
		},
		{
			name: "suffix",
			config: RuleConfig[customConfig]{
				hosts: []HostMatcher{
					{
						matchType: Suffix,
						host:      ".example.com",
					},
				},
			},
			host:   "www.example.com",
			result: true,
		},
		{
			name: "suffix failed",
			config: RuleConfig[customConfig]{
				hosts: []HostMatcher{
					{
						matchType: Suffix,
						host:      ".example.com",
					},
				},
			},
			host:   "example.com",
			result: false,
		},
		{
			name: "exact",
			config: RuleConfig[customConfig]{
				hosts: []HostMatcher{
					{
						matchType: Exact,
						host:      "www.example.com",
					},
				},
			},
			host:   "www.example.com",
			result: true,
		},
		{
			name: "exact failed",
			config: RuleConfig[customConfig]{
				hosts: []HostMatcher{
					{
						matchType: Exact,
						host:      "www.example.com",
					},
				},
			},
			host:   "example.com",
			result: false,
		},
		{
			name: "any",
			config: RuleConfig[customConfig]{
				hosts: []HostMatcher{
					{
						matchType: Suffix,
						host:      "",
					},
				},
			},
			host:   "www.example.com",
			result: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var m RuleMatcher[customConfig]
			assert.Equal(t, c.result, m.hostMatch(c.config, c.host))
		})
	}
}

func TestParseRuleConfig(t *testing.T) {
	cases := []struct {
		name     string
		config   string
		errMsg   string
		expected RuleMatcher[customConfig]
	}{
		{
			name:   "global config",
			config: `{"name":"john", "age":18}`,
			expected: RuleMatcher[customConfig]{
				globalConfig: customConfig{
					name: "john",
					age:  18,
				},
				hasGlobalConfig: true,
			},
		},
		{
			name:   "rules config",
			config: `{"_rules_":[{"_match_domain_":["*.example.com","www.*","*","www.abc.com"],"name":"john", "age":18},{"_match_route_":["test1","test2"],"name":"ann", "age":16}]}`,
			expected: RuleMatcher[customConfig]{
				ruleConfig: []RuleConfig[customConfig]{
					{
						category: Host,
						hosts: []HostMatcher{
							{
								matchType: Suffix,
								host:      ".example.com",
							},
							{
								matchType: Prefix,
								host:      "www.",
							},
							{
								matchType: Suffix,
								host:      "",
							},
							{
								matchType: Exact,
								host:      "www.abc.com",
							},
						},
						routes: map[string]struct{}{},
						config: customConfig{
							name: "john",
							age:  18,
						},
					},
					{
						category: Route,
						routes: map[string]struct{}{
							"test1": {},
							"test2": {},
						},
						config: customConfig{
							name: "ann",
							age:  16,
						},
					},
				},
			},
		},
		{
			name:   "no rule",
			config: `{"_rules_":[]}`,
			errMsg: "parse config failed, no valid rules",
		},
		{
			name:   "invalid rule",
			config: `{"_rules_":[{"_match_domain_":["*"],"_match_route_":["test"]}]}`,
			errMsg: "there is only one of  '_match_route_' and '_match_domain_' can present in configuration.",
		},
		{
			name:   "invalid rule",
			config: `{"_rules_":[{"age":16}]}`,
			errMsg: "there is only one of  '_match_route_' and '_match_domain_' can present in configuration.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var actual RuleMatcher[customConfig]
			err := actual.ParseRuleConfig(gjson.Parse(c.config), parseConfig)
			if err != nil {
				if c.errMsg == "" {
					t.Errorf("parse failed: %v", err)
				}
				if err.Error() != c.errMsg {
					t.Errorf("expect err: %s, actual err: %s", c.errMsg,
						err.Error())
				}
				return
			}
			assert.Equal(t, c.expected, actual)
		})
	}
}
