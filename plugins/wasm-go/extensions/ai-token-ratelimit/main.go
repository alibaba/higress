package main

import (
	"errors"
	"strconv"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

func main() {
	wrapper.SetCtx(
		"ai-token-manage",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamingBody),
	)
}

type TokenManageConfig struct {
	client  wrapper.RedisClient
	tpm     int
	metrics map[string]proxywasm.MetricCounter
}

func (config *TokenManageConfig) incrementCounter(metricName string, inc uint64) {
	counter, ok := config.metrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.metrics[metricName] = counter
	}
	counter.Increment(inc)
}

const (
	redisKey = "higress-token-ratelimit"
)

func parseConfig(json gjson.Result, config *TokenManageConfig, log wrapper.Log) error {
	config.metrics = make(map[string]proxywasm.MetricCounter)
	config.tpm = int(json.Get("tpm").Int())

	serviceSource := json.Get("serviceSource").String()
	serviceName := json.Get("serviceName").String()
	servicePort := json.Get("servicePort").Int()
	username := json.Get("username").String()
	password := json.Get("password").String()
	timeout := json.Get("timeout").Int()
	if serviceName == "" || servicePort == 0 {
		return errors.New("invalid service config")
	}
	switch serviceSource {
	case "ip":
		config.client = wrapper.NewRedisClusterClient(wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Port:        servicePort,
		})
	case "dns":
		domain := json.Get("domain").String()
		config.client = wrapper.NewRedisClusterClient(wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      domain,
		})
	default:
		return errors.New("unknown service source: " + serviceSource)
	}
	return config.client.Init(username, password, timeout)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config TokenManageConfig, log wrapper.Log) types.Action {
	config.incrementCounter("ai_token_ratelimit_request_total", 1)
	err := config.client.Get(redisKey, func(response resp.Value) {
		if response.Error() != nil {
			log.Errorf("redisCall Get error: %v", response.Error())
			proxywasm.ResumeHttpRequest()
		} else {
			if !response.IsNull() && response.Integer() > config.tpm {
				config.incrementCounter("ai_token_ratelimit_request_deny", 1)
				proxywasm.SendHttpResponse(429, nil, []byte("Too many requests\n"), -1)
			} else {
				proxywasm.ResumeHttpRequest()
			}
		}
	})
	if err != nil {
		log.Errorf("Error occured while calling Get.")
		return types.ActionContinue
	} else {
		return types.ActionPause
	}
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config TokenManageConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	if !endOfStream {
		return data
	}
	inputTokenStr, err := proxywasm.GetProperty([]string{"filter_state", "wasm.input_token"})
	if err != nil {
		return data
	}
	outputTokenStr, err := proxywasm.GetProperty([]string{"filter_state", "wasm.output_token"})
	if err != nil {
		return data
	}

	inputToken, err := strconv.Atoi(string(inputTokenStr))
	if err != nil {
		return data
	}

	outputToken, err := strconv.Atoi(string(outputTokenStr))
	if err != nil {
		return data
	}

	script := "local current = redis.call('incrby', KEYS[1], ARGV[1]) if tonumber(current) == tonumber(ARGV[1]) then redis.call('expire', KEYS[1], ARGV[2]) end return current"
	err = config.client.Eval(script, 1, []interface{}{redisKey}, []interface{}{inputToken + outputToken, 60}, func(response resp.Value) {
		if response.Error() != nil {
			log.Errorf("call Eval error: %v", response.Error())
		}
		proxywasm.ResumeHttpResponse()
	})
	if err != nil {
		log.Errorf("Error occured while calling IncrBy.")
		return data
	} else {
		return data
	}
}
