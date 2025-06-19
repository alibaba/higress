package prefix_cache

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"math/rand"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

const (
	RedisLua = `-- 将十六进制字符串转换为字节数组
local function hex_to_bytes(hex)
    local bytes = {}
    for i = 1, #hex, 2 do
        local byte_str = hex:sub(i, i+1)
        local byte_val = tonumber(byte_str, 16)
        table.insert(bytes, byte_val)
    end
    return bytes
end

-- 字节转回十六进制字符串
local function bytes_to_hex(bytes)
    local result = ""
    for _, byte in ipairs(bytes) do
        result = result .. string.format("%02X", byte)
    end
    return result
end

-- byte XOR
local function byte_xor(a, b)
    local result = 0
    for i = 0, 7 do
        local bit_val = 2^i
        if ((a % (bit_val * 2)) >= bit_val) ~= ((b % (bit_val * 2)) >= bit_val) then
            result = result + bit_val
        end
    end
    return result
end

-- hex string XOR
local function hex_xor(a, b)
    if #a ~= #b then
        error("Hex strings must be of equal length, first is " .. a .. " second is " .. b)
    end

    local a_bytes = hex_to_bytes(a)
    local b_bytes = hex_to_bytes(b)

    local result_bytes = {}
    for i = 1, #a_bytes do
        table.insert(result_bytes, byte_xor(a_bytes[i], b_bytes[i]))
    end

    return bytes_to_hex(result_bytes)
end

local target = ""
local key = ""
local current_key = ""
local count = #ARGV
local ttl = KEYS[1]
local default_target = KEYS[2]

if count == 0 then
    return target
end

-- find longest prefix
local index = 1
while index <= count do
    if current_key == "" then
        current_key = ARGV[index]
    else
        current_key = hex_xor(current_key, ARGV[index])
    end
    if redis.call("EXISTS", current_key) == 1 then
        key = current_key
        target = redis.call("GET", key)
        -- update ttl for exist keys
        redis.call("EXPIRE", key, ttl)
        index = index + 1
    else
        break
    end
end

-- default_target should be passed outside
if target == "" then
    target = default_target
end

-- add tree-path
while index <= count do
    if key == "" then
        key = ARGV[index]
    else
        key = hex_xor(key, ARGV[index])
    end
    redis.call("SET", key, target)
    redis.call("EXPIRE", key, ttl)
    index = index + 1
end

return target`
)

type PrefixCacheLoadBalancer struct {
	redisClient wrapper.RedisClient
	redisKeyTTL int
}

func NewPrefixCacheLoadBalancer(json gjson.Result) (PrefixCacheLoadBalancer, error) {
	lb := PrefixCacheLoadBalancer{}
	serviceFQDN := json.Get("serviceFQDN").String()
	servicePort := json.Get("servicePort").Int()
	if serviceFQDN == "" || servicePort == 0 {
		log.Errorf("invalid redis service, serviceFQDN: %s, servicePort: %d", serviceFQDN, servicePort)
		return lb, errors.New("invalid redis service config")
	}
	lb.redisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: serviceFQDN,
		Port: servicePort,
	})
	username := json.Get("username").String()
	password := json.Get("password").String()
	timeout := json.Get("timeout").Int()
	if timeout == 0 {
		timeout = 3000
	}
	// database default is 0
	database := json.Get("database").Int()
	if json.Get("redisKeyTTL").Int() == 0 {
		lb.redisKeyTTL = int(json.Get("redisKeyTTL").Int())
	} else {
		lb.redisKeyTTL = 1800
	}
	return lb, lb.redisClient.Init(username, password, int64(timeout), wrapper.WithDataBase(int(database)))
}

func (lb PrefixCacheLoadBalancer) HandleHttpRequestHeaders(ctx wrapper.HttpContext) types.Action {
	// If return types.ActionContinue, SetUpstreamOverrideHost will not take effect
	return types.HeaderStopIteration
}

func (lb PrefixCacheLoadBalancer) HandleHttpRequestBody(ctx wrapper.HttpContext, body []byte) types.Action {
	var err error
	hostInfos, err := proxywasm.GetUpstreamHosts()
	if err != nil {
		log.Error("get upstream cluster endpoints failed")
		return types.ActionContinue
	}
	healthyHosts := []string{}
	for _, hostInfo := range hostInfos {
		if gjson.Get(hostInfo[1], "health_status").String() == "Healthy" {
			healthyHosts = append(healthyHosts, hostInfo[0])
		}
	}
	if len(healthyHosts) == 0 {
		log.Info("upstream cluster has no healthy endpoints")
		return types.ActionContinue
	}
	defaultHost := healthyHosts[rand.Intn(len(healthyHosts))]
	params := []interface{}{}
	rawStr := ""
	messages := gjson.GetBytes(body, "messages").Array()
	for index, obj := range messages {
		if !obj.Get("role").Exists() || !obj.Get("content").Exists() {
			log.Info("cannot extract role or content from request body, skip llm load balancing")
			return types.ActionContinue
		}
		role := obj.Get("role").String()
		content := obj.Get("content").String()
		rawStr += role + ":" + content
		if role == "user" || index == len(messages)-1 {
			sha1Str := computeSHA1(rawStr)
			params = append(params, sha1Str)
			rawStr = ""
		}
	}
	if len(params) == 0 {
		return types.ActionContinue
	}
	err = lb.redisClient.Eval(RedisLua, 2, []interface{}{lb.redisKeyTTL, defaultHost}, params, func(response resp.Value) {
		if err := response.Error(); err != nil {
			log.Errorf("Redis eval failed: %+v", err)
			proxywasm.ResumeHttpRequest()
			return
		}
		hostSelected := response.String()
		if err := proxywasm.SetUpstreamOverrideHost([]byte(hostSelected)); err != nil {
			log.Errorf("override upstream host failed, fallback to default lb policy, error informations: %+v", err)
		}
		proxywasm.ResumeHttpRequest()
	})
	if err != nil {
		return types.ActionContinue
	}
	return types.ActionPause
}

func (lb PrefixCacheLoadBalancer) HandleHttpResponseHeaders(ctx wrapper.HttpContext) types.Action {
	ctx.DontReadResponseBody()
	return types.ActionContinue
}

func (lb PrefixCacheLoadBalancer) HandleHttpStreamingResponseBody(ctx wrapper.HttpContext, data []byte, endOfStream bool) []byte {
	return data
}

func (lb PrefixCacheLoadBalancer) HandleHttpResponseBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}

func computeSHA1(data string) string {
	hasher := sha1.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}
