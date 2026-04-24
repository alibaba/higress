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
constexpr std::string_view SetEncoderBufferLimitKey =
    "set_encoder_buffer_limit";
constexpr std::string_view DefaultMaxBodyBytes = "104857600";
constexpr std::string_view EventStreamContentType = "text/event-stream";

}  // namespace

bool rewriteModelFieldInJson(nlohmann::json& payload, const std::string& key,
                             std::string_view upstream_model,
                             std::string_view client_model) {
  bool rewritten = false;
  if (payload.contains(key)) {
    auto& model_value = payload[key];
    if (model_value.is_string() &&
        model_value.get<std::string>() == upstream_model) {
      model_value = client_model;
      rewritten = true;
    }
  }

  // Anthropic SSE message_start uses nested message.model.
  if (payload.contains("message") && payload["message"].is_object() &&
      payload["message"].contains(key)) {
    auto& nested_model = payload["message"][key];
    if (nested_model.is_string() &&
        nested_model.get<std::string>() == upstream_model) {
      nested_model = client_model;
      rewritten = true;
    }
  }
  return rewritten;
}

std::pair<size_t, size_t> findSseEventSeparator(std::string_view data) {
  auto lf_pos = data.find("\n\n");
  auto crlf_pos = data.find("\r\n\r\n");
  if (lf_pos == std::string_view::npos) {
    if (crlf_pos == std::string_view::npos) {
      return {std::string_view::npos, 0};
    }
    return {crlf_pos, 4};
  }
  if (crlf_pos == std::string_view::npos || lf_pos < crlf_pos) {
    return {lf_pos, 2};
  }
  return {crlf_pos, 4};
}

std::string rewriteSseEvent(std::string_view raw_event, const std::string& key,
                            std::string_view upstream_model,
                            std::string_view client_model) {
  std::string result;
  size_t line_start = 0;
  while (line_start <= raw_event.size()) {
    auto line_end = raw_event.find('\n', line_start);
    std::string_view line;
    bool has_newline = line_end != std::string_view::npos;
    if (has_newline) {
      line = raw_event.substr(line_start, line_end - line_start);
      line_start = line_end + 1;
    } else {
      line = raw_event.substr(line_start);
      line_start = raw_event.size() + 1;
    }
    if (!line.empty() && line.back() == '\r') {
      line.remove_suffix(1);
    }
    if (!absl::StartsWith(line, "data:")) {
      result.append(line.data(), line.size());
      if (has_newline) {
        result.push_back('\n');
      }
      continue;
    }

    auto payload = absl::StripPrefix(line, "data:");
    auto payload_trimmed = absl::StripPrefix(payload, " ");
    if (payload_trimmed == "[DONE]") {
      result.append(line.data(), line.size());
      if (has_newline) {
        result.push_back('\n');
      }
      continue;
    }

    auto payload_json_opt = ::Wasm::Common::JsonParse(payload_trimmed);
    if (!payload_json_opt) {
      result.append(line.data(), line.size());
      if (has_newline) {
        result.push_back('\n');
      }
      continue;
    }
    auto payload_json = payload_json_opt.value();
    if (!rewriteModelFieldInJson(payload_json, key, upstream_model,
                                 client_model)) {
      result.append(line.data(), line.size());
      if (has_newline) {
        result.push_back('\n');
      }
      continue;
    }
    result.append("data: ");
    result.append(payload_json.dump());
    if (has_newline) {
      result.push_back('\n');
    }
  }
  return result;
}

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
    if (!JsonObjectIterate(model_mapping, [&](std::string key) -> bool {
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
        })) {
      return false;
    }
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

void PluginRootContext::incrementRequestCount() {
  request_count_++;
  if (request_count_ >= REBUILD_THRESHOLD) {
    LOG_DEBUG("Request count reached threshold, triggering rebuild");
    setFilterState("wasm_need_rebuild", "true");
    request_count_ = 0;  // Reset counter after setting rebuild flag
  }
}

