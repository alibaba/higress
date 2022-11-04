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
	"strings"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

type Category int

const (
	Route Category = iota
	Host
)

type MatchType int

const (
	Prefix MatchType = iota
	Exact
	Suffix
)

const (
	RULES_KEY        = "_rules_"
	MATCH_ROUTE_KEY  = "_match_route_"
	MATCH_DOMAIN_KEY = "_match_domain_"
)

type HostMatcher struct {
	matchType MatchType
	host      string
}

type RuleConfig[PluginConfig any] struct {
	category Category
	routes   map[string]struct{}
	hosts    []HostMatcher
	config   PluginConfig
}

type RuleMatcher[PluginConfig any] struct {
	ruleConfig      []RuleConfig[PluginConfig]
	globalConfig    PluginConfig
	hasGlobalConfig bool
}

func (m RuleMatcher[PluginConfig]) GetMatchConfig() (*PluginConfig, error) {
	host, err := proxywasm.GetHttpRequestHeader(":authority")
	if err != nil {
		return nil, err
	}
	routeName, err := proxywasm.GetProperty([]string{"route_name"})
	if err != nil {
		return nil, err
	}
	for _, rule := range m.ruleConfig {
		if rule.category == Host {
			if m.hostMatch(rule, host) {
				return &rule.config, nil
			}
		}
		// category == Route
		if _, ok := rule.routes[string(routeName)]; ok {
			return &rule.config, nil
		}
	}
	if m.hasGlobalConfig {
		return &m.globalConfig, nil
	}
	return nil, nil
}

func (m *RuleMatcher[PluginConfig]) ParseRuleConfig(config gjson.Result,
	parsePluginConfig func(gjson.Result, *PluginConfig) error) error {
	var rules []gjson.Result
	obj := config.Map()
	keyCount := len(obj)
	if keyCount == 0 {
		// enable globally for empty config
		m.hasGlobalConfig = true
		return nil
	}
	if rulesJson, ok := obj[RULES_KEY]; ok {
		rules = rulesJson.Array()
		keyCount--
	}
	var pluginConfig PluginConfig
	if keyCount > 0 {
		err := parsePluginConfig(config, &pluginConfig)
		if err != nil {
			proxywasm.LogInfof("parse global config failed, err:%v", err)
		} else {
			m.globalConfig = pluginConfig
			m.hasGlobalConfig = true
		}
	}
	if len(rules) == 0 {
		if m.hasGlobalConfig {
			return nil
		}
		return errors.New("parse config failed, no valid rules")
	}
	for _, ruleJson := range rules {
		var rule RuleConfig[PluginConfig]
		err := parsePluginConfig(ruleJson, &rule.config)
		if err != nil {
			return err
		}
		rule.routes = m.parseRouteMatchConfig(ruleJson)
		rule.hosts = m.parseHostMatchConfig(ruleJson)
		noRoute := len(rule.routes) == 0
		noHosts := len(rule.hosts) == 0
		if (noRoute && noHosts) || (!noRoute && !noHosts) {
			return errors.New("there is only one of  '_match_route_' and '_match_domain_' can present in configuration.")
		}
		if !noRoute {
			rule.category = Route
		} else {
			rule.category = Host
		}
		m.ruleConfig = append(m.ruleConfig, rule)
	}
	return nil
}

func (m RuleMatcher[PluginConfig]) parseRouteMatchConfig(config gjson.Result) map[string]struct{} {
	keys := config.Get(MATCH_ROUTE_KEY).Array()
	routes := make(map[string]struct{})
	for _, item := range keys {
		routeName := item.String()
		if routeName != "" {
			routes[routeName] = struct{}{}
		}
	}
	return routes
}

func (m RuleMatcher[PluginConfig]) parseHostMatchConfig(config gjson.Result) []HostMatcher {
	keys := config.Get(MATCH_DOMAIN_KEY).Array()
	var hostMatchers []HostMatcher
	for _, item := range keys {
		host := item.String()
		var hostMatcher HostMatcher
		if strings.HasPrefix(host, "*") {
			hostMatcher.matchType = Suffix
			hostMatcher.host = host[1:]
		} else if strings.HasSuffix(host, "*") {
			hostMatcher.matchType = Prefix
			hostMatcher.host = host[:len(host)-1]
		} else {
			hostMatcher.matchType = Exact
			hostMatcher.host = host
		}
		hostMatchers = append(hostMatchers, hostMatcher)
	}
	return hostMatchers
}

func (m RuleMatcher[PluginConfig]) hostMatch(rule RuleConfig[PluginConfig], reqHost string) bool {
	for _, hostMatch := range rule.hosts {
		switch hostMatch.matchType {
		case Suffix:
			if strings.HasSuffix(reqHost, hostMatch.host) {
				return true
			}
		case Prefix:
			if strings.HasPrefix(reqHost, hostMatch.host) {
				return true
			}
		case Exact:
			if reqHost == hostMatch.host {
				return true
			}
		default:
			return false
		}
	}
	return false
}
