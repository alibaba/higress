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
namespace model_mapper {

#endif

struct ModelMapperConfigRule {
  std::string model_key_ = "model";
  std::map<std::string, std::string> exact_model_mapping_;
  std::vector<std::pair<std::string, std::string>> prefix_model_mapping_;
  std::string default_model_mapping_;
  std::vector<std::string> enable_on_path_suffix_ = {
      "/completions",  "/embeddings",       "/images/generations",
      "/audio/speech", "/fine_tuning/jobs", "/moderations"};
};

// PluginRootContext is the root context for all streams processed by the
// thread. It has the same lifetime as the worker thread and acts as target for
// interactions that outlives individual stream, e.g. timer, async calls.
class PluginRootContext : public RootContext,
                          public RouteRuleMatcher<ModelMapperConfigRule> {
 public:
  PluginRootContext(uint32_t id, std::string_view root_id)
      : RootContext(id, root_id) {}
  ~PluginRootContext() {}
  bool onConfigure(size_t) override;
  FilterHeadersStatus onHeader(const ModelMapperConfigRule&);
  FilterDataStatus onBody(const ModelMapperConfigRule&, std::string_view);
  bool configure(size_t);

 private:
  bool parsePluginConfig(const json&, ModelMapperConfigRule&) override;
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
  const ModelMapperConfigRule* config_ = nullptr;
};

#ifdef NULL_PLUGIN

}  // namespace model_mapper
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