FilterHeadersStatus PluginRootContext::onHeader(
    const ModelMapperConfigRule& rule) {
  // Increment request count and check for rebuild
  incrementRequestCount();

  // Check memory threshold and trigger rebuild if needed
  std::string value;
  if (getValue({"plugin_vm_memory"}, &value)) {
    // The value is stored as binary uint64_t, convert to string for logging
    if (value.size() == sizeof(uint64_t)) {
      uint64_t memory_size;
      memcpy(&memory_size, value.data(), sizeof(uint64_t));
      LOG_DEBUG(absl::StrCat("vm memory size is ", memory_size));
      if (memory_size >= MEMORY_THRESHOLD_BYTES) {
        LOG_INFO(absl::StrCat("Memory threshold reached (", memory_size, " >= ",
                              MEMORY_THRESHOLD_BYTES, "), triggering rebuild"));
        setFilterState("wasm_need_rebuild", "true");
      }
    } else {
      LOG_ERROR("invalid memory size format");
    }
  } else {
    LOG_ERROR("get vm memory size failed");
  }

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
  removeRequestHeader(Wasm::Common::Http::Header::ContentLength);
  setFilterState(SetDecoderBufferLimitKey, DefaultMaxBodyBytes);
  LOG_INFO(absl::StrCat("SetRequestBodyBufferLimit: ", DefaultMaxBodyBytes));
  return FilterHeadersStatus::StopIteration;
}

FilterDataStatus PluginRootContext::onBody(const ModelMapperConfigRule& rule,
                                           std::string_view body,
                                           PluginContext& stream) {
  stream.response_model_rewrite_enabled_ = false;
  const auto& exact_model_mapping = rule.exact_model_mapping_;
  const auto& prefix_model_mapping = rule.prefix_model_mapping_;
  const auto& default_model_mapping = rule.default_model_mapping_;
  const auto& model_key = rule.model_key_;
  auto body_json_opt = ::Wasm::Common::JsonParse(body);
  if (!body_json_opt) {
    LOG_WARN(absl::StrCat("cannot parse body to JSON string: ", body));
    return FilterDataStatus::Continue;
  }
  auto body_json = body_json_opt.value();
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
    if (!old_model.empty()) {
      stream.response_model_key_ = model_key;
      stream.response_client_model_ = old_model;
      stream.response_upstream_model_ = model;
      stream.response_model_rewrite_enabled_ = true;
    }
    body_json[model_key] = model;
    setBuffer(WasmBufferType::HttpRequestBody, 0,
              std::numeric_limits<size_t>::max(), body_json.dump());
    LOG_DEBUG(
        absl::StrCat("model mapped, before:", old_model, ", after:", model));
  }
  return FilterDataStatus::Continue;
}

FilterHeadersStatus PluginRootContext::onResponseHeader(PluginContext& stream) {
  if (!stream.response_model_rewrite_enabled_) {
    return FilterHeadersStatus::Continue;
  }
  stream.response_rewrite_mode_ = ResponseRewriteMode::None;
  auto content_type_value =
      getResponseHeader(Wasm::Common::Http::Header::ContentType);
  if (absl::StrContains(content_type_value->view(),
                        Wasm::Common::Http::ContentTypeValues::Json)) {
    stream.response_rewrite_mode_ = ResponseRewriteMode::Json;
    removeResponseHeader(Wasm::Common::Http::Header::ContentLength);
    setFilterState(SetEncoderBufferLimitKey, DefaultMaxBodyBytes);
    LOG_INFO(absl::StrCat("SetResponseBodyBufferLimit: ", DefaultMaxBodyBytes));
    return FilterHeadersStatus::StopIteration;
  }
  if (absl::StrContains(content_type_value->view(), EventStreamContentType)) {
    stream.response_rewrite_mode_ = ResponseRewriteMode::EventStream;
    removeResponseHeader(Wasm::Common::Http::Header::ContentLength);
    return FilterHeadersStatus::Continue;
  }
  stream.response_model_rewrite_enabled_ = false;
  return FilterHeadersStatus::Continue;
}

