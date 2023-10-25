// Copyright (c) 2023 Alibaba Group Holding Ltd.
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

package common

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

var (
	errMatchRouteDomainNotFound = errors.New("Must one of _match_route_ and _match_domain_")
)

type Rule struct {
	MatchRoute  []string `yaml:"_match_route_,omitempty"`
	MatchDomain []string `yaml:"_match_domain_,omitempty"`
	Allow       []string `yaml:"allow"`
}

func ParseRuleConfig(json gjson.Result, config *Rule, log wrapper.Log) error {
	routes := json.Get("_match_route_")
	domains := json.Get("_match_domain_")
	if !routes.Exists() && !domains.Exists() {
		return errMatchRouteDomainNotFound
	}

	if routes.Exists() {
		for _, item := range routes.Array() {
			config.MatchRoute = append(config.MatchRoute, item.String())
		}
	}

	if domains.Exists() {
		for _, item := range domains.Array() {
			config.MatchDomain = append(config.MatchDomain, item.String())
		}
	}

	allows := json.Get("allow")
	if allows.Exists() {
		for _, allow := range allows.Array() {
			config.Allow = append(config.Allow, allow.String())
		}
	}

	return nil
}

type Rules struct {
	Rules []*Rule `yaml:"_rules_,omitempty"`
}

func ParseRulesConfig(json gjson.Result, config *Rules, log wrapper.Log) error {
	rules := json.Get("_rules_")
	if rules.Exists() {
		for index, rule := range rules.Array() {
			err := ParseRuleConfig(rule, config.Rules[index], log)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
