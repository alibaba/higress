package main

import (
	"bytes"
	"encoding/json"
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
	Key         string `json:"key"`
	ValueSource string `json:"value_source"`
	Value       string `json:"value"`
	Rule        string `json:"rule,omitempty"`
	ApplyToLog  bool   `json:"apply_to_log,omitempty"`
	ApplyToSpan bool   `json:"apply_to_span,omitempty"`
}

type AIStatisticsConfig struct {
	// Metrics
	// TODO: add more metrics in Gauge and Histogram format
	counterMetrics map[string]proxywasm.MetricCounter
	// Attributes to be recorded in log & span
	attributes []Attribute
	// If there exist attributes extracted from streaming body, chunks should be buffered
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
	attributeConfigs := configJson.Get("attributes").Array()
	config.attributes = make([]Attribute, len(attributeConfigs))
	for i, attributeConfig := range attributeConfigs {
		attribute := Attribute{}
		err := json.Unmarshal([]byte(attributeConfig.Raw), &attribute)
		if err != nil {
			log.Errorf("parse config failed, %v", err)
			return err
		}
		if attribute.ValueSource == "response_streaming_body" {
			config.shouldBufferStreamingBody = true
		}
		log.Infof("%v", attribute)
		config.attributes[i] = attribute
	}
	// Metric settings
	config.counterMetrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	ctx.SetContext("attributes", map[string]string{})
	ctx.SetContext("logAttributes", map[string]string{})
	// Set user defined log & span attributes which type is fixed_value
	setAttributeBySource(ctx, config, "fixed_value", nil, log)
	// Set user defined log & span attributes which type is request_header
	setAttributeBySource(ctx, config, "request_header", nil, log)
	// Set request start time.
	ctx.SetContext(StatisticsRequestStartTime, time.Now().UnixMilli())

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	// Set user defined log & span attributes.
	setAttributeBySource(ctx, config, "request_body", body, log)
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if !strings.Contains(contentType, "text/event-stream") {
		ctx.BufferResponseBody()
	}

	// Set user defined log & span attributes.
	setAttributeBySource(ctx, config, "response_header", nil, log)

	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	// Get attributes from http context
	attributes, ok := ctx.GetContext("attributes").(map[string]string)
	if !ok {
		log.Error("failed to get attributes from http context")
		return data
	}

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
	}

	// Set infomations about this request
	route, _ := getRouteName()
	cluster, _ := getClusterName()
	if model, inputToken, outputToken, ok := getUsage(data); ok {
		// Record Log Attributes
		attributes["model"] = model
		attributes["input_token"] = fmt.Sprint(inputToken)
		attributes["output_token"] = fmt.Sprint(outputToken)
		// Set attributes to http context
		ctx.SetContext("attributes", attributes)
	}
	// If the end of the stream is reached, record metrics/logs/spans.
	if endOfStream {
		responseEndTime := time.Now().UnixMilli()
		llm_first_token_duration := uint64(firstTokenTime - requestStartTime)
		llm_service_duration := uint64(responseEndTime - requestStartTime)

		// Set user defined log & span attributes.
		if config.shouldBufferStreamingBody {
			streamingBodyBuffer, ok := ctx.GetContext("streamingBodyBuffer").([]byte)
			if !ok {
				return data
			}
			setAttributeBySource(ctx, config, "response_streaming_body", streamingBodyBuffer, log)
		}
		// Get updated(maybe) attributes from http context
		attributes, _ := ctx.GetContext("attributes").(map[string]string)

		// Inner filter states which can be used by other plugins such as ai-token-ratelimit
		setFilterState("model", attributes["model"], log)
		setFilterState("input_token", attributes["input_token"], log)
		setFilterState("output_token", attributes["output_token"], log)

		// Inner log attribute
		setLogAttribute(ctx, "model", attributes["model"], log)
		setLogAttribute(ctx, "input_token", attributes["input_token"], log)
		setLogAttribute(ctx, "output_token", attributes["output_token"], log)
		setLogAttribute(ctx, "llm_first_token_duration", llm_first_token_duration, log)
		setLogAttribute(ctx, "llm_service_duration", llm_service_duration, log)

		// Write log
		writeLog(ctx, log)

		// Set metrics
		inputTokenUint64, err := strconv.ParseUint(attributes["input_token"], 10, 0)
		if err != nil || inputTokenUint64 == 0 {
			log.Errorf("input_token convert failed, value is %d, err msg is [%v]", inputTokenUint64, err)
			return data
		}
		outputTokenUint64, err := strconv.ParseUint(attributes["output_token"], 10, 0)
		if err != nil || outputTokenUint64 == 0 {
			log.Errorf("output_token convert failed, value is %d, err msg is [%v]", outputTokenUint64, err)
			return data
		}
		config.incrementCounter(generateMetricName(route, cluster, attributes["model"], "input_token"), inputTokenUint64)
		config.incrementCounter(generateMetricName(route, cluster, attributes["model"], "output_token"), outputTokenUint64)
		config.incrementCounter(generateMetricName(route, cluster, attributes["model"], "llm_first_token_duration"), llm_first_token_duration)
		config.incrementCounter(generateMetricName(route, cluster, attributes["model"], "llm_service_duration"), llm_service_duration)
		config.incrementCounter(generateMetricName(route, cluster, attributes["model"], "llm_duration_count"), 1)
	}
	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	// Get attributes from http context
	attributes, ok := ctx.GetContext("attributes").(map[string]string)
	if !ok {
		log.Error("failed to get attributes from http context")
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
	if ok {
		attributes["model"] = model
		attributes["input_token"] = fmt.Sprint(inputToken)
		attributes["output_token"] = fmt.Sprint(outputToken)
		// Update attributes
		ctx.SetContext("attributes", attributes)
	}

	// Set user defined log & span attributes.
	setAttributeBySource(ctx, config, "response_body", body, log)

	// Get updated(maybe) attributes from http context
	attributes, _ = ctx.GetContext("attributes").(map[string]string)

	// Inner filter states which can be used by other plugins such as ai-token-ratelimit
	setFilterState("model", attributes["model"], log)
	setFilterState("input_token", attributes["input_token"], log)
	setFilterState("output_token", attributes["output_token"], log)

	// Inner log attribute
	setLogAttribute(ctx, "model", attributes["model"], log)
	setLogAttribute(ctx, "input_token", attributes["input_token"], log)
	setLogAttribute(ctx, "output_token", attributes["output_token"], log)
	setLogAttribute(ctx, "llm_service_duration", llm_service_duration, log)

	// Write log
	writeLog(ctx, log)

	// Set metrics
	inputTokenUint64, err := strconv.ParseUint(attributes["input_token"], 10, 0)
	if err != nil || inputTokenUint64 == 0 {
		log.Errorf("input_token convert failed, value is %d, err msg is [%v]", inputTokenUint64, err)
		return types.ActionContinue
	}
	outputTokenUint64, err := strconv.ParseUint(attributes["output_token"], 10, 0)
	if err != nil || outputTokenUint64 == 0 {
		log.Errorf("output_token convert failed, value is %d, err msg is [%v]", outputTokenUint64, err)
		return types.ActionContinue
	}
	config.incrementCounter(generateMetricName(route, cluster, attributes["model"], "input_token"), inputTokenUint64)
	config.incrementCounter(generateMetricName(route, cluster, attributes["model"], "output_token"), outputTokenUint64)
	config.incrementCounter(generateMetricName(route, cluster, attributes["model"], "llm_service_duration"), llm_service_duration)
	config.incrementCounter(generateMetricName(route, cluster, attributes["model"], "llm_duration_count"), 1)

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

// fetches the tracing span value from the specified source.
func setAttributeBySource(ctx wrapper.HttpContext, config AIStatisticsConfig, source string, body []byte, log wrapper.Log) {
	attributes, ok := ctx.GetContext("attributes").(map[string]string)
	if !ok {
		log.Error("failed to get attributes from http context")
		return
	}
	for _, attribute := range config.attributes {
		if source == attribute.ValueSource {
			switch source {
			case "fixed_value":
				log.Debugf("[attribute] source type: %s, key: %s, value: %s", source, attribute.Key, attribute.Value)
				attributes[attribute.Key] = attribute.Value
			case "request_header":
				if value, err := proxywasm.GetHttpRequestHeader(attribute.Value); err == nil {
					log.Debugf("[attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
					attributes[attribute.Key] = value
				}
			case "request_body":
				raw := gjson.GetBytes(body, attribute.Value).Raw
				var value string
				if len(raw) > 2 {
					value = raw[1 : len(raw)-1]
				}
				log.Debugf("[attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
				attributes[attribute.Key] = value
			case "response_header":
				if value, err := proxywasm.GetHttpResponseHeader(attribute.Value); err == nil {
					log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
					attributes[attribute.Key] = value
				}
			case "response_streaming_body":
				value := extractStreamingBodyByJsonPath(body, attribute.Value, attribute.Rule, log)
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
				attributes[attribute.Key] = value
			case "response_body":
				value := gjson.GetBytes(body, attribute.Value).Raw
				if len(value) > 2 && value[0] == '"' && value[len(value)-1] == '"' {
					value = value[1 : len(value)-1]
				}
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
				attributes[attribute.Key] = value
			default:
				log.Errorf("source type %s is error", source)
			}
		}
		if attribute.ApplyToLog {
			setLogAttribute(ctx, attribute.Key, attributes[attribute.Key], log)
		}
		if attribute.ApplyToSpan {
			setSpanAttribute(attribute.Key, attributes[attribute.Key], log)
		}
	}
	ctx.SetContext("attributes", attributes)
}

func extractStreamingBodyByJsonPath(data []byte, jsonPath string, rule string, log wrapper.Log) string {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	var value string
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
			if len(raw) > 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
				value += raw[1 : len(raw)-1]
			}
		}
	} else {
		log.Errorf("unsupported rule type: %s", rule)
	}
	return value
}

