// Copyright (c) 2024 Alibaba Group Holding Ltd.
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

use higress_wasm_rust::dispatcher::HttpDispatcher;
use higress_wasm_rust::log::Log;
use higress_wasm_rust::rule_matcher::{on_configure, SharedRuleMatcher};
use higress_wasm_rust::{hostcalls, rule_matcher};
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{ContextType, HeaderAction, LogLevel};
use serde::Deserialize;
use std::ops::DerefMut;
use std::rc::Rc;
use std::time::Duration;

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_|Box::new(HttpPromiseExampleRoot::new()));
}}

struct HttpPromiseExampleRoot {
    log: Rc<Log>,
    rule_matcher: SharedRuleMatcher<HttpPromiseExampleConfig>,
}

struct HttpPromiseExample {
    log: Rc<Log>,
    rule_matcher: SharedRuleMatcher<HttpPromiseExampleConfig>,
    http_dispatcher: HttpDispatcher,
}

#[derive(Default, Clone, Debug, Deserialize)]
struct HttpPromiseExampleConfig {}

impl HttpPromiseExampleRoot {
    fn new() -> Self {
        HttpPromiseExampleRoot {
            log: Rc::new(Log::new("http-promise-example".to_string())),
            rule_matcher: rule_matcher::new_shared(),
        }
    }
}

impl Context for HttpPromiseExampleRoot {}

impl RootContext for HttpPromiseExampleRoot {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        on_configure(
            self,
            _plugin_configuration_size,
            self.rule_matcher.borrow_mut().deref_mut(),
            &self.log,
        )
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(HttpPromiseExample {
            log: self.log.clone(),
            rule_matcher: self.rule_matcher.clone(),
            http_dispatcher: Default::default(),
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl Context for HttpPromiseExample {
    fn on_http_call_response(
        &mut self,
        _token_id: u32,
        _num_headers: usize,
        _body_size: usize,
        _num_trailers: usize,
    ) {
        self.http_dispatcher
            .callback(_token_id, _num_headers, _body_size, _num_trailers)
    }
}

impl HttpContext for HttpPromiseExample {
    fn on_http_request_headers(&mut self, _: usize, _: bool) -> HeaderAction {
        let log = self.log.clone();

        self.http_dispatcher
            .dispatch(
                "httpbin",
                vec![
                    (":method", "GET"),
                    (":path", "/bytes/1"),
                    (":authority", "httpbin.org"),
                ],
                None,
                vec![],
                Duration::from_secs(1),
            )
            .then(move |(_, _, _body_size, _)| {
                if let Some(body) = hostcalls::get_http_call_response_body(0, _body_size) {
                    if !body.is_empty() && body[0] % 2 == 0 {
                        log.info("Access granted.");
                        hostcalls::resume_http_request();
                        return;
                    }
                }
                log.info("Access forbidden.");
                hostcalls::send_http_response(
                    403,
                    vec![("Powered-By", "proxy-wasm")],
                    Some(b"Access forbidden.\n"),
                );
            });
        HeaderAction::StopIteration
    }

    fn on_http_response_headers(&mut self, _: usize, _: bool) -> HeaderAction {
        self.set_http_response_header("Powered-By", Some("proxy-wasm"));
        HeaderAction::Continue
    }
}
