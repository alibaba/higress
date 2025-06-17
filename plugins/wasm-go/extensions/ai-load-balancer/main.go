package main

import (
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"

	global_least_request "ai-load-balancer/global_least_request"
	least_busy "ai-load-balancer/least_busy"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"ai-load-balancer",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
		wrapper.ProcessResponseBody(onHttpResponseBody),
	)
}

type LoadBalancer interface {
	HandleHttpRequestHeaders(ctx wrapper.HttpContext) types.Action
	HandleHttpRequestBody(ctx wrapper.HttpContext, body []byte) types.Action
	HandleHttpResponseHeaders(ctx wrapper.HttpContext) types.Action
	HandleHttpStreamingResponseBody(ctx wrapper.HttpContext, data []byte, endOfStream bool) []byte
	HandleHttpResponseBody(ctx wrapper.HttpContext, body []byte) types.Action
}

type Config struct {
	policy string
	lb     LoadBalancer
}

const (
	LeastBusyLoadBalancerPolicy          = "least_busy"
	GlobalLeastRequestLoadBalancerPolicy = "global_least_request"
)

func parseConfig(json gjson.Result, config *Config) error {
	config.policy = json.Get("lb_policy").String()
	var err error
	if config.policy == LeastBusyLoadBalancerPolicy {
		config.lb, err = least_busy.NewLeastBusyLoadBalancer(json.Get("lb_config"))
	} else if config.policy == GlobalLeastRequestLoadBalancerPolicy {
		config.lb, err = global_least_request.NewGlobalLeastRequestLoadBalancer(json.Get("lb_config"))
	}
	return err
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config) types.Action {
	return config.lb.HandleHttpRequestHeaders(ctx)
}

func onHttpRequestBody(ctx wrapper.HttpContext, config Config, body []byte) types.Action {
	return config.lb.HandleHttpRequestBody(ctx, body)
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config Config) types.Action {
	return config.lb.HandleHttpRequestHeaders(ctx)
}

func onHttpStreamingResponseBody(ctx wrapper.HttpContext, config Config, data []byte, endOfStream bool) []byte {
	return config.lb.HandleHttpStreamingResponseBody(ctx, data, endOfStream)
}

func onHttpResponseBody(ctx wrapper.HttpContext, config Config, body []byte) types.Action {
	return config.lb.HandleHttpRequestBody(ctx, body)
}
