#!/bin/sh

name=''
keywords=''
description=''
testing=false
testing_port=10000

help() {
  echo "$0 is used to generated Rust WASM extensions."
  echo "Usage:"
  echo "  $0 [arguments]"
  echo "The arguments are:"
  echo "  --name NAME               the name of extension"
  echo "  --keywords KEYWORDS       the keywords of extension [optional]"
  echo "  --description DESCRIPTION the description of extension [optional]"
  echo "  --testing                 generate docker-compose.yaml and envoy.yaml for testing [optional]"
  echo "  --testing-port            expose port in generated docker-compose.yaml for testing, -testing-port=10000 by default [optional] "
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
    --testing)
      testing=true
      shift # past argument
      ;;
    --testing-port)
      testing_port="$2"
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

workdir=extensions/"$name"
srcdir="$workdir"/src

mkdir -p "$workdir"
mkdir -p "$srcdir"

cat >"$workdir/README.md"<<EOF
$name

---
title: $name
keywords: $keywords
description: $description
---
EOF

cat >"$workdir/Makefile"<<EOF
BUILD_OPTS="--release"

.DEFAULT:
build:
	cargo build --target wasm32-wasi \${BUILD_OPTS}

copy: build
	find target -name "*.wasm" -d 3 -exec cp "{}" plugin.wasm \;

clean:
	@cargo clean
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
}

#[derive(Default, Clone, Debug, Deserialize)]
struct $struct_config_ {
}

impl $struct_root_ {
    fn new() -> Self {
        $struct_root_ {
            log: Rc::new(Log::new("$name".to_string())),
            rule_matcher: rule_matcher::shared(),
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
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl Context for $struct_ {}

impl HttpContext for $struct_ {}
EOF

if [ "$testing" = true ]; then
  cat >>"$workdir"/Makefile<<EOF

docker-compose: copy
	@docker-compose up
EOF

  cat >"$workdir"/docker-compose.yaml<<EOF
# Copyright (c) $(date +"%Y") Alibaba Group Holding Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

services:
  envoy:
    image: higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/all-in-one:latest
    entrypoint: /usr/local/bin/envoy
    command: -c /etc/envoy/envoy.yaml --component-log-level wasm:debug
    hostname: envoy
    ports:
      - "$testing_port:10000"
    volumes:
      - ./envoy.yaml:/etc/envoy/envoy.yaml
      - ./plugin.wasm:/etc/envoy/plugin.wasm
    networks:
      - envoymesh
networks:
  envoymesh: {}
EOF

  cat >"$workdir"/envoy.yaml<<EOF
# Copyright (c) 2023 Alibaba Group Holding Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

static_resources:
  listeners:
    - name: listener_0
      address:
        socket_address:
          protocol: TCP
          address: 0.0.0.0
          port_value: 10000
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: local_service
                      domains: ["*"]
                      routes:
                        - name: index
                          match:
                            prefix: "/"
                          direct_response:
                            status: 200
                http_filters:
                  - name: envoy.filters.http.wasm
                    typed_config:
                      "@type": type.googleapis.com/udpa.type.v1.TypedStruct
                      type_url: type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
                      value:
                        config:
                          name: "http_body"
                          configuration:
                            "@type": type.googleapis.com/google.protobuf.StringValue
                            # TODO adjust it for WASM extensions
                            value: |-
                              {
                                "name": "$name",
                                "_rules_": []
                              }
                          vm_config:
                            runtime: "envoy.wasm.runtime.v8"
                            code:
                              local:
                                filename: "/etc/envoy/plugin.wasm"
                  - name: envoy.filters.http.router
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
EOF

fi