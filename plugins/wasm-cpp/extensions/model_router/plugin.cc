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
#include <regex>

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
    "set_decoder_buffer_limit";
constexpr std::string_view DefaultMaxBodyBytes = "104857600";

}  // namespace

bool PluginRootContext::parsePluginConfig(const json& configuration,
                                          ModelRouterConfigRule& rule) {
  if (auto it = configuration.find("modelKey"); it != configuration.end()) {
    if (it->is_string()) {
      rule.model_key_ = it->get<std::string>();
    } else {
      LOG_ERROR("Invalid type for modelKey. Expected string.");
      return false;
    }
  }

  if (auto it = configuration.find("addProviderHeader");
      it != configuration.end()) {
    if (it->is_string()) {
      rule.add_provider_header_ = it->get<std::string>();
    } else {
      LOG_ERROR("Invalid type for addProviderHeader. Expected string.");
      return false;
    }
  }

  if (auto it = configuration.find("modelToHeader");
      it != configuration.end()) {
    if (it->is_string()) {
      rule.model_to_header_ = it->get<std::string>();
    } else {
      LOG_ERROR("Invalid type for modelToHeader. Expected string.");
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
    LOG_ERROR("Invalid type for item in enableOnPathSuffix. Expected string.");
    return false;
  }

  return true;
}

bool PluginRootContext::onConfigure(size_t size) {
  // Parse configuration JSON string.
  if (size > 0 && !configure(size)) {
    LOG_ERROR("configuration has errors initialization will not continue.");
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
    LOG_ERROR(absl::StrCat("cannot parse plugin configuration JSON string: ",
                           configuration_data->view()));
    return false;
  }
  if (!parseRuleConfig(result.value())) {
    LOG_ERROR(absl::StrCat("cannot parse plugin configuration JSON string: ",
                           configuration_data->view()));
    return false;
  }
  return true;
}

FilterHeadersStatus PluginRootContext::onHeader(
    PluginContext& ctx,
    const ModelRouterConfigRule& rule) {
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
    // Support wildcard "*" to enable for all paths
    if (enable_suffix == "*") {
      enable = true;
      break;
    }
    if (absl::EndsWith({path.c_str(), uri_end}, enable_suffix)) {
      enable = true;
      break;
    }
  }
  if (!enable) {
    return FilterHeadersStatus::Continue;
  }
  auto content_type_ptr =
      getRequestHeader(Wasm::Common::Http::Header::ContentType);
  auto content_type_value = content_type_ptr->view();
  LOG_DEBUG(absl::StrCat("Content-Type: ", content_type_value));
  if (absl::StrContains(content_type_value,
                         Wasm::Common::Http::ContentTypeValues::Json)) {
    ctx.mode_ = MODE_JSON;
    LOG_DEBUG("Enable JSON mode.");
    removeRequestHeader(Wasm::Common::Http::Header::ContentLength);
    setFilterState(SetDecoderBufferLimitKey, DefaultMaxBodyBytes);
    LOG_INFO(absl::StrCat("SetRequestBodyBufferLimit: ", DefaultMaxBodyBytes));
    return FilterHeadersStatus::StopIteration;
  }
  if (absl::StrContains(content_type_value,
                         Wasm::Common::Http::ContentTypeValues::MultipartFormData)) {
    // Get the boundary from the content type
    auto boundary_start = content_type_value.find("boundary=");
    if (boundary_start == std::string::npos) {
      LOG_WARN(absl::StrCat("No boundary found in a multipart/form-data content-type: ", content_type_value));
      return FilterHeadersStatus::Continue;
    }
    boundary_start += 9;
    auto boundary_end = content_type_value.find(';', boundary_start);
    if (boundary_end == std::string::npos) {
      boundary_end = content_type_value.size();
    }
    auto boundary_length = boundary_end - boundary_start;
    if (boundary_length < 1 || boundary_length > 70) {
      // See https://www.w3.org/Protocols/rfc1341/7_2_Multipart.html
      LOG_WARN(absl::StrCat("Invalid boundary value in a multipart/form-data content-type: ", content_type_value));
      return FilterHeadersStatus::Continue;
    }
    auto boundary_value = content_type_value.substr(boundary_start, boundary_end - boundary_start);
    ctx.mode_ = MODE_MULTIPART;
    ctx.boundary_ = boundary_value;
    LOG_DEBUG(absl::StrCat("Enable multipart/form-data mode. Boundary=", boundary_value));
    return FilterHeadersStatus::StopIteration;
  }
  return FilterHeadersStatus::Continue;
}

