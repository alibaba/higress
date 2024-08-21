package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

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
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamingBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

var route, cluster, model, traceId, requestId string

type AIStatisticsConfig struct {
	enable               bool
	responseMetricConfig []ResponseMetricConfig
	metricsCounter       map[string]proxywasm.MetricCounter
	metricsGauge         map[string]proxywasm.MetricGauge
}

/**
    trace_tag:
		path: "body.cost"
		metric_name: "cost"
		metric_type: "gauge"
*/

type ResponseMetricConfig struct {
	path       string
	metricName string
	metricType string
}

type ResponseMetricData struct {
	path       string
	metricName string
	metricType string
	value      string
}

func (config *AIStatisticsConfig) incrementCounter(metricName string, inc uint64, log wrapper.Log) {
	counter, ok := config.metricsCounter[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.metricsCounter[metricName] = counter
	}
	counter.Increment(inc)
}

func parseConfig(json gjson.Result, config *AIStatisticsConfig, log wrapper.Log) error {
	config.enable = json.Get("enable").Bool()

	traceTagArrayStr := json.Get("trace_tag").Array()
	config.responseMetricConfig = make([]ResponseMetricConfig, len(traceTagArrayStr))
	for i, TraceTagArray := range traceTagArrayStr {
		config.responseMetricConfig[i].path = TraceTagArray.Get("path").String()
		config.responseMetricConfig[i].metricName = TraceTagArray.Get("metric_name").String()
		config.responseMetricConfig[i].metricType = TraceTagArray.Get("metric_type").String()
	}

	config.metricsCounter = make(map[string]proxywasm.MetricCounter)
	config.metricsGauge = make(map[string]proxywasm.MetricGauge)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	// The request does not have a body.
	if contentType == "" {
		return types.ActionContinue
	}
	if !strings.Contains(contentType, "application/json") {
		log.Warnf("content is not json, can't process:%s", contentType)
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	// The request has a body and requires delaying the header transmission until a cache miss occurs,
	// at which point the header should be sent.
	return types.HeaderStopIteration
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

	if endOfStream {
		// Get first token time from response header.
		firstTokenTime, _ := proxywasm.GetHttpResponseHeader("req-cost-time")
		if e := proxywasm.SetProperty([]string{"trace_span_tag.first_token_time"}, []byte(fmt.Sprintf("%d", firstTokenTime))); e != nil {
			log.Errorf("failed to set first_token_time in filter state: %v", e)
		}

		// calculate total time.
		statisticsStartTimeStr, _ := proxywasm.GetHttpResponseHeader("req-arrive-time")
		statisticsStartTime, _ := strconv.ParseInt(statisticsStartTimeStr, 10, 64)
		tm := time.Unix(0, statisticsStartTime*int64(time.Millisecond))
		end := time.Now()

		duration := end.Sub(tm)
		totalTime := int(duration.Nanoseconds() / 1e6)

		if e := proxywasm.SetProperty([]string{"trace_span_tag.total_time"}, []byte(fmt.Sprintf("%d", totalTime))); e != nil {
			log.Errorf("failed to set total_time in filter state: %v", e)
		}

		getBasicInfo()
		defineGauge(config, "first_token_time", log)
		defineGauge(config, "total_time", log)

		ResponseMetricDataResult := getCustomUsage(config, data)

		for _, metricDataResult := range ResponseMetricDataResult {
			switch metricDataResult.metricType {
			case "counter":
				defineCounter(config, metricDataResult.metricName, log)
				break
			case "gauge":
				defineGauge(config, metricDataResult.metricName, log)
				break
			case "string":
				break
			default:
				log.Errorf("unknown metric type: %s", metricDataResult.metricType)
			}
		}
	}

	model, inputToken, outputToken, ok := getUsage(data)
	if !ok {
		return data
	}
	setFilterStateData(config, model, inputToken, outputToken, log)
	incrementCounter(config, model, inputToken, outputToken, log)

	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {

	// calculate total time.
	statisticsStartTimeStr, _ := proxywasm.GetHttpResponseHeader("req-arrive-time")
	statisticsStartTime, _ := strconv.ParseInt(statisticsStartTimeStr, 10, 64)
	tm := time.Unix(0, statisticsStartTime*int64(time.Millisecond))
	end := time.Now()

	duration := end.Sub(tm)
	totalTime := int(duration.Nanoseconds() / 1e6)

	if e := proxywasm.SetProperty([]string{"trace_span_tag.total_time"}, []byte(fmt.Sprintf("%d", totalTime))); e != nil {
		log.Errorf("failed to set total_time in filter state: %v", e)
	}

	getBasicInfo()
	defineGauge(config, "total_time", log)
	ResponseMetricDataResult := getCustomUsage(config, body)

	for _, metricDataResult := range ResponseMetricDataResult {
		switch metricDataResult.metricType {
		case "counter":
			defineCounter(config, metricDataResult.metricName, log)
			break
		case "gauge":
			defineGauge(config, metricDataResult.metricName, log)
			break
		case "string":
			break
		default:
			log.Errorf("unknown metric type: %s", metricDataResult.metricType)
		}
	}

	model, inputToken, outputToken, ok := getUsage(body)
	if !ok {
		return types.ActionContinue
	}
	setFilterStateData(config, model, inputToken, outputToken, log)
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

func getCustomUsage(config AIStatisticsConfig, data []byte) (result []ResponseMetricData) {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	for _, chunk := range chunks {
		// the feature strings are used to identify the usage data, like:
		// {"model":"gpt2","usage":{"prompt_tokens":1,"completion_tokens":1}}
		for _, metric := range config.responseMetricConfig {
			value := gjson.GetBytes(chunk, metric.path)
			if value.Exists() {
				result = append(result, ResponseMetricData{
					path:       metric.path,
					metricName: metric.metricName,
					metricType: metric.metricType,
					value:      value.String(),
				})
				_ = proxywasm.SetProperty([]string{"trace_span_tag." + metric.metricName}, []byte(value.String()))
			}
		}
	}
	return
}

// setFilterData sets the input_token and output_token in the filter state.
// ai-token-ratelimit will use these values to calculate the total token usage.
func setFilterStateData(config AIStatisticsConfig, model string, inputToken int64, outputToken int64, log wrapper.Log) {
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
	var routeName, clusterName string
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err == nil {
		routeName = string(raw)
	}
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err == nil {
		clusterName = string(raw)
	}
	config.incrementCounter("route."+routeName+".upstream."+clusterName+".model."+model+".input_token", uint64(inputToken), log)
	config.incrementCounter("route."+routeName+".upstream."+clusterName+".model."+model+".output_token", uint64(outputToken), log)
}

// function to add Gauge metric
func (config *AIStatisticsConfig) defineGaugeMetric(metricName string, val uint64, log wrapper.Log) {
	log.Infof(metricName + ":" + strconv.FormatUint(val, 10))
	gauge, ok := config.metricsGauge[metricName]
	if !ok {
		gauge = proxywasm.DefineGaugeMetric(metricName)
		config.metricsGauge[metricName] = gauge
	}
	gauge.SetValue(int64(val))
}

func getBasicInfo() {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err == nil {
		route = string(raw)
	}
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err == nil {
		cluster = string(raw)
	}
	if raw, err := proxywasm.GetProperty([]string{"trace_span_tag.model"}); err == nil {
		model = string(raw)
	}
	if raw, err := proxywasm.GetProperty([]string{"trace_span_tag.trace_id"}); err == nil {
		traceId = string(raw)
	}
	if raw, err := proxywasm.GetProperty([]string{"trace_span_tag.request_id"}); err == nil {
		requestId = string(raw)
	}
}

func defineCounter(config AIStatisticsConfig, metricName string, log wrapper.Log) {
	if raw, err := proxywasm.GetProperty([]string{"trace_span_tag." + metricName}); err == nil {
		metricValue := binary.BigEndian.Uint64(raw)
		config.incrementCounter("route."+route+".upstream."+cluster+".model."+model+".traceid."+traceId+".requestid."+requestId+"."+metricName, metricValue, log)
	}
}

func defineGauge(config AIStatisticsConfig, metricName string, log wrapper.Log) {
	if raw, err := proxywasm.GetProperty([]string{"trace_span_tag." + metricName}); err == nil {
		metricValue := binary.BigEndian.Uint64(raw)
		config.defineGaugeMetric("route."+route+".upstream."+cluster+".model."+model+".traceid."+traceId+".requestid."+requestId+"."+metricName, metricValue, log)
	}
}
