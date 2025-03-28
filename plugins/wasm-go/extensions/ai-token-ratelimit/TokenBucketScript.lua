redis.replicate_commands()

local tokens_key = KEYS[1]
local timestamp_key = tokens_key .. ".timestamp"

local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = redis.call('TIME')[1]
local requested = 0
if #ARGV == 3 then
    requested = tonumber(ARGV[3])
end

local last_tokens = tonumber(redis.call("get", tokens_key))
if last_tokens == nil then
    last_tokens = capacity
end

local last_refreshed = tonumber(redis.call("get", timestamp_key))
if last_refreshed == nil then
    last_refreshed = 0
end

local delta = math.max(0, now-last_refreshed)
local filled_tokens = math.min(capacity, last_tokens+(delta*rate))
local new_tokens = filled_tokens - requested

local ttl
if new_tokens < 0 then
    ttl = math.max(1, math.floor((-new_tokens*2)/rate))
else
    local fill_time = capacity/rate
    ttl = math.max(1, math.floor(fill_time*2))
end
redis.call("setex", tokens_key, ttl, new_tokens)
redis.call("setex", timestamp_key, ttl, now)

if new_tokens >= 0 then
    return {tokens_key, new_tokens, 0}
else
    return {tokens_key, new_tokens, (-new_tokens)/rate}
end
