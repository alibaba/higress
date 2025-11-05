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

#include <assert.h>

#include <cstdint>
#include <string>
#include <unordered_map>

#include "bucket.h"
#include "common/http_util.h"
#include "common/route_rule_matcher.h"
#define ASSERT(_X) assert(_X)

#ifndef NULL_PLUGIN

#include "proxy_wasm_intrinsics.h"

#else

#include "include/proxy-wasm/null_plugin.h"

namespace proxy_wasm {
namespace null_plugin {
namespace key_rate_limit {

#endif

struct KeyRateLimitConfigRule {
  std::unordered_map<std::string, LimitItem> limit_keys;
  std::string limit_by_header;
  std::string limit_by_param;
};

// PluginRootContext is the root context for all streams processed by the
// thread. It has the same lifetime as the worker thread and acts as target for
// interactions that outlives individual stream, e.g. timer, async calls.
class PluginRootContext : public RootContext,
                          public RouteRuleMatcher<KeyRateLimitConfigRule> {
 public:
  PluginRootContext(uint32_t id, std::string_view root_id)
      : RootContext(id, root_id) {}
  ~PluginRootContext() {}
  bool onConfigure(size_t) override;
  void onTick() override;
  bool checkPlugin(int, const KeyRateLimitConfigRule&);
  bool configure(size_t);

 private:
  bool parsePluginConfig(const json&, KeyRateLimitConfigRule&) override;

  std::vector<std::pair<int, LimitItem>> limits_;

  friend class KeyRateLimitTest_Config_Test;
  friend class KeyRateLimitTest_RuleConfig_Test;
};

// Per-stream context.
class PluginContext : public Context {
 public:
  explicit PluginContext(uint32_t id, RootContext* root) : Context(id, root) {}
  FilterHeadersStatus onRequestHeaders(uint32_t, bool) override;

 private:
  inline PluginRootContext* rootContext() {
    return dynamic_cast<PluginRootContext*>(this->root());
  }
};

#ifdef NULL_PLUGIN

}  // namespace key_rate_limit
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
