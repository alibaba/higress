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

use crate::hostcalls;
use crate::promise::Promise;
use std::collections::HashMap;
use std::rc::Rc;
use std::time::Duration;

#[derive(Default)]
pub struct HttpDispatcher {
    m: HashMap<u32, Rc<Promise<(u32, usize, usize, usize)>>>,
}

impl HttpDispatcher {
    pub fn dispatch(
        &mut self,
        upstream: &str,
        headers: Vec<(&str, &str)>,
        body: Option<&[u8]>,
        trailers: Vec<(&str, &str)>,
        timeout: Duration,
    ) -> Rc<Promise<(u32, usize, usize, usize)>> {
        let token =
            hostcalls::dispatch_http_call(upstream, headers, body, trailers, timeout).unwrap();
        let promise = Promise::new();
        self.m.insert(token, promise.clone());
        promise
    }

    pub fn callback(
        &mut self,
        _token_id: u32,
        _num_headers: usize,
        _body_size: usize,
        _num_trailers: usize,
    ) {
        let promise = self.m.remove(&_token_id);
        promise
            .unwrap()
            .fulfill((_token_id, _num_headers, _body_size, _num_trailers))
    }
}
