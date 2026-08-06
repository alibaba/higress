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

#include <initializer_list>

#include "absl/strings/str_format.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "include/proxy-wasm/context.h"
#include "include/proxy-wasm/null.h"

namespace proxy_wasm {
namespace null_plugin {
namespace sni_misdirect {

NullPluginRegistry* context_registry_;
RegisterNullVmPluginFactory register_sni_misdirect_plugin(
    "sni_misdirect", []() {
      return std::make_unique<NullPlugin>(sni_misdirect::context_registry_);
    });

class MockContext : public proxy_wasm::ContextBase {
 public:
  MockContext(WasmBase* wasm) : ContextBase(wasm) {}

  MOCK_METHOD(BufferInterface*, getBuffer, (WasmBufferType));
  MOCK_METHOD(WasmResult, log, (uint32_t, std::string_view));
  MOCK_METHOD(WasmResult, getHeaderMapValue,
              (WasmHeaderMapType /* type */, std::string_view /* key */,
               std::string_view* /*result */));
  MOCK_METHOD(WasmResult, sendLocalResponse,
              (uint32_t /* response_code */, std::string_view /* body */,
               Pairs /* additional_headers */, uint32_t /* grpc_status */,
               std::string_view /* details */));
  MOCK_METHOD(WasmResult, getProperty, (std::string_view, std::string*));
};

class SNIMisdirectTest : public ::testing::Test {
 protected:
  SNIMisdirectTest() {
    // Initialize test VM
    test_vm_ = createNullVm();
    wasm_base_ = std::make_unique<WasmBase>(
        std::move(test_vm_), "test-vm", "", "",
        std::unordered_map<std::string, std::string>{},
        AllowedCapabilitiesMap{});
    wasm_base_->load("sni_misdirect");
    wasm_base_->initialize();

    // Initialize host side context
    mock_context_ = std::make_unique<MockContext>(wasm_base_.get());
    current_context_ = mock_context_.get();

    ON_CALL(*mock_context_, log(testing::_, testing::_))
        .WillByDefault([](uint32_t, std::string_view m) {
          std::cerr << m << "\n";
          return WasmResult::Ok;
        });

    ON_CALL(*mock_context_, getHeaderMapValue(WasmHeaderMapType::RequestHeaders,
                                              testing::_, testing::_))
        .WillByDefault([&](WasmHeaderMapType, std::string_view header,
                           std::string_view* result) {
          if (header == ":authority") {
            *result = authority_;
          }
          if (header == "content-type") {
            *result = content_type_;
          }
          return WasmResult::Ok;
        });

    ON_CALL(*mock_context_, getProperty(testing::_, testing::_))
        .WillByDefault([&](std::string_view path, std::string* result) {
          if (path == absl::StrFormat("%s%c%s%c", "connection", 0,
                                      "requested_server_name", 0)) {
            *result = sni_;
          }
          if (path ==
              absl::StrFormat("%s%c%s%c", "request", 0, "protocol", 0)) {
            *result = protocol_;
          }
          if (path == absl::StrFormat("%s%c%s%c", "request", 0, "scheme", 0)) {
            *result = scheme_;
          }
          return WasmResult::Ok;
        });

    // Initialize Wasm sandbox context
    root_context_ = std::make_unique<PluginRootContext>(0, "");
    context_ = std::make_unique<PluginContext>(1, root_context_.get());
  }
  ~SNIMisdirectTest() override {}

  std::unique_ptr<WasmBase> wasm_base_;
  std::unique_ptr<WasmVm> test_vm_;
  std::unique_ptr<MockContext> mock_context_;

  std::unique_ptr<PluginRootContext> root_context_;
  std::unique_ptr<PluginContext> context_;

  std::string protocol_ = "HTTP/2";
  std::string authority_;
  std::string sni_;
  std::string content_type_;
  std::string scheme_ = "https";
};

TEST_F(SNIMisdirectTest, OnMatch) {
  authority_ = "a.example.com";
  sni_ = "b.example.com";
  EXPECT_CALL(*mock_context_, sendLocalResponse(421, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  authority_ = "a.example.com";
  sni_ = "a.example.com";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  authority_ = "a.example.com:80";
  sni_ = "a.example.com";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  authority_ = "a.test.com";
  sni_ = "*.example.com";
  EXPECT_CALL(*mock_context_, sendLocalResponse(421, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  authority_ = "a.example.com";
  sni_ = "*.example.com";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  authority_ = "a.example.com";
  sni_ = "b.example.com";
  protocol_ = "HTTP/1.1";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  authority_ = "a.example.com";
  sni_ = "";
  protocol_ = "HTTP/2";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  authority_ = "a.example.com";
  sni_ = "b.example.com";
  protocol_ = "HTTP/2";
  content_type_ = "application/grpc";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  authority_ = "a.example.com";
  sni_ = "b.example.com";
  protocol_ = "HTTP/2";
  content_type_ = "application/grpc+";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  authority_ = "a.example.com";
  sni_ = "b.example.com";
  protocol_ = "HTTP/2";
  content_type_ = "application/grpc-web";
  EXPECT_CALL(*mock_context_, sendLocalResponse(421, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  authority_ = "a.example.com";
  sni_ = "b.example.com";
  protocol_ = "HTTP/2";
  scheme_ = "http";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);
}

}  // namespace sni_misdirect
}  // namespace null_plugin
}  // namespace proxy_wasm
