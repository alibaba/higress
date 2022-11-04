#include "extensions/key_rate_limit/bucket.h"

#include <string>
#include <unordered_map>

#include "absl/strings/str_cat.h"
#include "absl/strings/str_join.h"

namespace {

const int maxGetTokenRetry = 20;

// Key-prefix for token bucket shared data.
std::string tokenBucketPrefix = "mse.token_bucket";

// Key-prefix for token bucket last updated time.
std::string lastRefilledPrefix = "mse.last_refilled";

}  // namespace

bool getToken(int rule_id, const std::string &key) {
  WasmDataPtr token_bucket_data;
  uint32_t cas;
  std::string tokenBucketKey =
      std::to_string(rule_id) + tokenBucketPrefix + key;
  for (int i = 0; i < maxGetTokenRetry; i++) {
    if (WasmResult::Ok !=
        getSharedData(tokenBucketKey, &token_bucket_data, &cas)) {
      return false;
    }
    uint64_t token_left =
        *reinterpret_cast<const uint64_t *>(token_bucket_data->data());
    if (token_left == 0) {
      return false;
    }
    token_left -= 1;
    auto res = setSharedData(
        tokenBucketKey,
        {reinterpret_cast<const char *>(&token_left), sizeof(token_left)}, cas);
    if (res == WasmResult::Ok) {
      return true;
    }
    if (res == WasmResult::CasMismatch) {
      continue;
    }
    return false;
  }

  LOG_WARN("get token failed with cas mismatch");
  return true;
}

void refillToken(const std::vector<std::pair<int, LimitItem>> &rules) {
  uint32_t last_update_cas;
  WasmDataPtr last_update_data;
  for (const auto &rule : rules) {
    auto id = std::to_string(rule.first);
    std::string lastRefilledKey = id + lastRefilledPrefix + rule.second.key;
    std::string tokenBucketKey = id + tokenBucketPrefix + rule.second.key;
    auto result =
        getSharedData(lastRefilledKey, &last_update_data, &last_update_cas);
    if (result != WasmResult::Ok) {
      LOG_WARN(
          absl::StrCat("failed to get last update time of the local rate limit "
                       "token bucket ",
                       toString(result)));
      continue;
    }
    uint64_t last_update =
        *reinterpret_cast<const uint64_t *>(last_update_data->data());
    uint64_t now = getCurrentTimeNanoseconds();
    if (now - last_update < rule.second.refill_interval_nanosec) {
      continue;
    }
    // Otherwise, try set last updated time. If updated failed because of cas
    // mismatch, the bucket is going to be refilled by other VMs.
    auto res = setSharedData(
        lastRefilledKey, {reinterpret_cast<const char *>(&now), sizeof(now)},
        last_update_cas);
    if (res == WasmResult::CasMismatch) {
      continue;
    }
    do {
      if (WasmResult::Ok !=
          getSharedData(tokenBucketKey, &last_update_data, &last_update_cas)) {
        LOG_WARN("failed to get current local rate limit token bucket");
        break;
      }
      uint64_t token_left =
          *reinterpret_cast<const uint64_t *>(last_update_data->data());
      // Refill tokens, and update bucket with cas. If update failed because of
      // cas mismatch, retry refilling.
      token_left += rule.second.tokens_per_refill;
      if (token_left > rule.second.max_tokens) {
        token_left = rule.second.max_tokens;
      }
      if (WasmResult::CasMismatch ==
          setSharedData(
              tokenBucketKey,
              {reinterpret_cast<const char *>(&token_left), sizeof(token_left)},
              last_update_cas)) {
        continue;
      }
      break;
    } while (true);
  }
}

bool initializeTokenBucket(
    const std::vector<std::pair<int, LimitItem>> &rules) {
  uint32_t last_update_cas;
  WasmDataPtr last_update_data;
  uint64_t initial_value = 0;
  for (const auto &rule : rules) {
    auto id = std::to_string(rule.first);
    std::string lastRefilledKey = id + lastRefilledPrefix + rule.second.key;
    std::string tokenBucketKey = id + tokenBucketPrefix + rule.second.key;
    auto res =
        getSharedData(lastRefilledKey, &last_update_data, &last_update_cas);
    if (res == WasmResult::NotFound) {
      setSharedData(lastRefilledKey,
                    {reinterpret_cast<const char *>(&initial_value),
                     sizeof(initial_value)});
      setSharedData(tokenBucketKey,
                    {reinterpret_cast<const char *>(&rule.second.max_tokens),
                     sizeof(uint64_t)});
      continue;
    }
    // reconfigure
    do {
      if (WasmResult::Ok !=
          getSharedData(lastRefilledKey, &last_update_data, &last_update_cas)) {
        LOG_WARN("failed to get lastRefilled");
        return false;
      }
      if (WasmResult::CasMismatch ==
          setSharedData(lastRefilledKey,
                        {reinterpret_cast<const char *>(&initial_value),
                         sizeof(initial_value)},
                        last_update_cas)) {
        continue;
      }
      break;
    } while (true);
    do {
      if (WasmResult::Ok !=
          getSharedData(tokenBucketKey, &last_update_data, &last_update_cas)) {
        LOG_WARN("failed to get tokenBucket");
        return false;
      }
      if (WasmResult::CasMismatch ==
          setSharedData(
              tokenBucketKey,
              {reinterpret_cast<const char *>(&rule.second.max_tokens),
               sizeof(uint64_t)},
              last_update_cas)) {
        continue;
      }
      break;
    } while (true);
  }
  return true;
}
