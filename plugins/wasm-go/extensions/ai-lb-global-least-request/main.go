package main

import (
	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"global-least-request",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
	)
}

const (
	RedisKey = "higress:global_least_request_table"
)

type GlobalLeastRequestLBConfig struct {
	redisClient wrapper.RedisClient
}

func parseConfig(json gjson.Result, config *GlobalLeastRequestLBConfig) error {
	serviceFQDN := json.Get("serviceFQDN").String()
	servicePort := json.Get("servicePort").Int()
	if serviceFQDN == "" || servicePort == 0 {
		log.Errorf("invalid redis service, serviceFQDN: %s, servicePort: %d", serviceFQDN, servicePort)
	}
	config.redisClient = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
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
	return config.redisClient.Init(username, password, int64(timeout), wrapper.WithDataBase(int(database)))
}

// Callbacks which are called in request path
func onHttpRequestHeaders(ctx wrapper.HttpContext, config GlobalLeastRequestLBConfig) types.Action {
	// If return types.ActionContinue, SetUpstreamOverrideHost will failed
	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, config GlobalLeastRequestLBConfig, body []byte) types.Action {
	hostInfos, err := proxywasm.GetUpstreamHosts()
	// log.Infof("%+v", hostInfos)
	if err != nil {
		return types.ActionContinue
	}
	// Only healthy host can be selected
	hostRqCount := make(map[string]int)
	for _, hostInfo := range hostInfos {
		if gjson.Get(hostInfo[1], "health_status").String() == "Healthy" {
			hostRqCount[hostInfo[0]] = 0
		}
	}
	// log.Infof("hostRqCount initial: %+v", hostRqCount)
	err = config.redisClient.HGetAll(RedisKey, func(response resp.Value) {
		// log.Infof("HGetAll response: %+v", response)
		if err := response.Error(); err != nil {
			log.Errorf("HGetAll failed: %+v", err)
			proxywasm.ResumeHttpRequest()
			return
		}
		// update ongoing request number for each healthy host
		// redis response format is [addr_1 count_1 addr_2 count_2 ...]
		index := 0
		arr := response.Array()
		for index < len(arr)-1 {
			host := arr[index].String()
			count := arr[index+1].Integer()
			hostRqCount[host] = count
			index += 2
		}
		// log.Infof("hostRqCount final: %+v", hostRqCount)
		hostSelected := ""
		for h, c := range hostRqCount {
			if hostSelected == "" {
				hostSelected = h
			} else if c < hostRqCount[hostSelected] {
				hostSelected = h
			}
		}
		log.Debugf("host_selected: %s", hostSelected)
		ctx.SetContext("host_selected", hostSelected)
		if err := proxywasm.SetUpstreamOverrideHost([]byte(hostSelected)); err != nil {
			log.Errorf("override upstream host failed, fallback to default lb policy, error informations: %+v", err)
		}
		err := config.redisClient.HIncrBy(RedisKey, hostSelected, 1, func(response resp.Value) {
			if err := response.Error(); err != nil {
				log.Errorf("HIncrBy failed on request phase: %+v", err)
			}
			proxywasm.ResumeHttpRequest()
		})
		if err != nil {
			proxywasm.ResumeHttpRequest()
		}
	})
	if err != nil {
		return types.ActionContinue
	}
	return types.ActionPause
}

func onHttpStreamingResponseBody(ctx wrapper.HttpContext, config GlobalLeastRequestLBConfig, data []byte, endOfStream bool) []byte {
	if endOfStream {
		host_selected, _ := ctx.GetContext("host_selected").(string)
		if host_selected == "" {
			log.Errorf("get host_selected failed")
		} else {
			config.redisClient.HIncrBy(RedisKey, host_selected, -1, nil)
		}
	}
	return data
}
