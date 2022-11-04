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
