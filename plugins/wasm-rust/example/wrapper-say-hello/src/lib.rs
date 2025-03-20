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
use higress_wasm_rust::plugin_wrapper::{HttpContextWrapper, RootContextWrapper};
use higress_wasm_rust::rule_matcher::{on_configure, RuleMatcher, SharedRuleMatcher};
use multimap::MultiMap;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{ContextType, HeaderAction, LogLevel};
use serde::Deserialize;
use std::cell::RefCell;
use std::ops::DerefMut;
use std::rc::{Rc, Weak};

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_|Box::new(SayHelloRoot::new()));
}}

const PLUGIN_NAME: &str = "wrapper-demo-wasm";
struct SayHelloRoot {
    log: Log,
    rule_matcher: SharedRuleMatcher<SayHelloConfig>,
}

struct SayHello {
    rule_matcher: SharedRuleMatcher<SayHelloConfig>,
    log: Log,
    config: Option<Rc<SayHelloConfig>>,
    weak: Weak<RefCell<Box<dyn HttpContextWrapper<SayHelloConfig>>>>,
}

#[derive(Default, Debug, Deserialize, Clone)]
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
        self.log.info("DemoWasmRoot::on_configure");
        on_configure(
            self,
            _plugin_configuration_size,
            self.rule_matcher.borrow_mut().deref_mut(),
            &self.log,
        )
    }
    fn create_http_context(&self, context_id: u32) -> Option<Box<dyn HttpContext>> {
        self.log.info(&format!(
            "DemoWasmRoot::create_http_context({})",
            context_id
        ));
        self.create_http_context_use_wrapper(context_id)
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl RootContextWrapper<SayHelloConfig> for SayHelloRoot {
    fn rule_matcher(&self) -> &SharedRuleMatcher<SayHelloConfig> {
        &self.rule_matcher
    }

    fn create_http_context_wrapper(
        &self,
        _context_id: u32,
    ) -> Option<Box<dyn HttpContextWrapper<SayHelloConfig>>> {
        Some(Box::new(SayHello {
            log: Log::new(PLUGIN_NAME.to_string()),
            config: None,
            rule_matcher: Rc::new(RefCell::new(RuleMatcher::default())),
            weak: Default::default(),
        }))
    }
}

impl Context for SayHello {}

impl HttpContext for SayHello {}

impl HttpContextWrapper<SayHelloConfig> for SayHello {
    fn init_self_weak(
        &mut self,
        self_weak: Weak<RefCell<Box<dyn HttpContextWrapper<SayHelloConfig>>>>,
    ) {
        self.weak = self_weak;
        self.log.info("init_self_rc");
    }
    fn log(&self) -> &Log {
        &self.log
    }
    fn on_config(&mut self, config: Rc<SayHelloConfig>) {
        // 获取config
        self.log.info(&format!("on_config {}", config.name));
        self.config = Some(config.clone());
    }
    fn on_http_request_complete_headers(
        &mut self,
        _headers: &multimap::MultiMap<String, String>,
    ) -> HeaderAction {
        let binding = self.rule_matcher.borrow();
        let config = match binding.get_match_config() {
            None => {
                self.send_http_response(200, vec![], Some("Hello, World!".as_bytes()));
                return HeaderAction::Continue;
            }
            Some(config) => config.1,
        };

        self.send_http_response(
            200,
            vec![],
            Some(format!("Hello, {}!", config.name).as_bytes()),
        );
        HeaderAction::Continue
    }

    fn on_http_response_complete_headers(
        &mut self,
        _headers: &MultiMap<String, String>,
    ) -> HeaderAction {
        self.set_response_body_buffer_limit(381881);
        HeaderAction::Continue
    }
}
