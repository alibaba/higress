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
	"sort"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/iface"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
)

type Category int

const (
	Route Category = iota
	Host
	Service
	RoutePrefix
	RouteAndService
)

type MatchType int

const (
	Prefix MatchType = iota
	Exact
	Suffix
)

const (
	RULES_KEY              = "_rules_"
	MATCH_ROUTE_KEY        = "_match_route_"
	MATCH_DOMAIN_KEY       = "_match_domain_"
	MATCH_SERVICE_KEY      = "_match_service_"
	MATCH_ROUTE_PREFIX_KEY = "_match_route_prefix_"
)

type HostMatcher struct {
	matchType MatchType
	host      string
}

type RuleConfig[PluginConfig any] struct {
	category     Category
	routes       map[string]struct{}
	services     map[string]struct{}
	routePrefixs map[string]struct{}
	hosts        []HostMatcher
	config       PluginConfig
}

// GenerateHashKey generates a hash key for the rule config based on matching conditions
func (r *RuleConfig[PluginConfig]) GenerateHashKey() string {
	var keyParts []string

	// Add category
	keyParts = append(keyParts, fmt.Sprintf("cat:%d", r.category))

	// Add routes (sorted for stability)
	if len(r.routes) > 0 {
		var routes []string
		for route := range r.routes {
			routes = append(routes, route)
		}
		sort.Strings(routes)
		keyParts = append(keyParts, fmt.Sprintf("routes:%s", strings.Join(routes, ",")))
	}

	// Add services (sorted for stability)
	if len(r.services) > 0 {
		var services []string
		for service := range r.services {
			services = append(services, service)
		}
		sort.Strings(services)
		keyParts = append(keyParts, fmt.Sprintf("services:%s", strings.Join(services, ",")))
	}

	// Add route prefixes (sorted for stability)
	if len(r.routePrefixs) > 0 {
		var prefixes []string
		for prefix := range r.routePrefixs {
			prefixes = append(prefixes, prefix)
		}
		sort.Strings(prefixes)
		keyParts = append(keyParts, fmt.Sprintf("prefixes:%s", strings.Join(prefixes, ",")))
	}

	// Add hosts (already in slice order, but sort for consistency)
	if len(r.hosts) > 0 {
		var hosts []string
		for _, hostMatcher := range r.hosts {
			hosts = append(hosts, fmt.Sprintf("%d:%s", hostMatcher.matchType, hostMatcher.host))
		}
		sort.Strings(hosts)
		keyParts = append(keyParts, fmt.Sprintf("hosts:%s", strings.Join(hosts, ",")))
	}

	return strings.Join(keyParts, "|")
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
		// category == Service
		if rule.category == Service {
			if m.serviceMatch(rule, string(serviceName)) {
				return &rule.config, nil
			}
		}
		// category == RouteAndService
		if rule.category == RouteAndService {
			if _, ok := rule.routes[string(routeName)]; ok {
				if m.serviceMatch(rule, string(serviceName)) {
					return &rule.config, nil
				}
			}
		}
		// category == RoutePrefix
		if rule.category == RoutePrefix {
			for routePrefix := range rule.routePrefixs {
				if strings.HasPrefix(string(routeName), routePrefix) {
					return &rule.config, nil
				}
			}
		}
	}
	if m.hasGlobalConfig {
		return &m.globalConfig, nil
	}
	return nil, nil
}

