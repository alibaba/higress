package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
	// Trace span prefix
	TracePrefix = "trace_span_tag."
	// Context consts
	StatisticsRequestStartTime = "ai-statistics-request-start-time"
	StatisticsFirstTokenTime   = "ai-statistics-first-token-time"
	CtxGeneralAtrribute        = "attributes"
	CtxLogAtrribute            = "logAttributes"
	CtxStreamingBodyBuffer     = "streamingBodyBuffer"

	// Source Type
	FixedValue            = "fixed_value"
	RequestHeader         = "request_header"
	RequestBody           = "request_body"
	ResponseHeader        = "response_header"
	ResponseStreamingBody = "response_streaming_body"
	ResponseBody          = "response_body"

	// Inner metric & log attributes name
	Model                 = "model"
	InputToken            = "input_token"
	OutputToken           = "output_token"
	LLMFirstTokenDuration = "llm_first_token_duration"
	LLMServiceDuration    = "llm_service_duration"
	LLMDurationCount      = "llm_duration_count"

	// Extract Rule
	RuleFirst   = "first"
	RuleReplace = "replace"
	RuleAppend  = "append"
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
		if attribute.ValueSource == ResponseStreamingBody {
			config.shouldBufferStreamingBody = true
		}
		if attribute.Rule != "" && attribute.Rule != RuleFirst && attribute.Rule != RuleReplace && attribute.Rule != RuleAppend {
			return errors.New("value of rule must be one of [nil, first, replace, append]")
		}
		config.attributes[i] = attribute
	}
	// Metric settings
	config.counterMetrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	ctx.SetContext(CtxGeneralAtrribute, map[string]string{})
	ctx.SetContext(CtxLogAtrribute, map[string]string{})
	ctx.SetContext(StatisticsRequestStartTime, time.Now().UnixMilli())

	// Set user defined log & span attributes which type is fixed_value
	setAttributeBySource(ctx, config, FixedValue, nil, log)
	// Set user defined log & span attributes which type is request_header
	setAttributeBySource(ctx, config, RequestHeader, nil, log)
	// Set request start time.

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	// Set user defined log & span attributes.
	setAttributeBySource(ctx, config, RequestBody, body, log)
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if !strings.Contains(contentType, "text/event-stream") {
		ctx.BufferResponseBody()
	}

	// Set user defined log & span attributes.
	setAttributeBySource(ctx, config, ResponseHeader, nil, log)

	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	// Buffer stream body for record log & span attributes
	if config.shouldBufferStreamingBody {
		var streamingBodyBuffer []byte
		streamingBodyBuffer, ok := ctx.GetContext(CtxStreamingBodyBuffer).([]byte)
		if !ok {
			streamingBodyBuffer = data
		} else {
			streamingBodyBuffer = append(streamingBodyBuffer, data...)
		}
		ctx.SetContext(CtxStreamingBodyBuffer, streamingBodyBuffer)
	}

	// Get requestStartTime from http context
	requestStartTime, ok := ctx.GetContext(StatisticsRequestStartTime).(int64)
	if !ok {
		log.Error("failed to get requestStartTime from http context")
		return data
	}

	// If this is the first chunk, record first token duration metric and span attribute
	if ctx.GetContext(StatisticsFirstTokenTime) == nil {
		firstTokenTime := time.Now().UnixMilli()
		ctx.SetContext(StatisticsFirstTokenTime, firstTokenTime)
		attributes, _ := ctx.GetContext(CtxGeneralAtrribute).(map[string]string)
		attributes[LLMFirstTokenDuration] = fmt.Sprint(firstTokenTime - requestStartTime)
		ctx.SetContext(CtxGeneralAtrribute, attributes)
	}

	// Set information about this request

	if model, inputToken, outputToken, ok := getUsage(data); ok {
		attributes, _ := ctx.GetContext(CtxGeneralAtrribute).(map[string]string)
		// Record Log Attributes
		attributes[Model] = model
		attributes[InputToken] = fmt.Sprint(inputToken)
		attributes[OutputToken] = fmt.Sprint(outputToken)
		// Set attributes to http context
		ctx.SetContext(CtxGeneralAtrribute, attributes)
	}
	// If the end of the stream is reached, record metrics/logs/spans.
	if endOfStream {
		responseEndTime := time.Now().UnixMilli()
		attributes, _ := ctx.GetContext(CtxGeneralAtrribute).(map[string]string)
		attributes[LLMServiceDuration] = fmt.Sprint(responseEndTime - requestStartTime)
		ctx.SetContext(CtxGeneralAtrribute, attributes)

		// Set user defined log & span attributes.
		if config.shouldBufferStreamingBody {
			streamingBodyBuffer, ok := ctx.GetContext(CtxStreamingBodyBuffer).([]byte)
			if !ok {
				return data
			}
			setAttributeBySource(ctx, config, ResponseStreamingBody, streamingBodyBuffer, log)
		}

		// Write inner filter states which can be used by other plugins such as ai-token-ratelimit
		writeFilterStates(ctx, log)

		// Write log
		writeLog(ctx, log)

		// Write metrics
		writeMetric(ctx, config, log)
	}
	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log wrapper.Log) types.Action {
	// Get attributes from http context
	attributes, _ := ctx.GetContext(CtxGeneralAtrribute).(map[string]string)

	// Get requestStartTime from http context
	requestStartTime, _ := ctx.GetContext(StatisticsRequestStartTime).(int64)

	responseEndTime := time.Now().UnixMilli()
	attributes[LLMServiceDuration] = fmt.Sprint(responseEndTime - requestStartTime)

	// Set information about this request
	model, inputToken, outputToken, ok := getUsage(body)
	if ok {
		attributes[Model] = model
		attributes[InputToken] = fmt.Sprint(inputToken)
		attributes[OutputToken] = fmt.Sprint(outputToken)
		// Update attributes
		ctx.SetContext(CtxGeneralAtrribute, attributes)
	}

	// Set user defined log & span attributes.
	setAttributeBySource(ctx, config, ResponseBody, body, log)

	// Write inner filter states which can be used by other plugins such as ai-token-ratelimit
	writeFilterStates(ctx, log)

	// Write log
	writeLog(ctx, log)

	// Write metrics
	writeMetric(ctx, config, log)

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
	attributes, ok := ctx.GetContext(CtxGeneralAtrribute).(map[string]string)
	if !ok {
		log.Error("failed to get attributes from http context")
		return
	}
	for _, attribute := range config.attributes {
		if source == attribute.ValueSource {
			switch source {
			case FixedValue:
				log.Debugf("[attribute] source type: %s, key: %s, value: %s", source, attribute.Key, attribute.Value)
				attributes[attribute.Key] = attribute.Value
			case RequestHeader:
				if value, err := proxywasm.GetHttpRequestHeader(attribute.Value); err == nil {
					log.Debugf("[attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
					attributes[attribute.Key] = value
				}
			case RequestBody:
				raw := gjson.GetBytes(body, attribute.Value).Raw
				var value string
				if len(raw) > 2 {
					value = raw[1 : len(raw)-1]
				}
				log.Debugf("[attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
				attributes[attribute.Key] = value
			case ResponseHeader:
				if value, err := proxywasm.GetHttpResponseHeader(attribute.Value); err == nil {
					log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
					attributes[attribute.Key] = value
				}
			case ResponseStreamingBody:
				value := extractStreamingBodyByJsonPath(body, attribute.Value, attribute.Rule, log)
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
				attributes[attribute.Key] = value
			case ResponseBody:
				value := gjson.GetBytes(body, attribute.Value).Raw
				if len(value) > 2 && value[0] == '"' && value[len(value)-1] == '"' {
					value = value[1 : len(value)-1]
				}
				log.Debugf("[log attribute] source type: %s, key: %s, value: %s", source, attribute.Key, value)
				attributes[attribute.Key] = value
			default:
			}
		}
		if attribute.ApplyToLog {
			setLogAttribute(ctx, attribute.Key, attributes[attribute.Key], log)
		}
		if attribute.ApplyToSpan {
			setSpanAttribute(attribute.Key, attributes[attribute.Key], log)
		}
	}
	ctx.SetContext(CtxGeneralAtrribute, attributes)
}

func extractStreamingBodyByJsonPath(data []byte, jsonPath string, rule string, log wrapper.Log) string {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	var value string
	if rule == RuleFirst {
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				value = jsonObj.String()
				break
			}
		}
	} else if rule == RuleReplace {
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				value = jsonObj.String()
			}
		}
	} else if rule == RuleAppend {
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
	logAttributes, ok := ctx.GetContext(CtxLogAtrribute).(map[string]string)
	if !ok {
		log.Error("failed to get logAttributes from http context")
		return
	}
	logAttributes[key] = fmt.Sprint(value)
	ctx.SetContext(CtxLogAtrribute, logAttributes)
}

