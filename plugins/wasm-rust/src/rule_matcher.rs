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
use proxy_wasm::traits::RootContext;
use serde::de::DeserializeOwned;
use serde_json::{from_slice, Map, Value};
use std::borrow::Borrow;
use std::cell::RefCell;
use std::collections::HashSet;
use std::rc::Rc;

#[derive(PartialEq)]
enum Category {
    Route,
    Host,
    RoutePrefix,
    Service,
}

#[derive(PartialEq)]
enum MatchType {
    Prefix,
    Exact,
    Suffix,
}

const RULES_KEY: &str = "_rules_";
const MATCH_ROUTE_KEY: &str = "_match_route_";
const MATCH_DOMAIN_KEY: &str = "_match_domain_";
const MATCH_SERVICE_KEY: &str = "_match_service_";
const MATCH_ROUTE_PREFIX_KEY: &str = "_match_route_prefix_";

pub type SharedRuleMatcher<PluginConfig> = Rc<RefCell<RuleMatcher<PluginConfig>>>;

#[derive(PartialEq)]
struct HostMatcher {
    match_type: MatchType,
    host: String,
}

struct RuleConfig<PluginConfig> {
    category: Category,
    routes: HashSet<String>,
    hosts: Vec<HostMatcher>,
    route_prefixes: HashSet<String>,
    services: HashSet<String>,
    config: Rc<PluginConfig>,
}

#[derive(Default)]
pub struct RuleMatcher<PluginConfig> {
    rule_config: Vec<RuleConfig<PluginConfig>>,
    global_config: Option<Rc<PluginConfig>>,
}

