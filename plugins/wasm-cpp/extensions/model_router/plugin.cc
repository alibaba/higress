// Copyright (c) 2022 Alibaba Group Holding Ltd.
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

#include "extensions/model_router/plugin.h"

#include <array>
#include <limits>

#include "absl/strings/str_cat.h"
#include "absl/strings/str_split.h"
#include "common/http_util.h"
#include "common/json_util.h"

using ::nlohmann::json;
using ::Wasm::Common::JsonArrayIterate;
using ::Wasm::Common::JsonGetField;
using ::Wasm::Common::JsonObjectIterate;
using ::Wasm::Common::JsonValueAs;

#ifdef NULL_PLUGIN

namespace proxy_wasm {
namespace null_plugin {
namespace model_router {

PROXY_WASM_NULL_PLUGIN_REGISTRY

#endif

static RegisterContextFactory register_ModelRouter(
    CONTEXT_FACTORY(PluginContext), ROOT_FACTORY(PluginRootContext));

namespace {

constexpr std::string_view SetDecoderBufferLimitKey =
    "SetRequestBodyBufferLimit";
constexpr std::string_view DefaultMaxBodyBytes = "10485760";

}  // namespace

bool PluginRootContext::parsePluginConfig(const json& configuration,
                                          ModelRouterConfigRule& rule) {
  if (auto it = configuration.find("enable"); it != configuration.end()) {
    if (it->is_boolean()) {
      rule.enable_ = it->get<bool>();
    } else {
      LOG_WARN("Invalid type for enable. Expected boolean.");
      return false;
    }
  }

  if (auto it = configuration.find("model_key"); it != configuration.end()) {
    if (it->is_string()) {
      rule.model_key_ = it->get<std::string>();
    } else {
      LOG_WARN("Invalid type for model_key. Expected string.");
      return false;
    }
  }

  if (auto it = configuration.find("add_header_key");
      it != configuration.end()) {
    if (it->is_string()) {
      rule.add_header_key_ = it->get<std::string>();
    } else {
      LOG_WARN("Invalid type for add_header_key. Expected string.");
      return false;
    }
  }

  return true;
}

bool PluginRootContext::onConfigure(size_t size) {
  // Parse configuration JSON string.
  if (size > 0 && !configure(size)) {
    LOG_WARN("configuration has errors initialization will not continue.");
    return false;
  }
  return true;
}

bool PluginRootContext::configure(size_t configuration_size) {
  auto configuration_data = getBufferBytes(WasmBufferType::PluginConfiguration,
                                           0, configuration_size);
  // Parse configuration JSON string.
  auto result = ::Wasm::Common::JsonParse(configuration_data->view());
  if (!result) {
    LOG_WARN(absl::StrCat("cannot parse plugin configuration JSON string: ",
                          configuration_data->view()));
    return false;
  }
  if (!parseRuleConfig(result.value())) {
    LOG_WARN(absl::StrCat("cannot parse plugin configuration JSON string: ",
                          configuration_data->view()));
    return false;
  }
  return true;
}

FilterHeadersStatus PluginRootContext::onHeader(
    const ModelRouterConfigRule& rule) {
  if (!rule.enable_ || !Wasm::Common::Http::hasRequestBody()) {
    return FilterHeadersStatus::Continue;
  }
  auto content_type_value =
      getRequestHeader(Wasm::Common::Http::Header::ContentType);
  if (!absl::StrContains(content_type_value->view(),
                         Wasm::Common::Http::ContentTypeValues::Json)) {
    return FilterHeadersStatus::Continue;
  }
  removeRequestHeader(Wasm::Common::Http::Header::ContentLength);
  setFilterState(SetDecoderBufferLimitKey, DefaultMaxBodyBytes);
  return FilterHeadersStatus::StopIteration;
}

FilterDataStatus PluginRootContext::onBody(const ModelRouterConfigRule& rule,
                                           std::string_view body) {
  const auto& model_key = rule.model_key_;
  const auto& add_header_key = rule.add_header_key_;
  auto body_json_opt = ::Wasm::Common::JsonParse(body);
  if (!body_json_opt) {
    LOG_WARN(absl::StrCat("cannot parse body to JSON string: ", body));
    return FilterDataStatus::Continue;
  }
  auto body_json = body_json_opt.value();
  if (body_json.contains(model_key)) {
    std::string model_value = body_json[model_key];
    auto pos = model_value.find('/');
    if (pos != std::string::npos) {
      const auto& provider = model_value.substr(0, pos);
      const auto& model = model_value.substr(pos + 1);
      replaceRequestHeader(add_header_key, provider);
      body_json[model_key] = model;
      setBuffer(WasmBufferType::HttpRequestBody, 0,
                std::numeric_limits<size_t>::max(), body_json.dump());
      LOG_DEBUG(absl::StrCat("model route to provider:", provider,
                             ", model:", model));
    } else {
      LOG_DEBUG(absl::StrCat("model route not work, model:", model_value));
    }
  }
  return FilterDataStatus::Continue;
}

FilterHeadersStatus PluginContext::onRequestHeaders(uint32_t, bool) {
  auto* rootCtx = rootContext();
  return rootCtx->onHeaders([rootCtx, this](const auto& config) {
    auto ret = rootCtx->onHeader(config);
    if (ret == FilterHeadersStatus::StopIteration) {
      this->config_ = &config;
    }
    return ret;
  });
}

FilterDataStatus PluginContext::onRequestBody(size_t body_size,
                                              bool end_stream) {
  if (config_ == nullptr) {
    return FilterDataStatus::Continue;
  }
  body_total_size_ += body_size;
  if (!end_stream) {
    return FilterDataStatus::StopIterationAndBuffer;
  }
  auto body =
      getBufferBytes(WasmBufferType::HttpRequestBody, 0, body_total_size_);
  auto* rootCtx = rootContext();
  return rootCtx->onBody(*config_, body->view());
}

#ifdef NULL_PLUGIN

}  // namespace model_router
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
