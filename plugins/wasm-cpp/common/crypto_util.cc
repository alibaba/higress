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

#include "crypto_util.h"

#include <crypt.h>
#include <openssl/sha.h>

#include <array>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <string_view>

#include "absl/strings/ascii.h"
#include "absl/strings/match.h"
#include "absl/strings/str_cat.h"
#include "base64.h"

extern "C" {
char* __crypt_blowfish(const char* key, const char* setting, char* output);
}

namespace Wasm::Common::Crypto {

namespace {
size_t getDigestLength(std::string_view name) {
  if (name == "sha1") {
    return SHA_DIGEST_LENGTH;
  }
  if (name == "sha224") {
    return SHA224_DIGEST_LENGTH;
  }
  if (name == "sha256") {
    return SHA256_DIGEST_LENGTH;
  }
  if (name == "sha384") {
    return SHA384_DIGEST_LENGTH;
  }
  if (name == "sha512") {
    return SHA512_DIGEST_LENGTH;
  }
  return 0;
}
const EVP_MD* getHashFunction(std::string_view name) {
  // Hash algorithms set refers
  // https://github.com/google/boringssl/blob/master/include/openssl/digest.h
  if (name == "sha1") {
    return EVP_sha1();
  }
  if (name == "sha224") {
    return EVP_sha224();
  }
  if (name == "sha256") {
    return EVP_sha256();
  }
  if (name == "sha384") {
    return EVP_sha384();
  }
  if (name == "sha512") {
    return EVP_sha512();
  }
  return nullptr;
}
}  // namespace

std::vector<uint8_t> getShaHmac(std::string_view hash_type,
                                std::string_view key,
                                std::string_view message) {
  auto length = getDigestLength(hash_type);
  if (length == 0) {
    return {};
  }
  const auto* hashFunc = getHashFunction(hash_type);
  if (hashFunc == nullptr) {
    return {};
  }
  std::vector<uint8_t> hmac(length);
  HMAC(hashFunc, key.data(), key.size(),
       reinterpret_cast<const uint8_t*>(message.data()), message.size(),
       hmac.data(), nullptr);
  return hmac;
}

std::string getShaHmacBase64(std::string_view hash_type, std::string_view key,
                             std::string_view message) {
  auto hmac = getShaHmac(hash_type, key, message);
  return Base64::encode(reinterpret_cast<const char*>(hmac.data()),
                        hmac.size());
}

std::vector<uint8_t> getMD5(std::string_view message) {
  std::vector<uint8_t> md5(MD5_DIGEST_LENGTH);
  MD5(reinterpret_cast<const uint8_t*>(message.data()), message.size(),
      md5.data());
  return md5;
}

std::string getMD5Base64(std::string_view message) {
  auto md5 = getMD5(message);
  return Base64::encode(reinterpret_cast<const char*>(md5.data()), md5.size());
}

bool libc_crypt(const std::string& key, const std::string& salt,
                std::string& encrypted) {
  char* value;
  struct crypt_data cd;
  cd.initialized = 0;
  value = crypt_r(key.data(), salt.data(), &cd);
  if (value != nullptr) {
    encrypted = value;
    return true;
  }
  return false;
}

void crypt_to64(std::string& encrypted, uint32_t v, size_t n) {
  static u_char itoa64[] =
      "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
  while (n-- > 0) {
    encrypted.push_back(itoa64[v & 0x3f]);
    v >>= 6;
  }
}

bool crypt_apr1(const std::string& key, const std::string& salt,
                std::string& encrypted) {
  const char* salt_data;
  const char* last;
  std::array<u_char, 16> final_data;
  salt_data = salt.data();
  salt_data += sizeof("$apr1$") - 1;
  // true salt: no magic, max 8 chars, stop at first $
  last = salt_data + 8;
  const char* p;
  for (p = salt_data; *p != 0 && *p != '$' && p < last; p++) {
    /* void */
  }
  size_t salt_len = p - salt_data;
  MD5_CTX md5;
  MD5_Init(&md5);
  MD5_Update(&md5, key.data(), key.size());
  MD5_Update(&md5, "$apr1$", sizeof("$apr1$") - 1);
  MD5_Update(&md5, salt_data, salt_len);

  MD5_CTX ctx1;
  MD5_Init(&ctx1);
  MD5_Update(&ctx1, key.data(), key.size());
  MD5_Update(&ctx1, salt_data, salt_len);
  MD5_Update(&ctx1, key.data(), key.size());
  MD5_Final(final_data.data(), &ctx1);

  for (int i = key.size(); i > 0; i -= 16) {
    MD5_Update(&md5, final_data.data(), i > 16 ? 16 : i);
  }
  final_data.fill(0);
  for (auto i = key.size(); i != 0; i >>= 1) {
    if ((i & 1) != 0) {
      MD5_Update(&md5, final_data.data(), 1);
    } else {
      MD5_Update(&md5, key.data(), 1);
    }
  }
  MD5_Final(final_data.data(), &md5);
  for (auto i = 0; i < 1000; i++) {
    MD5_Init(&ctx1);
    if ((i & 1) != 0) {
      MD5_Update(&ctx1, key.data(), key.size());
    } else {
      MD5_Update(&ctx1, final_data.data(), 16);
    }
    if (i % 3 != 0) {
      MD5_Update(&ctx1, salt_data, salt_len);
    }
    if (i % 7 != 0) {
      MD5_Update(&ctx1, key.data(), key.size());
    }
    if ((i & 1) != 0) {
      MD5_Update(&ctx1, final_data.data(), 16);
    } else {
      MD5_Update(&ctx1, key.data(), key.size());
    }
    MD5_Final(final_data.data(), &ctx1);
  }
  encrypted =
      absl::StrCat("$apr1$", absl::string_view{salt_data, salt_len}, "$");
  crypt_to64(encrypted,
             (final_data[0] << 16) | (final_data[6] << 8) | final_data[12], 4);
  crypt_to64(encrypted,
             (final_data[1] << 16) | (final_data[7] << 8) | final_data[13], 4);
  crypt_to64(encrypted,
             (final_data[2] << 16) | (final_data[8] << 8) | final_data[14], 4);
  crypt_to64(encrypted,
             (final_data[3] << 16) | (final_data[9] << 8) | final_data[15], 4);
  crypt_to64(encrypted,
             (final_data[4] << 16) | (final_data[10] << 8) | final_data[5], 4);
  crypt_to64(encrypted, final_data[11], 2);
  return true;
}

bool crypt_plain(const std::string& key, const std::string& salt,
                 std::string& encrypted) {
  encrypted = absl::StrCat("{PLAIN}", key);
  return true;
}

bool crypt_ssha(const std::string& key, const std::string& salt,
                std::string& encrypted) {
  auto decoded = Base64::decodeWithoutPadding(
      {salt.data() + sizeof("{SSHA}") - 1, salt.size() - sizeof("{SSHA}") + 1});
  if (decoded.empty()) {
    return false;
  }
  if (decoded.size() < 20) {
    decoded.resize(20);
  }
  SHA_CTX sha1;
  SHA1_Init(&sha1);
  SHA1_Update(&sha1, key.data(), key.size());
  SHA1_Update(&sha1, decoded.data() + 20, decoded.size() - 20);
  SHA1_Final((u_char*)decoded.data(), &sha1);

  encrypted =
      absl::StrCat("{SSHA}", Base64::encode(decoded.data(), decoded.size()));
  return true;
}

bool crypt_sha(const std::string& key, const std::string& salt,
               std::string& encrypted) {
  SHA_CTX sha1;
  std::array<u_char, 20> digest;
  SHA1_Init(&sha1);
  SHA1_Update(&sha1, key.data(), key.size());
  SHA1_Final(digest.data(), &sha1);
  encrypted = absl::StrCat("{SHA}",
                           Base64::encode((char*)digest.data(), digest.size()));
  return true;
}

bool bcrypt(const std::string& key, const std::string& salt,
            std::string& encrypted) {
  struct crypt_data cd;
  cd.initialized = 0;
  char* value = __crypt_blowfish(key.data(), salt.data(), (char*)&cd);
  if (value != nullptr) {
    encrypted = value;
    return true;
  }
  return false;
}

bool crypt(const std::string& key, const std::string& salt,
           std::string& encrypted) {
  if (absl::StartsWith(salt, "$apr1$")) {
    return crypt_apr1(key, salt, encrypted);
  }
  if (absl::StartsWith(salt, "{SHA}")) {
    return crypt_sha(key, salt, encrypted);
  }
  if (absl::StartsWith(salt, "{SSHA}")) {
    return crypt_ssha(key, salt, encrypted);
  }
  if (absl::StartsWith(salt, "{PLAIN}")) {
    return crypt_plain(key, salt, encrypted);
  }
  if (salt.size() > 3 && salt[1] == '2' && salt[3] == '$') {
    return bcrypt(key, salt, encrypted);
  }
  // fallback to libc crypt()
  return libc_crypt(key, salt, encrypted);
}
}  // namespace Wasm::Common::Crypto
