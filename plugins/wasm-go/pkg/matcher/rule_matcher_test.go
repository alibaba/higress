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
	"errors"
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
			name: "exact port",
			config: RuleConfig[customConfig]{
				hosts: []HostMatcher{
					{
						matchType: Exact,
						host:      "www.example.com",
					},
				},
			},
			host:   "www.example.com:8080",
			result: true,
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

func TestServiceMatch(t *testing.T) {
	cases := []struct {
		name    string
		config  RuleConfig[customConfig]
		service string
		result  bool
	}{
		{
			name: "fqdn",
			config: RuleConfig[customConfig]{
				services: map[string]struct{}{
					"qwen.dns": {},
				},
			},
			service: "outbound|443||qwen.dns",
			result:  true,
		},
		{
			name: "fqdn with port",
			config: RuleConfig[customConfig]{
				services: map[string]struct{}{
					"qwen.dns:443": {},
				},
			},
			service: "outbound|443||qwen.dns",
			result:  true,
		},
		{
			name: "not match",
			config: RuleConfig[customConfig]{
				services: map[string]struct{}{
					"moonshot.dns:443": {},
				},
			},
			service: "outbound|443||qwen.dns",
			result:  false,
		},
		{
			name: "error config format",
			config: RuleConfig[customConfig]{
				services: map[string]struct{}{
					"qwen.dns:": {},
				},
			},
			service: "outbound|443||qwen.dns",
			result:  false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var m RuleMatcher[customConfig]
			assert.Equal(t, c.result, m.serviceMatch(c.config, c.service))
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
			config: `{"_rules_":[{"_match_domain_":["*.example.com","www.*","*","www.abc.com"],"name":"john", "age":18},{"_match_route_":["test1","test2"],"name":"ann", "age":16},{"_match_service_":["test1.dns","test2.static:8080"],"name":"ann", "age":16}]}`,
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
						routes:   map[string]struct{}{},
						services: map[string]struct{}{},
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
						services: map[string]struct{}{},
						config: customConfig{
							name: "ann",
							age:  16,
						},
					},
					{
						category: Service,
						services: map[string]struct{}{
							"test1.dns":         {},
							"test2.static:8080": {},
						},
						routes: map[string]struct{}{},
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
			errMsg: "parse config failed, no valid rules; global config parse error:<nil>",
		},
		{
			name:   "invalid rule",
			config: `{"_rules_":[{"_match_domain_":["*"],"_match_route_":["test"]}]}`,
			errMsg: "there is only one of  '_match_route_', '_match_domain_' and '_match_service_' can present in configuration.",
		},
		{
			name:   "invalid rule",
			config: `{"_rules_":[{"_match_domain_":["*"],"_match_service_":["test.dns"]}]}`,
			errMsg: "there is only one of  '_match_route_', '_match_domain_' and '_match_service_' can present in configuration.",
		},
		{
			name:   "invalid rule",
			config: `{"_rules_":[{"age":16}]}`,
			errMsg: "there is only one of  '_match_route_', '_match_domain_' and '_match_service_' can present in configuration.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var actual RuleMatcher[customConfig]
			err := actual.ParseRuleConfig(gjson.Parse(c.config), parseConfig, nil)
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

type completeConfig struct {
	// global config
	consumers []string
	// rule config
	allow []string
}

func parseGlobalConfig(json gjson.Result, global *completeConfig) error {
	if json.Get("consumers").Exists() && json.Get("allow").Exists() {
		return errors.New("consumers and allow should not be configured at the same level")
	}

	for _, item := range json.Get("consumers").Array() {
		global.consumers = append(global.consumers, item.String())
	}

	return nil
}

func parseOverrideRuleConfig(json gjson.Result, global completeConfig, config *completeConfig) error {
	if json.Get("consumers").Exists() && json.Get("allow").Exists() {
		return errors.New("consumers and allow should not be configured at the same level")
	}

	// override config via global
	*config = global

	for _, item := range json.Get("allow").Array() {
		config.allow = append(config.allow, item.String())
	}

	return nil
}

func TestParseOverrideConfig(t *testing.T) {
	cases := []struct {
		name     string
		config   string
		errMsg   string
		expected RuleMatcher[completeConfig]
	}{
		{
			name:   "override rule config",
			config: `{"consumers":["c1","c2","c3"],"_rules_":[{"_match_route_":["r1","r2"],"allow":["c1","c3"]}]}`,
			expected: RuleMatcher[completeConfig]{
				ruleConfig: []RuleConfig[completeConfig]{
					{
						category: Route,
						routes: map[string]struct{}{
							"r1": {},
							"r2": {},
						},
						services: map[string]struct{}{},
						config: completeConfig{
							consumers: []string{"c1", "c2", "c3"},
							allow:     []string{"c1", "c3"},
						},
					},
				},
				globalConfig: completeConfig{
					consumers: []string{"c1", "c2", "c3"},
				},
				hasGlobalConfig: true,
			},
		},
		{
			name:   "invalid config",
			config: `{"consumers":["c1","c2","c3"],"allow":["c1"]}`,
			errMsg: "parse config failed, no valid rules; global config parse error:consumers and allow should not be configured at the same level",
		},
		{
			name:   "invalid config",
			config: `{"_rules_":[{"_match_route_":["r1","r2"],"consumers":["c1","c2"],"allow":["c1"]}]}`,
			errMsg: "consumers and allow should not be configured at the same level",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var actual RuleMatcher[completeConfig]
			err := actual.ParseRuleConfig(gjson.Parse(c.config), parseGlobalConfig, parseOverrideRuleConfig)
			if err != nil {
				if c.errMsg == "" {
					t.Errorf("parse failed: %v", err)
				}
				if err.Error() != c.errMsg {
					t.Errorf("expect err: %s, actual err: %s", c.errMsg, err.Error())
				}
				return
			}
			assert.Equal(t, c.expected, actual)
		})
	}
}
