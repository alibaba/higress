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

// modified base on envoy/source/extensions/filters/http/jwt_authn/extractor.h
#pragma once
#include <map>
#include <memory>
#include <string>
#include <vector>

#ifndef NULL_PLUGIN

#include "proxy_wasm_intrinsics.h"
#else

#include "include/proxy-wasm/null_plugin.h"

namespace proxy_wasm {
namespace null_plugin {
namespace jwt_auth {

#endif

#define PURE = 0

/**
 * JwtLocation stores following token information:
 *
 * * extracted token string,
 * * the location where the JWT is extracted from,
 * * list of issuers specified the location.
 *
 */
class JwtLocation {
 public:
  virtual ~JwtLocation() = default;

  // Get the token string
  virtual const std::string& token() const PURE;

  // Check if claim has specified the location.
  virtual bool isClaimAllowed(const std::string& key,
                              const std::string& value) const PURE;

  // Remove the token from the headers
  virtual void removeJwt() const PURE;

  // Store the claim to header
  virtual void addClaimToHeader(const std::string& header,
                                const std::string& value,
                                bool override) const PURE;

  // Set claim to request header
  virtual void claimsToHeaders() const PURE;
};

using JwtLocationConstPtr = std::unique_ptr<const JwtLocation>;

class Extractor;
using ExtractorConstPtr = std::unique_ptr<const Extractor>;

struct Consumer;
/**
 * Extracts JWT from locations specified in the config.
 *
 * Usage example:
 *
 *  auto extractor = Extractor::create(config);
 *  auto tokens = extractor->extract(headers);
 *  for (token : tokens) {
 *     Jwt jwt;
 *     if (jwt.parseFromString(token->token()) != Status::Ok) {
 *       // Handle JWT parsing failure.
 *     }
 *
 *     if (need_to_remove) {
 *        // remove the JWT
 *        token->removeJwt(headers);
 *     }
 *  }
 *
 */
class Extractor {
 public:
  virtual ~Extractor() = default;

  /**
   * Extract all JWT tokens from the headers. If set of header_keys or
   * param_keys is not empty only those in the matching locations will be
   * returned.
   *
   * @param headers is the HTTP request headers.
   * @return list of extracted Jwt location info.
   */
  virtual std::vector<JwtLocationConstPtr> extract() const PURE;

  /**
   * Create an instance of Extractor for a given config.
   * @param from_headers header location config.
   * @param from_params query param location config.
   * @return the extractor object.
   */
  static ExtractorConstPtr create(const Consumer& provider);
};

#ifdef NULL_PLUGIN

}  // namespace jwt_auth
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