impl<PluginConfig> RuleMatcher<PluginConfig>
where
    PluginConfig: Default + DeserializeOwned,
{
    pub fn override_config(
        &mut self,
        override_func: fn(config: &PluginConfig, global: &PluginConfig) -> PluginConfig,
    ) {
        if let Some(global) = &self.global_config {
            for rule_config in &mut self.rule_config {
                rule_config.config = Rc::new(override_func(rule_config.config.borrow(), global));
            }
        }
    }
    pub fn parse_rule_config(&mut self, config: &Value) -> Result<(), WasmRustError> {
        let empty_object = Map::new();
        let empty_vec = Vec::new();

        let object = config.as_object().unwrap_or(&empty_object);
        let mut key_count = object.len();

        if object.is_empty() {
            self.global_config = Some(Rc::new(PluginConfig::default()));
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
                    self.global_config = Some(Rc::new(plugin_config));
                }
                Err(err) => {
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
            let services = RuleMatcher::<PluginConfig>::parse_service_match_config(rule_json);
            let route_prefixes =
                RuleMatcher::<PluginConfig>::parse_route_prefix_match_config(rule_json);

            let no_routes = routes.is_empty();
            let no_hosts = hosts.is_empty();
            let no_service = services.is_empty();
            let no_route_prefix = route_prefixes.is_empty();
            if [no_routes, no_hosts, no_service, no_route_prefix]
                .iter()
                .filter(|&x| *x)
                .count()
                != 3
            {
                return Err(WasmRustError::new("there is only one of  '_match_route_', '_match_domain_', '_match_service_' and '_match_route_prefix_' can present in configuration.".to_string()));
            }

            let category = if !no_routes {
                Category::Route
            } else if !no_hosts {
                Category::Host
            } else if !no_service {
                Category::Service
            } else {
                Category::RoutePrefix
            };

            self.rule_config.push(RuleConfig {
                category,
                routes,
                hosts,
                route_prefixes,
                services,
                config: Rc::new(config),
            })
        }

        Ok(())
    }

    pub fn get_match_config(&self) -> Option<(i64, Rc<PluginConfig>)> {
        let host = get_http_request_header(":authority").unwrap_or_default();
        let route_name = String::from_utf8(get_property(vec!["route_name"]).unwrap_or_default())
            .unwrap_or_else(|_| "".to_string());
        let service_name =
            String::from_utf8(get_property(vec!["cluster_name"]).unwrap_or_default())
                .unwrap_or_else(|_| "".to_string());

        for (i, rule) in self.rule_config.iter().enumerate() {
            match rule.category {
                Category::Host => {
                    if self.host_match(rule, host.as_str()) {
                        return Some((i as i64, rule.config.clone()));
                    }
                }
                Category::Route => {
                    if rule.routes.contains(route_name.as_str()) {
                        return Some((i as i64, rule.config.clone()));
                    }
                }
                Category::RoutePrefix => {
                    for route_prefix in &rule.route_prefixes {
                        if route_name.starts_with(route_prefix) {
                            return Some((i as i64, rule.config.clone()));
                        }
                    }
                }
                Category::Service => {
                    if self.service_match(rule, &service_name) {
                        return Some((i as i64, rule.config.clone()));
                    }
                }
            }
        }

        self.global_config
            .as_ref()
            .map(|config| (usize::MAX as i64, config.clone()))
    }

    pub fn rewrite_config(&mut self, rewrite: fn(config: &PluginConfig) -> PluginConfig) {
        if let Some(global_config) = &self.global_config {
            self.global_config = Some(Rc::new(rewrite(global_config.borrow())));
        }

        for rule_config in &mut self.rule_config {
            rule_config.config = Rc::new(rewrite(rule_config.config.borrow()));
        }
    }

    fn parse_match_config(json_key: &str, config: &Value) -> HashSet<String> {
        let empty_vec = Vec::new();
        let keys = config[json_key].as_array().unwrap_or(&empty_vec);
        let mut values = HashSet::new();
        for key in keys {
            let value = key.as_str().unwrap_or("").to_string();
            if !value.is_empty() {
                values.insert(value);
            }
        }
        values
    }
    fn parse_route_match_config(config: &Value) -> HashSet<String> {
        Self::parse_match_config(MATCH_ROUTE_KEY, config)
    }
    fn parse_service_match_config(config: &Value) -> HashSet<String> {
        Self::parse_match_config(MATCH_SERVICE_KEY, config)
    }
    fn parse_route_prefix_match_config(config: &Value) -> HashSet<String> {
        Self::parse_match_config(MATCH_ROUTE_PREFIX_KEY, config)
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
    fn strip_port_from_host(req_host: &str) -> String {
        // Port removing code is inspired by
        // https://github.com/envoyproxy/envoy/blob/v1.17.0/source/common/http/header_utility.cc#L219
        if let Some(port_start) = req_host.rfind(':') {
            // According to RFC3986 v6 address is always enclosed in "[]".
            // section 3.2.2.
            let v6_end_index = req_host.rfind(']');
            if v6_end_index.map_or(true, |idx| idx < port_start) && port_start < req_host.len() {
                return req_host[..port_start].to_string();
            }
        }
        req_host.to_string()
    }
    fn host_match(&self, rule: &RuleConfig<PluginConfig>, request_host: &str) -> bool {
        let request_host = Self::strip_port_from_host(request_host);
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
    fn service_match(&self, rule: &RuleConfig<PluginConfig>, service_name: &str) -> bool {
        let parts = service_name.split("|").collect::<Vec<&str>>();
        if parts.len() != 4 {
            return false;
        }
        let port = parts[1];
        let fqdn = parts[3];
        for config_service_name in &rule.services {
            if let Some(colon_index) = config_service_name.rfind(':') {
                if fqdn == &config_service_name[..colon_index]
                    && port == &config_service_name[colon_index + 1..]
                {
                    return true;
                }
            } else if fqdn == config_service_name {
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

    if let Err(err) = rule_matcher.parse_rule_config(&value) {
        log.error(format!("parse_rule_config fail {}", err).as_str());
        false
    } else {
        true
    }
}

#[cfg(test)]
mod tests {
    use std::vec;

    use serde::Deserialize;

    use super::*;

    #[derive(Default, Deserialize, PartialEq, Eq)]
    struct CustomConfig {
        #[serde(default)]
        name: String,
        #[serde(default)]
        age: i64,
    }

    impl CustomConfig {
        fn new(name: &str, age: i64) -> Self {
            CustomConfig {
                name: name.to_string(),
                age,
            }
        }
    }
    struct RuleConfigBuilder<Config> {
        config: RuleConfig<Config>,
    }

    impl<Config> RuleConfigBuilder<Config> {
        fn new(category: Category, config: Rc<Config>) -> Self {
            RuleConfigBuilder {
                config: RuleConfig {
                    category,
                    config,
                    routes: HashSet::default(),
                    hosts: Vec::default(),
                    route_prefixes: HashSet::default(),
                    services: HashSet::default(),
                },
            }
        }
        fn add_host(mut self, match_type: MatchType, host: &str) -> Self {
            self.config.hosts.push(HostMatcher {
                match_type,
                host: host.to_string(),
            });
            self
        }
        fn add_route(mut self, route: &str) -> Self {
            self.config.routes.insert(route.to_string());
            self
        }
        fn add_route_prefix(mut self, route_prefix: &str) -> Self {
            self.config.route_prefixes.insert(route_prefix.to_string());
            self
        }
        fn add_service(mut self, service_name: &str) -> Self {
            self.config.services.insert(service_name.to_string());
            self
        }
        fn config(self) -> RuleConfig<Config> {
            self.config
        }
    }
    struct MatchTestCase<Config> {
        name: String,
        config: RuleConfig<Config>,
        key: String,
        result: bool,
    }
    impl<Config> MatchTestCase<Config> {
        fn new(name: &str, key: &str, result: bool, config: RuleConfig<Config>) -> Self {
            MatchTestCase {
                name: name.to_string(),
                key: key.to_string(),
                result,
                config,
            }
        }
    }
    #[test]
    fn test_host_match() {
        let config = Rc::new(CustomConfig::new("test", 1));
        let cases = vec![
            MatchTestCase::new(
                "prefix",
                "www.test.com",
                true,
                RuleConfigBuilder::new(Category::Host, config.clone())
                    .add_host(MatchType::Prefix, "www.")
                    .config(),
            ),
            MatchTestCase::new(
                "prefix failed",
                "test.com",
                false,
                RuleConfigBuilder::new(Category::Host, config.clone())
                    .add_host(MatchType::Prefix, "www.")
                    .config(),
            ),
            MatchTestCase::new(
                "suffix",
                "www.example.com",
                true,
                RuleConfigBuilder::new(Category::Host, config.clone())
                    .add_host(MatchType::Suffix, ".example.com")
                    .config(),
            ),
            MatchTestCase::new(
                "suffix failed",
                "example.com",
                false,
                RuleConfigBuilder::new(Category::Host, config.clone())
                    .add_host(MatchType::Suffix, ".example.com")
                    .config(),
            ),
            MatchTestCase::new(
                "exact",
                "www.example.com",
                true,
                RuleConfigBuilder::new(Category::Host, config.clone())
                    .add_host(MatchType::Exact, "www.example.com")
                    .config(),
            ),
            MatchTestCase::new(
                "exact failed",
                "example.com",
                false,
                RuleConfigBuilder::new(Category::Host, config.clone())
                    .add_host(MatchType::Exact, "www.example.com")
                    .config(),
            ),
            MatchTestCase::new(
                "exact port",
                "www.example.com:8080",
                true,
                RuleConfigBuilder::new(Category::Host, config.clone())
                    .add_host(MatchType::Exact, "www.example.com")
                    .config(),
            ),
            MatchTestCase::new(
                "any",
                "www.example.com",
                true,
                RuleConfigBuilder::new(Category::Host, config.clone())
                    .add_host(MatchType::Suffix, "")
                    .config(),
            ),
        ];
        for case in &cases {
            println!("test {} start", case.name);
            let rule = RuleMatcher::default();
            assert_eq!(
                case.result,
                rule.host_match(&case.config, case.key.as_str())
            );
        }
    }

    #[test]
    fn test_service_match() {
        let config = Rc::new(CustomConfig::new("test", 1));
        let cases = vec![
            MatchTestCase::new(
                "fqdn",
                "outbound|443||qwen.dns",
                true,
                RuleConfigBuilder::new(Category::Service, config.clone())
                    .add_service("qwen.dns")
                    .config(),
            ),
            MatchTestCase::new(
                "fqdn with port",
                "outbound|443||qwen.dns",
                true,
                RuleConfigBuilder::new(Category::Service, config.clone())
                    .add_service("qwen.dns:443")
                    .config(),
            ),
            MatchTestCase::new(
                "not match",
                "outbound|443||qwen.dns",
                false,
                RuleConfigBuilder::new(Category::Service, config.clone())
                    .add_service("moonshot.dns:443")
                    .config(),
            ),
            MatchTestCase::new(
                "error config format",
                "outbound|443||qwen.dns",
                false,
                RuleConfigBuilder::new(Category::Service, config.clone())
                    .add_service("qwen.dns:")
                    .config(),
            ),
        ];
        for case in &cases {
            println!("test {} start", case.name);
            let rule = RuleMatcher::default();
            assert_eq!(
                case.result,
                rule.service_match(&case.config, case.key.as_str())
            );
        }
    }

    struct ParseTestCase<Config> {
        name: String,
        config: String,
        err_msg: String,
        expected: RuleMatcher<Config>,
    }

    impl<Config> ParseTestCase<Config>
    where
        Config: DeserializeOwned + PartialEq + Eq + Default,
    {
        fn new(name: &str, config: &str, err_msg: &str) -> Self {
            ParseTestCase {
                name: name.to_string(),
                config: config.to_string(),
                err_msg: err_msg.to_string(),
                expected: RuleMatcher::default(),
            }
        }
        fn global_config(mut self, config: Config) -> Self {
            self.expected.global_config = Some(Rc::new(config));
            self
        }

        fn rule_config(mut self, config: RuleConfig<Config>) -> Self {
            self.expected.rule_config.push(config);
            self
        }
        fn is_eq(&self, other: &RuleMatcher<Config>) -> bool {
            if self.expected.global_config.is_some() != other.global_config.is_some() {
                return false;
            }
            if let (Some(a), Some(b)) = (&self.expected.global_config, &other.global_config) {
                if a != b {
                    return false;
                }
            }
            if self.expected.rule_config.len() != other.rule_config.len() {
                return false;
            }
            for (s, o) in self
                .expected
                .rule_config
                .iter()
                .zip(other.rule_config.iter())
            {
                if s.category != o.category
                    || s.config != o.config
                    || s.routes != o.routes
                    || s.hosts != o.hosts
                    || s.route_prefixes != o.route_prefixes
                    || s.services != o.services
                {
                    return false;
                }
            }
            true
        }
    }

    #[test]
    fn test_parse_rule_config() {
        let cases = vec![
            ParseTestCase::new("global config", r#"{"name":"john", "age":18}"#, "").global_config(CustomConfig::new("john", 18)),
            ParseTestCase::new("no rule", r#"{"_rules_":[]}"#, "parse config failed, no valid rules; global config parse error:"),
            ParseTestCase::new("invalid rule", r#"{"_rules_":[{"_match_domain_":["*"],"_match_route_":["test"]}]}"#, "there is only one of  '_match_route_', '_match_domain_', '_match_service_' and '_match_route_prefix_' can present in configuration."),
            ParseTestCase::new("invalid rule", r#"{"_rules_":[{"_match_domain_":["*"],"_match_service_":["test.dns"]}]}"#, "there is only one of  '_match_route_', '_match_domain_', '_match_service_' and '_match_route_prefix_' can present in configuration."),
            ParseTestCase::new("invalid rule", r#"{"_rules_":[{"age":16}]}"#, "there is only one of  '_match_route_', '_match_domain_', '_match_service_' and '_match_route_prefix_' can present in configuration."),
            ParseTestCase::new("rules config", r#"{"_rules_":[{"_match_domain_":["*.example.com","www.*","*","www.abc.com"],"name":"john", "age":18},{"_match_route_":["test1","test2"],"name":"ann", "age":16},{"_match_service_":["test1.dns","test2.static:8080"],"name":"ann", "age":16},{"_match_route_prefix_":["api1","api2"],"name":"ann", "age":16}]}"#, "")
                .rule_config(RuleConfigBuilder::new(Category::Host, Rc::new(CustomConfig::new("john", 18))).add_host(MatchType::Suffix, ".example.com").add_host(MatchType::Prefix, "www.").add_host(MatchType::Suffix, "").add_host(MatchType::Exact, "www.abc.com").config())
                .rule_config(RuleConfigBuilder::new(Category::Route, Rc::new(CustomConfig::new("ann", 16))).add_route("test1").add_route("test2").config())
                .rule_config(RuleConfigBuilder::new(Category::Service, Rc::new(CustomConfig::new("ann", 16))).add_service("test1.dns").add_service("test2.static:8080").config())
                .rule_config(RuleConfigBuilder::new(Category::RoutePrefix, Rc::new(CustomConfig::new("ann", 16))).add_route_prefix("api1").add_route_prefix("api2").config())
        ];
        for case in &cases {
            println!("test {} start", case.name);

            let mut rule = RuleMatcher::default();

            let res = rule.parse_rule_config(&serde_json::from_str(&case.config).unwrap());
            if let Err(e) = res {
                assert_eq!(case.err_msg, e.to_string());
            } else {
                assert!(case.err_msg.is_empty())
            }
            assert!(case.is_eq(&rule));
        }
    }

    #[derive(Default, Clone, Deserialize, PartialEq, Eq)]
    struct CompleteConfig {
        // global config
        #[serde(default)]
        consumers: Vec<String>,
        // rule config
        #[serde(default)]
        allow: Vec<String>,
    }
    impl CompleteConfig {
        fn new(consumers: Vec<&str>, allow: Vec<&str>) -> Self {
            CompleteConfig {
                consumers: consumers.iter().map(|s| s.to_string()).collect(),
                allow: allow.iter().map(|s| s.to_string()).collect(),
            }
        }
    }
    fn override_config(config: &CompleteConfig, global: &CompleteConfig) -> CompleteConfig {
        let mut new_config = global.clone();
        new_config.allow.extend(config.allow.clone());
        new_config
    }

    #[test]
    fn test_parse_override_config() {
        let cases = vec![
            ParseTestCase::new("override rule config", r#"{"consumers":["c1","c2","c3"],"_rules_":[{"_match_route_":["r1","r2"],"allow":["c1","c3"]}]}"#, "")
                .global_config(CompleteConfig::new(vec!["c1", "c2", "c3"], vec![]))
                .rule_config(RuleConfigBuilder::new(Category::Route, Rc::new(CompleteConfig::new(vec!["c1", "c2", "c3"], vec!["c1", "c3"]))).add_route("r1").add_route("r2").config())
        ];
        for case in &cases {
            println!("test {} start", case.name);

            let mut rule = RuleMatcher::default();

            let res = rule.parse_rule_config(&serde_json::from_str(&case.config).unwrap());
            if res.is_ok() {
                rule.override_config(override_config);
            }
            if let Err(e) = res {
                assert_eq!(case.err_msg, e.to_string());
            } else {
                assert!(case.err_msg.is_empty())
            }
            assert!(case.is_eq(&rule));
        }
    }
}
