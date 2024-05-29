package main

import (
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	regexp "github.com/wasilibs/go-re2"
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

type AIStatisticsConfig struct {
	client     wrapper.HttpClient
	metrics    map[string]proxywasm.MetricCounter
	qwenRegExp *regexp.Regexp
	gptRegExp  *regexp.Regexp
}

func (config *AIStatisticsConfig) incrementCounter(metricName string, inc uint64) {
	counter, ok := config.metrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.metrics[metricName] = counter
	}
	counter.Increment(inc)
}

func parseConfig(json gjson.Result, config *AIStatisticsConfig, log wrapper.Log) error {
	config.metrics = make(map[string]proxywasm.MetricCounter)
	config.qwenRegExp, _ = regexp.Compile("qwen.*")
	config.gptRegExp, _ = regexp.Compile("gpt.*")
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

	if config.gptRegExp.MatchString(model) {
		if gjson.GetBytes(body, "stream").Exists() && gjson.GetBytes(body, "stream").Bool() {
			ctx.BufferResponseBody()
		}
	} else if config.qwenRegExp.MatchString(model) {
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
		proxywasm.SetProperty([]string{"model"}, []byte(model))
		proxywasm.SetProperty([]string{"input_token"}, []byte(fmt.Sprint(input_token)))
		proxywasm.SetProperty([]string{"output_token"}, []byte(fmt.Sprint(output_token)))
	}

	if config.gptRegExp.MatchString(model) {
		usage := gjson.GetBytes(data, "usage")
		if usage.Exists() {
			input_token := usage.Get("prompt_tokens").Int()
			ctx.SetContext("input_token", input_token)
			output_token := usage.Get("completion_tokens").Int()
			ctx.SetContext("output_token", output_token)
			log.Infof("input_token: %d, output_token: %d", input_token, output_token)
		}
	} else if config.qwenRegExp.MatchString(model) {
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

	if config.gptRegExp.MatchString(model) {
		usage := gjson.GetBytes(body, "usage")
		if usage.Exists() {
			input_token := usage.Get("prompt_tokens").Int()
			ctx.SetContext("input_token", input_token)
			output_token := usage.Get("completion_tokens").Int()
			ctx.SetContext("output_token", output_token)
			log.Infof("input_token: %d, output_token: %d", input_token, output_token)
		}
	} else if config.qwenRegExp.MatchString(model) {
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

	proxywasm.SetProperty([]string{"model"}, []byte(model))
	proxywasm.SetProperty([]string{"input_token"}, []byte(fmt.Sprint(input_token)))
	proxywasm.SetProperty([]string{"output_token"}, []byte(fmt.Sprint(output_token)))

	return types.ActionContinue
}
