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
#include <unordered_set>

#include "common/route_rule_matcher.h"
#include "jwt-cpp/jwt.h"
#define ASSERT(_X) assert(_X)

#ifndef NULL_PLUGIN

#include "proxy_wasm_intrinsics.h"

#else

#include "include/proxy-wasm/null_plugin.h"

namespace proxy_wasm {
namespace null_plugin {
namespace oauth {

#endif

struct Consumer {
  std::string name;
  std::string client_id;
  std::string client_secret;
};

struct OAuthConfigRule {
  std::unordered_map<std::string, Consumer> consumers;
  std::string issuer = "Higress-Gateway";
  std::string auth_header_name = "Authorization";
  std::string auth_path = "/oauth2/token";
  bool global_credentials = true;
  uint64_t token_ttl = 7200;
  bool keep_token = true;
  uint64_t clock_skew = 60;
};

// PluginRootContext is the root context for all streams processed by the
// thread. It has the same lifetime as the worker thread and acts as target for
// interactions that outlives individual stream, e.g. timer, async calls.
class PluginRootContext : public RootContext,
                          public RouteRuleMatcher<OAuthConfigRule> {
 public:
  PluginRootContext(uint32_t id, std::string_view root_id)
      : RootContext(id, root_id) {}
  ~PluginRootContext() {}
  bool onConfigure(size_t) override;
  bool checkPlugin(const OAuthConfigRule&,
                   const std::optional<std::unordered_set<std::string>>&,
                   const std::string&);
  bool configure(size_t);
  bool generateToken(const OAuthConfigRule& rule, const std::string& route_name,
                     const absl::string_view& raw_params, std::string* token,
                     std::string* err_msg);

 private:
  bool parsePluginConfig(const json&, OAuthConfigRule&) override;
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

  std::string route_name_;
  std::optional<std::reference_wrapper<OAuthConfigRule>> config_;
  bool check_body_params_ = false;
  size_t body_total_size_ = 0;
};

#ifdef NULL_PLUGIN

}  // namespace oauth
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