func writeFilterStates(ctx wrapper.HttpContext, log wrapper.Log) {
	attributes, _ := ctx.GetContext(CtxGeneralAtrribute).(map[string]string)
	setFilterState(Model, attributes[Model], log)
	setFilterState(InputToken, attributes[InputToken], log)
	setFilterState(OutputToken, attributes[OutputToken], log)
}

func writeMetric(ctx wrapper.HttpContext, config AIStatisticsConfig, log wrapper.Log) {
	attributes, _ := ctx.GetContext(CtxGeneralAtrribute).(map[string]string)
	route, _ := getRouteName()
	cluster, _ := getClusterName()
	model, ok := attributes["model"]
	if !ok {
		log.Errorf("Get model failed")
		return
	}
	if inputToken, ok := attributes[InputToken]; ok {
		inputTokenUint64, err := strconv.ParseUint(inputToken, 10, 0)
		if err != nil || inputTokenUint64 == 0 {
			log.Errorf("inputToken convert failed, value is %d, err msg is [%v]", inputTokenUint64, err)
			return
		}
		config.incrementCounter(generateMetricName(route, cluster, model, InputToken), inputTokenUint64)
	}
	if outputToken, ok := attributes[OutputToken]; ok {
		outputTokenUint64, err := strconv.ParseUint(outputToken, 10, 0)
		if err != nil || outputTokenUint64 == 0 {
			log.Errorf("outputToken convert failed, value is %d, err msg is [%v]", outputTokenUint64, err)
			return
		}
		config.incrementCounter(generateMetricName(route, cluster, model, OutputToken), outputTokenUint64)
	}
	if llmFirstTokenDuration, ok := attributes[LLMFirstTokenDuration]; ok {
		llmFirstTokenDurationUint64, err := strconv.ParseUint(llmFirstTokenDuration, 10, 0)
		if err != nil || llmFirstTokenDurationUint64 == 0 {
			log.Errorf("llmFirstTokenDuration convert failed, value is %d, err msg is [%v]", llmFirstTokenDurationUint64, err)
			return
		}
		config.incrementCounter(generateMetricName(route, cluster, model, LLMFirstTokenDuration), llmFirstTokenDurationUint64)
	}
	if llmServiceDuration, ok := attributes[LLMServiceDuration]; ok {
		llmServiceDurationUint64, err := strconv.ParseUint(llmServiceDuration, 10, 0)
		if err != nil || llmServiceDurationUint64 == 0 {
			log.Errorf("llmServiceDuration convert failed, value is %d, err msg is [%v]", llmServiceDurationUint64, err)
			return
		}
		config.incrementCounter(generateMetricName(route, cluster, model, LLMServiceDuration), llmServiceDurationUint64)
	}
	config.incrementCounter(generateMetricName(route, cluster, model, LLMDurationCount), 1)
}

func writeLog(ctx wrapper.HttpContext, log wrapper.Log) {
	attributes, _ := ctx.GetContext(CtxGeneralAtrribute).(map[string]string)
	logAttributes, _ := ctx.GetContext(CtxLogAtrribute).(map[string]string)
	// Set inner log fields
	if attributes[Model] != "" {
		logAttributes[Model] = attributes[Model]
	}
	if attributes[InputToken] != "" {
		logAttributes[InputToken] = attributes[InputToken]
	}
	if attributes[OutputToken] != "" {
		logAttributes[OutputToken] = attributes[OutputToken]
	}
	if attributes[LLMFirstTokenDuration] != "" {
		logAttributes[LLMFirstTokenDuration] = attributes[LLMFirstTokenDuration]
	}
	if attributes[LLMServiceDuration] != "" {
		logAttributes[LLMServiceDuration] = attributes[LLMServiceDuration]
	}
	// Traverse log fields
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
