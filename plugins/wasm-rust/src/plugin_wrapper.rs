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

use std::cell::RefCell;
use std::collections::HashMap;
use std::rc::{Rc, Weak};
use std::time::Duration;

use crate::cluster_wrapper::Cluster;
use crate::log::Log;
use crate::rule_matcher::SharedRuleMatcher;
use http::{method::Method, Uri};
use lazy_static::lazy_static;
use multimap::MultiMap;
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, Bytes, DataAction, HeaderAction, Status};
use serde::de::DeserializeOwned;

lazy_static! {
    static ref LOG: Log = Log::new("plugin_wrapper".to_string());
}

thread_local! {
    static HTTP_CALLBACK_DISPATCHER: HttpCallbackDispatcher = HttpCallbackDispatcher::new();
}

pub trait RootContextWrapper<PluginConfig>: RootContext
where
    PluginConfig: Default + DeserializeOwned + Clone + 'static,
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

pub type HttpCallbackFn = dyn FnOnce(u16, &MultiMap<String, String>, Option<Vec<u8>>);

pub struct HttpCallbackDispatcher {
    call_fns: RefCell<HashMap<u32, Box<HttpCallbackFn>>>,
}

impl Default for HttpCallbackDispatcher {
    fn default() -> Self {
        Self::new()
    }
}

impl HttpCallbackDispatcher {
    pub fn new() -> Self {
        HttpCallbackDispatcher {
            call_fns: RefCell::new(HashMap::new()),
        }
    }

    pub fn set(&self, token_id: u32, arg: Box<HttpCallbackFn>) {
        self.call_fns.borrow_mut().insert(token_id, arg);
    }

    pub fn pop(&self, token_id: u32) -> Option<Box<HttpCallbackFn>> {
        self.call_fns.borrow_mut().remove(&token_id)
    }
}

pub trait HttpContextWrapper<PluginConfig>: HttpContext
where
    PluginConfig: Default + DeserializeOwned + Clone + 'static,
{
    fn init_self_weak(
        &mut self,
        _self_weak: Weak<RefCell<Box<dyn HttpContextWrapper<PluginConfig>>>>,
    ) {
    }

    fn log(&self) -> &Log {
        &LOG
    }

    fn on_config(&mut self, _config: Rc<PluginConfig>) {}

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

    fn set_request_body_buffer_limit(&self, limit: u32) {
        self.log()
            .infof(format_args!("SetRequestBodyBufferLimit:{}", limit));
        self.set_property(
            vec!["set_decoder_buffer_limit"],
            Some(limit.to_string().as_bytes()),
        );
    }

    fn set_response_body_buffer_limit(&self, limit: u32) {
        self.log()
            .infof(format_args!("SetResponseBodyBufferLimit:{}", limit));
        self.set_property(
            vec!["set_encoder_buffer_limit"],
            Some(limit.to_string().as_bytes()),
        );
    }

    #[allow(clippy::too_many_arguments)]
    fn http_call(
        &mut self,
        cluster: &dyn Cluster,
        method: &Method,
        raw_url: &str,
        headers: MultiMap<String, String>,
        body: Option<&[u8]>,
        call_fn: Box<HttpCallbackFn>,
        timeout: Duration,
    ) -> Result<u32, Status> {
        if let Ok(uri) = raw_url.parse::<Uri>() {
            let mut authority = cluster.host_name();
            if let Some(host) = uri.host() {
                authority = host.to_string();
            }
            let mut path = uri.path().to_string();
            if let Some(query) = uri.query() {
                path = format!("{}?{}", path, query);
            }
            let mut headers_vec = Vec::new();
            for (k, v) in headers.iter() {
                headers_vec.push((k.as_str(), v.as_str()));
            }
            headers_vec.push((":method", method.as_str()));
            headers_vec.push((":path", &path));
            headers_vec.push((":authority", &authority));
            let ret = self.dispatch_http_call(
                &cluster.cluster_name(),
                headers_vec,
                body,
                Vec::new(),
                timeout,
            );

            if let Ok(token_id) = ret {
                HTTP_CALLBACK_DISPATCHER.with(|dispatcher| dispatcher.set(token_id, call_fn));
                self.log().debugf(
                    format_args!(
                        "http call start, id: {}, cluster: {}, method: {}, url: {}, body: {:?}, timeout: {:?}",
                        token_id, cluster.cluster_name(), method.as_str(), raw_url, body, timeout
                    )
                );
            }
            ret
        } else {
            self.log()
                .criticalf(format_args!("invalid raw_url:{}", raw_url));
            Err(Status::ParseFailure)
        }
    }
}

