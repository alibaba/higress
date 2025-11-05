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

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "include/proxy-wasm/context.h"
#include "include/proxy-wasm/null.h"

namespace proxy_wasm {
namespace null_plugin {
namespace custom_response {

NullPluginRegistry* context_registry_;
RegisterNullVmPluginFactory register_custom_response_plugin(
    "custom_response", []() {
      return std::make_unique<NullPlugin>(custom_response::context_registry_);
    });

class MockContext : public proxy_wasm::ContextBase {
 public:
  MockContext(WasmBase* wasm) : ContextBase(wasm) {}

  MOCK_METHOD(BufferInterface*, getBuffer, (WasmBufferType));
  MOCK_METHOD(WasmResult, log, (uint32_t, std::string_view));
  MOCK_METHOD(WasmResult, getHeaderMapValue,
              (WasmHeaderMapType /* type */, std::string_view /* key */,
               std::string_view* /*result */));
  MOCK_METHOD(WasmResult, replaceHeaderMapValue,
              (WasmHeaderMapType /* type */, std::string_view /* key */,
               std::string_view /* value */));
  MOCK_METHOD(WasmResult, sendLocalResponse,
              (uint32_t /* response_code */, std::string_view /* body */,
               Pairs /* additional_headers */, uint32_t /* grpc_status */,
               std::string_view /* details */));
  MOCK_METHOD(WasmResult, getProperty, (std::string_view, std::string*));
};

class CustomResponseTest : public ::testing::Test {
 protected:
  CustomResponseTest() {
    // Initialize test VM
    test_vm_ = createNullVm();
    wasm_base_ = std::make_unique<WasmBase>(
        std::move(test_vm_), "test-vm", "", "",
        std::unordered_map<std::string, std::string>{},
        AllowedCapabilitiesMap{});
    wasm_base_->load("custom_response");
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
          return WasmResult::Ok;
        });

    ON_CALL(*mock_context_,
            replaceHeaderMapValue(WasmHeaderMapType::RequestHeaders, testing::_,
                                  testing::_))
        .WillByDefault([&](WasmHeaderMapType, std::string_view key,
                           std::string_view value) { return WasmResult::Ok; });

    ON_CALL(*mock_context_,
            getHeaderMapValue(WasmHeaderMapType::ResponseHeaders, testing::_,
                              testing::_))
        .WillByDefault([&](WasmHeaderMapType, std::string_view header,
                           std::string_view* result) {
          if (header == ":status") {
            *result = status_code_;
          }
          return WasmResult::Ok;
        });

    ON_CALL(*mock_context_, getProperty(testing::_, testing::_))
        .WillByDefault([&](std::string_view path, std::string* result) {
          *result = route_name_;
          return WasmResult::Ok;
        });

    // Initialize Wasm sandbox context
    root_context_ = std::make_unique<PluginRootContext>(0, "");
    context_ = std::make_unique<PluginContext>(1, root_context_.get());
  }
  ~CustomResponseTest() override {}

  std::unique_ptr<WasmBase> wasm_base_;
  std::unique_ptr<WasmVm> test_vm_;
  std::unique_ptr<MockContext> mock_context_;

  std::unique_ptr<PluginRootContext> root_context_;
  std::unique_ptr<PluginContext> context_;

  std::string authority_;
  std::string route_name_;
  std::string status_code_;
};

TEST_F(CustomResponseTest, EnableOnStatus) {
  std::string configuration = R"(
{
   "enable_on_status": [429],
   "headers": ["abc=123","zty=test"],
   "status_code": 233,
   "body": "{\"abc\":123}"
})";

  BufferBase buffer;
  buffer.set({configuration.data(), configuration.size()});

  EXPECT_CALL(*mock_context_, getBuffer(WasmBufferType::PluginConfiguration))
      .WillOnce([&buffer](WasmBufferType) { return &buffer; });
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  status_code_ = "200";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);
  EXPECT_EQ(context_->onResponseHeaders(0, false),
            FilterHeadersStatus::Continue);

  status_code_ = "429";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);
  EXPECT_CALL(
      *mock_context_,
      sendLocalResponse(
          233, testing::_,
          testing::ElementsAre(
              testing::Pair("abc", "123"), testing::Pair("zty", "test"),
              testing::Pair("content-type", "application/json; charset=utf-8")),
          testing::_, testing::_));
  EXPECT_EQ(context_->onResponseHeaders(0, false),
            FilterHeadersStatus::StopIteration);
}

TEST_F(CustomResponseTest, ContentTypePlain) {
  std::string configuration = R"(
{
   "status_code": 200,
   "body": "abc"
})";

  BufferBase buffer;
  buffer.set({configuration.data(), configuration.size()});

  EXPECT_CALL(*mock_context_, getBuffer(WasmBufferType::PluginConfiguration))
      .WillOnce([&buffer](WasmBufferType) { return &buffer; });
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  EXPECT_CALL(
      *mock_context_,
      sendLocalResponse(200, testing::_,
                        testing::ElementsAre(testing::Pair(
                            "content-type", "text/plain; charset=utf-8")),
                        testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
}

TEST_F(CustomResponseTest, ContentTypeCustom) {
  std::string configuration = R"(
{
   "status_code": 200,
   "headers": ["content-type=application/custom"],
   "body": "abc"
})";

  BufferBase buffer;
  buffer.set({configuration.data(), configuration.size()});

  EXPECT_CALL(*mock_context_, getBuffer(WasmBufferType::PluginConfiguration))
      .WillOnce([&buffer](WasmBufferType) { return &buffer; });
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  EXPECT_CALL(*mock_context_,
              sendLocalResponse(200, testing::_,
                                testing::ElementsAre(testing::Pair(
                                    "content-type", "application/custom")),
                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
}

TEST_F(CustomResponseTest, NoGlobalRule) {
  std::string configuration = R"(
{
   "_rules_": [{
     "_match_route_": ["test"],
     "headers": ["abc=123","zty=test"],
     "status_code": 233,
     "body": "{\"abc\":123}"
   }]
})";

  BufferBase buffer;
  buffer.set({configuration.data(), configuration.size()});

  EXPECT_CALL(*mock_context_, getBuffer(WasmBufferType::PluginConfiguration))
      .WillOnce([&buffer](WasmBufferType) { return &buffer; });
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);
  EXPECT_EQ(context_->onResponseHeaders(0, false),
            FilterHeadersStatus::Continue);

  route_name_ = "test";
  EXPECT_CALL(*mock_context_, sendLocalResponse(233, testing::_, testing::_,
                                                testing::_, testing::_));
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
}
}  // namespace custom_response
}  // namespace null_plugin
}  // namespace proxy_wasm
