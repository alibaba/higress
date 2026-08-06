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

#include "absl/strings/str_join.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "include/proxy-wasm/context.h"
#include "include/proxy-wasm/null.h"

namespace proxy_wasm {
namespace null_plugin {
namespace key_rate_limit {

NullPluginRegistry* context_registry_;
RegisterNullVmPluginFactory register_key_rate_limit_plugin(
    "key_rate_limit", []() {
      return std::make_unique<NullPlugin>(key_rate_limit::context_registry_);
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

class KeyRateLimitTest : public ::testing::Test {
 protected:
  KeyRateLimitTest() {
    // Initialize test VM
    test_vm_ = createNullVm();
    wasm_base_ = std::make_unique<WasmBase>(
        std::move(test_vm_), "test-vm", "", "",
        std::unordered_map<std::string, std::string>{},
        AllowedCapabilitiesMap{});
    wasm_base_->load("key_rate_limit");
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
  ~KeyRateLimitTest() override {}

  std::unique_ptr<WasmBase> wasm_base_;
  std::unique_ptr<WasmVm> test_vm_;
  std::unique_ptr<MockContext> mock_context_;

  std::unique_ptr<PluginRootContext> root_context_;
  std::unique_ptr<PluginContext> context_;

  std::string authority_;
  std::string route_name_;
  std::string status_code_;
};

TEST_F(KeyRateLimitTest, Config) {
  std::string configuration = R"(
{
    "limit_by_header": "x-api-key",
    "limit_keys": [
      {
        "key": "a",
        "query_per_second": 1
      },
      {
        "key": "b",
        "query_per_minute": 1
      },
      {
        "key": "c",
        "query_per_hour": 1
      },
      {
        "key": "d",
        "query_per_day": 1
      }
    ],
    "_rules_" : [
      {
        "_match_route_":["test"],
        "limit_by_param": "apikey",
        "limit_keys": [
          {
            "key": "a",
            "query_per_second": 10
          }
        ]
      }
    ]
})";

  BufferBase buffer;
  buffer.set({configuration.data(), configuration.size()});

  EXPECT_CALL(*mock_context_, getBuffer(WasmBufferType::PluginConfiguration))
      .WillOnce([&buffer](WasmBufferType) { return &buffer; });
  EXPECT_TRUE(root_context_->onConfigure(configuration.size()));
  EXPECT_EQ(root_context_->limits_.size(), 5);
  EXPECT_EQ(root_context_->limits_[0].first, 0);
  EXPECT_EQ(root_context_->limits_[1].first, 0);
  EXPECT_EQ(root_context_->limits_[2].first, 0);
  EXPECT_EQ(root_context_->limits_[3].first, 0);
  EXPECT_EQ(root_context_->limits_[4].first, 1);
}

TEST_F(KeyRateLimitTest, RuleConfig) {
  std::string configuration = R"(
{
    "_rules_" : [
      {
        "_match_route_":["test"],
        "limit_by_param": "apikey",
        "limit_keys": [
          {
            "key": "a",
            "query_per_second": 10
          }
        ]
      },
      {
        "_match_route_":["abc"],
        "limit_by_param": "apikey",
        "limit_keys": [
          {
            "key": "a",
            "query_per_second": 100
          }
        ]
      }
    ]
})";

  BufferBase buffer;
  buffer.set({configuration.data(), configuration.size()});

  EXPECT_CALL(*mock_context_, getBuffer(WasmBufferType::PluginConfiguration))
      .WillOnce([&buffer](WasmBufferType) { return &buffer; });
  EXPECT_TRUE(root_context_->onConfigure(configuration.size()));
  EXPECT_EQ(root_context_->limits_.size(), 2);
  EXPECT_EQ(root_context_->limits_[0].first, 1);
  EXPECT_EQ(root_context_->limits_[1].first, 2);
}

}  // namespace key_rate_limit
}  // namespace null_plugin
}  // namespace proxy_wasm