downcast_rs::impl_downcast!(HttpContextWrapper<PluginConfig> where PluginConfig: Default + DeserializeOwned + Clone);

pub struct PluginHttpWrapper<PluginConfig> {
    req_body_len: usize,
    res_body_len: usize,
    config: Option<Rc<PluginConfig>>,
    rule_matcher: SharedRuleMatcher<PluginConfig>,
    http_content: Rc<RefCell<Box<dyn HttpContextWrapper<PluginConfig>>>>,
}

impl<PluginConfig> PluginHttpWrapper<PluginConfig>
where
    PluginConfig: Default + DeserializeOwned + Clone + 'static,
{
    pub fn new(
        rule_matcher: &SharedRuleMatcher<PluginConfig>,
        http_content: Box<dyn HttpContextWrapper<PluginConfig>>,
    ) -> Self {
        let rc_content = Rc::new(RefCell::new(http_content));
        rc_content
            .borrow_mut()
            .init_self_weak(Rc::downgrade(&rc_content));
        PluginHttpWrapper {
            req_body_len: 0,
            res_body_len: 0,
            config: None,
            rule_matcher: rule_matcher.clone(),
            http_content: rc_content,
        }
    }

    fn get_http_call_fn(&mut self, token_id: u32) -> Option<Box<HttpCallbackFn>> {
        HTTP_CALLBACK_DISPATCHER.with(|dispatcher| dispatcher.pop(token_id))
    }
}

impl<PluginConfig> Context for PluginHttpWrapper<PluginConfig>
where
    PluginConfig: Default + DeserializeOwned + Clone + 'static,
{
    fn on_http_call_response(
        &mut self,
        token_id: u32,
        num_headers: usize,
        body_size: usize,
        num_trailers: usize,
    ) {
        if let Some(call_fn) = self.get_http_call_fn(token_id) {
            let body = self.get_http_call_response_body(0, body_size);
            let mut headers = MultiMap::new();
            let mut status_code = 502;
            let mut normal_response = false;
            for (k, v) in self.get_http_call_response_headers_bytes() {
                match String::from_utf8(v) {
                    Ok(header_value) => {
                        if k == ":status" {
                            if let Ok(code) = header_value.parse::<u16>() {
                                status_code = code;
                                normal_response = true;
                            } else {
                                self.http_content.borrow().log().errorf(format_args!(
                                    "failed to parse status: {}",
                                    header_value
                                ));
                                status_code = 500;
                            }
                        }
                        headers.insert(k, header_value);
                    }
                    Err(_) => {
                        self.http_content.borrow().log().warnf(format_args!(
                            "http call response header contains non-ASCII characters header: {}",
                            k
                        ));
                    }
                }
            }
            self.http_content.borrow().log().debugf(format_args!(
                "http call end, id: {}, code: {}, normal: {}, body: {:?}", /*  */
                token_id, status_code, normal_response, body
            ));
            call_fn(status_code, &headers, body)
        } else {
            self.http_content.borrow_mut().on_http_call_response(
                token_id,
                num_headers,
                body_size,
                num_trailers,
            )
        }
    }

    fn on_grpc_call_response(&mut self, token_id: u32, status_code: u32, response_size: usize) {
        self.http_content
            .borrow_mut()
            .on_grpc_call_response(token_id, status_code, response_size)
    }

    fn on_grpc_stream_initial_metadata(&mut self, token_id: u32, num_elements: u32) {
        self.http_content
            .borrow_mut()
            .on_grpc_stream_initial_metadata(token_id, num_elements)
    }

    fn on_grpc_stream_message(&mut self, token_id: u32, message_size: usize) {
        self.http_content
            .borrow_mut()
            .on_grpc_stream_message(token_id, message_size)
    }

    fn on_grpc_stream_trailing_metadata(&mut self, token_id: u32, num_elements: u32) {
        self.http_content
            .borrow_mut()
            .on_grpc_stream_trailing_metadata(token_id, num_elements)
    }

    fn on_grpc_stream_close(&mut self, token_id: u32, status_code: u32) {
        self.http_content
            .borrow_mut()
            .on_grpc_stream_close(token_id, status_code)
    }

    fn on_done(&mut self) -> bool {
        self.http_content.borrow_mut().on_done()
    }
}

