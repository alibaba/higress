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

use higress_wasm_rust::event_stream::EventStream;
use higress_wasm_rust::log::Log;
use higress_wasm_rust::rule_matcher::{on_configure, RuleMatcher, SharedRuleMatcher};
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{ContextType, DataAction, HeaderAction, LogLevel};
use serde::Deserialize;
use std::cell::RefCell;
use std::ops::DerefMut;
use std::rc::Rc;
use std::str::from_utf8;
use std::time::{Duration, SystemTime};

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_|Box::new(SseTimingRoot::new()));
}}

struct SseTimingRoot {
    log: Rc<Log>,
    rule_matcher: SharedRuleMatcher<SseTimingConfig>,
}

struct SseTiming {
    log: Rc<Log>,
    rule_matcher: SharedRuleMatcher<SseTimingConfig>,
    vendor: String,
    is_event_stream: bool,
    event_stream: EventStream,
    start_time: SystemTime,
}

#[derive(Default, Clone, Debug, Deserialize)]
struct SseTimingConfig {
    vendor: Option<String>,
}

impl SseTimingRoot {
    fn new() -> Self {
        SseTimingRoot {
            log: Rc::new(Log::new("sse_timing".to_string())),
            rule_matcher: Rc::new(RefCell::new(RuleMatcher::default())),
        }
    }
}

impl Context for SseTimingRoot {}

impl RootContext for SseTimingRoot {
    fn on_configure(&mut self, plugin_configuration_size: usize) -> bool {
        on_configure(
            self,
            plugin_configuration_size,
            self.rule_matcher.borrow_mut().deref_mut(),
            &self.log,
        )
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(SseTiming {
            log: self.log.clone(),
            rule_matcher: self.rule_matcher.clone(),
            vendor: "higress".into(),
            is_event_stream: false,
            event_stream: EventStream::default(),
            start_time: self.get_current_time(),
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl Context for SseTiming {}

impl HttpContext for SseTiming {
    fn on_http_request_headers(
        &mut self,
        _num_headers: usize,
        _end_of_stream: bool,
    ) -> HeaderAction {
        self.start_time = self.get_current_time();

        let binding = self.rule_matcher.borrow();
        let config = match binding.get_match_config() {
            None => {
                return HeaderAction::Continue;
            }
            Some(config) => config.1,
        };
        match config.vendor.clone() {
            None => {}
            Some(vendor) => self.vendor = vendor,
        }
        HeaderAction::Continue
    }

    fn on_http_response_headers(
        &mut self,
        _num_headers: usize,
        _end_of_stream: bool,
    ) -> HeaderAction {
        match self.get_http_response_header("Content-Type") {
            None => self
                .log
                .warn("upstream response is not set Content-Type, skipped"),
            Some(content_type) => {
                if content_type.starts_with("text/event-stream") {
                    self.is_event_stream = true
                } else {
                    self.log.warn(format!("upstream response Content-Type is not text/event-stream, but {}, skipped", content_type).as_str())
                }
            }
        }
        HeaderAction::Continue
    }

    fn on_http_response_body(&mut self, body_size: usize, end_of_stream: bool) -> DataAction {
        if !self.is_event_stream {
            return DataAction::Continue;
        }

        let body = self
            .get_http_response_body(0, body_size)
            .unwrap_or_default();
        self.event_stream.update(body);
        self.process_event_stream(end_of_stream)
    }
}

impl SseTiming {
    fn process_event_stream(&mut self, end_of_stream: bool) -> DataAction {
        let mut modified_events = Vec::new();

        loop {
            match self.event_stream.next() {
                None => break,
                Some(raw_event) => {
                    if !raw_event.is_empty() {
                        // according to spec, event-stream must be utf-8 encoding
                        let event = from_utf8(raw_event.as_slice()).unwrap();
                        let processed_event = self.process_event(event.to_string());
                        modified_events.push(processed_event);
                    }
                }
            }
        }

        if end_of_stream {
            match self.event_stream.flush() {
                None => {}
                Some(raw_event) => {
                    if !raw_event.is_empty() {
                        // according to spec, event-stream must be utf-8 encoding
                        let event = from_utf8(raw_event.as_slice()).unwrap();
                        let modified_event = self.process_event(event.into());
                        modified_events.push(modified_event);
                    }
                }
            }
        }

        if !modified_events.is_empty() {
            let modified_body = modified_events.concat();
            self.set_http_response_body(0, modified_body.len(), modified_body.as_bytes());
            DataAction::Continue
        } else {
            DataAction::StopIterationNoBuffer
        }
    }

    fn process_event(&self, event: String) -> String {
        let duration = self
            .get_current_time()
            .duration_since(self.start_time)
            .unwrap_or(Duration::ZERO);
        format!(
            ": server-timing: {};dur={}\n{}\n\n",
            self.vendor,
            duration.as_millis(),
            event
        )
    }
}
