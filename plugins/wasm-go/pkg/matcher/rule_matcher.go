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
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

type Category int

const (
	Route Category = iota
	Host
	Service
)

type MatchType int

const (
	Prefix MatchType = iota
	Exact
	Suffix
)

const (
	RULES_KEY         = "_rules_"
	MATCH_ROUTE_KEY   = "_match_route_"
	MATCH_DOMAIN_KEY  = "_match_domain_"
	MATCH_SERVICE_KEY = "_match_service_"
)

type HostMatcher struct {
	matchType MatchType
	host      string
}

type RuleConfig[PluginConfig any] struct {
	category Category
	routes   map[string]struct{}
	services map[string]struct{}
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
	if err != nil && err != types.ErrorStatusNotFound {
		return nil, err
	}
	serviceName, err := proxywasm.GetProperty([]string{"cluster_name"})
	if err != nil && err != types.ErrorStatusNotFound {
		return nil, err
	}
	for _, rule := range m.ruleConfig {
		// category == Host
		if rule.category == Host {
			if m.hostMatch(rule, host) {
				return &rule.config, nil
			}
		}
		// category == Route
		if rule.category == Route {
			if _, ok := rule.routes[string(routeName)]; ok {
				return &rule.config, nil
			}
		}
		// category == Cluster
		if m.serviceMatch(rule, string(serviceName)) {
			return &rule.config, nil
		}
	}
	if m.hasGlobalConfig {
		return &m.globalConfig, nil
	}
	return nil, nil
}

func (m *RuleMatcher[PluginConfig]) ParseRuleConfig(config gjson.Result,
	parsePluginConfig func(gjson.Result, *PluginConfig) error,
	parseOverrideConfig func(gjson.Result, PluginConfig, *PluginConfig) error) error {
	var rules []gjson.Result
	obj := config.Map()
	keyCount := len(obj)
	if keyCount == 0 {
		// enable globally for empty config
		m.hasGlobalConfig = true
		parsePluginConfig(config, &m.globalConfig)
		return nil
	}
	if rulesJson, ok := obj[RULES_KEY]; ok {
		rules = rulesJson.Array()
		keyCount--
	}
	var pluginConfig PluginConfig
	var globalConfigError error
	if keyCount > 0 {
		err := parsePluginConfig(config, &pluginConfig)
		if err != nil {
			globalConfigError = err
		} else {
			m.globalConfig = pluginConfig
			m.hasGlobalConfig = true
		}
	}
	if len(rules) == 0 {
		if m.hasGlobalConfig {
			return nil
		}
		return fmt.Errorf("parse config failed, no valid rules; global config parse error:%v", globalConfigError)
	}
	for _, ruleJson := range rules {
		var (
			rule RuleConfig[PluginConfig]
			err  error
		)
		if parseOverrideConfig != nil {
			err = parseOverrideConfig(ruleJson, m.globalConfig, &rule.config)
		} else {
			err = parsePluginConfig(ruleJson, &rule.config)
		}
		if err != nil {
			return err
		}
		rule.routes = m.parseRouteMatchConfig(ruleJson)
		rule.hosts = m.parseHostMatchConfig(ruleJson)
		rule.services = m.parseServiceMatchConfig(ruleJson)
		noRoute := len(rule.routes) == 0
		noHosts := len(rule.hosts) == 0
		noService := len(rule.services) == 0
		if boolToInt(noRoute)+boolToInt(noService)+boolToInt(noHosts) != 2 {
			return errors.New("there is only one of  '_match_route_', '_match_domain_' and '_match_service_' can present in configuration.")
		}
		if !noRoute {
			rule.category = Route
		} else if !noHosts {
			rule.category = Host
		} else {
			rule.category = Service
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

func (m RuleMatcher[PluginConfig]) parseServiceMatchConfig(config gjson.Result) map[string]struct{} {
	keys := config.Get(MATCH_SERVICE_KEY).Array()
	clusters := make(map[string]struct{})
	for _, item := range keys {
		clusterName := item.String()
		if clusterName != "" {
			clusters[clusterName] = struct{}{}
		}
	}
	return clusters
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

func stripPortFromHost(reqHost string) string {
	// Port removing code is inspired by
	// https://github.com/envoyproxy/envoy/blob/v1.17.0/source/common/http/header_utility.cc#L219
	portStart := strings.LastIndexByte(reqHost, ':')
	if portStart != -1 {
		// According to RFC3986 v6 address is always enclosed in "[]".
		// section 3.2.2.
		v6EndIndex := strings.LastIndexByte(reqHost, ']')
		if v6EndIndex == -1 || v6EndIndex < portStart {
			if portStart+1 <= len(reqHost) {
				return reqHost[:portStart]
			}
		}
	}
	return reqHost
}

func (m RuleMatcher[PluginConfig]) hostMatch(rule RuleConfig[PluginConfig], reqHost string) bool {
	reqHost = stripPortFromHost(reqHost)
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

func (m RuleMatcher[PluginConfig]) serviceMatch(rule RuleConfig[PluginConfig], serviceName string) bool {
	parts := strings.Split(serviceName, "|")
	if len(parts) != 4 {
		return false
	}
	port := parts[1]
	fqdn := parts[3]
	for configServiceName := range rule.services {
		colonIndex := strings.LastIndexByte(configServiceName, ':')
		if colonIndex != -1 && fqdn == string(configServiceName[:colonIndex]) && port == string(configServiceName[colonIndex+1:]) {
			return true
		} else if fqdn == string(configServiceName) {
			return true
		}
	}
	return false
}
