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

use std::error::Error;
use std::fmt::{Display, Formatter};

#[derive(Debug, Default)]
pub struct WasmRustError {
    message: String,
}

impl WasmRustError {
    pub const fn new(message: String) -> Self {
        WasmRustError { message }
    }
}

impl Display for WasmRustError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.message)
    }
}

impl Error for WasmRustError {
    fn description(&self) -> &str {
        &self.message
    }
}
