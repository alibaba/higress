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
