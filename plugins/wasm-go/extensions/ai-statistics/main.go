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
type Attribute struct {
	Key         string
	ValueSource string
	Value       string
	Rule        string
}

type AIStatisticsConfig struct {
	// Metrics
	counterMetrics map[string]proxywasm.MetricCounter
	// TODO: add more metrics in Gauge and Histogram format
	logAttributes             []Attribute
	spanAttributes            []Attribute
	shouldBufferStreamingBody bool
}

func generateMetricName(route, cluster, model, metricName string) string {
	return fmt.Sprintf("route.%s.upstream.%s.model.%s.metric.%s", route, cluster, model, metricName)
}

func getRouteName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		return "-", err
	} else {
		return string(raw), nil
	}
}

func getClusterName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err != nil {
		return "-", err
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
	// Parse tracing span attributes setting.
	tracingSpanConfigArray := configJson.Get("spanAttributes").Array()
	config.spanAttributes = make([]Attribute, len(tracingSpanConfigArray))
	for i, traceAttributeConfig := range tracingSpanConfigArray {
		spanAttribute := Attribute{
			Key:         traceAttributeConfig.Get("key").String(),
			ValueSource: traceAttributeConfig.Get("value_source").String(),
			Value:       traceAttributeConfig.Get("value").String(),
			Rule:        traceAttributeConfig.Get("rule").String(),
		}
		if spanAttribute.ValueSource == "response_streaming_body" {
			config.shouldBufferStreamingBody = true
		}
		config.spanAttributes[i] = spanAttribute
	}
	// Parse log attributes setting.
	logConfigArray := configJson.Get("logAttributes").Array()
	config.logAttributes = make([]Attribute, len(logConfigArray))
	for i, logAttributeConfig := range logConfigArray {
		logAttribute := Attribute{
			Key:         logAttributeConfig.Get("key").String(),
			ValueSource: logAttributeConfig.Get("value_source").String(),
			Value:       logAttributeConfig.Get("value").String(),
			Rule:        logAttributeConfig.Get("rule").String(),
		}
		if logAttribute.ValueSource == "response_streaming_body" {
			config.shouldBufferStreamingBody = true
		}
		config.logAttributes[i] = logAttribute
	}
	// Metric settings
	config.counterMetrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	logAttributes := make(map[string]string)
	ctx.SetContext("logAttributes", logAttributes)
	// Set base span attributes
	setTracingSpanValue("gen_ai.span.kind", "LLM", log)
	// Set user defined log & span attributes which type is request_header
	setTraceAttributeValueBySource(config, "request_header", nil, log)
	setLogAttributeValueBySource(ctx, config, "request_header", nil, log)
	// Set user defined log & span attributes which type is fixed_value
	setTraceAttributeValueBySource(config, "fixed_value", nil, log)
	setLogAttributeValueBySource(ctx, config, "fixed_value", nil, log)
	// Set request start time.
	ctx.SetContext(StatisticsRequestStartTime, time.Now().UnixMilli())

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	// Set user defined log & span attributes.
	setTraceAttributeValueBySource(config, "request_body", body, log)
	setLogAttributeValueBySource(ctx, config, "request_body", body, log)

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if !strings.Contains(contentType, "text/event-stream") {
		ctx.BufferResponseBody()
	}

	// Set user defined log & span attributes.
	setTraceAttributeValueBySource(config, "response_header", nil, log)
	setLogAttributeValueBySource(ctx, config, "response_header", nil, log)

	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	// Buffer stream body for record log & span attributes
	if config.shouldBufferStreamingBody {
		var streamingBodyBuffer []byte
		streamingBodyBuffer, ok := ctx.GetContext("streamingBodyBuffer").([]byte)
		if !ok {
			streamingBodyBuffer = data
		} else {
			streamingBodyBuffer = append(streamingBodyBuffer, data...)
		}
		ctx.SetContext("streamingBodyBuffer", streamingBodyBuffer)
	}

	// Get requestStartTime from http context
	requestStartTime, ok := ctx.GetContext(StatisticsRequestStartTime).(int64)
	if !ok {
		log.Error("failed to get requestStartTime from http context")
		return data
	}

	// If this is the first chunk, record first token duration metric and span attribute
	firstTokenTime, ok := ctx.GetContext(StatisticsFirstTokenTime).(int64)
	if !ok {
		firstTokenTime = time.Now().UnixMilli()
		ctx.SetContext(StatisticsFirstTokenTime, firstTokenTime)
		setTracingSpanValue("llm_first_token_duration", fmt.Sprint(firstTokenTime-requestStartTime), log)
	}

	// Set infomations about this request
	route, _ := getRouteName()
	cluster, _ := getClusterName()
	if model, inputToken, outputToken, ok := getUsage(data); ok {
		// Get logAttributes from http context
		logAttributes, ok := ctx.GetContext("logAttributes").(map[string]string)
		if !ok {
			log.Error("failed to get logAttributes from http context")
			return data
		}
		// Record Log Attributes
		logAttributes["route"] = route
		logAttributes["cluster"] = cluster
		logAttributes["model"] = model
		logAttributes["input_token"] = fmt.Sprint(inputToken)
		logAttributes["output_token"] = fmt.Sprint(outputToken)
		// Set logAttributes to http context
		ctx.SetContext("logAttributes", logAttributes)
	}
	// If the end of the stream is reached, record metrics/logs/spans.
	if endOfStream {
		// Get logAttributes from http context
		logAttributes, ok := ctx.GetContext("logAttributes").(map[string]string)
		if !ok {
			log.Error("failed to get logAttributes from http context")
			return data
		}
		responseEndTime := time.Now().UnixMilli()
		llm_first_token_duration := uint64(firstTokenTime - requestStartTime)
		llm_service_duration := uint64(responseEndTime - requestStartTime)
		logAttributes["llm_first_token_duration"] = fmt.Sprint(llm_first_token_duration)
		logAttributes["llm_service_duration"] = fmt.Sprint(llm_service_duration)
		inputTokenUint64, err := strconv.ParseUint(logAttributes["input_token"], 10, 0)
		if err != nil || inputTokenUint64 == 0 {
			log.Errorf("input_token convert failed, value is %d, err msg is [%v]", inputTokenUint64, err)
			return data
		}
		outputTokenUint64, err := strconv.ParseUint(logAttributes["output_token"], 10, 0)
		if err != nil || outputTokenUint64 == 0 {
			log.Errorf("output_token convert failed, value is %d, err msg is [%v]", outputTokenUint64, err)
			return data
		}
		// Set filter states which can be used by other plugins such as ai-token-ratelimit
		setFilterState("model", logAttributes["model"], log)
		setFilterState("input_token", logAttributes["input_token"], log)
		setFilterState("output_token", logAttributes["output_token"], log)
		// Set metrics
		config.incrementCounter(generateMetricName(route, cluster, logAttributes["model"], "input_token"), inputTokenUint64)
		config.incrementCounter(generateMetricName(route, cluster, logAttributes["model"], "output_token"), outputTokenUint64)
		config.incrementCounter(generateMetricName(route, cluster, logAttributes["model"], "llm_first_token_duration"), llm_first_token_duration)
		config.incrementCounter(generateMetricName(route, cluster, logAttributes["model"], "llm_service_duration"), llm_service_duration)
		config.incrementCounter(generateMetricName(route, cluster, logAttributes["model"], "llm_duration_count"), 1)
		// Set tracing span attributes.
		setTracingSpanValue("gen_ai.model_name", logAttributes["model"], log)
		setTracingSpanValue("gen_ai.usage.input_tokens", logAttributes["input_token"], log)
		setTracingSpanValue("gen_ai.usage.output_tokens", logAttributes["output_token"], log)
		setTracingSpanValue("gen_ai.usage.total_tokens", fmt.Sprint(inputTokenUint64+outputTokenUint64), log)
		setTracingSpanValue("llm_service_duration", fmt.Sprint(responseEndTime-requestStartTime), log)
		// Set user defined log & span attributes.
		if config.shouldBufferStreamingBody {
			streamingBodyBuffer, ok := ctx.GetContext("streamingBodyBuffer").([]byte)
			if !ok {
				return data
			}
			setTraceAttributeValueBySource(config, "response_streaming_body", streamingBodyBuffer, log)
			setLogAttributeValueBySource(ctx, config, "response_streaming_body", streamingBodyBuffer, log)
		}
		writeLog(logAttributes, log)
	}
	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	// Get logAttributes from http context
	logAttributes, ok := ctx.GetContext("logAttributes").(map[string]string)
	if !ok {
		log.Error("failed to get logAttributes from http context")
		return types.ActionContinue
	}

	// Get requestStartTime from http context
	requestStartTime, ok := ctx.GetContext(StatisticsRequestStartTime).(int64)
	if !ok {
		log.Error("failed to get requestStartTime from http context")
		return types.ActionContinue
	}

	responseEndTime := time.Now().UnixMilli()
	llm_service_duration := uint64(responseEndTime - requestStartTime)

	// Get infomations about this request
	route, _ := getRouteName()
	cluster, _ := getClusterName()
	model, inputToken, outputToken, ok := getUsage(body)
	if !ok {
		return types.ActionContinue
	}
	// Set filter states which can be used by other plugins such as ai-token-ratelimit
	setFilterState("model", model, log)
	setFilterState("input_token", inputToken, log)
	setFilterState("output_token", outputToken, log)
	// Set metrics
	config.incrementCounter(generateMetricName(route, cluster, model, "input_token"), uint64(inputToken))
	config.incrementCounter(generateMetricName(route, cluster, model, "output_token"), uint64(outputToken))
	config.incrementCounter(generateMetricName(route, cluster, model, "llm_service_duration"), llm_service_duration)
	config.incrementCounter(generateMetricName(route, cluster, model, "llm_duration_count"), 1)
	// Set tracing span tag input_tokens and output_tokens.
	setTracingSpanValue("gen_ai.model_name", model, log)
	setTracingSpanValue("gen_ai.usage.input_tokens", fmt.Sprint(inputToken), log)
	setTracingSpanValue("gen_ai.usage.output_tokens", fmt.Sprint(outputToken), log)
	setTracingSpanValue("gen_ai.usage.total_tokens", fmt.Sprint(inputToken+outputToken), log)
	setTracingSpanValue("llm_service_duration", fmt.Sprint(responseEndTime-requestStartTime), log)
	// Set Log Attributes
	logAttributes["route"] = route
	logAttributes["cluster"] = cluster
	logAttributes["model"] = model
	logAttributes["input_token"] = fmt.Sprint(inputToken)
	logAttributes["output_token"] = fmt.Sprint(outputToken)
	logAttributes["llm_service_duration"] = fmt.Sprint(llm_service_duration)
	// Write log
	writeLog(logAttributes, log)
	// Set user defined log & span attributes.
	setTraceAttributeValueBySource(config, "response_body", body, log)
	setLogAttributeValueBySource(ctx, config, "response_body", body, log)
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

func extractStreamingBodyByJsonPath(data []byte, jsonPath string, rule string, log wrapper.Log) string {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	value := ""
	if rule == "first" {
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				value = jsonObj.String()
				break
			}
		}
	} else if rule == "replace" {
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				value = jsonObj.String()
			}
		}
	} else if rule == "append" {
		// extract llm response
		for _, chunk := range chunks {
			raw := gjson.GetBytes(chunk, jsonPath).Raw
			if len(raw) > 2 {
				value += raw[1 : len(raw)-1]
			}
		}
	} else {
		log.Errorf("unsupported rule type: %s", rule)
	}
	return value
}

