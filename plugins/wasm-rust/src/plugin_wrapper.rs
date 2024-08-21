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

use crate::rule_matcher::SharedRuleMatcher;
use multimap::MultiMap;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, Bytes, DataAction, HeaderAction};
use serde::de::DeserializeOwned;

pub trait RootContextWrapper<PluginConfig>: RootContext
where
    PluginConfig: Default + DeserializeOwned + 'static + Clone,
{
    // fn create_http_context(&self, context_id: u32) -> Option<Box<dyn HttpContext>> {
    fn create_http_context_use_wrapper(&self, context_id: u32) -> Option<Box<dyn HttpContext>> {
        // trait 继承没法重写 RootContext 的 create_http_context，先写个函数让上层调下吧
        match self.create_http_context_wrapper(context_id) {
            Some(http_context) => Some(Box::new(PluginHttpWrapper::new(
                self.rule_matcher(),
                http_context,
            ))),
            None => None,
        }
    }
    fn rule_matcher(&self) -> &SharedRuleMatcher<PluginConfig>;
    fn create_http_context_wrapper(
        &self,
        _context_id: u32,
    ) -> Option<Box<dyn HttpContextWrapper<PluginConfig>>> {
        None
    }
}
pub trait HttpContextWrapper<PluginConfig>: HttpContext {
    fn on_config(&mut self, _config: &PluginConfig) {}
    fn on_http_request_complete_headers(
        &mut self,
        _headers: &MultiMap<String, String>,
    ) -> HeaderAction {
        HeaderAction::Continue
    }
    fn on_http_response_complete_headers(
        &mut self,
        _headers: &MultiMap<String, String>,
    ) -> HeaderAction {
        HeaderAction::Continue
    }
    fn cache_request_body(&self) -> bool {
        false
    }
    fn cache_response_body(&self) -> bool {
        false
    }
    fn on_http_request_complete_body(&mut self, _req_body: &Bytes) -> DataAction {
        DataAction::Continue
    }
    fn on_http_response_complete_body(&mut self, _res_body: &Bytes) -> DataAction {
        DataAction::Continue
    }
    fn replace_http_request_body(&mut self, body: &[u8]) {
        self.set_http_request_body(0, i32::MAX as usize, body)
    }
    fn replace_http_response_body(&mut self, body: &[u8]) {
        self.set_http_response_body(0, i32::MAX as usize, body)
    }
}
pub struct PluginHttpWrapper<PluginConfig> {
    req_headers: MultiMap<String, String>,
    res_headers: MultiMap<String, String>,
    req_body_len: usize,
    res_body_len: usize,
    config: Option<PluginConfig>,
    rule_matcher: SharedRuleMatcher<PluginConfig>,
    http_content: Box<dyn HttpContextWrapper<PluginConfig>>,
}
impl<PluginConfig> PluginHttpWrapper<PluginConfig> {
    pub fn new(
        rule_matcher: &SharedRuleMatcher<PluginConfig>,
        http_content: Box<dyn HttpContextWrapper<PluginConfig>>,
    ) -> Self {
        PluginHttpWrapper {
            req_headers: MultiMap::new(),
            res_headers: MultiMap::new(),
            req_body_len: 0,
            res_body_len: 0,
            config: None,
            rule_matcher: rule_matcher.clone(),
            http_content,
        }
    }
}
impl<PluginConfig> Context for PluginHttpWrapper<PluginConfig> {
    fn on_http_call_response(
        &mut self,
        token_id: u32,
        num_headers: usize,
        body_size: usize,
        num_trailers: usize,
    ) {
        self.http_content
            .on_http_call_response(token_id, num_headers, body_size, num_trailers)
    }

    fn on_grpc_call_response(&mut self, token_id: u32, status_code: u32, response_size: usize) {
        self.http_content
            .on_grpc_call_response(token_id, status_code, response_size)
    }
    fn on_grpc_stream_initial_metadata(&mut self, token_id: u32, num_elements: u32) {
        self.http_content
            .on_grpc_stream_initial_metadata(token_id, num_elements)
    }
    fn on_grpc_stream_message(&mut self, token_id: u32, message_size: usize) {
        self.http_content
            .on_grpc_stream_message(token_id, message_size)
    }
    fn on_grpc_stream_trailing_metadata(&mut self, token_id: u32, num_elements: u32) {
        self.http_content
            .on_grpc_stream_trailing_metadata(token_id, num_elements)
    }
    fn on_grpc_stream_close(&mut self, token_id: u32, status_code: u32) {
        self.http_content
            .on_grpc_stream_close(token_id, status_code)
    }

    fn on_done(&mut self) -> bool {
        self.http_content.on_done()
    }
}
impl<PluginConfig> HttpContext for PluginHttpWrapper<PluginConfig>
where
    PluginConfig: Default + DeserializeOwned + Clone,
{
    fn on_http_request_headers(&mut self, num_headers: usize, end_of_stream: bool) -> HeaderAction {
        let binding = self.rule_matcher.borrow();
        self.config = binding.get_match_config().map(|config| config.1.clone());
        for (k, v) in self.get_http_request_headers() {
            self.req_headers.insert(k, v);
        }
        if let Some(config) = &self.config {
            self.http_content.on_config(config);
        }
        let ret = self
            .http_content
            .on_http_request_headers(num_headers, end_of_stream);
        if ret != HeaderAction::Continue {
            return ret;
        }
        self.http_content
            .on_http_request_complete_headers(&self.req_headers)
    }

    fn on_http_request_body(&mut self, body_size: usize, end_of_stream: bool) -> DataAction {
        if !self.http_content.cache_request_body() {
            return self
                .http_content
                .on_http_request_body(body_size, end_of_stream);
        }
        self.req_body_len += body_size;
        if !end_of_stream {
            return DataAction::StopIterationAndBuffer;
        }
        let mut req_body = Bytes::new();
        if self.req_body_len > 0 {
            if let Some(body) = self.get_http_request_body(0, self.req_body_len) {
                req_body = body;
            }
        }
        self.http_content.on_http_request_complete_body(&req_body)
    }

    fn on_http_request_trailers(&mut self, num_trailers: usize) -> Action {
        self.http_content.on_http_request_trailers(num_trailers)
    }

    fn on_http_response_headers(
        &mut self,
        num_headers: usize,
        end_of_stream: bool,
    ) -> HeaderAction {
        for (k, v) in self.get_http_response_headers() {
            self.res_headers.insert(k, v);
        }
        let ret = self
            .http_content
            .on_http_response_headers(num_headers, end_of_stream);
        if ret != HeaderAction::Continue {
            return ret;
        }
        self.http_content
            .on_http_response_complete_headers(&self.res_headers)
    }

    fn on_http_response_body(&mut self, body_size: usize, end_of_stream: bool) -> DataAction {
        if !self.http_content.cache_response_body() {
            return self
                .http_content
                .on_http_response_body(body_size, end_of_stream);
        }
        self.res_body_len += body_size;

        if !end_of_stream {
            return DataAction::StopIterationAndBuffer;
        }

        let mut res_body = Bytes::new();
        if self.res_body_len > 0 {
            if let Some(body) = self.get_http_response_body(0, self.res_body_len) {
                res_body = body;
            }
        }
        self.http_content.on_http_response_complete_body(&res_body)
    }

    fn on_http_response_trailers(&mut self, num_trailers: usize) -> Action {
        self.http_content.on_http_response_trailers(num_trailers)
    }

    fn on_log(&mut self) {
        self.http_content.on_log()
    }
}