impl<PluginConfig> HttpContext for PluginHttpWrapper<PluginConfig>
where
    PluginConfig: Default + DeserializeOwned + Clone + 'static,
{
    fn on_http_request_headers(&mut self, num_headers: usize, end_of_stream: bool) -> HeaderAction {
        let binding = self.rule_matcher.borrow();
        self.config = binding.get_match_config().map(|config| config.1.clone());
        if self.config.is_none() {
            return HeaderAction::Continue;
        }

        let mut req_headers = MultiMap::new();
        for (k, v) in self.get_http_request_headers_bytes() {
            match String::from_utf8(v) {
                Ok(header_value) => {
                    req_headers.insert(k, header_value);
                }
                Err(_) => {
                    self.http_content.borrow().log().warnf(format_args!(
                        "request http header contains non-ASCII characters header: {}",
                        k
                    ));
                }
            }
        }

        if let Some(config) = &self.config {
            self.http_content.borrow_mut().on_config(config.clone());
        }
        let ret = self
            .http_content
            .borrow_mut()
            .on_http_request_headers(num_headers, end_of_stream);
        if ret != HeaderAction::Continue {
            return ret;
        }
        self.http_content
            .borrow_mut()
            .on_http_request_complete_headers(&req_headers)
    }

    fn on_http_request_body(&mut self, body_size: usize, end_of_stream: bool) -> DataAction {
        if self.config.is_none() {
            return DataAction::Continue;
        }
        if !self.http_content.borrow().cache_request_body() {
            return self
                .http_content
                .borrow_mut()
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
        self.http_content
            .borrow_mut()
            .on_http_request_complete_body(&req_body)
    }

    fn on_http_request_trailers(&mut self, num_trailers: usize) -> Action {
        if self.config.is_none() {
            return Action::Continue;
        }
        self.http_content
            .borrow_mut()
            .on_http_request_trailers(num_trailers)
    }

    fn on_http_response_headers(
        &mut self,
        num_headers: usize,
        end_of_stream: bool,
    ) -> HeaderAction {
        if self.config.is_none() {
            return HeaderAction::Continue;
        }

        let mut res_headers = MultiMap::new();
        for (k, v) in self.get_http_response_headers_bytes() {
            match String::from_utf8(v) {
                Ok(header_value) => {
                    res_headers.insert(k, header_value);
                }
                Err(_) => {
                    self.http_content.borrow().log().warnf(format_args!(
                        "response http header contains non-ASCII characters header: {}",
                        k
                    ));
                }
            }
        }

        let ret = self
            .http_content
            .borrow_mut()
            .on_http_response_headers(num_headers, end_of_stream);
        if ret != HeaderAction::Continue {
            return ret;
        }
        self.http_content
            .borrow_mut()
            .on_http_response_complete_headers(&res_headers)
    }

    fn on_http_response_body(&mut self, body_size: usize, end_of_stream: bool) -> DataAction {
        if self.config.is_none() {
            return DataAction::Continue;
        }
        if !self.http_content.borrow().cache_response_body() {
            return self
                .http_content
                .borrow_mut()
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
        self.http_content
            .borrow_mut()
            .on_http_response_complete_body(&res_body)
    }

    fn on_http_response_trailers(&mut self, num_trailers: usize) -> Action {
        if self.config.is_none() {
            return Action::Continue;
        }
        self.http_content
            .borrow_mut()
            .on_http_response_trailers(num_trailers)
    }

    fn on_log(&mut self) {
        self.http_content.borrow_mut().on_log()
    }
}
