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

use crate::error::WasmRustError;
use crate::internal::{get_http_request_header, get_property};
use crate::log::Log;
use proxy_wasm::hostcalls::log;
use proxy_wasm::traits::RootContext;
use proxy_wasm::types::LogLevel;
use serde::de::DeserializeOwned;
use serde_json::{from_slice, Map, Value};
use std::cell::RefCell;
use std::collections::HashSet;
use std::rc::Rc;

enum Category {
    Route,
    Host,
}

enum MatchType {
    Prefix,
    Exact,
    Suffix,
}

const RULES_KEY: &str = "_rules_";
const MATCH_ROUTE_KEY: &str = "_match_route_";
const MATCH_DOMAIN_KEY: &str = "_match_domain_";

pub type SharedRuleMatcher<PluginConfig> = Rc<RefCell<RuleMatcher<PluginConfig>>>;

struct HostMatcher {
    match_type: MatchType,
    host: String,
}

struct RuleConfig<PluginConfig> {
    category: Category,
    routes: HashSet<String>,
    hosts: Vec<HostMatcher>,
    config: PluginConfig,
}

#[derive(Default)]
pub struct RuleMatcher<PluginConfig> {
    rule_config: Vec<RuleConfig<PluginConfig>>,
    global_config: Option<PluginConfig>,
}

impl<PluginConfig> RuleMatcher<PluginConfig>
where
    PluginConfig: Default + DeserializeOwned,
{
    pub fn parse_rule_config(&mut self, config: &Value) -> Result<(), WasmRustError> {
        let empty_object = Map::new();
        let empty_vec = Vec::new();

        let object = config.as_object().unwrap_or(&empty_object);
        let mut key_count = object.len();

        if object.is_empty() {
            self.global_config = Some(PluginConfig::default());
            return Ok(());
        }

        let rules = if object.contains_key(RULES_KEY) {
            key_count -= 1;
            object[RULES_KEY].as_array().unwrap_or(&empty_vec)
        } else {
            &empty_vec
        };

        let mut global_config_error: WasmRustError = WasmRustError::default();
        if key_count > 0 {
            match serde_json::from_value::<PluginConfig>(config.clone()) {
                Ok(plugin_config) => {
                    self.global_config = Some(plugin_config);
                }
                Err(err) => {
                    log(
                        LogLevel::Warn,
                        format!("parse global config failed, err:{:?}", err).as_str(),
                    )
                    .unwrap();
                    global_config_error = WasmRustError::new(err.to_string());
                }
            }
        }

        if rules.is_empty() {
            return match self.global_config {
                Some(_) => Ok(()),
                None => Err(WasmRustError::new(format!(
                    "parse config failed, no valid rules; global config parse error:{}",
                    global_config_error
                ))),
            };
        }

        for rule_json in rules {
            let config = match serde_json::from_value::<PluginConfig>(rule_json.clone()) {
                Ok(config) => config,
                Err(error) => return Err(WasmRustError::new(error.to_string())),
            };
            let routes = RuleMatcher::<PluginConfig>::parse_route_match_config(rule_json);
            let hosts = RuleMatcher::<PluginConfig>::parse_host_match_config(rule_json);

            let no_routes = routes.is_empty();
            let no_hosts = hosts.is_empty();

            if (no_routes && no_hosts) || (!no_routes && !no_hosts) {
                return Err(WasmRustError::new("there is only one of  '_match_route_' and '_match_domain_' can present in configuration.".to_string()));
            }

            let category = if no_routes {
                Category::Host
            } else {
                Category::Route
            };

            self.rule_config.push(RuleConfig {
                category,
                routes,
                hosts,
                config,
            })
        }

        Ok(())
    }

    pub fn get_match_config(&self) -> Option<(i64, &PluginConfig)> {
        let host = get_http_request_header(":authority").unwrap_or_default();
        let route_name = get_property(vec!["route_name"]).unwrap_or_default();

        for (i, rule) in self.rule_config.iter().enumerate() {
            match rule.category {
                Category::Host => {
                    if self.host_match(rule, host.as_str()) {
                        return Some((i as i64, &rule.config));
                    }
                }
                Category::Route => {
                    if rule.routes.contains(
                        String::from_utf8(route_name.to_vec())
                            .unwrap_or_else(|_| "".to_string())
                            .as_str(),
                    ) {
                        return Some((i as i64, &rule.config));
                    }
                }
            }
        }

        self.global_config
            .as_ref()
            .map(|config| (usize::MAX as i64, config))
    }

    pub fn rewrite_config(&mut self, rewrite: fn(config: &PluginConfig) -> PluginConfig) {
        self.global_config = self.global_config.as_ref().map(rewrite);

        for rule_config in &mut self.rule_config {
            rule_config.config = rewrite(&rule_config.config);
        }
    }

    fn parse_route_match_config(config: &Value) -> HashSet<String> {
        let empty_vec = Vec::new();
        let keys = config[MATCH_ROUTE_KEY].as_array().unwrap_or(&empty_vec);
        let mut routes = HashSet::new();
        for key in keys {
            let route_name = key.as_str().unwrap_or("").to_string();
            if !route_name.is_empty() {
                routes.insert(route_name);
            }
        }
        routes
    }

    fn parse_host_match_config(config: &Value) -> Vec<HostMatcher> {
        let empty_vec = Vec::new();
        let keys = config[MATCH_DOMAIN_KEY].as_array().unwrap_or(&empty_vec);
        let mut host_matchers: Vec<HostMatcher> = Vec::new();
        for key in keys {
            let host = key.as_str().unwrap_or("").to_string();
            let mut host_matcher = HostMatcher {
                match_type: MatchType::Prefix,
                host: String::new(),
            };
            if let Some(suffix) = host.strip_prefix('*') {
                host_matcher.match_type = MatchType::Suffix;
                host_matcher.host = suffix.to_string()
            } else if let Some(prefix) = host.strip_suffix('*') {
                host_matcher.match_type = MatchType::Prefix;
                host_matcher.host = prefix.to_string();
            } else {
                host_matcher.match_type = MatchType::Exact;
                host_matcher.host = host
            }
            host_matchers.push(host_matcher)
        }
        host_matchers
    }

    fn host_match(&self, rule: &RuleConfig<PluginConfig>, request_host: &str) -> bool {
        for host in &rule.hosts {
            let matched = match host.match_type {
                MatchType::Prefix => request_host.starts_with(host.host.as_str()),
                MatchType::Suffix => request_host.ends_with(host.host.as_str()),
                MatchType::Exact => request_host == host.host.as_str(),
            };
            if matched {
                return true;
            }
        }
        false
    }
}

pub fn on_configure<RC: RootContext, PluginConfig: Default + DeserializeOwned>(
    root_context: &RC,
    _plugin_configuration_size: usize,
    rule_matcher: &mut RuleMatcher<PluginConfig>,
    log: &Log,
) -> bool {
    let config_buffer = match root_context.get_plugin_configuration() {
        None => {
            log.error("Error when configuring RootContext, no configuration supplied");
            return false;
        }
        Some(bytes) => bytes,
    };

    let value = match from_slice::<Value>(config_buffer.as_slice()) {
        Err(error) => {
            log.error(format!("cannot parse plugin configuration JSON string: {}", error).as_str());
            return false;
        }
        Ok(value) => value,
    };

    rule_matcher.parse_rule_config(&value).is_ok()
}