func (m *RuleMatcher[PluginConfig]) ParseRuleConfig(context iface.PluginContext, config gjson.Result,
	parsePluginConfig func(gjson.Result, *PluginConfig) error,
	parseOverrideConfig func(gjson.Result, PluginConfig, *PluginConfig) error) error {
	var rules []gjson.Result
	obj := config.Map()
	keyCount := len(obj)
	if keyCount == 0 {
		// enable globally for empty config
		m.hasGlobalConfig = true
		return parsePluginConfig(config, &m.globalConfig)
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
	// Check if rule level config isolation is enabled
	isRuleLevelIsolation := context.IsRuleLevelConfigIsolation()

	var successfulRules []RuleConfig[PluginConfig]
	var hasAnyRuleParseError bool

	for _, ruleJson := range rules {
		var (
			rule RuleConfig[PluginConfig]
			err  error
		)
		rule.routes = m.parseRouteMatchConfig(ruleJson)
		rule.hosts = m.parseHostMatchConfig(ruleJson)
		rule.services = m.parseServiceMatchConfig(ruleJson)
		rule.routePrefixs = m.parseRoutePrefixMatchConfig(ruleJson)
		hasRoute := len(rule.routes) != 0
		hasHosts := len(rule.hosts) != 0
		hasService := len(rule.services) != 0
		hasRoutePrefix := len(rule.routePrefixs) != 0
		if boolToInt(hasRoute)+boolToInt(hasService)+boolToInt(hasHosts)+boolToInt(hasRoutePrefix) == 0 {
			return errors.New("there is at least one of  '_match_route_', '_match_domain_', '_match_service_' and '_match_route_prefix_' can present in configuration.")
		}
		if hasRoute {
			rule.category = Route
			if hasService {
				rule.category = RouteAndService
			}
		} else if hasHosts {
			rule.category = Host
		} else if hasService {
			rule.category = Service
		} else {
			rule.category = RoutePrefix
		}

		// Try to parse the rule config
		if parseOverrideConfig != nil {
			err = parseOverrideConfig(ruleJson, m.globalConfig, &rule.config)
		} else {
			err = parsePluginConfig(ruleJson, &rule.config)
		}

		if err != nil {
			hasAnyRuleParseError = true
			if isRuleLevelIsolation {
				log.Warnf("parse rule config failed for rule %s: %v, trying to load from backup", ruleJson.Raw, err)
				// Try to load from backup
				if loadedRuleJson := m.loadRuleJsonFromBackup(context, rule); loadedRuleJson.Exists() {
					log.Infof("found backup rule config: %s", loadedRuleJson.Raw)
					// Try to parse the rule config from loaded JSON
					if parseOverrideConfig != nil {
						err = parseOverrideConfig(loadedRuleJson, m.globalConfig, &rule.config)
					} else {
						err = parsePluginConfig(loadedRuleJson, &rule.config)
					}
					if err == nil {
						successfulRules = append(successfulRules, rule)
						log.Infof("successfully loaded rule from backup: %s", ruleJson.Raw)
						continue
					}
					log.Errorf("failed to parse backup rule config: %v", err)
				}
				log.Errorf("failed to load rule from backup, skipping rule: %s", ruleJson.Raw)
				continue
			} else {
				return err
			}
		}

		// Store successful rule to backup if rule level isolation is enabled
		if isRuleLevelIsolation {
			log.Debugf("storing rule to backup: %s", ruleJson.Raw)
			if storeErr := m.storeRuleToBackup(context, ruleJson, rule); storeErr != nil {
				log.Errorf("failed to store rule to backup: %v, rule: %s", storeErr, ruleJson.Raw)
			} else {
				log.Debugf("successfully stored rule to backup: %s", ruleJson.Raw)
			}
		}

		successfulRules = append(successfulRules, rule)
	}

	m.ruleConfig = successfulRules

	// If no successful rules and rule level isolation is enabled, return error
	if isRuleLevelIsolation && len(successfulRules) == 0 && hasAnyRuleParseError {
		return fmt.Errorf("all rules failed to parse and no previous successful config available")
	} else if !isRuleLevelIsolation && len(successfulRules) == 0 {
		return fmt.Errorf("no valid rules parsed")
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

func (m RuleMatcher[PluginConfig]) parseRoutePrefixMatchConfig(config gjson.Result) map[string]struct{} {
	keys := config.Get(MATCH_ROUTE_PREFIX_KEY).Array()
	routePrefixs := make(map[string]struct{})
	for _, item := range keys {
		routePrefix := item.String()
		if routePrefix != "" {
			routePrefixs[routePrefix] = struct{}{}
		}
	}
	return routePrefixs
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

var gRuleBackupStore map[string]string

func init() {
	gRuleBackupStore = make(map[string]string)
}

// storeRuleToSharedMemory stores a successful rule configuration to backup store
func (m *RuleMatcher[PluginConfig]) storeRuleToBackup(context iface.PluginContext, ruleJson gjson.Result, rule RuleConfig[PluginConfig]) error {
	hashKey := rule.GenerateHashKey()
	gRuleBackupStore[hashKey] = ruleJson.Raw
	log.Infof("store rule to backup, key[%s]", hashKey)
	return nil
}

// loadRuleJsonFromSharedMemory loads a rule JSON from backup store
func (m *RuleMatcher[PluginConfig]) loadRuleJsonFromBackup(context iface.PluginContext, rule RuleConfig[PluginConfig]) gjson.Result {
	hashKey := rule.GenerateHashKey()
	data, ok := gRuleBackupStore[hashKey]
	if !ok || data == "" {
		log.Infof("load rule from backup failed, key[%s]", hashKey)
		return gjson.Result{}
	}
	log.Infof("load rule from backup success, key[%s]", hashKey)
	return gjson.Parse(data)
}
