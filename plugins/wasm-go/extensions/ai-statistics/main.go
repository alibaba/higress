package main

import (
	"bytes"
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
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamingBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

const (
	StatisticsRequestStartTime = "ai-statistics-request-start-time"
	StatisticsFirstTokenTime   = "ai-statistics-first-token-time"
	TracePrefix                = "trace_span_tag."
)

// TracingSpan is the tracing span configuration.
type TracingSpan struct {
	Key         string `required:"true" yaml:"key" json:"key"`
	ValueSource string `required:"true" yaml:"valueSource" json:"valueSource"`
	Value       string `required:"true" yaml:"value" json:"value"`
}

type AIStatisticsConfig struct {
	// Metrics
	counterMetrics map[string]proxywasm.MetricCounter
	// TODO: add more metrics in Gauge and Histogram format
	TracingSpan []TracingSpan
}

func generateMetricName(route, cluster, model, metricName string) string {
	return fmt.Sprintf("route.%s.upstream.%s.model.%s.metric.%s", route, cluster, model, metricName)
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

func (config *AIStatisticsConfig) incrementCounter(metricName string, inc uint64) {
	counter, ok := config.counterMetrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.counterMetrics[metricName] = counter
	}
	counter.Increment(inc)
}

func parseConfig(configJson gjson.Result, config *AIStatisticsConfig, log wrapper.Log) error {
	// Parse tracing span.
	tracingSpanConfigArray := configJson.Get("tracing_span").Array()
	config.TracingSpan = make([]TracingSpan, len(tracingSpanConfigArray))
	for i, tracingSpanConfig := range tracingSpanConfigArray {
		tracingSpan := TracingSpan{
			Key:         tracingSpanConfig.Get("key").String(),
			ValueSource: tracingSpanConfig.Get("value_source").String(),
			Value:       tracingSpanConfig.Get("value").String(),
		}
		config.TracingSpan[i] = tracingSpan
	}

	config.counterMetrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	// Fetch request header tracing span value.
	setTracingSpanValueBySource(config, "request_header", nil, log)
	// Fetch request process proxy wasm property.
	// Warn: The property may be modified by response process , so the value of the property may be overwritten.
	setTracingSpanValueBySource(config, "property", nil, log)

	// Set request start time.
	ctx.SetContext(StatisticsRequestStartTime, time.Now().UnixMilli())

	// The request has a body and requires delaying the header transmission until a cache miss occurs,
	// at which point the header should be sent.
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	// Set request body tracing span value.
	setTracingSpanValueBySource(config, "request_body", body, log)
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if !strings.Contains(contentType, "text/event-stream") {
		ctx.BufferResponseBody()
	}

	// Set response header tracing span value.
	setTracingSpanValueBySource(config, "response_header", nil, log)

	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	requestStartTime, ok := ctx.GetContext(StatisticsRequestStartTime).(int64)
	if !ok {
		return data
	}

	// If this is the first chunk, record first token duration metric and span attribute
	firstTokenTime, ok := ctx.GetContext(StatisticsFirstTokenTime).(int64)
	if !ok {
		firstTokenTime = time.Now().UnixMilli()
		ctx.SetContext(StatisticsFirstTokenTime, firstTokenTime)
		setTracingSpanValue("llm_first_token_duration", fmt.Sprint(firstTokenTime-requestStartTime), log)
	}

	// If the end of the stream is reached, calculate the total time and set metric and span attribute.
	if endOfStream {
		if model, ok := ctx.GetContext("model").(string); ok {
			route, err := getRouteName()
			if err != nil {
				return data
			}
			cluster, err := getClusterName()
			if err != nil {
				return data
			}
			responseEndTime := time.Now().UnixMilli()
			setTracingSpanValue("llm_service_duration", fmt.Sprint(responseEndTime-requestStartTime), log)
			config.incrementCounter(generateMetricName(route, cluster, model, "llm_duration_count"), 1)
			llm_first_token_duration := uint64(firstTokenTime - requestStartTime)
			config.incrementCounter(generateMetricName(route, cluster, model, "llm_first_token_duration"), llm_first_token_duration)
			llm_service_duration := uint64(responseEndTime - requestStartTime)
			config.incrementCounter(generateMetricName(route, cluster, model, "llm_service_duration"), llm_service_duration)
		}
	}

	// Get infomations about this request
	model, inputToken, outputToken, ok := getUsage(data)
	if !ok {
		return data
	}
	route, err := getRouteName()
	if err != nil {
		return data
	}
	cluster, err := getClusterName()
	if err != nil {
		return data
	}
	// Set model context used in the last chunk which can be empty
	if ctx.GetContext("model") == nil {
		ctx.SetContext("model", model)
	}

	// Set token usage metrics
	config.incrementCounter(generateMetricName(route, cluster, model, "input_token"), uint64(inputToken))
	config.incrementCounter(generateMetricName(route, cluster, model, "output_token"), uint64(outputToken))
	// Set filter states which can be used by other plugins.
	setFilterState("model", model, log)
	setFilterState("input_token", inputToken, log)
	setFilterState("output_token", outputToken, log)
	// Set tracing span tag input_token and output_token.
	setTracingSpanValue("input_token", strconv.FormatInt(inputToken, 10), log)
	setTracingSpanValue("output_token", strconv.FormatInt(outputToken, 10), log)
	// Set response process proxy wasm property.
	setTracingSpanValueBySource(config, "property", nil, log)

	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	// Calculate the total time and set tracing span tag total_time.
	requestStartTime, ok := ctx.GetContext(StatisticsRequestStartTime).(int64)
	if !ok {
		return types.ActionContinue
	}
	responseEndTime := time.Now().UnixMilli()
	setTracingSpanValue("llm_service_duration", fmt.Sprint(responseEndTime-requestStartTime), log)
	// Get infomations about this request
	model, inputToken, outputToken, ok := getUsage(body)
	if !ok {
		return types.ActionContinue
	}
	route, err := getRouteName()
	if err != nil {
		return types.ActionContinue
	}
	cluster, err := getClusterName()
	if err != nil {
		return types.ActionContinue
	}
	// Set metrics
	llm_service_duration := uint64(responseEndTime - requestStartTime)
	config.incrementCounter(generateMetricName(route, cluster, model, "llm_service_duration"), llm_service_duration)
	config.incrementCounter(generateMetricName(route, cluster, model, "llm_duration_count"), 1)
	config.incrementCounter(generateMetricName(route, cluster, model, "input_token"), uint64(inputToken))
	config.incrementCounter(generateMetricName(route, cluster, model, "output_token"), uint64(outputToken))
	// Set filter states which can be used by other plugins.
	setFilterState("model", model, log)
	setFilterState("input_token", inputToken, log)
	setFilterState("output_token", outputToken, log)
	// Set tracing span tag input_tokens and output_tokens.
	setTracingSpanValue("input_token", strconv.FormatInt(inputToken, 10), log)
	setTracingSpanValue("output_token", strconv.FormatInt(outputToken, 10), log)
	// Set response process proxy wasm property.
	setTracingSpanValueBySource(config, "property", nil, log)
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

func setFilterState(key string, value interface{}, log wrapper.Log) {
	if e := proxywasm.SetProperty([]string{key}, []byte(fmt.Sprint(value))); e != nil {
		log.Errorf("failed to set %s in filter state: %v", key, e)
	}
}

// fetches the tracing span value from the specified source.
func setTracingSpanValueBySource(config AIStatisticsConfig, tracingSource string, body []byte, log wrapper.Log) {
	for _, tracingSpanEle := range config.TracingSpan {
		if tracingSource == tracingSpanEle.ValueSource {
			switch tracingSource {
			case "response_header":
				if value, err := proxywasm.GetHttpResponseHeader(tracingSpanEle.Value); err == nil {
					setTracingSpanValue(tracingSpanEle.Key, value, log)
				}
			case "request_body":
				bodyJson := gjson.ParseBytes(body)
				value := bodyJson.Get(tracingSpanEle.Value).String()
				setTracingSpanValue(tracingSpanEle.Key, value, log)
			case "request_header":
				if value, err := proxywasm.GetHttpRequestHeader(tracingSpanEle.Value); err == nil {
					setTracingSpanValue(tracingSpanEle.Key, value, log)
				}
			case "property":
				if raw, err := proxywasm.GetProperty([]string{tracingSpanEle.Value}); err == nil {
					setTracingSpanValue(tracingSpanEle.Key, string(raw), log)
				}
			default:

			}
		}
	}
}

// Set the tracing span with value.
func setTracingSpanValue(tracingKey, tracingValue string, log wrapper.Log) {
	log.Debugf("try to set trace span [%s] with value [%s].", tracingKey, tracingValue)

	if tracingValue != "" {
		traceSpanTag := TracePrefix + tracingKey

		if raw, err := proxywasm.GetProperty([]string{traceSpanTag}); err == nil {
			if raw != nil {
				log.Warnf("trace span [%s] already exists, value will be overwrite, orign value: %s.", traceSpanTag, string(raw))
			}
		}

		if e := proxywasm.SetProperty([]string{traceSpanTag}, []byte(tracingValue)); e != nil {
			log.Errorf("failed to set %s in filter state: %v", traceSpanTag, e)
		}
		log.Debugf("successed to set trace span [%s] with value [%s].", traceSpanTag, tracingValue)
	}
}
