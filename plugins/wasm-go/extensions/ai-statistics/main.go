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

// TracingLabel is the tracing label configuration.
type TracingLabel struct {
	Key         string `required:"true" yaml:"key" json:"key"`
	ValueSource string `required:"true" yaml:"valueSource" json:"valueSource"`
	Value       string `required:"true" yaml:"value" json:"value"`
}

type AIStatisticsConfig struct {
	Enable bool `required:"true" yaml:"enable" json:"enable"`
	// TracingLabel array define the tracing label.
	TracingLabel map[string]TracingLabel            `required:"true" yaml:"tracingLabel" json:"tracingLabel"`
	Metrics      map[string]proxywasm.MetricCounter `required:"true" yaml:"metrics" json:"metrics"`
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

	// Parse tracing label.
	traceLabelConfigArray := configJson.Get("tracing_label").Array()
	config.TracingLabel = make(map[string]TracingLabel)
	for _, traceLabel := range traceLabelConfigArray {
		traceLabel := TracingLabel{
			Key:         traceLabel.Get("key").String(),
			ValueSource: traceLabel.Get("value_source").String(),
			Value:       traceLabel.Get("value").String(),
		}
		config.TracingLabel[traceLabel.ValueSource] = traceLabel
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
	// Fetch request header tracing label value.
	fetchTracingLabelValue(config, "request_header", nil, "", log)
	// Fetch request process proxy wasm property.
	// Warn: The property may be modified by response process , so the value of the property may be overwritten.
	fetchTracingLabelValue(config, "property", nil, "", log)

	// The request has a body and requires delaying the header transmission until a cache miss occurs,
	// at which point the header should be sent.
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	// Fetch request body tracing label value.
	fetchTracingLabelValue(config, "request_body", body, "", log)
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

	// calculate total cost time and set tracing span tag.
	startTimeStr, _ := proxywasm.GetHttpResponseHeader("req-arrive-time")
	startTime, _ := strconv.ParseInt(startTimeStr, 10, 64)
	endTimeStr, _ := proxywasm.GetHttpResponseHeader("resp-start-time")
	endTime, _ := strconv.ParseInt(endTimeStr, 10, 64)
	totalTime := endTime - startTime

	fetchTracingLabelValue(config, "total_time", nil, fmt.Sprintf("%d", totalTime), log)

	// Fetch response body tracing label value.
	fetchTracingLabelValue(config, "response_header", nil, "", log)
	// Fetch response process proxy wasm property.
	fetchTracingLabelValue(config, "property", nil, "", log)

	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	// Get first token time from response header and set tracing span tag first_token_time.
	if endOfStream {
		firstTokenTime, _ := proxywasm.GetHttpResponseHeader("req-cost-time")
		fetchTracingLabelValue(config, "first_token_time", nil, firstTokenTime, log)
	}

	model, inputToken, outputToken, ok := getUsage(data)
	if !ok {
		return data
	}
	setFilterStateData(model, inputToken, outputToken, log)
	incrementCounter(config, model, inputToken, outputToken, log)
	// add inputTokens and outputTokens tracing span tag.
	setTracingTokenCostTag(config, inputToken, outputToken, log)
	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	model, inputToken, outputToken, ok := getUsage(body)
	if !ok {
		return types.ActionContinue
	}
	setFilterStateData(model, inputToken, outputToken, log)
	incrementCounter(config, model, inputToken, outputToken, log)
	// add inputTokens and outputTokens tracing span tag.
	setTracingTokenCostTag(config, inputToken, outputToken, log)
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

// sets the input_tokens and output_tokens tags in the tracing span.
func setTracingTokenCostTag(config AIStatisticsConfig, inputToken int64, outputToken int64, log wrapper.Log) {
	fetchTracingLabelValue(config, "input_tokens", nil, fmt.Sprintf("%d", inputToken), log)
	fetchTracingLabelValue(config, "output_tokens", nil, fmt.Sprintf("%d", outputToken), log)
}

// fetches the tracing label value from the specified source.
func fetchTracingLabelValue(config AIStatisticsConfig, tracingSource string, body []byte, defaultValue string, log wrapper.Log) {
	var tracingValue string
	var tracingKey string
	tracingLabelEle, ok := config.TracingLabel[tracingSource]
	if ok {
		switch tracingLabelEle.ValueSource {
		case "response_header":
			if value, err := proxywasm.GetHttpResponseHeader(tracingLabelEle.Value); err == nil {
				tracingValue = value
			}
		case "request_body":
			bodyJson := gjson.ParseBytes(body)
			tracingValue = trimQuote(bodyJson.Get(tracingLabelEle.Value).String())
		case "request_header":
			if value, err := proxywasm.GetHttpRequestHeader(tracingLabelEle.Value); err == nil {
				tracingValue = value
			}
		case "property":
			if raw, err := proxywasm.GetProperty([]string{tracingLabelEle.Value}); err == nil {
				tracingValue = string(raw)
			}
		default:
			tracingValue = ""
		}
		tracingKey = tracingLabelEle.Key
	} else {
		tracingValue = defaultValue
		tracingKey = tracingSource
	}

	log.Debugf("try to set trace label [%s] with value [%s].", tracingKey, tracingValue)

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

	return
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
