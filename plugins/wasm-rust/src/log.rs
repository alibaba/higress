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

use proxy_wasm::{hostcalls, types};
use std::fmt::Arguments;

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
        let level = types::LogLevel::from(level);
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

    fn logf(&self, level: LogLevel, format_args: Arguments) {
        let level = types::LogLevel::from(level);
        if let Ok(log_level) = hostcalls::get_log_level() {
            if (level as i32) < (log_level as i32) {
                return;
            }
            hostcalls::log(
                level,
                format!("[{}] {}", self.plugin_name, format_args).as_str(),
            )
            .unwrap();
        }
    }

    /// ```
    /// use higress_wasm_rust::log::Log;
    /// let log = Log::new("foobar".into_string());
    /// log.tracef(format_args!("Hello, {}!","World"));
    /// ```
    pub fn tracef(&self, format_args: Arguments) {
        self.logf(LogLevel::Trace, format_args)
    }

    /// ```
    /// use higress_wasm_rust::log::Log;
    /// let log = Log::new("foobar".into_string());
    /// log.debugf(format_args!("Hello, {}!","World"));
    /// ```
    pub fn debugf(&self, format_args: Arguments) {
        self.logf(LogLevel::Debug, format_args)
    }

    /// ```
    /// use higress_wasm_rust::log::Log;
    /// let log = Log::new("foobar".into_string());
    /// log.infof(format_args!("Hello, {}!","World"));
    /// ```
    pub fn infof(&self, format_args: Arguments) {
        self.logf(LogLevel::Info, format_args)
    }

    /// ```
    /// use higress_wasm_rust::log::Log;
    /// let log = Log::new("foobar".into_string());
    /// log.warnf(format_args!("Hello, {}!","World"));
    /// ```
    pub fn warnf(&self, format_args: Arguments) {
        self.logf(LogLevel::Warn, format_args)
    }

    /// ```
    /// use higress_wasm_rust::log::Log;
    /// let log = Log::new("foobar".into_string());
    /// log.errorf(format_args!("Hello, {}!","World"));
    /// ```
    pub fn errorf(&self, format_args: Arguments) {
        self.logf(LogLevel::Error, format_args)
    }

    /// ```
    /// use higress_wasm_rust::log::Log;
    /// let log = Log::new("foobar".into_string());
    /// log.criticalf(format_args!("Hello, {}!","World"));
    /// ```
    pub fn criticalf(&self, format_args: Arguments) {
        self.logf(LogLevel::Critical, format_args)
    }
}

impl From<LogLevel> for types::LogLevel {
    fn from(value: LogLevel) -> Self {
        match value {
            LogLevel::Trace => types::LogLevel::Trace,
            LogLevel::Debug => types::LogLevel::Debug,
            LogLevel::Info => types::LogLevel::Info,
            LogLevel::Warn => types::LogLevel::Warn,
            LogLevel::Error => types::LogLevel::Error,
            LogLevel::Critical => types::LogLevel::Critical,
        }
    }
}
