package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
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
	defaultMaxBodyBytes uint32 = 100 * 1024 * 1024
	// Context consts
	StatisticsRequestStartTime = "ai-statistics-request-start-time"
	StatisticsFirstTokenTime   = "ai-statistics-first-token-time"
	CtxGeneralAtrribute        = "attributes"
	CtxLogAtrribute            = "logAttributes"
	CtxStreamingBodyBuffer     = "streamingBodyBuffer"
	RouteName                  = "route"
	ClusterName                = "cluster"
	APIName                    = "api"
	ConsumerKey                = "x-mse-consumer"

	// Source Type
	FixedValue            = "fixed_value"
	RequestHeader         = "request_header"
	RequestBody           = "request_body"
	ResponseHeader        = "response_header"
	ResponseStreamingBody = "response_streaming_body"
	ResponseBody          = "response_body"

	// Inner metric & log attributes
	Model                  = "model"
	InputToken             = "input_token"
	OutputToken            = "output_token"
	LLMFirstTokenDuration  = "llm_first_token_duration"
	LLMServiceDuration     = "llm_service_duration"
	LLMDurationCount       = "llm_duration_count"
	LLMStreamDurationCount = "llm_stream_duration_count"
	ResponseType           = "response_type"
	ChatID                 = "chat_id"
	ChatRound              = "chat_round"

	// Inner span attributes
	ArmsSpanKind     = "gen_ai.span.kind"
	ArmsModelName    = "gen_ai.model_name"
	ArmsRequestModel = "gen_ai.request.model"
	ArmsInputToken   = "gen_ai.usage.input_tokens"
	ArmsOutputToken  = "gen_ai.usage.output_tokens"
	ArmsTotalToken   = "gen_ai.usage.total_tokens"

	// Extract Rule
	RuleFirst   = "first"
	RuleReplace = "replace"
	RuleAppend  = "append"
)

// TracingSpan is the tracing span configuration.
type Attribute struct {
	Key                string `json:"key"`
	ValueSource        string `json:"value_source"`
	Value              string `json:"value"`
	TraceSpanKey       string `json:"trace_span_key,omitempty"`
	DefaultValue       string `json:"default_value,omitempty"`
	Rule               string `json:"rule,omitempty"`
	ApplyToLog         bool   `json:"apply_to_log,omitempty"`
	ApplyToSpan        bool   `json:"apply_to_span,omitempty"`
	AsSeparateLogField bool   `json:"as_separate_log_field,omitempty"`
}

type AIStatisticsConfig struct {
	// Metrics
	// TODO: add more metrics in Gauge and Histogram format
	counterMetrics map[string]proxywasm.MetricCounter
	// Attributes to be recorded in log & span
	attributes []Attribute
	// If there exist attributes extracted from streaming body, chunks should be buffered
	shouldBufferStreamingBody bool
	// If disableOpenaiUsage is true, model/input_token/output_token logs will be skipped
	disableOpenaiUsage bool
}

func generateMetricName(route, cluster, model, consumer, metricName string) string {
	return fmt.Sprintf("route.%s.upstream.%s.model.%s.consumer.%s.metric.%s", route, cluster, model, consumer, metricName)
}

func getRouteName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		return "-", err
	} else {
		return string(raw), nil
	}
}

func getAPIName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		return "-", err
	} else {
		parts := strings.Split(string(raw), "@")
		if len(parts) != 5 {
			return "-", errors.New("not api type")
		} else {
			return strings.Join(parts[:3], "@"), nil
		}
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
	if inc == 0 {
		return
	}
	counter, ok := config.counterMetrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.counterMetrics[metricName] = counter
	}
	counter.Increment(inc)
}

