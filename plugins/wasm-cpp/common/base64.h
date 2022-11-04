/* Copyright 2019 Istio Authors. All Rights Reserved.
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
 *
 * From
 * https://github.com/envoyproxy/envoy/blob/master/source/common/common/base64.{h,cc}
 */

#pragma once

#include <string>

static const std::string EMPTY_STRING;

class Base64 {
 public:
  static std::string encode(const char* input, uint64_t length,
                            bool add_padding);
  static std::string encode(const char* input, uint64_t length) {
    return encode(input, length, true);
  }
  static std::string decodeWithoutPadding(std::string_view input);
};

// clang-format off
inline constexpr char CHAR_TABLE[] =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

inline constexpr unsigned char REVERSE_LOOKUP_TABLE[256] = {
    64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64,
    64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 62, 64, 64, 64, 63,
    52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 64, 64, 64, 64, 64, 64, 64, 0,  1,  2,  3,  4,  5,  6,
    7,  8,  9,  10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 64, 64, 64, 64, 64,
    64, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
    49, 50, 51, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64,
    64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64,
    64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64,
    64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64,
    64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64,
    64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64};
// clang-format on

inline bool decodeBase(const uint8_t cur_char, uint64_t pos, std::string& ret,
                       const unsigned char* const reverse_lookup_table) {
  const unsigned char c = reverse_lookup_table[static_cast<uint32_t>(cur_char)];
  if (c == 64) {
    // Invalid character
    return false;
  }

  switch (pos % 4) {
    case 0:
      ret.push_back(c << 2);
      break;
    case 1:
      ret.back() |= c >> 4;
      ret.push_back(c << 4);
      break;
    case 2:
      ret.back() |= c >> 2;
      ret.push_back(c << 6);
      break;
    case 3:
      ret.back() |= c;
      break;
  }
  return true;
}

inline bool decodeLast(const uint8_t cur_char, uint64_t pos, std::string& ret,
                       const unsigned char* const reverse_lookup_table) {
  const unsigned char c = reverse_lookup_table[static_cast<uint32_t>(cur_char)];
  if (c == 64) {
    // Invalid character
    return false;
  }

  switch (pos % 4) {
    case 0:
      return false;
    case 1:
      ret.back() |= c >> 4;
      return (c & 0b1111) == 0;
    case 2:
      ret.back() |= c >> 2;
      return (c & 0b11) == 0;
    case 3:
      ret.back() |= c;
      break;
  }
  return true;
}

inline void encodeBase(const uint8_t cur_char, uint64_t pos, uint8_t& next_c,
                       std::string& ret, const char* const char_table) {
  switch (pos % 3) {
    case 0:
      ret.push_back(char_table[cur_char >> 2]);
      next_c = (cur_char & 0x03) << 4;
      break;
    case 1:
      ret.push_back(char_table[next_c | (cur_char >> 4)]);
      next_c = (cur_char & 0x0f) << 2;
      break;
    case 2:
      ret.push_back(char_table[next_c | (cur_char >> 6)]);
      ret.push_back(char_table[cur_char & 0x3f]);
      next_c = 0;
      break;
  }
}

inline void encodeLast(uint64_t pos, uint8_t last_char, std::string& ret,
                       const char* const char_table, bool add_padding) {
  switch (pos % 3) {
    case 1:
      ret.push_back(char_table[last_char]);
      if (add_padding) {
        ret.push_back('=');
        ret.push_back('=');
      }
      break;
    case 2:
      ret.push_back(char_table[last_char]);
      if (add_padding) {
        ret.push_back('=');
      }
      break;
    default:
      break;
  }
}

inline std::string Base64::encode(const char* input, uint64_t length,
                                  bool add_padding) {
  uint64_t output_length = (length + 2) / 3 * 4;
  std::string ret;
  ret.reserve(output_length);

  uint64_t pos = 0;
  uint8_t next_c = 0;

  for (uint64_t i = 0; i < length; ++i) {
    encodeBase(input[i], pos++, next_c, ret, CHAR_TABLE);
  }

  encodeLast(pos, next_c, ret, CHAR_TABLE, add_padding);

  return ret;
}

inline std::string Base64::decodeWithoutPadding(std::string_view input) {
  if (input.empty()) {
    return EMPTY_STRING;
  }

  // At most last two chars can be '='.
  size_t n = input.length();
  if (input[n - 1] == '=') {
    n--;
    if (n > 0 && input[n - 1] == '=') {
      n--;
    }
  }
  // Last position before "valid" padding character.
  uint64_t last = n - 1;
  // Determine output length.
  size_t max_length = (n + 3) / 4 * 3;
  if (n % 4 == 3) {
    max_length -= 1;
  }
  if (n % 4 == 2) {
    max_length -= 2;
  }

  std::string ret;
  ret.reserve(max_length);
  for (uint64_t i = 0; i < last; ++i) {
    if (!decodeBase(input[i], i, ret, REVERSE_LOOKUP_TABLE)) {
      return EMPTY_STRING;
    }
  }

  if (!decodeLast(input[last], last, ret, REVERSE_LOOKUP_TABLE)) {
    return EMPTY_STRING;
  }

  ASSERT(ret.size() == max_length);
  return ret;
}
