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

#include <string>
#include <unordered_set>

#include "common/route_rule_matcher.h"
#define ASSERT(_X) assert(_X)

#ifndef NULL_PLUGIN

#include "proxy_wasm_intrinsics.h"

#else

#include "include/proxy-wasm/null_plugin.h"

namespace proxy_wasm {
namespace null_plugin {
namespace basic_auth {

#endif

struct BasicAuthConfigRule {
  std::unordered_map<std::string, std::string> encrypted_credentials;
  std::unordered_set<std::string> encoded_credentials;
  std::unordered_map<std::string, std::string> credential_to_name;
  std::string realm = "MSE Gateway";
  bool passwd_encrypted = false;
};

// PluginRootContext is the root context for all streams processed by the
// thread. It has the same lifetime as the worker thread and acts as target for
// interactions that outlives individual stream, e.g. timer, async calls.
class PluginRootContext : public RootContext,
                          public RouteRuleMatcher<BasicAuthConfigRule> {
 public:
  PluginRootContext(uint32_t id, std::string_view root_id)
      : RootContext(id, root_id) {}
  ~PluginRootContext() {}
  bool onConfigure(size_t) override;
  bool checkPlugin(const BasicAuthConfigRule&,
                   const std::optional<std::unordered_set<std::string>>&);
  bool configure(size_t);

 private:
  bool parsePluginConfig(const json&, BasicAuthConfigRule&) override;
  bool addBasicAuthConfigRule(BasicAuthConfigRule& rule,
                              const std::string& credential,
                              const std::optional<std::string>& name,
                              bool base64_encoded);
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

}  // namespace basic_auth
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
