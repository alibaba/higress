/*
 * Copyright (c) 2022 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#pragma once

#ifndef NULL_PLUGIN

#include "proxy_wasm_intrinsics.h"

#else

#include "include/proxy-wasm/null_plugin.h"
using namespace proxy_wasm::null_plugin;
using proxy_wasm::WasmResult;

#endif

struct LimitItem {
  std::string key;
  uint64_t tokens_per_refill;
  uint64_t refill_interval_nanosec;
  uint64_t max_tokens;
};

bool getToken(int rule_id, const std::string& key);
void refillToken(const std::vector<std::pair<int, LimitItem>>& rules);
bool initializeTokenBucket(const std::vector<std::pair<int, LimitItem>>& rules);
