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

#include "extensions/key_rate_limit/plugin.h"

#include <array>
#include <vector>

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
namespace key_rate_limit {

PROXY_WASM_NULL_PLUGIN_REGISTRY

#endif

static RegisterContextFactory register_KeyRateLimit(
    CONTEXT_FACTORY(PluginContext), ROOT_FACTORY(PluginRootContext));

namespace {

constexpr uint64_t second_nano = 1000 * 1000 * 1000;
constexpr uint64_t minute_nano = 60 * second_nano;
constexpr uint64_t hour_nano = 60 * minute_nano;
constexpr uint64_t day_nano = 24 * hour_nano;

// tooManyRequest returns a 429 response code.
void tooManyRequest() {
  sendLocalResponse(429, "Too many requests", "rate_limited", {});
}

}  // namespace

bool PluginRootContext::parsePluginConfig(const json& configuration,
                                          KeyRateLimitConfigRule& rule) {
  if (!JsonArrayIterate(
          configuration, "limit_keys", [&](const json& item) -> bool {
            std::string key =
                Wasm::Common::JsonGetField<std::string>(item, "key").value();
            uint64_t qps =
                Wasm::Common::JsonGetField<uint64_t>(item, "query_per_second")
                    .value_or(0);
            if (qps > 0) {
              rule.limit_keys.emplace(key, LimitItem{
                                               key,
                                               qps,
                                               second_nano,
                                               qps,
                                           });
              return true;
            }
            uint64_t qpm =
                Wasm::Common::JsonGetField<uint64_t>(item, "query_per_minute")
                    .value_or(0);
            if (qpm > 0) {
              rule.limit_keys.emplace(key, LimitItem{
                                               key,
                                               qpm,
                                               minute_nano,
                                               qpm,
                                           });
              return true;
            }
            uint64_t qph =
                Wasm::Common::JsonGetField<uint64_t>(item, "query_per_hour")
                    .value_or(0);
            if (qph > 0) {
              rule.limit_keys.emplace(key, LimitItem{
                                               key,
                                               qph,
                                               hour_nano,
                                               qph,
                                           });
              return true;
            }
            uint64_t qpd =
                Wasm::Common::JsonGetField<uint64_t>(item, "query_per_day")
                    .value_or(0);
            if (qpd > 0) {
              rule.limit_keys.emplace(key, LimitItem{
                                               key,
                                               qpd,
                                               day_nano,
                                               qpd,
                                           });
              return true;
            }
            LOG_WARN(
                "one of 'query_per_second', 'query_per_minute', "
                "'query_per_hour' or 'query_per_day' must be set");
            return false;
          })) {
    LOG_WARN("failed to parse configuration for limit_keys.");
    return false;
  }
  if (rule.limit_keys.empty()) {
    LOG_WARN("no limit keys found in configuration");
    return false;
  }
  auto it = configuration.find("limit_by_header");
  if (it != configuration.end()) {
    auto limit_by_header = JsonValueAs<std::string>(it.value());
    if (limit_by_header.second != Wasm::Common::JsonParserResultDetail::OK) {
      LOG_WARN("cannot parse limit_by_header");
      return false;
    }
    rule.limit_by_header = limit_by_header.first.value();
  }
  it = configuration.find("limit_by_param");
  if (it != configuration.end()) {
    auto limit_by_param = JsonValueAs<std::string>(it.value());
    if (limit_by_param.second != Wasm::Common::JsonParserResultDetail::OK) {
      LOG_WARN("cannot parse limit_by_param");
      return false;
    }
    rule.limit_by_param = limit_by_param.first.value();
  }
  auto emptyHeader = rule.limit_by_header.empty();
  auto emptyParam = rule.limit_by_param.empty();
  if ((emptyHeader && emptyParam) || (!emptyHeader && !emptyParam)) {
    LOG_WARN("only one of 'limit_by_param' and 'limit_by_header' can be set");
    return false;
  }
  return true;
}

bool PluginRootContext::checkPlugin(int rule_id,
                                    const KeyRateLimitConfigRule& config) {
  const auto& headerKey = config.limit_by_header;
  const auto& paramKey = config.limit_by_param;
  std::string key;
  if (!headerKey.empty()) {
    GET_HEADER_VIEW(headerKey, header);
    key = header;
  } else {
    // use paramKey which must not be empty
    GET_HEADER_VIEW(":path", path);
    const auto& params = Wasm::Common::Http::parseQueryString(path);
    auto it = params.find(paramKey);
    if (it != params.end()) {
      key = it->second;
    }
  }
  const auto& limit_keys = config.limit_keys;
  if (limit_keys.find(key) == limit_keys.end()) {
    return true;
  }
  if (!getToken(rule_id, key)) {
    LOG_INFO(absl::StrCat("request rate limited by key: ", key));
    tooManyRequest();
    return false;
  }
  return true;
}

void PluginRootContext::onTick() { refillToken(limits_); }

bool PluginRootContext::onConfigure(size_t size) {
  // Parse configuration JSON string.
  if (size > 0 && !configure(size)) {
    LOG_WARN("configuration has errors initialization will not continue.");
    return false;
  }
  const auto& rules = getRules();
  for (const auto& rule : rules) {
    for (auto& keyItem : rule.second.get().limit_keys) {
      limits_.emplace_back(rule.first, keyItem.second);
    }
  }
  initializeTokenBucket(limits_);
  proxy_set_tick_period_milliseconds(500);
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
  return rootCtx->checkRuleWithId([rootCtx](auto rule_id, const auto& config) {
    return rootCtx->checkPlugin(rule_id, config);
  })
             ? FilterHeadersStatus::Continue
             : FilterHeadersStatus::StopIteration;
}

#ifdef NULL_PLUGIN

}  // namespace key_rate_limit
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
