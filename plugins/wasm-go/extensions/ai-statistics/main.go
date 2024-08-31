package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"strconv"
	"strings"
	"time"
)

const (
	StatisticsRequestStartTime = "ai-statistics-request-start-time"
	StatisticsFirstTokenTime   = "ai-statistics-first-token-time"
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

// TracingSpan is the tracing span configuration.
type TracingSpan struct {
	Key         string `required:"true" yaml:"key" json:"key"`
	ValueSource string `required:"true" yaml:"valueSource" json:"valueSource"`
	Value       string `required:"true" yaml:"value" json:"value"`
}

type AIStatisticsConfig struct {
	Enable bool `required:"true" yaml:"enable" json:"enable"`
	// TracingSpan array define the tracing span.
	TracingSpan []TracingSpan                      `required:"true" yaml:"tracingSpan" json:"tracingSpan"`
	Metrics     map[string]proxywasm.MetricCounter `required:"true" yaml:"metrics" json:"metrics"`
}

func (config *AIStatisticsConfig) incrementCounter(metricName string, inc uint64, log wrapper.Log) {
	counter, ok := config.Metrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.Metrics[metricName] = counter
	}
	counter.Increment(inc)
}

func parseConfig(configJson gjson.Result, config *AIStatisticsConfig, log wrapper.Log) error {
	config.Enable = configJson.Get("enable").Bool()

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

	config.Metrics = make(map[string]proxywasm.MetricCounter)

	configStr, _ := json.Marshal(config)
	log.Infof("Init ai-statistics config success, config: %s.", configStr)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {

	if !config.Enable {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	// Fetch request header tracing span value.
	setTracingSpanValueBySource(config, "request_header", nil, log)
	// Fetch request process proxy wasm property.
	// Warn: The property may be modified by response process , so the value of the property may be overwritten.
	setTracingSpanValueBySource(config, "property", nil, log)

	// Set request start time.
	ctx.SetContext(StatisticsRequestStartTime, strconv.FormatUint(uint64(time.Now().UnixMilli()), 10))

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
	if !config.Enable {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if !strings.Contains(contentType, "text/event-stream") {
		ctx.BufferResponseBody()
	}

	// Set response header tracing span value.
	setTracingSpanValueBySource(config, "response_header", nil, log)

	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {

	// If the end of the stream is reached, calculate the total time and set tracing span tag total_time.
	// Otherwise, set tracing span tag first_token_time.
	if endOfStream {
		requestStartTimeStr := ctx.GetContext(StatisticsRequestStartTime).(string)
		requestStartTime, _ := strconv.ParseInt(requestStartTimeStr, 10, 64)
		responseEndTime := time.Now().UnixMilli()
		setTracingSpanValue("total_time", fmt.Sprintf("%d", responseEndTime-requestStartTime), log)
	} else {
		firstTokenTime := ctx.GetContext(StatisticsFirstTokenTime)
		if firstTokenTime == nil {
			firstTokenTimeStr := strconv.FormatInt(time.Now().UnixMilli(), 10)
			ctx.SetContext(StatisticsFirstTokenTime, firstTokenTimeStr)
			setTracingSpanValue("first_token_time", firstTokenTimeStr, log)
		}
	}

	model, inputToken, outputToken, ok := getUsage(data)
	if !ok {
		return data
	}
	setFilterStateData(model, inputToken, outputToken, log)
	incrementCounter(config, model, inputToken, outputToken, log)
	// Set tracing span tag input_tokens and output_tokens.
	setTracingSpanValue("input_tokens", strconv.FormatInt(inputToken, 10), log)
	setTracingSpanValue("output_tokens", strconv.FormatInt(outputToken, 10), log)
	// Set response process proxy wasm property.
	setTracingSpanValueBySource(config, "property", nil, log)

	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {

	// Calculate the total time and set tracing span tag total_time.
	requestStartTimeStr := ctx.GetContext(StatisticsRequestStartTime).(string)
	requestStartTime, _ := strconv.ParseInt(requestStartTimeStr, 10, 64)
	responseEndTime := time.Now().UnixMilli()
	setTracingSpanValue("total_time", fmt.Sprintf("%d", responseEndTime-requestStartTime), log)

	model, inputToken, outputToken, ok := getUsage(body)
	if !ok {
		return types.ActionContinue
	}
	setFilterStateData(model, inputToken, outputToken, log)
	incrementCounter(config, model, inputToken, outputToken, log)
	// Set tracing span tag input_tokens and output_tokens.
	setTracingSpanValue("input_tokens", strconv.FormatInt(inputToken, 10), log)
	setTracingSpanValue("output_tokens", strconv.FormatInt(outputToken, 10), log)
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
				value := trimQuote(bodyJson.Get(tracingSpanEle.Value).String())
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
		traceSpanTag := "trace_span_tag." + tracingKey

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

// trims the quote from the source string.
func trimQuote(source string) string {
	TempKey := strings.Trim(source, `"`)
	Key, _ := zhToUnicode([]byte(TempKey))
	return string(Key)
}

// converts the zh string to Unicode.
func zhToUnicode(raw []byte) ([]byte, error) {
	str, err := strconv.Unquote(strings.Replace(strconv.Quote(string(raw)), `\\u`, `\u`, -1))
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}
