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

#include <cstddef>
#include <regex>

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "include/proxy-wasm/context.h"
#include "include/proxy-wasm/null.h"

namespace proxy_wasm {
namespace null_plugin {
namespace model_router {

NullPluginRegistry* context_registry_;
RegisterNullVmPluginFactory register_model_router_plugin("model_router", []() {
  return std::make_unique<NullPlugin>(model_router::context_registry_);
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
class ModelRouterTest : public ::testing::Test {
 protected:
  ModelRouterTest() {
    // Initialize test VM
    test_vm_ = createNullVm();
    wasm_base_ = std::make_unique<WasmBase>(
        std::move(test_vm_), "test-vm", "", "",
        std::unordered_map<std::string, std::string>{},
        AllowedCapabilitiesMap{});
    wasm_base_->load("model_router");
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
            *result = content_type_;
          } else if (header == "content-length") {
            *result = "1024";
          } else if (header == ":path") {
            *result = path_;
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
          *result = route_name_;
          return WasmResult::Ok;
        });
    ON_CALL(*mock_context_, setProperty(testing::_, testing::_))
        .WillByDefault(
            [&](std::string_view, std::string_view) { return WasmResult::Ok; });
  }
  ~ModelRouterTest() override {}
  std::unique_ptr<WasmBase> wasm_base_;
  std::unique_ptr<WasmVm> test_vm_;
  std::unique_ptr<MockContext> mock_context_;
  std::unique_ptr<PluginRootContext> root_context_;
  std::unique_ptr<PluginContext> context_;
  std::string route_name_;
  std::string path_;
  std::string content_type_ = "application/json";
  BufferBase body_;
  BufferBase config_;
};

TEST_F(ModelRouterTest, RewriteModelAndHeader) {
  std::string configuration = R"(
{
  "addProviderHeader": "x-higress-llm-provider"
})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/v1/chat/completions";
  std::string request_json = R"({"model": "qwen/qwen-long"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long"})");
        return WasmResult::Ok;
      });

  EXPECT_CALL(*mock_context_,
              replaceHeaderMapValue(testing::_,
                                    std::string_view("x-higress-llm-provider"),
                                    std::string_view("qwen")));

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(request_json.length(), true),
            FilterDataStatus::Continue);
}

TEST_F(ModelRouterTest, ModelToHeader) {
  std::string configuration = R"(
{
  "modelToHeader": "x-higress-llm-model"
})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/v1/chat/completions";
  std::string request_json = R"({"model": "qwen-long"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .Times(0);

  EXPECT_CALL(
      *mock_context_,
      replaceHeaderMapValue(testing::_, std::string_view("x-higress-llm-model"),
                            std::string_view("qwen-long")));

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(request_json.length(), true),
            FilterDataStatus::Continue);
}

TEST_F(ModelRouterTest, IgnorePath) {
  std::string configuration = R"(
{
  "addProviderHeader": "x-higress-llm-provider"
})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/v1/chat/xxxx";
  std::string request_json = R"({"model": "qwen/qwen-long"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .Times(0);

  EXPECT_CALL(*mock_context_,
              replaceHeaderMapValue(testing::_,
                                    std::string_view("x-higress-llm-provider"),
                                    std::string_view("qwen")))
      .Times(0);

  body_.set(request_json);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::Continue);
  EXPECT_EQ(context_->onRequestBody(request_json.length(), true),
            FilterDataStatus::Continue);
}

TEST_F(ModelRouterTest, RouteLevelRewriteModelAndHeader) {
  std::string configuration = R"(
{
  "_rules_": [
    {
      "_match_route_": ["route-a"],
      "addProviderHeader": "x-higress-llm-provider"
    }
]})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/api/v1/chat/completions";
  std::string request_json = R"({"model": "qwen/qwen-long"})";
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t, size_t, std::string_view body) {
        EXPECT_EQ(body, R"({"model":"qwen-long"})");
        return WasmResult::Ok;
      });

  EXPECT_CALL(*mock_context_,
              replaceHeaderMapValue(testing::_,
                                    std::string_view("x-higress-llm-provider"),
                                    std::string_view("qwen")));

  body_.set(request_json);
  route_name_ = "route-a";
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);
  EXPECT_EQ(context_->onRequestBody(request_json.length(), true),
            FilterDataStatus::Continue);
}