func setFilterState(key string, value interface{}, log wrapper.Log) {
	if e := proxywasm.SetProperty([]string{key}, []byte(fmt.Sprint(value))); e != nil {
		log.Errorf("failed to set %s in filter state: %v", key, e)
	}
}

// fetches the tracing span value from the specified source.
func setTraceAttributeValueBySource(config AIStatisticsConfig, source string, body []byte, log wrapper.Log) {
	for _, spanAttribute := range config.spanAttributes {
		if source == spanAttribute.ValueSource {
			switch source {
			case "fixed_value":
				log.Debugf("[span attribute] source type: %s, key: %s, value: %s", source, spanAttribute.Key, spanAttribute.Value)
				setTracingSpanValue(spanAttribute.Key, spanAttribute.Value, log)
			case "request_header":
				if value, err := proxywasm.GetHttpRequestHeader(spanAttribute.Value); err == nil {
					log.Debugf("[span attribute] source type: %s, key: %s, value: %s", source, spanAttribute.Key, value)
					setTracingSpanValue(spanAttribute.Key, value, log)
				}
			case "request_body":
				value := gjson.GetBytes(body, spanAttribute.Value).String()
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, spanAttribute.Key, value)
				setTracingSpanValue(spanAttribute.Key, value, log)
			case "response_header":
				if value, err := proxywasm.GetHttpResponseHeader(spanAttribute.Value); err == nil {
					log.Debugf("[span attribute] source type: %s, key: %s, value: %s", source, spanAttribute.Key, value)
					setTracingSpanValue(spanAttribute.Key, value, log)
				}
			case "response_streaming_body":
				value := extractStreamingBodyByJsonPath(body, spanAttribute.Value, spanAttribute.Rule, log)
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, spanAttribute.Key, value)
				setTracingSpanValue(spanAttribute.Key, value, log)
			case "response_body":
				value := gjson.GetBytes(body, spanAttribute.Value).String()
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, spanAttribute.Key, value)
				setTracingSpanValue(spanAttribute.Key, value, log)
			default:
				log.Errorf("source type %s is error", source)
			}
		}
	}
}

