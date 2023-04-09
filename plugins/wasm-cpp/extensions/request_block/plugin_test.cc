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

#include "extensions/request_block/plugin.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "include/proxy-wasm/context.h"
#include "include/proxy-wasm/null.h"

namespace proxy_wasm {
namespace null_plugin {
namespace request_block {

NullPluginRegistry* context_registry_;
RegisterNullVmPluginFactory register_request_block_plugin(
    "request_block", []() {
      return std::make_unique<NullPlugin>(request_block::context_registry_);
    });

class MockContext : public proxy_wasm::ContextBase {
 public:
  MockContext(WasmBase* wasm) : ContextBase(wasm) {}

  MOCK_METHOD(BufferInterface*, getBuffer, (WasmBufferType));
  MOCK_METHOD(WasmResult, log, (uint32_t, std::string_view));
  MOCK_METHOD(WasmResult, getHeaderMapValue,
              (WasmHeaderMapType /* type */, std::string_view /* key */,
               std::string_view* /*result */));
  MOCK_METHOD(WasmResult, getHeaderMapPairs, (WasmHeaderMapType, Pairs*));
  MOCK_METHOD(WasmResult, sendLocalResponse,
              (uint32_t /* response_code */, std::string_view /* body */,
               Pairs /* additional_headers */, uint32_t /* grpc_status */,
               std::string_view /* details */));
  MOCK_METHOD(WasmResult, getProperty, (std::string_view, std::string*));
};

class RequestBlockTest : public ::testing::Test {
 protected:
  RequestBlockTest() {
    // Initialize test VM
    test_vm_ = createNullVm();
    wasm_base_ = std::make_unique<WasmBase>(
        std::move(test_vm_), "test-vm", "", "",
        std::unordered_map<std::string, std::string>{},
        AllowedCapabilitiesMap{});
    wasm_base_->load("request_block");
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
          if (header == ":path") {
            *result = path_;
          }
          return WasmResult::Ok;
        });

    ON_CALL(*mock_context_, getProperty(testing::_, testing::_))
        .WillByDefault([&](std::string_view path, std::string* result) {
          *result = route_name_;
          return WasmResult::Ok;
        });

    ON_CALL(*mock_context_,
            getHeaderMapPairs(WasmHeaderMapType::RequestHeaders, testing::_))
        .WillByDefault([&](WasmHeaderMapType, Pairs* result) {
          *result = headers_;
          return WasmResult::Ok;
        });

    ON_CALL(*mock_context_, getBuffer(testing::_))
        .WillByDefault([&](WasmBufferType type) {
          if (type == WasmBufferType::HttpRequestBody) {
            return &body_;
          }
          return &config_;
        });

    // Initialize Wasm sandbox context
    root_context_ = std::make_unique<PluginRootContext>(0, "");
    context_ = std::make_unique<PluginContext>(1, root_context_.get());
  }
  ~RequestBlockTest() override {}

  std::unique_ptr<WasmBase> wasm_base_;
  std::unique_ptr<WasmVm> test_vm_;
  std::unique_ptr<MockContext> mock_context_;

  std::unique_ptr<PluginRootContext> root_context_;
  std::unique_ptr<PluginContext> context_;

  std::string authority_;
  std::string route_name_;
  std::string path_;
  Pairs headers_;
  BufferBase body_;
  BufferBase config_;
};

TEST_F(RequestBlockTest, CaseSensitive) {
  std::string configuration = R"(
{
   "block_urls": ["?foo=bar", "swagger.html"],
   "block_headers": ["headerKey", "headerValue"],
   "block_bodys": ["Hello World"]
})";

  config_.set({configuration.data(), configuration.size()});
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/?foo=BAR";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  path_ = "/?foo=bar";
  EXPECT_CALL(*mock_context_, sendLocalResponse(403, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  path_ = "/swagger.html?foo=BAR";
  EXPECT_CALL(*mock_context_, sendLocalResponse(403, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  path_ = "";
  headers_ = {{"headerKey", "123"}};
  EXPECT_CALL(*mock_context_, sendLocalResponse(403, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  headers_ = {{"abc", "headerValue"}};
  EXPECT_CALL(*mock_context_, sendLocalResponse(403, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  headers_ = {{"abc", "123"}};
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  body_.set("Hello World");
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);
  EXPECT_CALL(*mock_context_, sendLocalResponse(403, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestBody(11, true),
            FilterDataStatus::StopIterationNoBuffer);

  body_.set("hello world");
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);
  EXPECT_EQ(context_->onRequestBody(11, true), FilterDataStatus::Continue);
}

TEST_F(RequestBlockTest, CaseInsensitive) {
  std::string configuration = R"(
{
   "case_sensitive": false,
   "blocked_code": 404,
   "block_urls": ["?foo=bar", "swagger.html"],
   "block_headers": ["headerKey", "headerValue"],
   "block_bodys": ["Hello World"]
})";

  config_.set({configuration.data(), configuration.size()});
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/?foo=BAR";
  EXPECT_CALL(*mock_context_, sendLocalResponse(404, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  path_ = "/swagger.html?foo=bar";
  EXPECT_CALL(*mock_context_, sendLocalResponse(404, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  path_ = "";
  headers_ = {{"headerkey", "123"}};
  EXPECT_CALL(*mock_context_, sendLocalResponse(404, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  headers_ = {{"abc", "headervalue"}};
  EXPECT_CALL(*mock_context_, sendLocalResponse(404, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  headers_ = {{"abc", "123"}};
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);

  body_.set("hello world");
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);
  EXPECT_CALL(*mock_context_, sendLocalResponse(404, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestBody(11, true),
            FilterDataStatus::StopIterationNoBuffer);
}

}  // namespace request_block
}  // namespace null_plugin
}  // namespace proxy_wasm