func parseConfig(configJson gjson.Result, config *AIStatisticsConfig, log log.Log) error {
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

	// Parse openai usage config setting.
	config.disableOpenaiUsage = configJson.Get("disable_openai_usage").Bool()

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log log.Log) types.Action {
	route, _ := getRouteName()
	cluster, _ := getClusterName()
	api, api_error := getAPIName()
	if api_error == nil {
		route = api
	}
	ctx.SetContext(RouteName, route)
	ctx.SetContext(ClusterName, cluster)
	ctx.SetUserAttribute(APIName, api)
	ctx.SetContext(StatisticsRequestStartTime, time.Now().UnixMilli())
	if consumer, _ := proxywasm.GetHttpRequestHeader(ConsumerKey); consumer != "" {
		ctx.SetContext(ConsumerKey, consumer)
	}
	hasRequestBody := wrapper.HasRequestBody()
	if hasRequestBody {
		_ = proxywasm.RemoveHttpRequestHeader("Content-Length")
		ctx.SetRequestBodyBufferLimit(defaultMaxBodyBytes)
	}

	// Set user defined log & span attributes which type is fixed_value
	setAttributeBySource(ctx, config, FixedValue, nil, log)
	// Set user defined log & span attributes which type is request_header
	setAttributeBySource(ctx, config, RequestHeader, nil, log)
	// Set span attributes for ARMS.
	setSpanAttribute(ArmsSpanKind, "LLM", log)

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log log.Log) types.Action {
	// Set user defined log & span attributes.
	setAttributeBySource(ctx, config, RequestBody, body, log)
	// Set span attributes for ARMS.
	requestModel := gjson.GetBytes(body, "model").String()
	if requestModel == "" {
		requestModel = "UNKNOWN"
	}
	setSpanAttribute(ArmsRequestModel, requestModel, log)
	// Set the number of conversation rounds
	if gjson.GetBytes(body, "messages").Exists() {
		userPromptCount := 0
		for _, msg := range gjson.GetBytes(body, "messages").Array() {
			if msg.Get("role").String() == "user" {
				userPromptCount += 1
			}
		}
		ctx.SetUserAttribute(ChatRound, userPromptCount)
	}

	// Write log
	ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config AIStatisticsConfig, log log.Log) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if !strings.Contains(contentType, "text/event-stream") {
		ctx.BufferResponseBody()
	}

	// Set user defined log & span attributes.
	setAttributeBySource(ctx, config, ResponseHeader, nil, log)

	return types.ActionContinue
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config AIStatisticsConfig, data []byte, endOfStream bool, log log.Log) []byte {
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

	ctx.SetUserAttribute(ResponseType, "stream")
	chatID := gjson.GetBytes(data, "id").String()
	if chatID != "" {
		ctx.SetUserAttribute(ChatID, chatID)
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
		ctx.SetUserAttribute(LLMFirstTokenDuration, firstTokenTime-requestStartTime)
	}

	// Set information about this request
	if !config.disableOpenaiUsage {
		if model, inputToken, outputToken, ok := getUsage(data); ok {
			ctx.SetUserAttribute(Model, model)
			ctx.SetUserAttribute(InputToken, inputToken)
			ctx.SetUserAttribute(OutputToken, outputToken)
			// Set span attributes for ARMS.
			setSpanAttribute(ArmsModelName, model, log)
			setSpanAttribute(ArmsInputToken, inputToken, log)
			setSpanAttribute(ArmsOutputToken, outputToken, log)
			setSpanAttribute(ArmsTotalToken, inputToken+outputToken, log)
		}
	}
	// If the end of the stream is reached, record metrics/logs/spans.
	if endOfStream {
		responseEndTime := time.Now().UnixMilli()
		ctx.SetUserAttribute(LLMServiceDuration, responseEndTime-requestStartTime)

		// Set user defined log & span attributes.
		if config.shouldBufferStreamingBody {
			streamingBodyBuffer, ok := ctx.GetContext(CtxStreamingBodyBuffer).([]byte)
			if !ok {
				return data
			}
			setAttributeBySource(ctx, config, ResponseStreamingBody, streamingBodyBuffer, log)
		}

		// Write log
		ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)

		// Write metrics
		writeMetric(ctx, config, log)
	}
	return data
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AIStatisticsConfig, body []byte, log log.Log) types.Action {
	// Get requestStartTime from http context
	requestStartTime, _ := ctx.GetContext(StatisticsRequestStartTime).(int64)

	responseEndTime := time.Now().UnixMilli()
	ctx.SetUserAttribute(LLMServiceDuration, responseEndTime-requestStartTime)

	ctx.SetUserAttribute(ResponseType, "normal")
	chatID := gjson.GetBytes(body, "id").String()
	if chatID != "" {
		ctx.SetUserAttribute(ChatID, chatID)
	}

	// Set information about this request
	if !config.disableOpenaiUsage {
		if model, inputToken, outputToken, ok := getUsage(body); ok {
			ctx.SetUserAttribute(Model, model)
			ctx.SetUserAttribute(InputToken, inputToken)
			ctx.SetUserAttribute(OutputToken, outputToken)
			// Set span attributes for ARMS.
			setSpanAttribute(ArmsModelName, model, log)
			setSpanAttribute(ArmsInputToken, inputToken, log)
			setSpanAttribute(ArmsOutputToken, outputToken, log)
			setSpanAttribute(ArmsTotalToken, inputToken+outputToken, log)
		}
	}

	// Set user defined log & span attributes.
	setAttributeBySource(ctx, config, ResponseBody, body, log)

	// Write log
	ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)

	// Write metrics
	writeMetric(ctx, config, log)

	return types.ActionContinue
}

