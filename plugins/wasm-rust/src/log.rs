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

use proxy_wasm::hostcalls;

pub enum LogLevel {
    Trace,
    Debug,
    Info,
    Warn,
    Error,
    Critical,
}

pub struct Log {
    plugin_name: String,
}

impl Log {
    pub fn new(plugin_name: String) -> Log {
        Log { plugin_name }
    }

    fn log(&self, level: LogLevel, msg: &str) {
        let msg = format!("[{}] {}", self.plugin_name, msg);
        let level = match level {
            LogLevel::Trace => proxy_wasm::types::LogLevel::Trace,
            LogLevel::Debug => proxy_wasm::types::LogLevel::Debug,
            LogLevel::Info => proxy_wasm::types::LogLevel::Info,
            LogLevel::Warn => proxy_wasm::types::LogLevel::Warn,
            LogLevel::Error => proxy_wasm::types::LogLevel::Error,
            LogLevel::Critical => proxy_wasm::types::LogLevel::Critical,
        };
        hostcalls::log(level, msg.as_str()).unwrap();
    }

    pub fn trace(&self, msg: &str) {
        self.log(LogLevel::Trace, msg)
    }

    pub fn debug(&self, msg: &str) {
        self.log(LogLevel::Debug, msg)
    }

    pub fn info(&self, msg: &str) {
        self.log(LogLevel::Info, msg)
    }

    pub fn warn(&self, msg: &str) {
        self.log(LogLevel::Warn, msg)
    }

    pub fn error(&self, msg: &str) {
        self.log(LogLevel::Error, msg)
    }

    pub fn critical(&self, msg: &str) {
        self.log(LogLevel::Critical, msg)
    }
}
