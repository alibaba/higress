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
	"fmt"
	"os"
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// testLogger is a simple logger implementation for validation mode
type testLogger struct{}

func (l *testLogger) Trace(msg string) { fmt.Fprintf(os.Stderr, "[TRACE] %s\n", msg) }
func (l *testLogger) Tracef(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[TRACE] "+format+"\n", args...)
}
func (l *testLogger) Debug(msg string) { fmt.Fprintf(os.Stderr, "[DEBUG] %s\n", msg) }
func (l *testLogger) Debugf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
}
func (l *testLogger) Info(msg string) { fmt.Fprintf(os.Stderr, "[INFO] %s\n", msg) }
func (l *testLogger) Infof(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
}
func (l *testLogger) Warn(msg string) { fmt.Fprintf(os.Stderr, "[WARN] %s\n", msg) }
func (l *testLogger) Warnf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
}
func (l *testLogger) Error(msg string) { fmt.Fprintf(os.Stderr, "[ERROR] %s\n", msg) }
func (l *testLogger) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}
func (l *testLogger) Critical(msg string) { fmt.Fprintf(os.Stderr, "[CRITICAL] %s\n", msg) }
func (l *testLogger) Criticalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[CRITICAL] "+format+"\n", args...)
}
func (l *testLogger) ResetID(pluginID string) {}

// init initializes the validator package
func init() {
	// Set a custom logger for validation mode to prevent panics
	log.SetPluginLog(&testLogger{})
}

type customConfig struct {
	name string
	age  int64
}

type mockPluginContext struct {
	ruleLevelIsolation bool
}

func (c *mockPluginContext) SetContext(key string, value interface{}) {}

func (c *mockPluginContext) GetContext(key string) interface{} { return nil }

func (c *mockPluginContext) EnableRuleLevelConfigIsolation() { c.ruleLevelIsolation = true }

func (c *mockPluginContext) IsRuleLevelConfigIsolation() bool { return c.ruleLevelIsolation }

func (c *mockPluginContext) GetFingerPrint() string { return "" }

