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

#include <cstdint>
#include <string>
#include <vector>

#include "openssl/hmac.h"
#include "openssl/md5.h"
#include "openssl/sha.h"

#define ASSERT(_X) assert(_X)

namespace Wasm::Common::Crypto {

std::vector<uint8_t> getShaHmac(std::string_view hash_type,
                                std::string_view key, std::string_view message);

std::string getShaHmacBase64(std::string_view hash_type, std::string_view key,
                             std::string_view message);

std::vector<uint8_t> getMD5(std::string_view message);

std::string getMD5Base64(std::string_view message);

bool crypt(const std::string& key, const std::string& salt,
           std::string& encrypted);

}  // namespace Wasm::Common::Crypto