func unifySSEChunk(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	return data
}

func getUsage(data []byte) (model string, inputTokenUsage int64, outputTokenUsage int64, ok bool) {
	chunks := bytes.Split(bytes.TrimSpace(unifySSEChunk(data)), []byte("\n\n"))
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
		if modelObj.Exists() {
			model = modelObj.String()
		} else {
			model = "unknown"
		}
		inputTokenObj := gjson.GetBytes(chunk, "usage.prompt_tokens")
		outputTokenObj := gjson.GetBytes(chunk, "usage.completion_tokens")
		if inputTokenObj.Exists() && outputTokenObj.Exists() {
			inputTokenUsage = inputTokenObj.Int()
			outputTokenUsage = outputTokenObj.Int()
			ok = true
			return
		}
	}
	return
}

// fetches the tracing span value from the specified source.
func setAttributeBySource(ctx wrapper.HttpContext, config AIStatisticsConfig, source string, body []byte, log log.Log) {
	for _, attribute := range config.attributes {
		var key string
		var value interface{}
		if source == attribute.ValueSource {
			key = attribute.Key
			switch source {
			case FixedValue:
				value = attribute.Value
			case RequestHeader:
				value, _ = proxywasm.GetHttpRequestHeader(attribute.Value)
			case RequestBody:
				value = gjson.GetBytes(body, attribute.Value).Value()
			case ResponseHeader:
				value, _ = proxywasm.GetHttpResponseHeader(attribute.Value)
			case ResponseStreamingBody:
				value = extractStreamingBodyByJsonPath(body, attribute.Value, attribute.Rule, log)
			case ResponseBody:
				value = gjson.GetBytes(body, attribute.Value).Value()
			default:
			}
			if (value == nil || value == "") && attribute.DefaultValue != "" {
				value = attribute.DefaultValue
			}
			log.Debugf("[attribute] source type: %s, key: %s, value: %+v", source, key, value)
			if attribute.ApplyToLog {
				if attribute.AsSeparateLogField {
					marshalledJsonStr := wrapper.MarshalStr(fmt.Sprint(value))
					if err := proxywasm.SetProperty([]string{key}, []byte(marshalledJsonStr)); err != nil {
						log.Warnf("failed to set %s in filter state, raw is %s, err is %v", key, marshalledJsonStr, err)
					}
				} else {
					ctx.SetUserAttribute(key, value)
				}
			}
			// for metrics
			if key == Model || key == InputToken || key == OutputToken {
				ctx.SetContext(key, value)
			}
			if attribute.ApplyToSpan {
				if attribute.TraceSpanKey != "" {
					key = attribute.TraceSpanKey
				}
				setSpanAttribute(key, value, log)
			}
		}
	}
}

func extractStreamingBodyByJsonPath(data []byte, jsonPath string, rule string, log log.Log) interface{} {
	chunks := bytes.Split(bytes.TrimSpace(unifySSEChunk(data)), []byte("\n\n"))
	var value interface{}
	if rule == RuleFirst {
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				value = jsonObj.Value()
				break
			}
		}
	} else if rule == RuleReplace {
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				value = jsonObj.Value()
			}
		}
	} else if rule == RuleAppend {
		// extract llm response
		var strValue string
		for _, chunk := range chunks {
			jsonObj := gjson.GetBytes(chunk, jsonPath)
			if jsonObj.Exists() {
				strValue += jsonObj.String()
			}
		}
		value = strValue
	} else {
		log.Errorf("unsupported rule type: %s", rule)
	}
	return value
}