func (c *mockPluginContext) DoLeaderElection() {}
func (c *mockPluginContext) IsLeader() bool    { return true }

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
			config: `{"_rules_":[{"_match_domain_":["*.example.com","www.*","*","www.abc.com"],"name":"john", "age":18},{"_match_route_":["test1","test2"],"name":"ann", "age":16},{"_match_service_":["test1.dns","test2.static:8080"],"name":"ann", "age":16},{"_match_route_prefix_":["api1","api2"],"name":"ann", "age":16}]}`,
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
						routes:       map[string]struct{}{},
						services:     map[string]struct{}{},
						routePrefixs: map[string]struct{}{},
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
						services:     map[string]struct{}{},
						routePrefixs: map[string]struct{}{},
						config: customConfig{
							name: "ann",
							age:  16,
						},
					},
					{
						category: Service,
						routes:   map[string]struct{}{},
						services: map[string]struct{}{
							"test1.dns":         {},
							"test2.static:8080": {},
						},
						routePrefixs: map[string]struct{}{},
						config: customConfig{
							name: "ann",
							age:  16,
						},
					},
					{
						category: RoutePrefix,
						routes:   map[string]struct{}{},
						services: map[string]struct{}{},
						routePrefixs: map[string]struct{}{
							"api1": {},
							"api2": {},
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
			errMsg: "parse config failed, no valid rules; global config parse error:<nil>",
		},
		{
			name:   "invalid rule",
			config: `{"_rules_":[{"age":16}]}`,
			errMsg: "there is at least one of  '_match_route_', '_match_domain_', '_match_service_' and '_match_route_prefix_' can present in configuration.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var actual RuleMatcher[customConfig]
			var ctx mockPluginContext
			err := actual.ParseRuleConfig(&ctx, gjson.Parse(c.config), parseConfig, nil)
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
						services:     map[string]struct{}{},
						routePrefixs: map[string]struct{}{},
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
			var ctx mockPluginContext
			err := actual.ParseRuleConfig(&ctx, gjson.Parse(c.config), parseGlobalConfig, parseOverrideRuleConfig)
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

func TestGenerateHashKey(t *testing.T) {
	// Test hash key stability - same rule should generate same hash key
	rule1 := RuleConfig[customConfig]{
		category: Route,
		routes: map[string]struct{}{
			"route2": {},
			"route1": {},
		},
		services: map[string]struct{}{
			"service2": {},
			"service1": {},
		},
		routePrefixs: map[string]struct{}{
			"prefix2": {},
			"prefix1": {},
		},
		hosts: []HostMatcher{
			{matchType: Exact, host: "host2.com"},
			{matchType: Exact, host: "host1.com"},
		},
	}

	rule2 := RuleConfig[customConfig]{
		category: Route,
		routes: map[string]struct{}{
			"route1": {},
			"route2": {},
		},
		services: map[string]struct{}{
			"service1": {},
			"service2": {},
		},
		routePrefixs: map[string]struct{}{
			"prefix1": {},
			"prefix2": {},
		},
		hosts: []HostMatcher{
			{matchType: Exact, host: "host1.com"},
			{matchType: Exact, host: "host2.com"},
		},
	}

	hash1 := rule1.GenerateHashKey()
	hash2 := rule2.GenerateHashKey()

	assert.Equal(t, hash1, hash2, "Same rule with different map order should generate same hash key")
	assert.NotEmpty(t, hash1, "Hash key should not be empty")

	// Test different rules generate different hash keys
	rule3 := RuleConfig[customConfig]{
		category: Host,
		hosts: []HostMatcher{
			{matchType: Exact, host: "different.com"},
		},
	}

	hash3 := rule3.GenerateHashKey()
	assert.NotEqual(t, hash1, hash3, "Different rules should generate different hash keys")
}

func TestBackupStore(t *testing.T) {
	// Clear backup store before test
	gRuleBackupStore = make(map[string]string)

	var matcher RuleMatcher[customConfig]
	var ctx mockPluginContext

	rule := RuleConfig[customConfig]{
		category: Route,
		routes: map[string]struct{}{
			"test-route": {},
		},
		config: customConfig{
			name: "test",
			age:  25,
		},
	}

	ruleJson := gjson.Parse(`{"_match_route_":["test-route"],"name":"test","age":25}`)

	// Test store rule to backup
	err := matcher.storeRuleToBackup(&ctx, ruleJson, rule)
	assert.NoError(t, err, "Store rule to backup should not fail")

	// Test load rule from backup
	loadedJson := matcher.loadRuleJsonFromBackup(&ctx, rule)
	assert.True(t, loadedJson.Exists(), "Loaded rule JSON should exist")
	assert.Equal(t, ruleJson.Raw, loadedJson.Raw, "Loaded rule JSON should match original")

	// Test load non-existent rule
	nonExistentRule := RuleConfig[customConfig]{
		category: Host,
		hosts: []HostMatcher{
			{matchType: Exact, host: "non-existent.com"},
		},
	}

	loadedJson2 := matcher.loadRuleJsonFromBackup(&ctx, nonExistentRule)
	assert.False(t, loadedJson2.Exists(), "Non-existent rule should not be found")
}

func parseConfigWithError(json gjson.Result, config *customConfig) error {
	if json.Get("name").String() == "error" {
		return errors.New("parse error")
	}
	return parseConfig(json, config)
}

func TestRuleLevelConfigIsolation(t *testing.T) {
	// Clear backup store before test
	gRuleBackupStore = make(map[string]string)

	cases := []struct {
		name              string
		config            string
		enableIsolation   bool
		expectedRuleCount int
		expectedError     bool
		setupBackup       bool
		backupRuleJson    string
		backupRule        RuleConfig[customConfig]
	}{
		{
			name:              "isolation disabled - parse error should fail",
			config:            `{"_rules_":[{"_match_route_":["test1"],"name":"error","age":18}]}`,
			enableIsolation:   false,
			expectedRuleCount: 0,
			expectedError:     true,
		},
		{
			name:              "isolation enabled - parse error with no backup should skip rule",
			config:            `{"_rules_":[{"_match_route_":["test1"],"name":"error","age":18}]}`,
			enableIsolation:   true,
			expectedRuleCount: 0,
			expectedError:     true,
		},
		{
			name:              "isolation enabled - parse error with backup should use backup",
			config:            `{"_rules_":[{"_match_route_":["test1"],"name":"error","age":18}]}`,
			enableIsolation:   true,
			expectedRuleCount: 1,
			expectedError:     false,
			setupBackup:       true,
			backupRuleJson:    `{"_match_route_":["test1"],"name":"backup","age":30}`,
			backupRule: RuleConfig[customConfig]{
				category: Route,
				routes: map[string]struct{}{
					"test1": {},
				},
			},
		},
		{
			name:              "isolation enabled - successful parse should store to backup",
			config:            `{"_rules_":[{"_match_route_":["test2"],"name":"success","age":25}]}`,
			enableIsolation:   true,
			expectedRuleCount: 1,
			expectedError:     false,
		},
		{
			name:              "isolation enabled - mixed success and failure",
			config:            `{"_rules_":[{"_match_route_":["test1"],"name":"success","age":25},{"_match_route_":["test2"],"name":"error","age":18}]}`,
			enableIsolation:   true,
			expectedRuleCount: 1,
			expectedError:     false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Clear backup store for each test
			gRuleBackupStore = make(map[string]string)

			var matcher RuleMatcher[customConfig]
			var ctx mockPluginContext

			if c.enableIsolation {
				ctx.EnableRuleLevelConfigIsolation()
			}

			// Setup backup if needed
			if c.setupBackup {
				err := matcher.storeRuleToBackup(&ctx, gjson.Parse(c.backupRuleJson), c.backupRule)
				assert.NoError(t, err, "Setup backup should not fail")
			}

			err := matcher.ParseRuleConfig(&ctx, gjson.Parse(c.config), parseConfigWithError, nil)

			if c.expectedError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Unexpected error: %v", err)
			}

			assert.Equal(t, c.expectedRuleCount, len(matcher.ruleConfig), "Rule count mismatch")

			// If backup was used, verify the config
			if c.setupBackup && !c.expectedError && c.expectedRuleCount > 0 {
				assert.Equal(t, "backup", matcher.ruleConfig[0].config.name, "Should use backup config")
				assert.Equal(t, int64(30), matcher.ruleConfig[0].config.age, "Should use backup config")
			}
		})
	}
}
