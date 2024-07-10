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


use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, Bytes};
use serde::de::DeserializeOwned;
use multimap::MultiMap;
use crate::rule_matcher::SharedRuleMatcher;

pub trait RootContextWrapper<PluginConfig> : RootContext
where
    PluginConfig: Default + DeserializeOwned + 'static + Clone,
{

    // fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
    fn create_http_context_use_wrapper(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        // trait 继承没法重写 RootContext 的 create_http_context，先写个函数让上层调下吧
        match self.create_http_context_wrapper(_context_id) {
            Some(http_context) => Some(Box::new(PluginHttpWrapper::new(self.rule_matcher(), http_context))),
            None => None,
        }
    }
    fn rule_matcher(&self) -> &SharedRuleMatcher<PluginConfig>;
    fn create_http_context_wrapper(&self, _context_id: u32) -> Option<Box<dyn HttpContextWrapper<PluginConfig>>> {
       None
    }
}
pub trait HttpContextWrapper<PluginConfig> : HttpContext{
    fn on_config(&mut self, _config: &PluginConfig){

    }
    fn on_http_request_headers_ok(&mut self, _headers: &MultiMap<String, String>) -> Action{
        Action::Continue
    }
    fn cache_request_body(&self) -> bool{
        false
    }
    fn cache_response_body(&self) -> bool{
        false
    }
    fn on_http_request_body_ok(&mut self, _req_body: &Bytes) -> Action{
        Action::Continue
    }
    fn on_http_response_body_ok(&mut self, _res_body: &Bytes) -> Action{
        Action::Continue
    }
}
pub struct PluginHttpWrapper<PluginConfig>{
    req_headers: MultiMap<String, String>,
    req_body: Bytes,
    res_body: Bytes,
    config: Option<PluginConfig>,
    rule_matcher: SharedRuleMatcher<PluginConfig>,
    http_content: Box<dyn HttpContextWrapper<PluginConfig>>
}
impl <PluginConfig> PluginHttpWrapper<PluginConfig>{
    pub fn new(rule_matcher: &SharedRuleMatcher<PluginConfig>, http_content:Box<dyn HttpContextWrapper<PluginConfig>>) -> Self{
        PluginHttpWrapper{
            req_headers: MultiMap::new(),
            req_body: Bytes::new(),
            res_body: Bytes::new(),
            config: None,
            rule_matcher: rule_matcher.clone(),
            http_content: http_content
        }
    }
}
impl <PluginConfig> Context for PluginHttpWrapper<PluginConfig> {}
impl <PluginConfig> HttpContext for PluginHttpWrapper<PluginConfig> 
where
    PluginConfig: Default + DeserializeOwned + Clone,
{
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        let binding = self.rule_matcher.borrow();
        self.config = match binding.get_match_config() {
            None => None,
            Some(config) => Some(config.1.clone()),
        };
        for (k, v) in self.get_http_request_headers(){
            
            self.req_headers.insert(k, v);
        }
        if let Some(config) = &self.config{
            self.http_content.on_config(config);
        }
        let ret = self.http_content.on_http_request_headers(_num_headers, _end_of_stream);
        if ret != Action::Continue{
            return ret;
        }
        self.http_content.on_http_request_headers_ok(&self.req_headers)

    }

    fn on_http_request_body(&mut self, _body_size: usize, _end_of_stream: bool) -> Action {
        let mut ret = self.http_content.on_http_request_body(_body_size, _end_of_stream);
        if !self.http_content.cache_request_body(){
            return ret
        }
        if _body_size > 0{
            match self.get_http_request_body(0, _body_size) {
                Some(body) => self.req_body.extend(body),
                None => {},
            }
        }
        if _end_of_stream && ret == Action::Continue{
            ret = self.http_content.on_http_request_body_ok(&self.req_body);
        }
        ret
    }

    fn on_http_request_trailers(&mut self, _num_trailers: usize) -> Action {
        self.http_content.on_http_request_trailers(_num_trailers)
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        self.http_content.on_http_response_headers(_num_headers, _end_of_stream)
    }

    fn on_http_response_body(&mut self, _body_size: usize, _end_of_stream: bool) -> Action {
        let mut ret = self.http_content.on_http_response_body(_body_size, _end_of_stream);
        if !self.http_content.cache_response_body(){
            return ret
        }
        if _body_size > 0 {
            if let Some(body) = self.get_http_response_body(0, _body_size) {
                self.res_body.extend(body);
            }
        }
        if _end_of_stream && ret == Action::Continue{
            ret = self.http_content.on_http_response_body_ok(&self.res_body);
        }
        ret
    }

    fn on_http_response_trailers(&mut self, _num_trailers: usize) -> Action {
        self.http_content.on_http_response_trailers(_num_trailers)
    }

    fn on_log(&mut self) {
        self.http_content.on_log()
    }
}