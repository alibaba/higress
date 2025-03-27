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

#include <cctype>
#include <cstddef>

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "include/proxy-wasm/context.h"
#include "include/proxy-wasm/null.h"

namespace proxy_wasm {
namespace null_plugin {
namespace model_mapper {

NullPluginRegistry* context_registry_;
RegisterNullVmPluginFactory register_model_mapper_plugin("model_mapper", []() {
  return std::make_unique<NullPlugin>(model_mapper::context_registry_);
});

class MockContext : public proxy_wasm::ContextBase {
 public:
  MockContext(WasmBase* wasm) : ContextBase(wasm) {}
  MOCK_METHOD(BufferInterface*, getBuffer, (WasmBufferType));
  MOCK_METHOD(WasmResult, log, (uint32_t, std::string_view));
  MOCK_METHOD(WasmResult, setBuffer,
              (WasmBufferType, size_t, size_t, std::string_view));
  MOCK_METHOD(WasmResult, getHeaderMapValue,
              (WasmHeaderMapType /* type */, std::string_view /* key */,
               std::string_view* /*result */));
  MOCK_METHOD(WasmResult, replaceHeaderMapValue,
              (WasmHeaderMapType /* type */, std::string_view /* key */,
               std::string_view /* value */));
  MOCK_METHOD(WasmResult, removeHeaderMapValue,
              (WasmHeaderMapType /* type */, std::string_view /* key */));
  MOCK_METHOD(WasmResult, addHeaderMapValue,
              (WasmHeaderMapType, std::string_view, std::string_view));
  MOCK_METHOD(WasmResult, getProperty, (std::string_view, std::string*));
  MOCK_METHOD(WasmResult, setProperty, (std::string_view, std::string_view));
};
class ModelMapperTest : public ::testing::Test {
 protected:
  ModelMapperTest() {
    // Initialize test VM
    test_vm_ = createNullVm();
    wasm_base_ = std::make_unique<WasmBase>(
        std::move(test_vm_), "test-vm", "", "",
        std::unordered_map<std::string, std::string>{},
        AllowedCapabilitiesMap{});
    wasm_base_->load("model_mapper");
    wasm_base_->initialize();
    // Initialize host side context
    mock_context_ = std::make_unique<MockContext>(wasm_base_.get());
    current_context_ = mock_context_.get();
    // Initialize Wasm sandbox context
    root_context_ = std::make_unique<PluginRootContext>(0, "");
    context_ = std::make_unique<PluginContext>(1, root_context_.get());

    ON_CALL(*mock_context_, log(testing::_, testing::_))
        .WillByDefault([](uint32_t, std::string_view m) {
          std::cerr << m << "\n";
          return WasmResult::Ok;
        });

    ON_CALL(*mock_context_, getBuffer(testing::_))
        .WillByDefault([&](WasmBufferType type) {
          if (type == WasmBufferType::HttpRequestBody) {
            return &body_;
          }
          return &config_;
        });
    ON_CALL(*mock_context_, getHeaderMapValue(WasmHeaderMapType::RequestHeaders,
                                              testing::_, testing::_))
        .WillByDefault([&](WasmHeaderMapType, std::string_view header,
                           std::string_view* result) {
          if (header == "content-type") {
            *result = "application/json";
          } else if (header == "content-length") {
            *result = "1024";
          } else if (header == ":path") {
            *result = path_;
          } else if (header == "x-mse-consumer") {
            *result = consumer_;
          }
          return WasmResult::Ok;
        });
    ON_CALL(*mock_context_,
            replaceHeaderMapValue(WasmHeaderMapType::RequestHeaders, testing::_,
                                  testing::_))
        .WillByDefault([&](WasmHeaderMapType, std::string_view key,
                           std::string_view value) { return WasmResult::Ok; });
    ON_CALL(*mock_context_,
            removeHeaderMapValue(WasmHeaderMapType::RequestHeaders, testing::_))
        .WillByDefault([&](WasmHeaderMapType, std::string_view key) {
          return WasmResult::Ok;
        });
    ON_CALL(*mock_context_, addHeaderMapValue(WasmHeaderMapType::RequestHeaders,
                                              testing::_, testing::_))
        .WillByDefault([&](WasmHeaderMapType, std::string_view header,
                           std::string_view value) { return WasmResult::Ok; });
    ON_CALL(*mock_context_, getProperty(testing::_, testing::_))
        .WillByDefault([&](std::string_view path, std::string* result) {
          if (absl::StartsWith(path, "route_name")) {
            *result = route_name_;
          } else if (absl::StartsWith(path, "cluster_name")) {
            *result = service_name_;
          }
          return WasmResult::Ok;
        });
    ON_CALL(*mock_context_, setProperty(testing::_, testing::_))
        .WillByDefault(
            [&](std::string_view, std::string_view) { return WasmResult::Ok; });
  }
  ~ModelMapperTest() override {}
  std::unique_ptr<WasmBase> wasm_base_;
  std::unique_ptr<WasmVm> test_vm_;
  std::unique_ptr<MockContext> mock_context_;
  std::unique_ptr<PluginRootContext> root_context_;
  std::unique_ptr<PluginContext> context_;
  std::string route_name_;
  std::string service_name_;
  std::string path_;
  std::string consumer_;
  BufferBase body_;
  BufferBase config_;
};

TEST_F(ModelMapperTest, ModelMappingTest) {
  std::string configuration = R"(
{
  "modelMapping": {
     "*": "qwen-long",
     "gpt-4*": "qwen-max",
     "gpt-4o": "qwen-turbo",
     "gpt-4o-mini": "qwen-plus",
     "text-embedding-v1": ""
  }
})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/v1/chat/completions";
  std::string request_json = R"({"model": "gpt-3.5"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-max"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4o"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-turbo"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4o-mini"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-plus"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "text-embedding-v1"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .Times(0);

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);
}

TEST_F(ModelMapperTest, ConditionalModelMappingTestNoConsumer) {
  std::string configuration = R"(
{
  "modelMapping": {
     "*": "qwen-long",
     "gpt-4*": "qwen-max",
     "gpt-4o": "qwen-turbo",
     "gpt-4o-mini": "qwen-plus",
     "text-embedding-v1": ""
  },
  "conditionalModelMappings": [
    {
      "consumers": [
        "consumer-1a",
        "consumer-1b"
      ],
      "modelMapping": {
        "*": "qwen-long-*",
        "gpt-4*": "qwen-max-1",
        "gpt-4x": "",
        "gpt-4o": "qwen-turbo-1",
        "gpt-4o-mini": "qwen-plus-1",
        "text-embedding-v1": "text-embedding-v1-1"
      }
    }
  ]
})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/v1/chat/completions";
  std::string request_json = R"({"model": "gpt-3.5"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-max"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4o"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-turbo"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4o-mini"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-plus"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "text-embedding-v1"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .Times(0);

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);
}