FilterDataStatus PluginRootContext::onResponseBody(PluginContext& stream,
                                                   std::string_view body,
                                                   bool end_stream) {
  if (!stream.response_model_rewrite_enabled_) {
    return FilterDataStatus::Continue;
  }
  if (stream.response_rewrite_mode_ == ResponseRewriteMode::None) {
    return FilterDataStatus::Continue;
  }

  if (stream.response_rewrite_mode_ == ResponseRewriteMode::Json) {
    auto body_json_opt = ::Wasm::Common::JsonParse(body);
    if (!body_json_opt) {
      LOG_WARN(absl::StrCat("cannot parse response body to JSON string: ", body));
      return FilterDataStatus::Continue;
    }
    auto body_json = body_json_opt.value();
    if (!rewriteModelFieldInJson(body_json, stream.response_model_key_,
                                 stream.response_upstream_model_,
                                 stream.response_client_model_)) {
      return FilterDataStatus::Continue;
    }
    setBuffer(WasmBufferType::HttpResponseBody, 0,
              std::numeric_limits<size_t>::max(), body_json.dump());
    LOG_DEBUG(absl::StrCat("response model mapped to client name:",
                           stream.response_client_model_));
    return FilterDataStatus::Continue;
  }

  stream.response_stream_pending_data_.append(body.data(), body.size());
  std::string output;
  while (true) {
    auto [event_pos, sep_size] =
        findSseEventSeparator(stream.response_stream_pending_data_);
    if (event_pos == std::string_view::npos) {
      break;
    }
    auto raw_event =
        std::string_view(stream.response_stream_pending_data_).substr(0, event_pos);
    output.append(rewriteSseEvent(raw_event, stream.response_model_key_,
                                  stream.response_upstream_model_,
                                  stream.response_client_model_));
    output.append(stream.response_stream_pending_data_.substr(event_pos, sep_size));
    stream.response_stream_pending_data_.erase(0, event_pos + sep_size);
  }
  if (end_stream && !stream.response_stream_pending_data_.empty()) {
    output.append(rewriteSseEvent(stream.response_stream_pending_data_,
                                  stream.response_model_key_,
                                  stream.response_upstream_model_,
                                  stream.response_client_model_));
    stream.response_stream_pending_data_.clear();
  }
  setBuffer(WasmBufferType::HttpResponseBody, 0, std::numeric_limits<size_t>::max(),
            output);
  return FilterDataStatus::Continue;
}

FilterHeadersStatus PluginContext::onRequestHeaders(uint32_t, bool) {
  body_total_size_ = 0;
  response_body_total_size_ = 0;
  response_model_rewrite_enabled_ = false;
  response_model_key_.clear();
  response_client_model_.clear();
  response_upstream_model_.clear();
  response_rewrite_mode_ = ResponseRewriteMode::None;
  response_stream_pending_data_.clear();
  auto* rootCtx = rootContext();
  return rootCtx->onHeaders([rootCtx, this](const auto& config) {
    auto ret = rootCtx->onHeader(config);
    if (ret == FilterHeadersStatus::StopIteration) {
      this->config_ = &config;
    } else {
      this->config_ = nullptr;
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
  return rootCtx->onBody(*config_, body->view(), *this);
}

FilterHeadersStatus PluginContext::onResponseHeaders(uint32_t, bool) {
  auto* rootCtx = rootContext();
  return rootCtx->onResponseHeader(*this);
}

FilterDataStatus PluginContext::onResponseBody(size_t body_size,
                                               bool end_stream) {
  if (!response_model_rewrite_enabled_) {
    return FilterDataStatus::Continue;
  }
  auto* rootCtx = rootContext();
  if (response_rewrite_mode_ == ResponseRewriteMode::EventStream) {
    auto body = getBufferBytes(WasmBufferType::HttpResponseBody, 0, body_size);
    return rootCtx->onResponseBody(*this, body->view(), end_stream);
  }
  response_body_total_size_ += body_size;
  if (!end_stream) {
    return FilterDataStatus::StopIterationAndBuffer;
  }
  auto body =
      getBufferBytes(WasmBufferType::HttpResponseBody, 0,
                     response_body_total_size_);
  return rootCtx->onResponseBody(*this, body->view(), true);
}

#ifdef NULL_PLUGIN

}  // namespace model_mapper
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