func setFilterState(key, value string, log wrapper.Log) {
	if value != "" {
		if e := proxywasm.SetProperty([]string{key}, []byte(fmt.Sprint(value))); e != nil {
			log.Errorf("failed to set %s in filter state: %v", key, e)
		}
	} else {
		log.Debugf("failed to write filter state [%s], because it's value is empty")
	}
}

// Set the tracing span with value.
func setSpanAttribute(key, value string, log wrapper.Log) {
	if value != "" {
		traceSpanTag := TracePrefix + key
		if e := proxywasm.SetProperty([]string{traceSpanTag}, []byte(value)); e != nil {
			log.Errorf("failed to set %s in filter state: %v", traceSpanTag, e)
		}
	} else {
		log.Debugf("failed to write span attribute [%s], because it's value is empty")
	}
}

// fetches the tracing span value from the specified source.
func setLogAttribute(ctx wrapper.HttpContext, key string, value interface{}, log wrapper.Log) {
	logAttributes, ok := ctx.GetContext("logAttributes").(map[string]string)
	if !ok {
		log.Error("failed to get logAttributes from http context")
		return
	}
	logAttributes[key] = fmt.Sprint(value)
	ctx.SetContext("logAttributes", logAttributes)
}

func writeLog(ctx wrapper.HttpContext, log wrapper.Log) {
	logAttributes, ok := ctx.GetContext("logAttributes").(map[string]string)
	if !ok {
		log.Error("failed to write log")
	}
	items := []string{}
	for k, v := range logAttributes {
		items = append(items, fmt.Sprintf(`"%s":"%s"`, k, v))
	}
	aiLogField := fmt.Sprintf(`{%s}`, strings.Join(items, ","))
	// log.Infof("ai request json log: %s", aiLogField)
	jsonMap := map[string]string{
		"ai_log": aiLogField,
	}
	serialized, _ := json.Marshal(jsonMap)
	jsonLogRaw := gjson.GetBytes(serialized, "ai_log").Raw
	jsonLog := jsonLogRaw[1 : len(jsonLogRaw)-1]
	if err := proxywasm.SetProperty([]string{"ai_log"}, []byte(jsonLog)); err != nil {
		log.Errorf("failed to set ai_log in filter state: %v", err)
	}
}