TEST_F(ModelMapperTest, ConditionalModelMappingTestConsumerConfigured) {
  std::string configuration = R"(
{
  "modelMapping": {
     "*": "qwen-long",
     "gpt-4*": "qwen-max",
     "gpt-4o": "qwen-turbo",
     "gpt-4o-mini": "qwen-plus",
     "text-embedding-v1": ""
  },
  "conditionalModelMappings": [
    {
      "consumers": [
        "consumer-1a",
        "consumer-1b"
      ],
      "modelMapping": {
        "*": "qwen-long-1",
        "gpt-4*": "qwen-max-1",
        "gpt-4x": "",
        "gpt-4o": "qwen-turbo-1",
        "gpt-4o-mini": "qwen-plus-1",
        "text-embedding-v1": "text-embedding-v1-1"
      }
    }
  ]
})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/v1/chat/completions";
  consumer_ = "consumer-1a";
  std::string request_json = R"({"model": "gpt-3.5"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long-1"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-max-1"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4x"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .Times(0);

  request_json = R"({"model": "gpt-4o"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-turbo-1"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4o-mini"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-plus-1"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "text-embedding-v1"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"text-embedding-v1-1"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long-1"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);
}

TEST_F(ModelMapperTest, ConditionalModelMappingTestConsumerNotConfigured) {
  std::string configuration = R"(
{
  "modelMapping": {
     "*": "qwen-long",
     "gpt-4*": "qwen-max",
     "gpt-4o": "qwen-turbo",
     "gpt-4o-mini": "qwen-plus",
     "text-embedding-v1": ""
  },
  "conditionalModelMappings": [
    {
      "consumers": [
        "consumer-1a",
        "consumer-1b"
      ],
      "modelMapping": {
        "*": "qwen-long-*",
        "gpt-4*": "qwen-max-1",
        "gpt-4x": "",
        "gpt-4o": "qwen-turbo-1",
        "gpt-4o-mini": "qwen-plus-1",
        "text-embedding-v1": "text-embedding-v1-1"
      }
    }
  ]
})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/v1/chat/completions";
  consumer_ = "consumer-2a";
  std::string request_json = R"({"model": "gpt-3.5"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-max"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4o"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-turbo"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "gpt-4o-mini"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-plus"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({"model": "text-embedding-v1"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .Times(0);

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  request_json = R"({})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long"})");
        return WasmResult::Ok;
      });

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);
}

TEST_F(ModelMapperTest, RouteLevelModelMappingTest) {
  std::string configuration = R"(
{
  "_rules_": [
    {
      "_match_route_": ["route-a"],
      "_match_service_": ["service-1"],
      "modelMapping": {
        "*": "qwen-long"
      }
    },
    {
      "_match_route_": ["route-b"],
      "_match_service_": ["service-2"],
      "modelMapping": {
        "*": "qwen-max"
      }
    },
    {
      "_match_route_": ["route-b"],
      "_match_service_": ["service-3"],
      "modelMapping": {
        "*": "qwen-turbo"
      }
    }
]})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/api/v1/chat/completions";
  std::string request_json = R"({"model": "gpt-4"})";
  body_.set(request_json);
  route_name_ = "route-a";
  service_name_ = "outbound|80||service-1";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long"})");
        return WasmResult::Ok;
      });
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  route_name_ = "route-b";
  service_name_ = "outbound|80||service-2";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-max"})");
        return WasmResult::Ok;
      });
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);

  route_name_ = "route-b";
  service_name_ = "outbound|80||service-3";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-turbo"})");
        return WasmResult::Ok;
      });
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(20, true), FilterDataStatus::Continue);
}

}  // namespace model_mapper
}  // namespace null_plugin
}  // namespace proxy_wasm
