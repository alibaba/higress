/* Copyright 2020 Istio Authors. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#pragma once

#include "common/nlohmann_json.hpp"

/**
 * Utilities for working with JSON without exceptions.
 */
namespace Wasm {
namespace Common {

using JsonObject = ::nlohmann::json;

enum JsonParserResultDetail {
  UNKNOWN,
  OK,
  OUT_OF_RANGE,
  TYPE_ERROR,
  INVALID_VALUE,
};

std::optional<JsonObject> JsonParse(std::string_view str);

template <typename T>
std::pair<std::optional<T>, JsonParserResultDetail> JsonValueAs(
    const JsonObject&) {
  static_assert(true, "Unsupported Type");
}

template <>
std::pair<std::optional<std::string_view>, JsonParserResultDetail>
JsonValueAs<std::string_view>(const JsonObject& j);

template <>
std::pair<std::optional<std::string>, JsonParserResultDetail>
JsonValueAs<std::string>(const JsonObject& j);

template <>
std::pair<std::optional<int64_t>, JsonParserResultDetail> JsonValueAs<int64_t>(
    const JsonObject& j);

template <>
std::pair<std::optional<uint64_t>, JsonParserResultDetail>
JsonValueAs<uint64_t>(const JsonObject& j);

template <>
std::pair<std::optional<bool>, JsonParserResultDetail> JsonValueAs<bool>(
    const JsonObject& j);

template <>
std::pair<std::optional<JsonObject>, JsonParserResultDetail>
JsonValueAs<JsonObject>(const JsonObject& j);

template <>
std::pair<std::optional<std::vector<std::string_view>>, JsonParserResultDetail>
JsonValueAs<std::vector<std::string_view>>(const JsonObject& j);

template <class T>
class JsonGetField {
 public:
  JsonGetField(const JsonObject& j, std::string_view field);
  const JsonParserResultDetail& detail() { return detail_; }
  T value() { return object_; }
  T value_or(T v) {
    if (detail_ != JsonParserResultDetail::OK)
      return v;
    else
      return object_;
  };

 private:
  JsonParserResultDetail detail_;
  T object_;
};

template <class T>
JsonGetField<T>::JsonGetField(const JsonObject& j, std::string_view field) {
  auto it = j.find(field);
  if (it == j.end()) {
    detail_ = JsonParserResultDetail::OUT_OF_RANGE;
    return;
  }
  auto value = JsonValueAs<T>(it.value());
  detail_ = value.second;
  if (value.first.has_value()) {
    object_ = value.first.value();
  }
}

// Iterate over an optional array field.
// Returns false if set and not an array, or any of the visitor calls returns
// false.
bool JsonArrayIterate(
    const JsonObject& j, std::string_view field,
    const std::function<bool(const JsonObject& elt)>& visitor);

// Iterate over an optional object field key set.
// Returns false if set and not an object, or any of the visitor calls returns
// false.
bool JsonObjectIterate(const JsonObject& j, std::string_view field,
                       const std::function<bool(std::string key)>& visitor);
bool JsonObjectIterate(const JsonObject& j,
                       const std::function<bool(std::string key)>& visitor);

}  // namespace Common
}  // namespace Wasm
