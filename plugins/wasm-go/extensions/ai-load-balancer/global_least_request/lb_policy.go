package global_least_request

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

const (
	RedisKeyFormat = "higress:global_least_request_table:%s:%s"
	RedisLua       = `local hset_key = KEYS[1]
local current_target = KEYS[2]
local current_count = 0

local function is_healthy(addr)
    for i = 3, #KEYS do
        if addr == KEYS[i] then
            return true
        end
    end
    return false
end

if redis.call('HEXISTS', hset_key, current_target) ~= 0 then
    current_count = redis.call('HGET', hset_key, current_target)
    local hash = redis.call('HGETALL', hset_key)
    for i = 1, #hash, 2 do
        local addr = hash[i]
        local count = hash[i+1]
        if count < current_count and is_healthy(addr) then
            current_target = addr
            current_count = count
        end
    end
end

redis.call("HINCRBY", hset_key, current_target, 1)

return current_target`
)

type GlobalLeastRequestLoadBalancer struct {
	redisClient wrapper.RedisClient
}

func getRouteName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		return "", err
	} else {
		return string(raw), nil
	}
}

func getClusterName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err != nil {
		return "", err
	} else {
		return string(raw), nil
	}
}

func NewGlobalLeastRequestLoadBalancer(json gjson.Result) (GlobalLeastRequestLoadBalancer, error) {
	lb := GlobalLeastRequestLoadBalancer{}
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
	return lb, lb.redisClient.Init(username, password, int64(timeout), wrapper.WithDataBase(int(database)))
}

func (lb GlobalLeastRequestLoadBalancer) HandleHttpRequestHeaders(ctx wrapper.HttpContext) types.Action {
	// If return types.ActionContinue, SetUpstreamOverrideHost will not take effect
	return types.HeaderStopIteration
}

func (lb GlobalLeastRequestLoadBalancer) HandleHttpRequestBody(ctx wrapper.HttpContext, body []byte) types.Action {
	routeName, err := getRouteName()
	if err != nil || routeName == "" {
		ctx.SetContext("error", true)
		return types.ActionContinue
	} else {
		ctx.SetContext("routeName", routeName)
	}
	clusterName, err := getClusterName()
	if err != nil || clusterName == "" {
		ctx.SetContext("error", true)
		return types.ActionContinue
	} else {
		ctx.SetContext("clusterName", clusterName)
	}
	hostInfos, err := proxywasm.GetUpstreamHosts()
	if err != nil {
		ctx.SetContext("error", true)
		return types.ActionContinue
	}
	// Only healthy host can be selected
	healthyHostMap := make(map[string]struct{})
	healthyHostArray := []string{}
	for _, hostInfo := range hostInfos {
		if gjson.Get(hostInfo[1], "health_status").String() == "Healthy" {
			healthyHostMap[hostInfo[0]] = struct{}{}
			healthyHostArray = append(healthyHostArray, hostInfo[0])
		}
	}
	if len(healthyHostArray) == 0 {
		ctx.SetContext("error", true)
		return types.ActionContinue
	}
	randomIndex := rand.Intn(len(healthyHostArray))
	hostSelected := healthyHostArray[randomIndex]
	keys := []interface{}{fmt.Sprintf(RedisKeyFormat, routeName, clusterName), hostSelected}
	for _, v := range healthyHostArray {
		keys = append(keys, v)
	}
	err = lb.redisClient.Eval(RedisLua, len(keys), keys, []interface{}{}, func(response resp.Value) {
		if err := response.Error(); err != nil {
			log.Errorf("HGetAll failed: %+v", err)
			proxywasm.ResumeHttpRequest()
			return
		}
		hostSelected = response.String()
		log.Debugf("host_selected: %s", hostSelected)
		ctx.SetContext("host_selected", hostSelected)
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

func (lb GlobalLeastRequestLoadBalancer) HandleHttpResponseHeaders(ctx wrapper.HttpContext) types.Action {
	return types.ActionContinue
}

func (lb GlobalLeastRequestLoadBalancer) HandleHttpStreamingResponseBody(ctx wrapper.HttpContext, data []byte, endOfStream bool) []byte {
	if endOfStream {
		isErr, _ := ctx.GetContext("error").(bool)
		if !isErr {
			routeName, _ := ctx.GetContext("routeName").(string)
			clusterName, _ := ctx.GetContext("clusterName").(string)
			host_selected, _ := ctx.GetContext("host_selected").(string)
			if host_selected == "" {
				log.Errorf("get host_selected failed")
			} else {
				lb.redisClient.HIncrBy(fmt.Sprintf(RedisKeyFormat, routeName, clusterName), host_selected, -1, nil)
			}
		}
	}
	return data
}

func (lb GlobalLeastRequestLoadBalancer) HandleHttpResponseBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}