FilterDataStatus PluginRootContext::onJsonBody(const ModelRouterConfigRule& rule,
                                           std::string_view body) {
  const auto& model_key = rule.model_key_;
  const auto& add_provider_header = rule.add_provider_header_;
  const auto& model_to_header = rule.model_to_header_;
  auto body_json_opt = ::Wasm::Common::JsonParse(body);
  if (!body_json_opt) {
    LOG_WARN(absl::StrCat("cannot parse body to JSON string: ", body));
    return FilterDataStatus::Continue;
  }
  auto body_json = body_json_opt.value();
  if (body_json.contains(model_key)) {
    std::string model_value = body_json[model_key];
    if (!model_to_header.empty()) {
      replaceRequestHeader(model_to_header, model_value);
    }
    if (!add_provider_header.empty()) {
      auto pos = model_value.find('/');
      if (pos != std::string::npos) {
        const auto& provider = model_value.substr(0, pos);
        const auto& model = model_value.substr(pos + 1);
        replaceRequestHeader(add_provider_header, provider);
        body_json[model_key] = model;
        setBuffer(WasmBufferType::HttpRequestBody, 0,
                  std::numeric_limits<size_t>::max(), body_json.dump());
        LOG_DEBUG(absl::StrCat("model route to provider:", provider,
                               ", model:", model));
      } else {
        LOG_DEBUG(absl::StrCat("model route to provider not work, model:",
                               model_value));
      }
    }
  }
  return FilterDataStatus::Continue;
}

FilterDataStatus PluginRootContext::onMultipartBody(
    PluginContext& ctx,
    const ModelRouterConfigRule& rule,
    WasmDataPtr& body,
    bool end_stream) {
  const auto& add_provider_header = rule.add_provider_header_;
  const auto& model_to_header = rule.model_to_header_;

  const auto boundary = ctx.boundary_;
  const auto body_view = body->view();
  const auto model_param_header = absl::StrCat("Content-Disposition: form-data; name=\"", rule.model_key_, "\"");

  for (size_t pos = 0; (pos = body_view.find(boundary, pos)) != std::string_view::npos;) {
    LOG_DEBUG(absl::StrCat("Found boundary at ", pos));
    pos += boundary.length();
    size_t end_pos = body_view.find(boundary, pos);
    if (end_pos == std::string_view::npos) {
      end_pos = body_view.length();
    }
    std::string_view part = body_view.substr(pos, end_pos - pos);
    LOG_DEBUG(absl::StrCat("Part: ", part));
    pos = end_pos;

    // Check if this part contains the model parameter
    if (!absl::StrContains(part, model_param_header)) {
      LOG_DEBUG("Part does not contain model parameter");
      continue;
    }
    size_t value_start = part.find(CRLF_CRLF);
    if (value_start == std::string_view::npos) {
      LOG_DEBUG("No value start found in part");
      break;
    }
    value_start += 4; // Skip the "\r\n\r\n"
    size_t value_end = part.find(CRLF, value_start);
    if (value_end == std::string_view::npos) {
      LOG_DEBUG("No value end found in part");
      break;
    }
    auto model_value = part.substr(value_start, value_end - value_start);
    LOG_DEBUG(absl::StrCat("Model value: ", model_value));
    if (!model_to_header.empty()) {
      replaceRequestHeader(model_to_header, model_value);
    }
    if (!add_provider_header.empty()) {
      auto pos = model_value.find('/');
      if (pos != std::string::npos) {
        const auto& provider = model_value.substr(0, pos);
        const auto& model = model_value.substr(pos + 1);
        replaceRequestHeader(add_provider_header, provider);
        setBuffer(WasmBufferType::HttpRequestBody, value_start,
                  value_end - value_start, model);
        LOG_DEBUG(absl::StrCat("model route to provider:", provider,
                               ", model:", model));
      } else {
        LOG_DEBUG(absl::StrCat("model route to provider not work, model:",
                               model_value));
      }
    }
    // We are done now. We can stop processing the body.
    LOG_DEBUG(absl::StrCat("Done processing multipart body after caching ", body_view.length() , " bytes."));
    ctx.mode_ = MODE_BYPASS;
    return FilterDataStatus::Continue;
  }
  if (end_stream) {
    LOG_DEBUG("No model parameter found in the body");
    return FilterDataStatus::Continue;
  }
  return FilterDataStatus::StopIterationAndBuffer;
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
  auto* rootCtx = rootContext();
  body_total_size_ += body_size;
  switch (mode_) {
    case MODE_JSON:
    {
      if (!end_stream) {
        return FilterDataStatus::StopIterationAndBuffer;
      }
      auto body =
          getBufferBytes(WasmBufferType::HttpRequestBody, 0, body_total_size_);
      return rootCtx->onJsonBody(*config_, body->view());
    }
    case MODE_MULTIPART:
    {
      auto body =
          getBufferBytes(WasmBufferType::HttpRequestBody, 0, body_total_size_);
      return rootCtx->onMultipartBody(*this, *config_, body, end_stream);
    }
    case MODE_BYPASS:
    default:
      return FilterDataStatus::Continue;
  }
}

#ifdef NULL_PLUGIN

}  // namespace model_router
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
