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

#include <functional>
#include <optional>
#include <string>
#include <unordered_map>

#include "common/http_util.h"
#include "common/route_rule_matcher.h"
#define ASSERT(_X) assert(_X)

#ifndef NULL_PLUGIN

#include "proxy_wasm_intrinsics.h"

#else

#include "include/proxy-wasm/null_plugin.h"

namespace proxy_wasm {
namespace null_plugin {
namespace request_block {

#endif

struct RequestBlockConfigRule {
  int blocked_code = 403;
  std::string blocked_message;
  bool case_sensitive = true;
  std::vector<std::string> block_urls;
  std::vector<std::string> block_headers;
  std::vector<std::string> block_bodys;
};

// PluginRootContext is the root context for all streams processed by the
// thread. It has the same lifetime as the worker thread and acts as target for
// interactions that outlives individual stream, e.g. timer, async calls.
class PluginRootContext : public RootContext,
                          public RouteRuleMatcher<RequestBlockConfigRule> {
 public:
  PluginRootContext(uint32_t id, std::string_view root_id)
      : RootContext(id, root_id) {}
  ~PluginRootContext() {}
  bool onConfigure(size_t) override;
  bool checkHeader(const RequestBlockConfigRule&, bool&);
  bool checkBody(const RequestBlockConfigRule&, std::string_view);
  bool configure(size_t);

 private:
  bool parsePluginConfig(const json&, RequestBlockConfigRule&) override;
};

// Per-stream context.
class PluginContext : public Context {
 public:
  explicit PluginContext(uint32_t id, RootContext* root) : Context(id, root) {}
  FilterHeadersStatus onRequestHeaders(uint32_t, bool) override;
  FilterDataStatus onRequestBody(size_t, bool) override;

 private:
  inline PluginRootContext* rootContext() {
    return dynamic_cast<PluginRootContext*>(this->root());
  }

  size_t body_total_size_ = 0;
  bool check_body_ = false;
  std::optional<std::reference_wrapper<RequestBlockConfigRule>> config_;
};

#ifdef NULL_PLUGIN

}  // namespace request_block
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
