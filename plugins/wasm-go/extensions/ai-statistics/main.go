package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"ai-statistics",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamingBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

type AIStatisticsConfig struct {
	enable  bool
	metrics map[string]proxywasm.MetricCounter
}

func (config *AIStatisticsConfig) incrementCounter(metricName string, inc uint64, log wrapper.Log) {
	counter, ok := config.metrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.metrics[metricName] = counter
	}
	counter.Increment(inc)
}

func parseConfig(json gjson.Result, config *AIStatisticsConfig, log wrapper.Log) error {
	config.enable = json.Get("enable").Bool()
	config.metrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	if !config.enable {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if !strings.Contains(contentType, "text/event-stream") {
		ctx.BufferResponseBody()
	}
	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	model, inputToken, outputToken, ok := getUsage(data)
	if !ok {
		return data
	}
	setFilterStateData(model, inputToken, outputToken, log)
	incrementCounter(config, model, inputToken, outputToken, log)
	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	model, inputToken, outputToken, ok := getUsage(body)
	if !ok {
		return types.ActionContinue
	}
	setFilterStateData(model, inputToken, outputToken, log)
	incrementCounter(config, model, inputToken, outputToken, log)
	return types.ActionContinue
}

func getUsage(data []byte) (model string, inputTokenUsage int64, outputTokenUsage int64, ok bool) {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	for _, chunk := range chunks {
		// the feature strings are used to identify the usage data, like:
		// {"model":"gpt2","usage":{"prompt_tokens":1,"completion_tokens":1}}
		if !bytes.Contains(chunk, []byte("prompt_tokens")) {
			continue
		}
		if !bytes.Contains(chunk, []byte("completion_tokens")) {
			continue
		}
		modelObj := gjson.GetBytes(chunk, "model")
		inputTokenObj := gjson.GetBytes(chunk, "usage.prompt_tokens")
		outputTokenObj := gjson.GetBytes(chunk, "usage.completion_tokens")
		if modelObj.Exists() && inputTokenObj.Exists() && outputTokenObj.Exists() {
			model = modelObj.String()
			inputTokenUsage = inputTokenObj.Int()
			outputTokenUsage = outputTokenObj.Int()
			ok = true
			return
		}
	}
	return
}

// setFilterData sets the input_token and output_token in the filter state.
// ai-token-ratelimit will use these values to calculate the total token usage.
func setFilterStateData(model string, inputToken int64, outputToken int64, log wrapper.Log) {
	if e := proxywasm.SetProperty([]string{"model"}, []byte(model)); e != nil {
		log.Errorf("failed to set model in filter state: %v", e)
	}
	if e := proxywasm.SetProperty([]string{"input_token"}, []byte(fmt.Sprintf("%d", inputToken))); e != nil {
		log.Errorf("failed to set input_token in filter state: %v", e)
	}
	if e := proxywasm.SetProperty([]string{"output_token"}, []byte(fmt.Sprintf("%d", outputToken))); e != nil {
		log.Errorf("failed to set output_token in filter state: %v", e)
	}
}

func incrementCounter(config AIStatisticsConfig, model string, inputToken int64, outputToken int64, log wrapper.Log) {
	var route, cluster string
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err == nil {
		route = string(raw)
	}
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err == nil {
		cluster = string(raw)
	}
	config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".input_token", uint64(inputToken), log)
	config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".output_token", uint64(outputToken), log)
}
