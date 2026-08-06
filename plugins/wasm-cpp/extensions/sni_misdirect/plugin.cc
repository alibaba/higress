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

#include "extensions/sni_misdirect/plugin.h"

#include "absl/strings/match.h"
#include "absl/strings/str_format.h"
#include "common/http_util.h"

#ifdef NULL_PLUGIN

namespace proxy_wasm {
namespace null_plugin {
namespace sni_misdirect {

PROXY_WASM_NULL_PLUGIN_REGISTRY

NullPluginRegistry* context_registry_;
RegisterNullVmPluginFactory register_sni_misdirect_plugin(
    "envoy.wasm.sni_misdirect", []() {
      return std::make_unique<NullPlugin>(sni_misdirect::context_registry_);
    });

#endif

static RegisterContextFactory register_SNIMisdirect(
    CONTEXT_FACTORY(PluginContext), ROOT_FACTORY(PluginRootContext));

namespace {

void misdirectedRequest() {
  sendLocalResponse(421, "Misdirected Request", "", {});
}

}  // namespace

FilterHeadersStatus PluginContext::onRequestHeaders(uint32_t, bool) {
  std::string protocol;
  // no need to check HTTP/1.0 and HTTP/1.1
  if (getValue({"request", "protocol"}, &protocol) &&
      absl::StartsWith(protocol, "HTTP/1")) {
    return FilterHeadersStatus::Continue;
  }
  // no need to check http scheme
  std::string scheme;
  if (getValue({"request", "scheme"}, &scheme) && scheme != "https") {
    return FilterHeadersStatus::Continue;
  }
  // no need to check grpc
  auto content_type_header =
      getRequestHeader(Wasm::Common::Http::Header::ContentType);
  auto content_type = content_type_header->view();
  auto grpc_value =
      absl::string_view(Wasm::Common::Http::ContentTypeValues::Grpc.data(),
                        Wasm::Common::Http::ContentTypeValues::Grpc.size());
  if (absl::StartsWith(
          absl::string_view(content_type.data(), content_type.size()),
          grpc_value) &&
      (content_type.size() == grpc_value.size() ||
       content_type.at(grpc_value.size()) == '+')) {
    LOG_DEBUG("ignore grpc");
    return FilterHeadersStatus::Continue;
  }
  std::string sni;
  if (!getValue({"connection", "requested_server_name"}, &sni) || sni.empty()) {
    LOG_DEBUG("failed to get sni");
    return FilterHeadersStatus::Continue;
  }

  auto host_header = getRequestHeader(":authority");
  auto host = host_header->view();
  if (host.empty()) {
    LOG_CRITICAL("failed to get authority");
    return FilterHeadersStatus::Continue;
  }
  host = Wasm::Common::Http::stripPortFromHost(host);
  LOG_DEBUG(absl::StrFormat("sni:%s authority:%s", sni, host));
  if (sni == host) {
    return FilterHeadersStatus::Continue;
  }
  auto isWildcardSNI = absl::StartsWith(sni, "*.");
  if (!isWildcardSNI) {
    misdirectedRequest();
    return FilterHeadersStatus::StopIteration;
  }
  if (!absl::StrContains(absl::string_view(host.data(), host.size()),
                         sni.substr(1))) {
    misdirectedRequest();
    return FilterHeadersStatus::StopIteration;
  }
  return FilterHeadersStatus::Continue;
}

#ifdef NULL_PLUGIN

}  // namespace sni_misdirect
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
