-- Mocking Redis environment
local redis_data = {
    hset = {},
    kv = {}
}

local redis = {
    call = function(cmd, ...)
        local args = {...}
        if cmd == "HGET" then
            local key, field = args[1], args[2]
            return redis_data.hset[field]
        elseif cmd == "HSET" then
            local key, field, val = args[1], args[2], args[3]
            redis_data.hset[field] = val
        elseif cmd == "HINCRBY" then
            local key, field, increment = args[1], args[2], args[3]
            local val = tonumber(redis_data.hset[field] or 0)
            redis_data.hset[field] = tostring(val + increment)
            return redis_data.hset[field]
        elseif cmd == "HKEYS" then
            local keys = {}
            for k, _ in pairs(redis_data.hset) do
                table.insert(keys, k)
            end
            return keys
        elseif cmd == "HDEL" then
            local key, field = args[1], args[2]
            redis_data.hset[field] = nil
        elseif cmd == "GET" then
            return redis_data.kv[args[1]]
        elseif cmd == "HMGET" then
            local key = args[1]
            local res = {}
            for i = 2, #args do
                table.insert(res, redis_data.hset[args[i]])
            end
            return res
        elseif cmd == "SET" then
            redis_data.kv[args[1]] = args[2]
        end
    end
}

-- The actual logic from lb_policy.go
local function run_lb_logic(KEYS)
    local seed = tonumber(KEYS[1])
    local hset_key = KEYS[2]
    local last_clean_key = KEYS[3]
    local clean_interval = tonumber(KEYS[4])
    local current_target = KEYS[5]
    local healthy_count = tonumber(KEYS[6])
    local enable_detail_log = KEYS[7]

    math.randomseed(seed)

    -- 1. Selection
    local current_count = 0
    local same_count_hits = 0

    for i = 8, 8 + healthy_count - 1 do
        local host = KEYS[i]
        local count = 0
        local val = redis.call('HGET', hset_key, host)
        if val then
            count = tonumber(val) or 0
        end
        
        if same_count_hits == 0 or count < current_count then
            current_target = host
            current_count = count
            same_count_hits = 1
        elseif count == current_count then
            same_count_hits = same_count_hits + 1
            if math.random(same_count_hits) == 1 then
                current_target = host
            end
        end
    end

    redis.call("HINCRBY", hset_key, current_target, 1)
    local new_count = redis.call("HGET", hset_key, current_target)

    -- Collect host counts for logging
    local host_details = {}
    if enable_detail_log == "1" then
        local fields = {}
        for i = 8, #KEYS do
            table.insert(fields, KEYS[i])
        end
        if #fields > 0 then
            local values = redis.call('HMGET', hset_key, (table.unpack or unpack)(fields))
            for i, val in ipairs(values) do
                table.insert(host_details, fields[i])
                table.insert(host_details, tostring(val or 0))
            end
        end
    end

    -- 2. Cleanup
    local current_time = math.floor(seed / 1000000)
    local last_clean_time = tonumber(redis.call('GET', last_clean_key) or 0)

    if current_time - last_clean_time >= clean_interval then
        local all_keys = redis.call('HKEYS', hset_key)
        if #all_keys > 0 then
            -- Create a lookup table for current hosts (from index 8 onwards)
            local current_hosts = {}
            for i = 8, #KEYS do
                current_hosts[KEYS[i]] = true
            end
            -- Remove keys not in current hosts
            for _, host in ipairs(all_keys) do
                if not current_hosts[host] then
                    redis.call('HDEL', hset_key, host)
                end
            end
        end
        redis.call('SET', last_clean_key, current_time)
    end

    return {current_target, new_count, host_details}
end

-- --- Test 1: Load Balancing Distribution ---
print("--- Test 1: Load Balancing Distribution ---")
local hosts = {"host1", "host2", "host3", "host4", "host5"}
local iterations = 100000
local results = {}
for _, h in ipairs(hosts) do results[h] = 0 end

-- Reset redis
redis_data.hset = {}
for _, h in ipairs(hosts) do redis_data.hset[h] = "0" end

print(string.format("Running %d iterations with %d hosts (all counts started at 0)...", iterations, #hosts))

for i = 1, iterations do
    local initial_host = hosts[math.random(#hosts)]
    -- KEYS structure: [seed, hset_key, last_clean_key, clean_interval, host_selected, healthy_count, enable_detail_log, ...healthy_hosts]
    local keys = {i * 1000000, "table_key", "clean_key", 3600, initial_host, #hosts, "1"}
    for _, h in ipairs(hosts) do table.insert(keys, h) end
    
    local res = run_lb_logic(keys)
    local selected = res[1]
    results[selected] = results[selected] + 1
end

for _, h in ipairs(hosts) do
    local percentage = (results[h] / iterations) * 100
    print(string.format("%s: %6d (%.2f%%)", h, results[h], percentage))
end

-- --- Test 2: IP Cleanup Logic ---
print("\n--- Test 2: IP Cleanup Logic ---")

local function test_cleanup()
    redis_data.hset = {
        ["host1"] = "10",
        ["host2"] = "5",
        ["old_ip_1"] = "1",
        ["old_ip_2"] = "1",
    }
    redis_data.kv["clean_key"] = "1000" -- Last cleaned at 1000s
    
    local current_hosts = {"host1", "host2"}
    local current_time_ms = 1000 * 1000000 + 500 * 1000000 -- 1500s (interval is 300s, let's say)
    local clean_interval = 300
    
    print("Initial Redis IPs:", table.concat((function() local res={} for k,_ in pairs(redis_data.hset) do table.insert(res, k) end return res end)(), ", "))
    
    -- Run logic (seed is microtime)
    local keys = {current_time_ms, "table_key", "clean_key", clean_interval, "host1", #current_hosts, "1"}
    for _, h in ipairs(current_hosts) do table.insert(keys, h) end
    
    run_lb_logic(keys)
    
    print("After Cleanup Redis IPs:", table.concat((function() local res={} for k,_ in pairs(redis_data.hset) do table.insert(res, k) end table.sort(res) return res end)(), ", "))
    
    local exists_old1 = redis_data.hset["old_ip_1"] ~= nil
    local exists_old2 = redis_data.hset["old_ip_2"] ~= nil
    
    if not exists_old1 and not exists_old2 then
        print("Success: Outdated IPs removed.")
    else
        print("Failure: Outdated IPs still exist.")
    end
    
    print("New last_clean_time:", redis_data.kv["clean_key"])
end

test_cleanup()

-- --- Test 3: No Cleanup if Interval Not Reached ---
print("\n--- Test 3: No Cleanup if Interval Not Reached ---")

local function test_no_cleanup()
    redis_data.hset = {
        ["host1"] = "10",
        ["old_ip_1"] = "1",
    }
    redis_data.kv["clean_key"] = "1000"
    
    local current_hosts = {"host1"}
    local current_time_ms = 1000 * 1000000 + 100 * 1000000 -- 1100s (interval 300s, not reached)
    local clean_interval = 300
    
    local keys = {current_time_ms, "table_key", "clean_key", clean_interval, "host1", #current_hosts, "0"}
    for _, h in ipairs(current_hosts) do table.insert(keys, h) end
    
    run_lb_logic(keys)
    
    if redis_data.hset["old_ip_1"] then
        print("Success: Cleanup not triggered as expected.")
    else
        print("Failure: Cleanup triggered unexpectedly.")
    end
end

test_no_cleanup()
