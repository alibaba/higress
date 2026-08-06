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

#include "extensions/custom_response/plugin.h"

#include <array>

#include "absl/strings/str_cat.h"
#include "absl/strings/str_split.h"
#include "common/json_util.h"

using ::nlohmann::json;
using ::Wasm::Common::JsonArrayIterate;
using ::Wasm::Common::JsonGetField;
using ::Wasm::Common::JsonObjectIterate;
using ::Wasm::Common::JsonValueAs;

#ifdef NULL_PLUGIN

namespace proxy_wasm {
namespace null_plugin {
namespace custom_response {

PROXY_WASM_NULL_PLUGIN_REGISTRY

#endif

static RegisterContextFactory register_CustomResponse(
    CONTEXT_FACTORY(PluginContext), ROOT_FACTORY(PluginRootContext));

bool PluginRootContext::parsePluginConfig(const json& configuration,
                                          CustomResponseConfigRule& rule) {
  if (!JsonArrayIterate(
          configuration, "enable_on_status", [&](const json& item) -> bool {
            auto status = JsonValueAs<int64_t>(item);
            if (status.second != Wasm::Common::JsonParserResultDetail::OK) {
              LOG_WARN("cannot parse enable_on_status");
              return false;
            }
            rule.enable_on_status.push_back(
                std::to_string(status.first.value()));
            return true;
          })) {
    LOG_WARN("failed to parse configuration for enable_on_status.");
    return false;
  }
  bool has_content_type = false;
  if (!JsonArrayIterate(
          configuration, "headers", [&](const json& item) -> bool {
            auto header = JsonValueAs<std::string>(item);
            if (header.second != Wasm::Common::JsonParserResultDetail::OK) {
              LOG_WARN("cannot parse header");
              return false;
            }
            std::vector<std::string> pair =
                absl::StrSplit(header.first.value(), absl::MaxSplits("=", 2));
            if (pair.size() != 2) {
              LOG_WARN("invalid header pair format");
            }
            if (absl::AsciiStrToLower(pair[0]) ==
                Wasm::Common::Http::Header::ContentLength) {
              return true;
            }
            if (absl::AsciiStrToLower(pair[0]) ==
                Wasm::Common::Http::Header::ContentType) {
              has_content_type = true;
            }
            rule.headers.emplace_back(pair[0], pair[1]);
            return true;
          })) {
    LOG_WARN("failed to parse configuration for headers.");
    return false;
  }
  auto it = configuration.find("status_code");
  if (it != configuration.end()) {
    auto status_code = JsonValueAs<int64_t>(it.value());
    if (status_code.second != Wasm::Common::JsonParserResultDetail::OK) {
      LOG_WARN("cannot parse status code");
      return false;
    }
    rule.status_code = status_code.first.value();
  }
  it = configuration.find("body");
  if (it != configuration.end()) {
    auto body_string = JsonValueAs<std::string>(it.value());
    if (body_string.second != Wasm::Common::JsonParserResultDetail::OK) {
      LOG_WARN("cannot parse body");
      return false;
    }
    rule.body = body_string.first.value();
  }
  if (!rule.body.empty() && !has_content_type) {
    auto try_decode_json = Wasm::Common::JsonParse(rule.body);
    if (try_decode_json.has_value()) {
      rule.headers.emplace_back(Wasm::Common::Http::Header::ContentType,
                                "application/json; charset=utf-8");
    } else {
      rule.headers.emplace_back(Wasm::Common::Http::Header::ContentType,
                                "text/plain; charset=utf-8");
    }
  }
  return true;
}

FilterHeadersStatus PluginRootContext::onRequest(
    const CustomResponseConfigRule& rule) {
  if (!rule.enable_on_status.empty()) {
    return FilterHeadersStatus::Continue;
  }
  sendLocalResponse(rule.status_code, "", rule.body, rule.headers);
  return FilterHeadersStatus::StopIteration;
}

FilterHeadersStatus PluginRootContext::onResponse(
    const CustomResponseConfigRule& rule) {
  GET_RESPONSE_HEADER_VIEW(":status", status_code);
  bool hit = false;
  for (const auto& status : rule.enable_on_status) {
    if (status_code == status) {
      hit = true;
      break;
    }
  }
  if (!hit) {
    return FilterHeadersStatus::Continue;
  }
  sendLocalResponse(rule.status_code, "", rule.body, rule.headers);
  return FilterHeadersStatus::StopIteration;
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
  if (!result.has_value()) {
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

FilterHeadersStatus PluginContext::onRequestHeaders(uint32_t, bool) {
  auto* rootCtx = rootContext();
  return rootCtx->onHeaders(
      [rootCtx](const auto& config) { return rootCtx->onRequest(config); });
}

FilterHeadersStatus PluginContext::onResponseHeaders(uint32_t, bool) {
  auto* rootCtx = rootContext();
  return rootCtx->onHeaders(
      [rootCtx](const auto& config) { return rootCtx->onResponse(config); });
}

#ifdef NULL_PLUGIN

}  // namespace custom_response
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
