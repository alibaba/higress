/*
 * Copyright (c) 2022 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#pragma once

#include <string_view>

#include "absl/strings/string_view.h"

namespace Wasm::Common {
inline absl::string_view stdToAbsl(const std::string_view& str) {
  return {str.data(), str.size()};
}
inline std::string_view abslToStd(const absl::string_view& str) {
  return {str.data(), str.size()};
}

const char WhitespaceChars[] = " \t\f\v\n\r";

inline std::string_view ltrim(std::string_view source) {
  const std::string_view::size_type pos =
      source.find_first_not_of(WhitespaceChars);
  if (pos != std::string_view::npos) {
    source.remove_prefix(pos);
  } else {
    source.remove_prefix(source.size());
  }
  return source;
}

inline std::string_view rtrim(std::string_view source) {
  const std::string_view::size_type pos =
      source.find_last_not_of(WhitespaceChars);
  if (pos != std::string_view::npos) {
    source.remove_suffix(source.size() - pos - 1);
  } else {
    source.remove_suffix(source.size());
  }
  return source;
}

inline std::string_view trim(std::string_view source) {
  return ltrim(rtrim(source));
}

}  // namespace Wasm::Common
