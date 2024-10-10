#!/bin/sh

plugin_root=$(cd "$(dirname $0)/..";pwd)

name=''
keywords=''
description=''
version="0.1.0"
output="$plugin_root/extensions"

help() {
  echo "$0 is used to generated Rust WASM extensions."
  echo "Usage:"
  echo "  $0 [arguments]"
  echo "The arguments are:"
  echo "  --name NAME               the name of extension"
  echo "  --keywords KEYWORDS       the keywords of extension [optional]"
  echo "  --description DESCRIPTION the description of extension [optional]"
  echo "  --version VERSION         the version of extension, default version is \"0.1.0\" [optional]"
  echo "  --output OUTPUT           the output folder of extension, default output is {PROJECT_ROOT}/plugins/wasm-rust/extensions/ [optional]"
  exit 1
}

while [[ $# -gt 0 ]]; do
  case $1 in
    --name)
      name="$2"
      shift # past argument
      shift # past value
      ;;
    --keywords)
      keywords="$2"
      shift # past argument
      shift # past value
      ;;
    --description)
      description="$2"
      shift # past argument
      shift # past value
      ;;
    --phase)
      phase="$2"
      shift # past argument
      shift # past value
      ;;
    --priority)
      priority="$2"
      shift # past argument
      shift # past value
      ;;
    --version)
      version="$2"
      shift # past argument
      shift # past value
      ;;
    --output)
      output="$2"
      shift # past argument
      shift # past value
      ;;
    *)
      help
  esac
done

if [ "$name" = "" ]; then
  help
fi

workdir=$output/"$name"
srcdir="$workdir"/src

mkdir -p "$workdir"
mkdir -p "$srcdir"

cat >"$workdir/Makefile"<<EOF
BUILD_OPTS="--release"

.DEFAULT:
build:
	rustup target add wasm32-wasi
	cargo build --target wasm32-wasi \${BUILD_OPTS}
	find target -name "*.wasm" -d 3 -exec cp "{}" plugin.wasm \;

clean:
	cargo clean
	rm -f plugin.wasm
EOF

cat >"$workdir"/Cargo.toml<<EOF
[package]
name = "$name"
version = "0.1.0"
edition = "2021"
publish = false

# See more keys and their definitions at https://doc.rust-lang.org/cargo/reference/manifest.html
[lib]
crate-type = ["cdylib"]

[dependencies]
higress-wasm-rust = { path = "../../", version = "0.1.0" }
proxy-wasm = { git="https://github.com/higress-group/proxy-wasm-rust-sdk", branch="main", version="0.2.2" }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
EOF

camelname=$(echo "$name" | tr '-' '_' | awk -F'_' '{for (i=1; i<=NF; i++) $i=toupper(substr($i,1,1)) substr($i,2)}1' OFS="")
struct_=$camelname
struct_root_="$camelname"Root
struct_config_="$camelname"Config

cat >"$srcdir"/lib.rs<<EOF
// Copyright (c) $(date +"%Y") Alibaba Group Holding Ltd.
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
use higress_wasm_rust::rule_matcher;
use higress_wasm_rust::rule_matcher::{on_configure, SharedRuleMatcher};
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{ContextType, LogLevel};
use serde::Deserialize;
use std::ops::DerefMut;
use std::rc::Rc;

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Trace);
    proxy_wasm::set_root_context(|_|Box::new($struct_root_::new()));
}}

struct $struct_root_ {
    log: Rc<Log>,
    rule_matcher: SharedRuleMatcher<$struct_config_>,
}

struct $struct_ {
    log: Rc<Log>,
    rule_matcher: SharedRuleMatcher<$struct_config_>,
    http_dispatcher: HttpDispatcher,
}

#[derive(Default, Clone, Debug, Deserialize)]
struct $struct_config_ {}

impl $struct_root_ {
    fn new() -> Self {
        $struct_root_ {
            log: Rc::new(Log::new("$name".to_string())),
            rule_matcher: rule_matcher::new_shared(),
        }
    }
}

impl Context for $struct_root_ {}

impl RootContext for $struct_root_ {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        on_configure(
            self,
            _plugin_configuration_size,
            self.rule_matcher.borrow_mut().deref_mut(),
            &self.log,
        )
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new($struct_ {
            log: self.log.clone(),
            rule_matcher: self.rule_matcher.clone(),
            http_dispatcher: Default::default(),
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl Context for $struct_ {
    fn on_http_call_response(&mut self, _token_id: u32, _num_headers: usize, _body_size: usize, _num_trailers: usize) {
        self.http_dispatcher.callback(_token_id, _num_headers, _body_size, _num_trailers)
    }
}

impl HttpContext for $struct_ {}
EOF

cat >"$workdir/VERSION"<<EOF
$version
EOF

cat >"$workdir/README.md"<<EOF
---
title: $name
keywords: $keywords
description: $description
---

## 功能说明

$description

## 运行属性

stage：\`默认阶段\`
level：\`10\`

### 配置说明

| Name     | Type     | Requirement | Default  | Description |
| -------- | -------- | --------    | -------- | --------    |
|          |          |             |          |             |

#### 配置示例

\`\`\`yaml

\`\`\`
EOF

cat >"$workdir/README_EN.md"<<EOF
---
title: $name
keywords: $keywords
description: $description
---

## Description

$description

## Runtime

phase：\`UNSPECIFIED_PHASE\`
priority：\`10\`

### Config

| Name     | Type     | Requirement | Default  | Description |
| -------- | -------- | --------    | -------- | --------    |
|          |          |             |          |             |

#### Example

\`\`\`yaml

\`\`\`
EOF