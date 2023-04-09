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

#include <stdexcept>
#include <string>

#include "re2/re2.h"

namespace Wasm::Common::Regex {

class CompiledGoogleReMatcher {
 public:
  CompiledGoogleReMatcher(const std::string& regex,
                          bool do_program_size_check = true)
      : regex_(regex, re2::RE2::Quiet) {
    if (!regex_.ok()) {
      error_ = regex_.error();
      return;
    }
    if (do_program_size_check) {
      const auto regex_program_size =
          static_cast<uint32_t>(regex_.ProgramSize());
      if (regex_program_size > 100) {
        error_ = "too complex regex: " + regex;
      }
    }
  }

  const std::string& error() const { return error_; }

  bool match(std::string_view value) const {
    return re2::RE2::FullMatch(re2::StringPiece(value.data(), value.size()),
                               regex_);
  }

  std::string replaceAll(std::string_view value,
                         std::string_view substitution) const {
    std::string result = std::string(value);
    re2::RE2::GlobalReplace(
        &result, regex_,
        re2::StringPiece(substitution.data(), substitution.size()));
    return result;
  }

 private:
  const re2::RE2 regex_;
  std::string error_;
};

}  // namespace Wasm::Common::Regex
