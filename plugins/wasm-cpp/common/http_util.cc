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

#include "http_util.h"

#include "absl/strings/ascii.h"
#include "absl/strings/match.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/str_format.h"
#include "absl/strings/str_split.h"

#include "common/common_util.h"

namespace Wasm::Common::Http {

std::string_view stripPortFromHost(std::string_view request_host) {
  // Remove port, if there is any. At Istio 1.10, port will be stripped
  // by default https://github.com/istio/istio/issues/25350.
  // Port removing code is inspired by
  // https://github.com/envoyproxy/envoy/blob/v1.17.0/source/common/http/header_utility.cc#L219
  const std::string_view::size_type port_start = request_host.rfind(':');
  if (port_start != std::string_view::npos) {
    // According to RFC3986 v6 address is always enclosed in "[]".
    // section 3.2.2.
    const auto v6_end_index = request_host.rfind("]");
    if (v6_end_index == std::string_view::npos || v6_end_index < port_start) {
      if ((port_start + 1) <= request_host.size()) {
        return request_host.substr(0, port_start);
      }
    }
  }
  return request_host;
}

std::string PercentEncoding::encode(absl::string_view value,
                                    absl::string_view reserved_chars) {
  std::unordered_set<char> reserved_char_set{reserved_chars.begin(),
                                             reserved_chars.end()};
  for (size_t i = 0; i < value.size(); ++i) {
    const char& ch = value[i];
    // The escaping characters are defined in
    // https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#responses.
    //
    // We do checking for each char in the string. If the current char is
    // included in the defined escaping characters, we jump to "the slow path"
    // (append the char [encoded or not encoded] to the returned string one by
    // one) started from the current index.
    if (ch < ' ' || ch >= '~' ||
        reserved_char_set.find(ch) != reserved_char_set.end()) {
      return PercentEncoding::encode(value, i, reserved_char_set);
    }
  }
  return std::string(value);
}

std::string PercentEncoding::encode(
    absl::string_view value, size_t index,
    const std::unordered_set<char>& reserved_char_set) {
  std::string encoded;
  if (index > 0) {
    absl::StrAppend(&encoded, value.substr(0, index));
  }

  for (size_t i = index; i < value.size(); ++i) {
    const char& ch = value[i];
    if (ch < ' ' || ch >= '~' ||
        reserved_char_set.find(ch) != reserved_char_set.end()) {
      // For consistency, URI producers should use uppercase hexadecimal digits
      // for all percent-encodings.
      // https://tools.ietf.org/html/rfc3986#section-2.1.
      absl::StrAppend(&encoded, absl::StrFormat("%02X", ch));
    } else {
      encoded.push_back(ch);
    }
  }
  return encoded;
}

std::string PercentEncoding::decode(absl::string_view encoded) {
  std::string decoded;
  decoded.reserve(encoded.size());
  for (size_t i = 0; i < encoded.size(); ++i) {
    char ch = encoded[i];
    if (ch == '%' && i + 2 < encoded.size()) {
      const char& hi = encoded[i + 1];
      const char& lo = encoded[i + 2];
      if (absl::ascii_isdigit(hi)) {
        ch = hi - '0';
      } else {
        ch = absl::ascii_toupper(hi) - 'A' + 10;
      }

      ch *= 16;
      if (absl::ascii_isdigit(lo)) {
        ch += lo - '0';
      } else {
        ch += absl::ascii_toupper(lo) - 'A' + 10;
      }
      i += 2;
    }
    decoded.push_back(ch);
  }
  return decoded;
}

SystemTime httpTime(std::string_view date) {
  absl::Time time;
  static constexpr std::array<absl::string_view, 4> rfc7231_date_formats = {
      "%a, %d %b %Y %H:%M:%S GMT", "%a, %d %b %Y %H:%M:%S GMT+00:00",
      "%A, %d-%b-%y %H:%M:%S GMT", "%a %b %e %H:%M:%S %Y"};
  for (auto format : rfc7231_date_formats) {
    if (absl::ParseTime(format, absl::string_view(date.data(), date.size()),
                        &time, nullptr)) {
      return absl::ToChronoTime(time);
    }
  }
  return {};
}

QueryParams parseQueryString(absl::string_view url) {
  size_t start = url.find('?');
  if (start == std::string::npos) {
    QueryParams params;
    return params;
  }

  start++;
  return parseParameters(url, start, /*decode_params=*/false);
}

QueryParams parseAndDecodeQueryString(absl::string_view url) {
  size_t start = url.find('?');
  if (start == std::string::npos) {
    QueryParams params;
    return params;
  }

  start++;
  return parseParameters(url, start, /*decode_params=*/true);
}

QueryParams parseFromBody(absl::string_view body) {
  return parseParameters(body, 0, /*decode_params=*/true);
}

inline std::string subspan(absl::string_view source, size_t start, size_t end) {
  return {source.data() + start, end - start};
}

QueryParams parseParameters(absl::string_view data, size_t start,
                            bool decode_params) {
  QueryParams params;

  while (start < data.size()) {
    size_t end = data.find('&', start);
    if (end == std::string::npos) {
      end = data.size();
    }
    absl::string_view param(data.data() + start, end - start);

    const size_t equal = param.find('=');
    if (equal != std::string::npos) {
      const auto param_name = subspan(data, start, start + equal);
      const auto param_value = subspan(data, start + equal + 1, end);
      params.emplace(
          decode_params ? PercentEncoding::decode(param_name) : param_name,
          decode_params ? PercentEncoding::decode(param_value) : param_value);
    } else {
      params.emplace(subspan(data, start, end), "");
    }

    start = end + 1;
  }

  return params;
}

std::vector<std::string> getAllOfHeader(std::string_view key) {
  std::vector<std::string> result;
  auto headers = getRequestHeaderPairs()->pairs();
  for (auto& header : headers) {
    if (absl::EqualsIgnoreCase(Wasm::Common::stdToAbsl(header.first), Wasm::Common::stdToAbsl(key))) {
      result.push_back(std::string(header.second));
    }
  }
  return result;
}

void forEachCookie(
    std::string_view cookie_header,
    const std::function<bool(std::string_view, std::string_view)>&
        cookie_consumer) {
  auto cookie_headers = getAllOfHeader(cookie_header);

  for (auto& cookie_header_value : cookie_headers) {
    // Split the cookie header into individual cookies.
    for (const auto& s :
         absl::StrSplit(cookie_header_value, ";", absl::SkipEmpty())) {
      // Find the key part of the cookie (i.e. the name of the cookie).
      size_t first_non_space = s.find_first_not_of(' ');
      size_t equals_index = s.find('=');
      if (equals_index == absl::string_view::npos) {
        // The cookie is malformed if it does not have an `=`. Continue
        // checking other cookies in this header.
        continue;
      }
      absl::string_view k =
          s.substr(first_non_space, equals_index - first_non_space);
      absl::string_view v = s.substr(equals_index + 1, s.size() - 1);

      // Cookie values may be wrapped in double quotes.
      // https://tools.ietf.org/html/rfc6265#section-4.1.1
      if (v.size() >= 2 && v.back() == '"' && v[0] == '"') {
        v = v.substr(1, v.size() - 2);
      }

      if (!cookie_consumer(Wasm::Common::abslToStd(k), Wasm::Common::abslToStd(v))) {
        return;
      }
    }
  }
}

std::unordered_map<std::string, std::string> parseCookies(
    const std::function<bool(std::string_view)>& key_filter) {
  std::unordered_map<std::string, std::string> cookies;
  forEachCookie(
      Header::Cookie,
      [&cookies, &key_filter](std::string_view k, std::string_view v) -> bool {
        if (key_filter(k)) {
          cookies.emplace(k, v);
        }

        // continue iterating until all cookies are processed.
        return true;
      });

  return cookies;
}

std::string buildOriginalUri(std::optional<uint32_t> max_path_length) {
  auto path_ptr = getRequestHeader(Header::Path);
  auto path = path_ptr->view();
  if (path.empty()) {
    return "";
  }
  auto envoy_path_ptr = getRequestHeader(Header::EnvoyOriginalPath);
  auto envoy_path = envoy_path_ptr->view();
  std::string_view final_path(envoy_path.empty() ? path : envoy_path);
  if (max_path_length && final_path.length() > max_path_length) {
    final_path = final_path.substr(0, max_path_length.value());
  }
  auto scheme_ptr = getRequestHeader(Header::Scheme);
  auto scheme = scheme_ptr->view();
  auto host_ptr = getRequestHeader(Header::Host);
  auto host = host_ptr->view();
  return absl::StrCat(Wasm::Common::stdToAbsl(scheme), "://", Wasm::Common::stdToAbsl(host),  Wasm::Common::stdToAbsl(final_path));
}

}  // namespace Wasm::Common::Http
