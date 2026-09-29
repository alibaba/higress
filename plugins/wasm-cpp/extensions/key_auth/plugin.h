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
namespace key_auth {

#endif

struct Consumer {
  std::string name;
  // The group this consumer belongs to. Downstream plugins such as ai-quota
  // can aggregate by group. Empty means no group.
  std::string group;
  std::unordered_set<std::string> credentials;
  std::optional<std::vector<std::string>> keys;
  std::optional<bool> in_query = std::nullopt;
  std::optional<bool> in_header = std::nullopt;

  // std::string debugString() const {
  //   std::string msg;
  //   msg += "name: " + name + "\n";
  //   msg += "  keys: \n";
  //   if (keys.has_value()) {
  //     for (const auto& item : keys.value()) {
  //       msg += "  - " + item + "\n";
  //     }
  //   }
  //   msg += "  credentials: \n";
  //   for (const auto& item : credentials) {
  //     msg += "  - " + item + "\n";
  //   }
  //   return msg;
  // }
};

struct KeyAuthConfigRule {
  std::vector<Consumer> consumers;
  std::unordered_set<std::string> credentials;
  std::unordered_map<std::string, std::string> credential_to_name;
  // Consumer name -> group, only contains consumers with a non-empty group.
  std::unordered_map<std::string, std::string> name_to_group;
  std::string realm = "MSE Gateway";
  std::vector<std::string> keys;
  bool in_query = true;
  bool in_header = true;

  // Route-level allow lists, parsed by key_auth itself from the rule JSON
  // (the shared RouteRuleMatcher keeps parsing `allow` into its own allow_set
  // and does not know about these fields). Empty sets mean "not configured".
  std::unordered_set<std::string> allow;
  std::unordered_set<std::string> allow_groups;
  std::unordered_set<std::string> allow_credentials;

  // std::string debugString(std::string prompt="") const {
  //   std::string msg;
  //   msg += prompt + "\n";
  //   msg += "realm: " + realm + "\n";
  //   msg += "keys: \n";
  //   for (const auto& item : keys) {
  //     msg += "- " + item + "\n";
  //   }
  //   msg += "credentials: \n";
  //   for (const auto& item : credentials) {
  //     msg += "- " + item + "\n";
  //   }
  //   msg += "credential_to_name: \n";
  //   for (const auto& item : credential_to_name) {
  //     msg += "- " + item.first + ": " + item.second + "\n";
  //   }
  //   msg += "consumers: \n";
  //   for (const auto& item : consumers) {
  //     msg += "- " + item.debugString();
  //   }

  //   return msg;
  // }
};

// PluginRootContext is the root context for all streams processed by the
// thread. It has the same lifetime as the worker thread and acts as target for
// interactions that outlives individual stream, e.g. timer, async calls.
class PluginRootContext : public RootContext,
                          public RouteRuleMatcher<KeyAuthConfigRule> {
 public:
  PluginRootContext(uint32_t id, std::string_view root_id)
      : RootContext(id, root_id) {}
  ~PluginRootContext() {}
  bool onConfigure(size_t) override;
  bool checkPlugin(const KeyAuthConfigRule&,
                   const std::optional<std::unordered_set<std::string>>&);
  bool configure(size_t);

 private:
  bool parsePluginConfig(const json&, KeyAuthConfigRule&) override;

  // Cached copy of the global-level parse. Rules that configure only allow
  // lists (no consumers/credentials of their own) resolve consumer identity
  // against this cache.
  KeyAuthConfigRule global_cache_;
  bool has_global_cache_ = false;

  std::string extractCredential(bool in_header, bool in_query,
                                const std::string& key) const;
  void addConsumerHeaders(const KeyAuthConfigRule&, const std::string& name,
                          const std::string& credential);
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

}  // namespace key_auth
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
