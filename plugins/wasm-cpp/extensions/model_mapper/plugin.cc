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

#include "extensions/model_mapper/plugin.h"

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
namespace model_mapper {

PROXY_WASM_NULL_PLUGIN_REGISTRY

#endif

static RegisterContextFactory register_ModelMapper(
    CONTEXT_FACTORY(PluginContext), ROOT_FACTORY(PluginRootContext));

namespace {

constexpr std::string_view SetDecoderBufferLimitKey =
    "set_decoder_buffer_limit";
constexpr std::string_view DefaultMaxBodyBytes = "104857600";

}  // namespace

#define CONSUMER_HEADER "x-mse-consumer"

bool PluginRootContext::parsePluginConfig(const json& configuration,
                                          ModelMapperConfigRule& rule) {
  if (auto it = configuration.find("modelKey"); it != configuration.end()) {
    if (it->is_string()) {
      rule.model_key_ = it->get<std::string>();
    } else {
      LOG_ERROR("Invalid type for modelKey. Expected string.");
      return false;
    }
  }

  if (auto it = configuration.find("modelMapping"); it != configuration.end()) {
    if (!it->is_object()) {
      LOG_ERROR("Invalid type for modelMapping. Expected object.");
      return false;
    }
    auto model_mapping = it->get<Wasm::Common::JsonObject>();
    if (!parseModelMappingRule(model_mapping, rule.default_rule_)) {
      return false;
    }
  }

  if (!JsonArrayIterate(
          configuration, "conditionalModelMappings", [&](const json& conditional_item) -> bool {
            if (!conditional_item.is_object()) {
              LOG_ERROR("Invalid type for conditionalModelMapping. Expected object.");
              return false;
            }
            ConditionalModelMappingRule conditional_rule;
            if (!JsonArrayIterate(
                    conditional_item, "consumers", [&](const json& consumer_item) -> bool {
                      if (consumer_item.is_string()) {
                        conditional_rule.consumers.emplace_back(consumer_item.get<std::string>());
                        return true;
                      }
                      return false;
                    })) {
              LOG_WARN("Invalid type for item in consumers. Expected string.");
              return false;
            }
            if (conditional_rule.empty()) {
              LOG_WARN("Ignore empty conditionalModelMapping.");
              return true;
            }
            if (auto it = conditional_item.find("modelMapping"); it != conditional_item.end()) {
              if (!it->is_object()) {
                LOG_ERROR("Invalid type for modelMapping. Expected object.");
                return false;
              }
              auto model_mapping = it->get<Wasm::Common::JsonObject>();
              if (!parseModelMappingRule(model_mapping, conditional_rule)) {
                return false;
              }
            }
            rule.conditional_rules_.emplace_back(std::move(conditional_rule));
            return true;
          })) {
    LOG_WARN("Invalid type for item in conditionalModelMappings. Expected object.");
    return false;
  }

  if (!JsonArrayIterate(
          configuration, "enableOnPathSuffix", [&](const json& item) -> bool {
            if (item.is_string()) {
              rule.enable_on_path_suffix_.emplace_back(item.get<std::string>());
              return true;
            }
            return false;
          })) {
    LOG_WARN("Invalid type for item in enableOnPathSuffix. Expected string.");
    return false;
  }
  return true;
}

bool PluginRootContext::parseModelMappingRule(const Wasm::Common::JsonObject& model_mapping, ModelMappingRule& rule) {
  return JsonObjectIterate(model_mapping, [&](std::string key) -> bool {
          auto model_json = model_mapping.find(key);
          if (!model_json->is_string()) {
            LOG_ERROR(
                "Invalid type for item in modelMapping. Expected string.");
            return false;
          }
          if (key == "*") {
            rule.default_model_mapping_ = model_json->get<std::string>();
            return true;
          }
          if (absl::EndsWith(key, "*")) {
            rule.prefix_model_mapping_.emplace_back(
                absl::StripSuffix(key, "*"), model_json->get<std::string>());
            return true;
          }
          auto ret = rule.exact_model_mapping_.emplace(
              key, model_json->get<std::string>());
          if (!ret.second) {
            LOG_ERROR("Duplicate key in modelMapping: " + key);
            return false;
          }
          return true;
  });
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

FilterHeadersStatus PluginRootContext::onHeader(PluginContext& ctx,
    const ModelMapperConfigRule& rule) {
  if (!Wasm::Common::Http::hasRequestBody()) {
    return FilterHeadersStatus::Continue;
  }
  auto path = getRequestHeader(Wasm::Common::Http::Header::Path)->toString();
  auto params_pos = path.find('?');
  size_t uri_end;
  if (params_pos == std::string::npos) {
    uri_end = path.size();
  } else {
    uri_end = params_pos;
  }
  bool enable = false;
  for (const auto& enable_suffix : rule.enable_on_path_suffix_) {
    if (absl::EndsWith({path.c_str(), uri_end}, enable_suffix)) {
      enable = true;
      break;
    }
  }
  if (!enable) {
    return FilterHeadersStatus::Continue;
  }
  auto content_type_value =
      getRequestHeader(Wasm::Common::Http::Header::ContentType);
  if (!absl::StrContains(content_type_value->view(),
                         Wasm::Common::Http::ContentTypeValues::Json)) {
    return FilterHeadersStatus::Continue;
  }

  ctx.active_rule_ = findActiveRule(rule);
  if (!ctx.active_rule_) {
    LOG_WARN("no active rule found");
    return FilterHeadersStatus::Continue;
  }

  removeRequestHeader(Wasm::Common::Http::Header::ContentLength);
  setFilterState(SetDecoderBufferLimitKey, DefaultMaxBodyBytes);
  LOG_INFO(absl::StrCat("SetRequestBodyBufferLimit: ", DefaultMaxBodyBytes));
  return FilterHeadersStatus::StopIteration;
}

const ModelMappingRule* PluginRootContext::findActiveRule(const ModelMapperConfigRule& rule) {
  if (!rule.conditional_rules_.empty()) {
    auto consumer = getRequestHeader(CONSUMER_HEADER)->toString();
    if (consumer.empty()) {
      LOG_DEBUG("no consumer found");
    } else {
      LOG_DEBUG("consumer found: " + consumer);
      for (auto &conditional_rule : rule.conditional_rules_) {
        if (std::find(conditional_rule.consumers.begin(), conditional_rule.consumers.end(), consumer)
                != conditional_rule.consumers.end()) {
          LOG_DEBUG("use conditional rule");
          return &conditional_rule;
        }
      }
    }
  }
  LOG_DEBUG("use default rule");
  return &rule.default_rule_;
}

FilterDataStatus PluginRootContext::onBody(PluginContext& ctx,
                                           const ModelMapperConfigRule& rule,
                                           std::string_view body) {
  const ModelMappingRule* active_rule = ctx.active_rule_;
  if (!active_rule) {
    LOG_WARN("no active rule found");
    return FilterDataStatus::Continue;
  }
  auto body_json_opt = ::Wasm::Common::JsonParse(body);
  if (!body_json_opt) {
    LOG_WARN(absl::StrCat("cannot parse body to JSON string: ", body));
    return FilterDataStatus::Continue;
  }
  doModelMapping(body_json_opt.value(), rule.model_key_, *active_rule);
  return FilterDataStatus::Continue;
}

void PluginRootContext::doModelMapping(Wasm::Common::JsonObject& body_json,
                                       const std::string model_key,
                                       const ModelMappingRule& rule) {
  const auto& exact_model_mapping = rule.exact_model_mapping_;
  const auto& prefix_model_mapping = rule.prefix_model_mapping_;
  const auto& default_model_mapping = rule.default_model_mapping_;
  std::string old_model;
  if (body_json.contains(model_key)) {
    old_model = body_json[model_key];
  }
  std::string model =
      default_model_mapping.empty() ? old_model : default_model_mapping;
  if (auto it = exact_model_mapping.find(old_model);
      it != exact_model_mapping.end()) {
    model = it->second;
  } else {
    for (auto& prefix_model_pair : prefix_model_mapping) {
      if (absl::StartsWith(old_model, prefix_model_pair.first)) {
        model = prefix_model_pair.second;
        break;
      }
    }
  }
  if (!model.empty() && model != old_model) {
    body_json[model_key] = model;
    setBuffer(WasmBufferType::HttpRequestBody, 0,
              std::numeric_limits<size_t>::max(), body_json.dump());
    LOG_DEBUG(
        absl::StrCat("model mapped, before:", old_model, ", after:", model));
  }
}

FilterHeadersStatus PluginContext::onRequestHeaders(uint32_t, bool) {
  auto* rootCtx = rootContext();
  return rootCtx->onHeaders([rootCtx, this](const auto& config) {
    auto ret = rootCtx->onHeader(*this, config);
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
  return rootCtx->onBody(*this, *config_, body->view());
}

#ifdef NULL_PLUGIN

}  // namespace model_mapper
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