TEST_F(ModelRouterTest, RewriteModelAndHeaderMultipartFormData) {
  std::string configuration = R"({
  "addProviderHeader": "x-higress-llm-provider"
})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/v1/chat/completions";
  content_type_ =
      "multipart/form-data; "
      "boundary=--------------------------100751621174704322650451";
  std::string request_data = std::regex_replace(
      R"(
----------------------------100751621174704322650451
Content-Disposition: form-data; name="purpose"

batch
----------------------------100751621174704322650451
Content-Disposition: form-data; name="model"

qwen/qwen-turbo
----------------------------100751621174704322650451
Content-Disposition: form-data; name="file"; filename="test-data.json"
Content-Type: application/json

[
]
----------------------------100751621174704322650451--
)",
      std::regex("\n"), "\r\n");  // Multipart data requires CRLF line endings
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .WillOnce([&](WasmBufferType, size_t start, size_t length,
                    std::string_view body) {
        std::cerr << "==============="
                  << "\n";
        std::cerr << body << "\n";
        std::cerr << "==============="
                  << "\n";
        EXPECT_EQ(start, 0);
        EXPECT_EQ(length, std::numeric_limits<size_t>::max());
        auto expected_body = std::regex_replace(
            R"(
----------------------------100751621174704322650451
Content-Disposition: form-data; name="purpose"

batch
----------------------------100751621174704322650451
Content-Disposition: form-data; name="model"

qwen-turbo
)",
            std::regex("\n"),
            "\r\n");  // Multipart data requires CRLF line endings
        EXPECT_EQ(body, expected_body);
        return WasmResult::Ok;
      });

  EXPECT_CALL(*mock_context_,
              replaceHeaderMapValue(testing::_,
                                    std::string_view("x-higress-llm-provider"),
                                    std::string_view("qwen")));

  body_.set(request_data);
  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  auto last_body_size = 0;

  auto body = request_data.substr(
      0, request_data.find("batch") + 5 + 2 /* batch + CRLF */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::StopIterationAndBuffer);
  last_body_size = body.size();

  body = request_data.substr(0, request_data.find("\"model\"") + 5 + 2 +
                                    2 /* "model" + CRLF + CRLF */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::StopIterationAndBuffer);
  last_body_size = body.size();

  body = request_data.substr(0, request_data.find("qwen") + 4 /* "qwen" */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::StopIterationAndBuffer);
  last_body_size = body.size();

  body = request_data.substr(
      0, request_data.find("qwen-turbo") + 10 /* "qwen-turbo" */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::StopIterationAndBuffer);
  last_body_size = body.size();

  body = request_data.substr(
      0, request_data.find("qwen-turbo") + 10 + 2 /* "qwen-turbo" + CRLF */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::Continue);
  last_body_size = body.size();

  body = request_data.substr(0, request_data.find("qwen-turbo") + 10 + 2 +
                                    50 /* "qwen-turbo" + CRLF + boundary */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::Continue);
  last_body_size = body.size();

  body_.set(request_data);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, true),
            FilterDataStatus::Continue);
}

TEST_F(ModelRouterTest, ModelToHeaderMultipartFormData) {
  std::string configuration = R"(
{
  "modelToHeader": "x-higress-llm-model"
})";

  config_.set(configuration);
  EXPECT_TRUE(root_context_->configure(configuration.size()));

  path_ = "/v1/chat/completions";
  content_type_ =
      "multipart/form-data; "
      "boundary=--------------------------100751621174704322650451";
  std::string request_data = std::regex_replace(
      R"(
----------------------------100751621174704322650451
Content-Disposition: form-data; name="purpose"

batch
----------------------------100751621174704322650451
Content-Disposition: form-data; name="model"

qwen-max
----------------------------100751621174704322650451
Content-Disposition: form-data; name="file"; filename="test-data.json"
Content-Type: application/json

[
]
----------------------------100751621174704322650451--
)",
      std::regex("\n"), "\r\n");  // Multipart data requires CRLF line endings
  EXPECT_CALL(*mock_context_,
              setBuffer(testing::_, testing::_, testing::_, testing::_))
      .Times(0);

  EXPECT_CALL(
      *mock_context_,
      replaceHeaderMapValue(testing::_, std::string_view("x-higress-llm-model"),
                            std::string_view("qwen-max")));

  EXPECT_EQ(context_->onRequestHeaders(0, false),
            FilterHeadersStatus::StopIteration);

  auto last_body_size = 0;

  auto body = request_data.substr(
      0, request_data.find("batch") + 5 + 2 /* batch + CRLF */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::StopIterationAndBuffer);
  last_body_size = body.size();

  body = request_data.substr(0, request_data.find("\"model\"") + 5 + 2 +
                                    2 /* "model" + CRLF + CRLF */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::StopIterationAndBuffer);
  last_body_size = body.size();

  body = request_data.substr(0, request_data.find("qwen") + 4 /* "qwen" */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::StopIterationAndBuffer);
  last_body_size = body.size();

  body = request_data.substr(
      0, request_data.find("qwen-max") + 8 /* "qwen-max" */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::StopIterationAndBuffer);
  last_body_size = body.size();

  body = request_data.substr(
      0, request_data.find("qwen-max") + 8 + 2 /* "qwen-max" + CRLF */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::Continue);
  last_body_size = body.size();

  body = request_data.substr(
      0, request_data.find("qwen-max") + 8 + 2 + 50 /* "qwen-max" + CRLF */);
  body_.set(body);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, false),
            FilterDataStatus::Continue);
  last_body_size = body.size();

  body_.set(request_data);
  EXPECT_EQ(context_->onRequestBody(body.size() - last_body_size, true),
            FilterDataStatus::Continue);
}

}  // namespace model_router
}  // namespace null_plugin
}  // namespace proxy_wasm