// Set the tracing span with value.
func setTracingSpanValue(tracingKey, tracingValue string, log wrapper.Log) {
	if tracingValue == "" {
		tracingValue = "-"
	}

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

// fetches the tracing span value from the specified source.
func setLogAttributeValueBySource(ctx wrapper.HttpContext, config AIStatisticsConfig, source string, body []byte, log wrapper.Log) {
	logAttributes, ok := ctx.GetContext("logAttributes").(map[string]string)
	if !ok {
		log.Error("failed to get logAttributes from http context")
		return
	}
	for _, logAttribute := range config.logAttributes {
		if source == logAttribute.ValueSource {
			switch source {
			case "fixed_value":
				log.Debugf("[span attribute] source type: %s, key: %s, value: %s", source, logAttribute.Key, logAttribute.Value)
				logAttributes[logAttribute.Key] = logAttribute.Value
			case "request_header":
				if value, err := proxywasm.GetHttpRequestHeader(logAttribute.Value); err == nil {
					log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, logAttribute.Key, value)
					logAttributes[logAttribute.Key] = value
				}
			case "request_body":
				value := gjson.GetBytes(body, logAttribute.Value).String()
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, logAttribute.Key, value)
				logAttributes[logAttribute.Key] = value
			case "response_header":
				if value, err := proxywasm.GetHttpResponseHeader(logAttribute.Value); err == nil {
					log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, logAttribute.Key, value)
					logAttributes[logAttribute.Key] = value
				}
			case "response_streaming_body":
				value := extractStreamingBodyByJsonPath(body, logAttribute.Value, logAttribute.Rule, log)
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, logAttribute.Key, value)
				logAttributes[logAttribute.Key] = value
			case "response_body":
				value := gjson.GetBytes(body, logAttribute.Value).String()
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, logAttribute.Key, value)
				logAttributes[logAttribute.Key] = value
			default:
				log.Errorf("source type %s is error", source)
			}
		}
	}
	ctx.SetContext("logAttributes", logAttributes)
}

func writeLog(logAttributes map[string]string, log wrapper.Log) {
	items := []string{}
	for k, v := range logAttributes {
		items = append(items, fmt.Sprintf(`"%s":"%s"`, k, v))
	}
	log.Infof("ai request json log: {%s}", strings.Join(items, ","))
}
