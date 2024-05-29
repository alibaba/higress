package main

import (
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"ai-statistics",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamingBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

var model2org map[string]string = map[string]string{
	"gpt-3.5-turbo": "openai",
	"qwen-turbo":    "qwen",
}

type AIStatisticsConfig struct {
	client  wrapper.HttpClient
	metrics map[string]proxywasm.MetricCounter
}

func (config *AIStatisticsConfig) incrementCounter(metricName string, inc uint64) {
	// TODO(jcchavezs): figure out if we are OK with dynamic creation of metrics
	// or we generate the metrics on before hand.
	counter, ok := config.metrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.metrics[metricName] = counter
	}
	counter.Increment(inc)
}

func parseConfig(json gjson.Result, config *AIStatisticsConfig, log wrapper.Log) error {
	config.metrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	if !gjson.GetBytes(body, "model").Exists() {
		ctx.SetContext("skip", true)
		return types.ActionContinue
	}

	model := gjson.GetBytes(body, "model").String()
	ctx.SetContext("model", model)
	ctx.SetContext("skip", false)

	switch model2org[model] {
	case "openai":
		if gjson.GetBytes(body, "stream").Exists() && gjson.GetBytes(body, "stream").Bool() {
			ctx.BufferResponseBody()
		}
	case "qwen":
		x_dashscope_sse, _ := proxywasm.GetHttpRequestHeader("X-DashScope-SSE")
		accept, _ := proxywasm.GetHttpRequestHeader("Accept")
		if x_dashscope_sse != "enable" && accept != "text/event-stream" {
			ctx.BufferResponseBody()
		}
	}

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	if ctx.GetContext("skip").(bool) {
		return data
	}

	model := ctx.GetContext("model").(string)

	if endOfStream {
		var route, cluster string
		if raw, err := proxywasm.GetProperty([]string{"route_name"}); err == nil {
			route = string(raw)
		}
		if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err == nil {
			cluster = string(raw)
		}
		input_token := ctx.GetContext("input_token").(int64)
		output_token := ctx.GetContext("output_token").(int64)
		config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".input_token", uint64(input_token))
		config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".output_token", uint64(output_token))
	}

	switch model2org[model] {
	case "openai":
		log.Infof("org is %s", model2org[model])
		usage := gjson.GetBytes(data, "usage")
		if usage.Exists() {
			input_token := usage.Get("prompt_tokens").Int()
			ctx.SetContext("input_token", input_token)
			output_token := usage.Get("completion_tokens").Int()
			ctx.SetContext("output_token", output_token)
			log.Infof("input_token: %d, output_token: %d", input_token, output_token)
		}
	case "qwen":
		log.Infof("org is %s", model2org[model])
		usage := gjson.GetBytes(data, "usage")
		if usage.Exists() {
			input_token := usage.Get("input_tokens").Int()
			ctx.SetContext("input_token", input_token)
			output_token := usage.Get("output_tokens").Int()
			ctx.SetContext("output_token", output_token)
			log.Infof("input_token: %d, output_token: %d", input_token, output_token)
		}
	}

	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	if ctx.GetContext("skip").(bool) {
		return types.ActionContinue
	}

	model := ctx.GetContext("model").(string)

	switch model2org[model] {
	case "openai":
		log.Infof("org is %s", model2org[model])
		usage := gjson.GetBytes(body, "usage")
		if usage.Exists() {
			input_token := usage.Get("prompt_tokens").Int()
			ctx.SetContext("input_token", input_token)
			output_token := usage.Get("completion_tokens").Int()
			ctx.SetContext("output_token", output_token)
			log.Infof("input_token: %d, output_token: %d", input_token, output_token)
		}
	case "qwen":
		log.Infof("org is %s", model2org[model])
		usage := gjson.GetBytes(body, "usage")
		if usage.Exists() {
			input_token := usage.Get("input_tokens").Int()
			ctx.SetContext("input_token", input_token)
			output_token := usage.Get("output_tokens").Int()
			ctx.SetContext("output_token", output_token)
			log.Infof("input_token: %d, output_token: %d", input_token, output_token)
		}
	}

	var route, cluster string
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err == nil {
		route = string(raw)
	}
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err == nil {
		cluster = string(raw)
	}
	input_token := ctx.GetContext("input_token").(int64)
	output_token := ctx.GetContext("output_token").(int64)
	config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".input_token", uint64(input_token))
	config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".output_token", uint64(output_token))

	return types.ActionContinue
}
