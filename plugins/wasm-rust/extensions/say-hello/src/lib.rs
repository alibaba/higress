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

use higress_wasm_rust::log::Log;
use higress_wasm_rust::rule_matcher::RuleMatcher;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, ContextType, LogLevel};
use serde::Deserialize;
use serde_json::{from_slice, Value};
use std::cell::RefCell;
use std::rc::Rc;

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_|Box::new(SayHelloRoot::new()));
}}

struct SayHelloRoot {
    log: Log,
    rule_matcher: Rc<RefCell<RuleMatcher<SayHelloConfig>>>,
}

struct SayHello {
    rule_matcher: Rc<RefCell<RuleMatcher<SayHelloConfig>>>,
}

#[derive(Default, Debug, Deserialize)]
struct SayHelloConfig {
    name: String,
}

impl SayHelloRoot {
    fn new() -> Self {
        SayHelloRoot {
            log: Log::new("say_hello".to_string()),
            rule_matcher: Rc::new(RefCell::new(RuleMatcher::default())),
        }
    }
}

impl Context for SayHelloRoot {}

impl RootContext for SayHelloRoot {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        let config_buffer = match self.get_plugin_configuration() {
            None => {
                self.log
                    .error("Error when configuring RootContext, no configuration supplied");
                return false;
            }
            Some(bytes) => bytes,
        };

        let value = match from_slice::<Value>(config_buffer.as_slice()) {
            Err(error) => {
                self.log.error(
                    format!("cannot parse plugin configuration JSON string: {}", error).as_str(),
                );
                return false;
            }
            Ok(value) => value,
        };

        self.rule_matcher
            .borrow_mut()
            .parse_rule_config(&value)
            .is_ok()
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(SayHello {
            rule_matcher: self.rule_matcher.clone(),
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl Context for SayHello {}

impl HttpContext for SayHello {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        let binding = self.rule_matcher.borrow();
        let config = match binding.get_match_config() {
            None => {
                self.send_http_response(200, vec![], Some("Hello, World!".as_bytes()));
                return Action::Continue;
            }
            Some(config) => config,
        };

        self.send_http_response(
            200,
            vec![],
            Some(format!("Hello, {}!", config.name).as_bytes()),
        );
        Action::Continue
    }
}
