package global_least_request

import (
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

const (
	RedisKeyFormat = "higress:global_least_request_table:%s"
)

type GlobalLeastRequestLoadBalancer struct {
	redisClient wrapper.RedisClient
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
	clusterName, err := getClusterName()
	if err != nil || clusterName == "" {
		ctx.SetContext("error", true)
		return types.ActionContinue
	} else {
		ctx.SetContext("clusterName", clusterName)
	}
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
	err = lb.redisClient.HGetAll(fmt.Sprintf(RedisKeyFormat, clusterName), func(response resp.Value) {
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
		err := lb.redisClient.HIncrBy(fmt.Sprintf(RedisKeyFormat, clusterName), hostSelected, 1, func(response resp.Value) {
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

func (lb GlobalLeastRequestLoadBalancer) HandleHttpResponseHeaders(ctx wrapper.HttpContext) types.Action {
	return types.ActionContinue
}

func (lb GlobalLeastRequestLoadBalancer) HandleHttpStreamingResponseBody(ctx wrapper.HttpContext, data []byte, endOfStream bool) []byte {
	if endOfStream {
		isErr, _ := ctx.GetContext("error").(bool)
		if !isErr {
			clusterName, _ := ctx.GetContext("clusterName").(string)
			host_selected, _ := ctx.GetContext("host_selected").(string)
			if host_selected == "" {
				log.Errorf("get host_selected failed")
			} else {
				lb.redisClient.HIncrBy(fmt.Sprintf(RedisKeyFormat, clusterName), host_selected, -1, nil)
			}
		}
	}
	return data
}

func (lb GlobalLeastRequestLoadBalancer) HandleHttpResponseBody(ctx wrapper.HttpContext, body []byte) types.Action {
	return types.ActionContinue
}
