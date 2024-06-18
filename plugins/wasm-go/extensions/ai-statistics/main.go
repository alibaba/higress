package main

import (
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
	log.Infof("metric: %s add %d", metricName, inc)
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

func getLastChunk(data []byte) []byte {
	chunks := strings.Split(strings.TrimSpace(string(data)), "\n\n")
	length := len(chunks)
	if length < 2 {
		return data
	}
	// ai-proxy append extra usage chunk
	return []byte(chunks[length-1])
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	lastChunk := getLastChunk(data)
	// log.Info(string(data))
	// log.Infof("LastChunk is: %s", string(lastChunk))
	modelObj := gjson.GetBytes(lastChunk, "model")
	inputTokenObj := gjson.GetBytes(lastChunk, "usage.prompt_tokens")
	outputTokenObj := gjson.GetBytes(lastChunk, "usage.completion_tokens")
	// log.Infof("model: %s, input_token: %s, output_token: %s", modelObj.Raw, inputTokenObj.Raw, outputTokenObj.Raw)
	if modelObj.Exists() && inputTokenObj.Exists() && outputTokenObj.Exists() {
		ctx.SetContext("model", modelObj.String())
		ctx.SetContext("input_token", inputTokenObj.Int())
		ctx.SetContext("output_token", outputTokenObj.Int())
	}

	if endOfStream {
		var route, cluster string
		if raw, err := proxywasm.GetProperty([]string{"route_name"}); err == nil {
			route = string(raw)
		}
		if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err == nil {
			cluster = string(raw)
		}
		model, ok := ctx.GetContext("model").(string)
		if !ok {
			log.Error("Get model failed!")
			return data
		}
		inputToken, ok := ctx.GetContext("input_token").(int64)
		if !ok {
			log.Error("Get input_token failed!")
			return data
		}
		outputToken, ok := ctx.GetContext("output_token").(int64)
		if !ok {
			log.Error("Get output_token failed!")
			return data
		}
		config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".input_token", uint64(inputToken), log)
		config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".output_token", uint64(outputToken), log)
		proxywasm.SetProperty([]string{"model"}, []byte(model))
		proxywasm.SetProperty([]string{"input_token"}, []byte(fmt.Sprint(inputToken)))
		proxywasm.SetProperty([]string{"output_token"}, []byte(fmt.Sprint(outputToken)))
	}

	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	modeObj := gjson.GetBytes(body, "model")
	inputTokenObj := gjson.GetBytes(body, "usage.prompt_tokens")
	outputTokenObj := gjson.GetBytes(body, "usage.completion_tokens")
	if !modeObj.Exists() {
		log.Error("Get model failed")
		return types.ActionContinue
	}
	if !inputTokenObj.Exists() {
		log.Error("Get input_token failed")
		return types.ActionContinue
	}
	if !outputTokenObj.Exists() {
		log.Error("Get output_token failed")
		return types.ActionContinue
	}
	model := modeObj.String()
	inputToken := inputTokenObj.Int()
	outputToken := outputTokenObj.Int()
	var route, cluster string
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err == nil {
		route = string(raw)
	}
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err == nil {
		cluster = string(raw)
	}
	config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".input_token", uint64(inputToken), log)
	config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".output_token", uint64(outputToken), log)

	proxywasm.SetProperty([]string{"model"}, []byte(model))
	proxywasm.SetProperty([]string{"input_token"}, []byte(fmt.Sprint(inputToken)))
	proxywasm.SetProperty([]string{"output_token"}, []byte(fmt.Sprint(outputToken)))

	return types.ActionContinue
}