// Set the tracing span with value.
func setSpanAttribute(key string, value interface{}, log log.Log) {
	if value != "" {
		traceSpanTag := wrapper.TraceSpanTagPrefix + key
		if e := proxywasm.SetProperty([]string{traceSpanTag}, []byte(fmt.Sprint(value))); e != nil {
			log.Warnf("failed to set %s in filter state: %v", traceSpanTag, e)
		}
	} else {
		log.Debugf("failed to write span attribute [%s], because it's value is empty")
	}
}

func writeMetric(ctx wrapper.HttpContext, config AIStatisticsConfig, log log.Log) {
	// Generate usage metrics
	var ok bool
	var route, cluster, model string
	var inputToken, outputToken uint64
	consumer := ctx.GetStringContext(ConsumerKey, "none")
	route, ok = ctx.GetContext(RouteName).(string)
	if !ok {
		log.Warnf("RouteName typd assert failed, skip metric record")
		return
	}
	cluster, ok = ctx.GetContext(ClusterName).(string)
	if !ok {
		log.Warnf("ClusterName typd assert failed, skip metric record")
		return
	}

	if config.disableOpenaiUsage {
		return
	}

	if ctx.GetUserAttribute(Model) == nil || ctx.GetUserAttribute(InputToken) == nil || ctx.GetUserAttribute(OutputToken) == nil {
		log.Warnf("get usage information failed, skip metric record")
		return
	}
	model, ok = ctx.GetUserAttribute(Model).(string)
	if !ok {
		log.Warnf("Model typd assert failed, skip metric record")
		return
	}
	inputToken, ok = convertToUInt(ctx.GetUserAttribute(InputToken))
	if !ok {
		log.Warnf("InputToken typd assert failed, skip metric record")
		return
	}
	outputToken, ok = convertToUInt(ctx.GetUserAttribute(OutputToken))
	if !ok {
		log.Warnf("OutputToken typd assert failed, skip metric record")
		return
	}
	if inputToken == 0 || outputToken == 0 {
		log.Warnf("inputToken and outputToken cannot equal to 0, skip metric record")
		return
	}
	config.incrementCounter(generateMetricName(route, cluster, model, consumer, InputToken), inputToken)
	config.incrementCounter(generateMetricName(route, cluster, model, consumer, OutputToken), outputToken)

	// Generate duration metrics
	var llmFirstTokenDuration, llmServiceDuration uint64
	// Is stream response
	if ctx.GetUserAttribute(LLMFirstTokenDuration) != nil {
		llmFirstTokenDuration, ok = convertToUInt(ctx.GetUserAttribute(LLMFirstTokenDuration))
		if !ok {
			log.Warnf("LLMFirstTokenDuration typd assert failed")
			return
		}
		config.incrementCounter(generateMetricName(route, cluster, model, consumer, LLMFirstTokenDuration), llmFirstTokenDuration)
		config.incrementCounter(generateMetricName(route, cluster, model, consumer, LLMStreamDurationCount), 1)
	}
	if ctx.GetUserAttribute(LLMServiceDuration) != nil {
		llmServiceDuration, ok = convertToUInt(ctx.GetUserAttribute(LLMServiceDuration))
		if !ok {
			log.Warnf("LLMServiceDuration typd assert failed")
			return
		}
		config.incrementCounter(generateMetricName(route, cluster, model, consumer, LLMServiceDuration), llmServiceDuration)
		config.incrementCounter(generateMetricName(route, cluster, model, consumer, LLMDurationCount), 1)
	}
}

func convertToUInt(val interface{}) (uint64, bool) {
	switch v := val.(type) {
	case float32:
		return uint64(v), true
	case float64:
		return uint64(v), true
	case int32:
		return uint64(v), true
	case int64:
		return uint64(v), true
	case uint32:
		return uint64(v), true
	case uint64:
		return v, true
	default:
		return 0, false
	}
}
