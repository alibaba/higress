// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wrapper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/matcher"
	"github.com/google/uuid"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

const (
	CustomLogKey       = "custom_log"
	AILogKey           = "ai_log"
	TraceSpanTagPrefix = "trace_span_tag."
	PluginIDKey        = "_plugin_id_"
)

type HttpContext interface {
	Scheme() string
	Host() string
	Path() string
	Method() string
	SetContext(key string, value interface{})
	GetContext(key string) interface{}
	GetBoolContext(key string, defaultValue bool) bool
	GetStringContext(key, defaultValue string) string
	GetByteSliceContext(key string, defaultValue []byte) []byte
	GetUserAttribute(key string) interface{}
	SetUserAttribute(key string, value interface{})
	SetUserAttributeMap(kvmap map[string]interface{})
	GetUserAttributeMap() map[string]interface{}
	// You can call this function to set custom log
	WriteUserAttributeToLog() error
	// You can call this function to set custom log with your specific key
	WriteUserAttributeToLogWithKey(key string) error
	// You can call this function to set custom trace span attribute
	WriteUserAttributeToTrace() error
	// If the onHttpRequestBody handle is not set, the request body will not be read by default
	DontReadRequestBody()
	// If the onHttpResponseBody handle is not set, the request body will not be read by default
	DontReadResponseBody()
	// If the onHttpStreamingRequestBody handle is not set, and the onHttpRequestBody handle is set, the request body will be buffered by default
	BufferRequestBody()
	// If the onHttpStreamingResponseBody handle is not set, and the onHttpResponseBody handle is set, the response body will be buffered by default
	BufferResponseBody()
	// If any request header is changed in onHttpRequestHeaders, envoy will re-calculate the route. Call this function to disable the re-routing.
	// You need to call this before making any header modification operations.
	DisableReroute()
	// Note that this parameter affects the gateway's memory usage！Support setting a maximum buffer size for each request body individually in request phase.
	SetRequestBodyBufferLimit(byteSize uint32)
	// Note that this parameter affects the gateway's memory usage! Support setting a maximum buffer size for each response body individually in response phase.
	SetResponseBodyBufferLimit(byteSize uint32)
	// Make a request to the target service of the current route using the specified URL and header.
	RouteCall(method, url string, headers [][2]string, body []byte, callback RouteResponseCallback) error
}

type oldParseConfigFunc[PluginConfig any] func(json gjson.Result, config *PluginConfig, log log.Log) error
type oldParseRuleConfigFunc[PluginConfig any] func(json gjson.Result, global PluginConfig, config *PluginConfig, log log.Log) error
type oldOnHttpHeadersFunc[PluginConfig any] func(context HttpContext, config PluginConfig, log log.Log) types.Action
type oldOnHttpBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, body []byte, log log.Log) types.Action
type oldOnHttpStreamingBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log log.Log) []byte
type oldOnHttpStreamDoneFunc[PluginConfig any] func(context HttpContext, config PluginConfig, log log.Log)

type ParseConfigFunc[PluginConfig any] func(json gjson.Result, config *PluginConfig) error
type ParseRawConfigFunc[PluginConfig any] func(configBytes []byte, config *PluginConfig) error
type ParseRuleConfigFunc[PluginConfig any] func(json gjson.Result, global PluginConfig, config *PluginConfig) error
type ParseRawRuleConfigFunc[PluginConfig any] func(configBytes []byte, global PluginConfig, config *PluginConfig) error
type onHttpHeadersFunc[PluginConfig any] func(context HttpContext, config PluginConfig) types.Action
type onHttpBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, body []byte) types.Action
type onHttpStreamingBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, chunk []byte, isLastChunk bool) []byte
type onHttpStreamDoneFunc[PluginConfig any] func(context HttpContext, config PluginConfig)

type CommonVmCtx[PluginConfig any] struct {
	types.DefaultVMContext
	pluginName                  string
	log                         log.Log
	hasCustomConfig             bool
	parseConfig                 ParseRawConfigFunc[PluginConfig]
	parseRuleConfig             ParseRawRuleConfigFunc[PluginConfig]
	onHttpRequestHeaders        onHttpHeadersFunc[PluginConfig]
	onHttpRequestBody           onHttpBodyFunc[PluginConfig]
	onHttpStreamingRequestBody  onHttpStreamingBodyFunc[PluginConfig]
	onHttpResponseHeaders       onHttpHeadersFunc[PluginConfig]
	onHttpResponseBody          onHttpBodyFunc[PluginConfig]
	onHttpStreamingResponseBody onHttpStreamingBodyFunc[PluginConfig]
	onHttpStreamDone            onHttpStreamDoneFunc[PluginConfig]
}

type TickFuncEntry struct {
	lastExecuted int64
	tickPeriod   int64
	tickFunc     func()
}

var globalOnTickFuncs []TickFuncEntry = []TickFuncEntry{}

// Registe multiple onTick functions. Parameters include:
// 1) tickPeriod: the execution period of tickFunc, this value should be a multiple of 100
// 2) tickFunc: the function to be executed
//
// You should call this function in parseConfig phase, for example:
//
//	func parseConfig(json gjson.Result, config *HelloWorldConfig, log log.Log) error {
//	  wrapper.RegisteTickFunc(1000, func() { proxywasm.LogInfo("onTick 1s") })
//		 wrapper.RegisteTickFunc(3000, func() { proxywasm.LogInfo("onTick 3s") })
//		 return nil
//	}
func RegisteTickFunc(tickPeriod int64, tickFunc func()) {
	globalOnTickFuncs = append(globalOnTickFuncs, TickFuncEntry{0, tickPeriod, tickFunc})
}

func SetCtx[PluginConfig any](pluginName string, options ...CtxOption[PluginConfig]) {
	proxywasm.SetVMContext(NewCommonVmCtx(pluginName, options...))
}

func SetCtxWithOptions[PluginConfig any](pluginName string, options ...CtxOption[PluginConfig]) {
	proxywasm.SetVMContext(NewCommonVmCtxWithOptions(pluginName, options...))
}

type CtxOption[PluginConfig any] interface {
	Apply(*CommonVmCtx[PluginConfig])
}

type parseConfigOption[PluginConfig any] struct {
	rawF ParseRawConfigFunc[PluginConfig]
	f    ParseConfigFunc[PluginConfig]
	oldF oldParseConfigFunc[PluginConfig]
}

func (o parseConfigOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.rawF != nil {
		ctx.parseConfig = o.rawF
	} else if o.f != nil {
		ctx.parseConfig = func(configBytes []byte, config *PluginConfig) error {
			return o.f(gjson.ParseBytes(configBytes), config)
		}
	} else {
		ctx.parseConfig = func(configBytes []byte, config *PluginConfig) error {
			return o.oldF(gjson.ParseBytes(configBytes), config, ctx.log)
		}
	}
}

// Deprecated: Please use `ParseConfig` instead.
func ParseConfigBy[PluginConfig any](f oldParseConfigFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &parseConfigOption[PluginConfig]{oldF: f}
}

func ParseConfig[PluginConfig any](f ParseConfigFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &parseConfigOption[PluginConfig]{f: f}
}

func ParseRawConfig[PluginConfig any](f ParseRawConfigFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &parseConfigOption[PluginConfig]{rawF: f}
}

type parseOverrideConfigOption[PluginConfig any] struct {
	parseRawConfigF     ParseRawConfigFunc[PluginConfig]
	parseRawRuleConfigF ParseRawRuleConfigFunc[PluginConfig]
	parseConfigF        ParseConfigFunc[PluginConfig]
	parseRuleConfigF    ParseRuleConfigFunc[PluginConfig]
	oldParseConfigF     oldParseConfigFunc[PluginConfig]
	oldParseRuleConfigF oldParseRuleConfigFunc[PluginConfig]
}

func (o *parseOverrideConfigOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.parseRawConfigF != nil && o.parseRawRuleConfigF != nil {
		ctx.parseConfig = o.parseRawConfigF
		ctx.parseRuleConfig = o.parseRawRuleConfigF
	} else if o.parseConfigF != nil && o.parseRuleConfigF != nil {
		ctx.parseConfig = func(configBytes []byte, config *PluginConfig) error {
			return o.parseConfigF(gjson.ParseBytes(configBytes), config)
		}
		ctx.parseRuleConfig = func(configBytes []byte, global PluginConfig, config *PluginConfig) error {
			return o.parseRuleConfigF(gjson.ParseBytes(configBytes), global, config)
		}
	} else {
		ctx.parseConfig = func(configBytes []byte, config *PluginConfig) error {
			return o.oldParseConfigF(gjson.ParseBytes(configBytes), config, ctx.log)
		}
		ctx.parseRuleConfig = func(configBytes []byte, global PluginConfig, config *PluginConfig) error {
			return o.oldParseRuleConfigF(gjson.ParseBytes(configBytes), global, config, ctx.log)
		}
	}
}

// Deprecated: Please use `ParseOverrideConfig` instead.
func ParseOverrideConfigBy[PluginConfig any](f oldParseConfigFunc[PluginConfig], g oldParseRuleConfigFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &parseOverrideConfigOption[PluginConfig]{
		oldParseConfigF:     f,
		oldParseRuleConfigF: g,
	}
}

func ParseOverrideConfig[PluginConfig any](f ParseConfigFunc[PluginConfig], g ParseRuleConfigFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &parseOverrideConfigOption[PluginConfig]{
		parseConfigF:     f,
		parseRuleConfigF: g,
	}
}

func ParseOverrideRawConfig[PluginConfig any](f ParseRawConfigFunc[PluginConfig], g ParseRawRuleConfigFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &parseOverrideConfigOption[PluginConfig]{
		parseRawConfigF:     f,
		parseRawRuleConfigF: g,
	}
}

type onProcessRequestHeadersOption[PluginConfig any] struct {
	f    onHttpHeadersFunc[PluginConfig]
	oldF oldOnHttpHeadersFunc[PluginConfig]
}

func (o *onProcessRequestHeadersOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.f != nil {
		ctx.onHttpRequestHeaders = o.f
	} else {
		ctx.onHttpRequestHeaders = func(context HttpContext, config PluginConfig) types.Action {
			return o.oldF(context, config, ctx.log)
		}
	}
}

// Deprecated: Please use `ProcessRequestHeaders` instead.
func ProcessRequestHeadersBy[PluginConfig any](f oldOnHttpHeadersFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessRequestHeadersOption[PluginConfig]{oldF: f}
}

func ProcessRequestHeaders[PluginConfig any](f onHttpHeadersFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessRequestHeadersOption[PluginConfig]{f: f}
}

type onProcessRequestBodyOption[PluginConfig any] struct {
	f    onHttpBodyFunc[PluginConfig]
	oldF oldOnHttpBodyFunc[PluginConfig]
}

func (o *onProcessRequestBodyOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.f != nil {
		ctx.onHttpRequestBody = o.f
	} else {
		ctx.onHttpRequestBody = func(context HttpContext, config PluginConfig, body []byte) types.Action {
			return o.oldF(context, config, body, ctx.log)
		}
	}
}

// Deprecated: Please use `ProcessRequestBody` instead.
func ProcessRequestBodyBy[PluginConfig any](f oldOnHttpBodyFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessRequestBodyOption[PluginConfig]{oldF: f}
}

func ProcessRequestBody[PluginConfig any](f onHttpBodyFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessRequestBodyOption[PluginConfig]{f: f}
}

type onProcessStreamingRequestBodyOption[PluginConfig any] struct {
	f    onHttpStreamingBodyFunc[PluginConfig]
	oldF oldOnHttpStreamingBodyFunc[PluginConfig]
}

func (o *onProcessStreamingRequestBodyOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.f != nil {
		ctx.onHttpStreamingRequestBody = o.f
	} else {
		ctx.onHttpStreamingRequestBody = func(context HttpContext, config PluginConfig, chunk []byte, isLastChunk bool) []byte {
			return o.oldF(context, config, chunk, isLastChunk, ctx.log)
		}
	}
}

// Deprecated: Please use `ProcessStreamingRequestBody` instead.
func ProcessStreamingRequestBodyBy[PluginConfig any](f oldOnHttpStreamingBodyFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessStreamingRequestBodyOption[PluginConfig]{oldF: f}
}

func ProcessStreamingRequestBody[PluginConfig any](f onHttpStreamingBodyFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessStreamingRequestBodyOption[PluginConfig]{f: f}
}

type onProcessResponseHeadersOption[PluginConfig any] struct {
	f    onHttpHeadersFunc[PluginConfig]
	oldF oldOnHttpHeadersFunc[PluginConfig]
}

func (o *onProcessResponseHeadersOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.f != nil {
		ctx.onHttpResponseHeaders = o.f
	} else {
		ctx.onHttpResponseHeaders = func(context HttpContext, config PluginConfig) types.Action {
			return o.oldF(context, config, ctx.log)
		}
	}
}

// Deprecated: Please use `ProcessResponseHeaders` instead.
func ProcessResponseHeadersBy[PluginConfig any](f oldOnHttpHeadersFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessResponseHeadersOption[PluginConfig]{oldF: f}
}

func ProcessResponseHeaders[PluginConfig any](f onHttpHeadersFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessResponseHeadersOption[PluginConfig]{f: f}
}

type onProcessResponseBodyOption[PluginConfig any] struct {
	f    onHttpBodyFunc[PluginConfig]
	oldF oldOnHttpBodyFunc[PluginConfig]
}

func (o *onProcessResponseBodyOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.f != nil {
		ctx.onHttpResponseBody = o.f
	} else {
		ctx.onHttpResponseBody = func(context HttpContext, config PluginConfig, body []byte) types.Action {
			return o.oldF(context, config, body, ctx.log)
		}
	}
}

// Deprecated: Please use `ProcessResponseBody` instead.
func ProcessResponseBodyBy[PluginConfig any](f oldOnHttpBodyFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessResponseBodyOption[PluginConfig]{oldF: f}
}

func ProcessResponseBody[PluginConfig any](f onHttpBodyFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessResponseBodyOption[PluginConfig]{f: f}
}

type onProcessStreamingResponseBodyOption[PluginConfig any] struct {
	f    onHttpStreamingBodyFunc[PluginConfig]
	oldF oldOnHttpStreamingBodyFunc[PluginConfig]
}

func (o *onProcessStreamingResponseBodyOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.f != nil {
		ctx.onHttpStreamingResponseBody = o.f
	} else {
		ctx.onHttpStreamingResponseBody = func(context HttpContext, config PluginConfig, chunk []byte, isLastChunk bool) []byte {
			return o.oldF(context, config, chunk, isLastChunk, ctx.log)
		}
	}
}

// Deprecated: Please use `ProcessStreamingResponseBody` instead.
func ProcessStreamingResponseBodyBy[PluginConfig any](f oldOnHttpStreamingBodyFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessStreamingResponseBodyOption[PluginConfig]{oldF: f}
}

func ProcessStreamingResponseBody[PluginConfig any](f onHttpStreamingBodyFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessStreamingResponseBodyOption[PluginConfig]{f: f}
}

type onProcessStreamDoneOption[PluginConfig any] struct {
	f    onHttpStreamDoneFunc[PluginConfig]
	oldF oldOnHttpStreamDoneFunc[PluginConfig]
}

func (o *onProcessStreamDoneOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.f != nil {
		ctx.onHttpStreamDone = o.f
	} else {
		ctx.onHttpStreamDone = func(context HttpContext, config PluginConfig) { o.oldF(context, config, ctx.log) }
	}

}

// Deprecated: Please use `ProcessStreamDoneBy` instead.
func ProcessStreamDoneBy[PluginConfig any](f oldOnHttpStreamDoneFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessStreamDoneOption[PluginConfig]{oldF: f}
}

func ProcessStreamDone[PluginConfig any](f onHttpStreamDoneFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &onProcessStreamDoneOption[PluginConfig]{f: f}
}

type logOption[PluginConfig any] struct {
	logger log.Log
}

func (o *logOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	log.SetPluginLog(o.logger)
	ctx.log = o.logger
}

func WithLogger[PluginConfig any](logger log.Log) CtxOption[PluginConfig] {
	return &logOption[PluginConfig]{logger}
}

func parseEmptyPluginConfig[PluginConfig any]([]byte, *PluginConfig) error {
	return nil
}

func NewCommonVmCtx[PluginConfig any](pluginName string, options ...CtxOption[PluginConfig]) *CommonVmCtx[PluginConfig] {
	logger := &DefaultLog{pluginName, "nil"}
	opts := []CtxOption[PluginConfig]{WithLogger[PluginConfig](logger)}
	for _, opt := range options {
		if opt == nil {
			continue
		}
		opts = append(opts, opt)
	}
	return NewCommonVmCtxWithOptions(pluginName, opts...)
}

func NewCommonVmCtxWithOptions[PluginConfig any](pluginName string, options ...CtxOption[PluginConfig]) *CommonVmCtx[PluginConfig] {
	ctx := &CommonVmCtx[PluginConfig]{
		pluginName:      pluginName,
		hasCustomConfig: true,
	}
	for _, opt := range options {
		opt.Apply(ctx)
	}
	if ctx.parseConfig == nil {
		var config PluginConfig
		if unsafe.Sizeof(config) != 0 {
			msg := "the `parseConfig` is missing in NewCommonVmCtx's arguments"
			panic(msg)
		}
		ctx.hasCustomConfig = false
		ctx.parseConfig = parseEmptyPluginConfig[PluginConfig]
	}
	return ctx
}

func (ctx *CommonVmCtx[PluginConfig]) NewPluginContext(uint32) types.PluginContext {
	return &CommonPluginCtx[PluginConfig]{
		vm: ctx,
	}
}

type CommonPluginCtx[PluginConfig any] struct {
	types.DefaultPluginContext
	matcher.RuleMatcher[PluginConfig]
	vm          *CommonVmCtx[PluginConfig]
	onTickFuncs []TickFuncEntry
}

func (ctx *CommonPluginCtx[PluginConfig]) OnPluginStart(int) types.OnPluginStartStatus {
	data, err := proxywasm.GetPluginConfiguration()
	globalOnTickFuncs = nil
	if err != nil && err != types.ErrorStatusNotFound {
		ctx.vm.log.Criticalf("error reading plugin configuration: %v", err)
		return types.OnPluginStartStatusFailed
	}
	var jsonData gjson.Result
	if len(data) == 0 {
		if ctx.vm.hasCustomConfig {
			ctx.vm.log.Warn("config is empty, but has ParseConfigFunc")
		}
	} else {
		if !gjson.ValidBytes(data) {
			ctx.vm.log.Warnf("the plugin configuration is not a valid json: %s", string(data))
			return types.OnPluginStartStatusFailed

		}
		jsonData = gjson.ParseBytes(data)
	}
	pluginID := jsonData.Get(PluginIDKey).String()
	if pluginID != "" {
		ctx.vm.log.ResetID(pluginID)
	}
	var parseOverrideConfig func(gjson.Result, PluginConfig, *PluginConfig) error
	if ctx.vm.parseRuleConfig != nil {
		parseOverrideConfig = func(js gjson.Result, global PluginConfig, cfg *PluginConfig) error {
			return ctx.vm.parseRuleConfig([]byte(js.Raw), global, cfg)
		}
	}
	err = ctx.ParseRuleConfig(jsonData,
		func(js gjson.Result, cfg *PluginConfig) error {
			return ctx.vm.parseConfig([]byte(js.Raw), cfg)
		},
		parseOverrideConfig,
	)
	if err != nil {
		ctx.vm.log.Warnf("parse rule config failed: %v", err)
		ctx.vm.log.Error("plugin start failed")
		return types.OnPluginStartStatusFailed
	}
	if globalOnTickFuncs != nil {
		ctx.onTickFuncs = globalOnTickFuncs
		if err := proxywasm.SetTickPeriodMilliSeconds(100); err != nil {
			ctx.vm.log.Error("SetTickPeriodMilliSeconds failed, onTick functions will not take effect.")
			ctx.vm.log.Error("plugin start failed")
			return types.OnPluginStartStatusFailed
		}
	}
	ctx.vm.log.Info("plugin start successfully")
	return types.OnPluginStartStatusOK
}

func (ctx *CommonPluginCtx[PluginConfig]) OnTick() {
	for i := range ctx.onTickFuncs {
		currentTimeStamp := time.Now().UnixMilli()
		if currentTimeStamp-ctx.onTickFuncs[i].lastExecuted >= ctx.onTickFuncs[i].tickPeriod {
			ctx.onTickFuncs[i].tickFunc()
			ctx.onTickFuncs[i].lastExecuted = currentTimeStamp
		}
	}
}

func (ctx *CommonPluginCtx[PluginConfig]) NewHttpContext(contextID uint32) types.HttpContext {
	httpCtx := &CommonHttpCtx[PluginConfig]{
		plugin:        ctx,
		contextID:     contextID,
		userContext:   map[string]interface{}{},
		userAttribute: map[string]interface{}{},
	}
	if ctx.vm.onHttpRequestBody != nil || ctx.vm.onHttpStreamingRequestBody != nil {
		httpCtx.needRequestBody = true
	}
	if ctx.vm.onHttpResponseBody != nil || ctx.vm.onHttpStreamingResponseBody != nil {
		httpCtx.needResponseBody = true
	}
	if ctx.vm.onHttpStreamingRequestBody != nil {
		httpCtx.streamingRequestBody = true
	}
	if ctx.vm.onHttpStreamingResponseBody != nil {
		httpCtx.streamingResponseBody = true
	}
	return httpCtx
}

type RouteResponseCallback func(sendDirectly bool, statusCode int, responseHeaders [][2]string, responseBody []byte)

type CommonHttpCtx[PluginConfig any] struct {
	types.DefaultHttpContext
	plugin                *CommonPluginCtx[PluginConfig]
	config                *PluginConfig
	needRequestBody       bool
	needResponseBody      bool
	streamingRequestBody  bool
	streamingResponseBody bool
	requestBodySize       int
	responseBodySize      int
	contextID             uint32
	userContext           map[string]interface{}
	userAttribute         map[string]interface{}
	responseCallback      RouteResponseCallback
}

func (ctx *CommonHttpCtx[PluginConfig]) SetContext(key string, value interface{}) {
	ctx.userContext[key] = value
}

func (ctx *CommonHttpCtx[PluginConfig]) GetContext(key string) interface{} {
	return ctx.userContext[key]
}

func (ctx *CommonHttpCtx[PluginConfig]) SetUserAttribute(key string, value interface{}) {
	ctx.userAttribute[key] = value
}

func (ctx *CommonHttpCtx[PluginConfig]) GetUserAttribute(key string) interface{} {
	return ctx.userAttribute[key]
}

func (ctx *CommonHttpCtx[PluginConfig]) SetUserAttributeMap(kvmap map[string]interface{}) {
	ctx.userAttribute = kvmap
}

func (ctx *CommonHttpCtx[PluginConfig]) GetUserAttributeMap() map[string]interface{} {
	return ctx.userAttribute
}

func (ctx *CommonHttpCtx[PluginConfig]) WriteUserAttributeToLog() error {
	return ctx.WriteUserAttributeToLogWithKey(CustomLogKey)
}

func (ctx *CommonHttpCtx[PluginConfig]) WriteUserAttributeToLogWithKey(key string) error {
	// e.g. {\"field1\":\"value1\",\"field2\":\"value2\"}
	preMarshalledJsonLogStr, _ := proxywasm.GetProperty([]string{key})
	newAttributeMap := map[string]interface{}{}
	if string(preMarshalledJsonLogStr) != "" {
		// e.g. {"field1":"value1","field2":"value2"}
		preJsonLogStr := unmarshalStr(fmt.Sprintf(`"%s"`, string(preMarshalledJsonLogStr)))
		err := json.Unmarshal([]byte(preJsonLogStr), &newAttributeMap)
		if err != nil {
			ctx.plugin.vm.log.Warnf("Unmarshal failed, will overwrite %s, pre value is: %s", key, string(preMarshalledJsonLogStr))
			return err
		}
	}
	// update customLog
	for k, v := range ctx.userAttribute {
		newAttributeMap[k] = v
	}
	// e.g. {"field1":"value1","field2":2,"field3":"value3"}
	jsonStr, _ := json.Marshal(newAttributeMap)
	// e.g. {\"field1\":\"value1\",\"field2\":2,\"field3\":\"value3\"}
	marshalledJsonStr := marshalStr(string(jsonStr))
	if err := proxywasm.SetProperty([]string{key}, []byte(marshalledJsonStr)); err != nil {
		ctx.plugin.vm.log.Warnf("failed to set %s in filter state, raw is %s, err is %v", key, marshalledJsonStr, err)
		return err
	}
	return nil
}

func (ctx *CommonHttpCtx[PluginConfig]) WriteUserAttributeToTrace() error {
	for k, v := range ctx.userAttribute {
		traceSpanTag := TraceSpanTagPrefix + k
		traceSpanValue := fmt.Sprint(v)
		var err error
		if traceSpanValue != "" {
			err = proxywasm.SetProperty([]string{traceSpanTag}, []byte(traceSpanValue))
		} else {
			err = fmt.Errorf("value of %s is empty", traceSpanTag)
		}
		if err != nil {
			ctx.plugin.vm.log.Warnf("Failed to set trace attribute - %s: %s, error message: %v", traceSpanTag, traceSpanValue, err)
		}
	}
	return nil
}

func (ctx *CommonHttpCtx[PluginConfig]) GetIntContext(key string, defaultValue int) int {
	if b, ok := ctx.userContext[key].(int); ok {
		return b
	}
	return defaultValue
}

func (ctx *CommonHttpCtx[PluginConfig]) GetBoolContext(key string, defaultValue bool) bool {
	if b, ok := ctx.userContext[key].(bool); ok {
		return b
	}
	return defaultValue
}

func (ctx *CommonHttpCtx[PluginConfig]) GetStringContext(key, defaultValue string) string {
	if s, ok := ctx.userContext[key].(string); ok {
		return s
	}
	return defaultValue
}

func (ctx *CommonHttpCtx[PluginConfig]) GetByteSliceContext(key string, defaultValue []byte) []byte {
	if s, ok := ctx.userContext[key].([]byte); ok {
		return s
	}
	return defaultValue
}

func (ctx *CommonHttpCtx[PluginConfig]) Scheme() string {
	proxywasm.SetEffectiveContext(ctx.contextID)
	return GetRequestScheme()
}

func (ctx *CommonHttpCtx[PluginConfig]) Host() string {
	proxywasm.SetEffectiveContext(ctx.contextID)
	return GetRequestHost()
}

func (ctx *CommonHttpCtx[PluginConfig]) Path() string {
	proxywasm.SetEffectiveContext(ctx.contextID)
	return GetRequestPath()
}

func (ctx *CommonHttpCtx[PluginConfig]) Method() string {
	proxywasm.SetEffectiveContext(ctx.contextID)
	return GetRequestMethod()
}

func (ctx *CommonHttpCtx[PluginConfig]) DontReadRequestBody() {
	ctx.needRequestBody = false
}

func (ctx *CommonHttpCtx[PluginConfig]) DontReadResponseBody() {
	ctx.needResponseBody = false
}

func (ctx *CommonHttpCtx[PluginConfig]) BufferRequestBody() {
	ctx.streamingRequestBody = false
}

func (ctx *CommonHttpCtx[PluginConfig]) BufferResponseBody() {
	ctx.streamingResponseBody = false
}

func (ctx *CommonHttpCtx[PluginConfig]) DisableReroute() {
	_ = proxywasm.SetProperty([]string{"clear_route_cache"}, []byte("off"))
}

func (ctx *CommonHttpCtx[PluginConfig]) SetRequestBodyBufferLimit(size uint32) {
	ctx.plugin.vm.log.Debugf("SetRequestBodyBufferLimit: %d", size)
	_ = proxywasm.SetProperty([]string{"set_decoder_buffer_limit"}, []byte(strconv.Itoa(int(size))))
}

func (ctx *CommonHttpCtx[PluginConfig]) SetResponseBodyBufferLimit(size uint32) {
	ctx.plugin.vm.log.Debugf("SetResponseBodyBufferLimit: %d", size)
	_ = proxywasm.SetProperty([]string{"set_encoder_buffer_limit"}, []byte(strconv.Itoa(int(size))))
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	defer recoverFunc()
	requestID, _ := proxywasm.GetHttpRequestHeader("x-request-id")
	_ = proxywasm.SetProperty([]string{"x_request_id"}, []byte(requestID))
	config, err := ctx.plugin.GetMatchConfig()
	if err != nil {
		ctx.plugin.vm.log.Errorf("get match config failed, err:%v", err)
		return types.ActionContinue
	}
	if config == nil {
		return types.ActionContinue
	}
	ctx.config = config
	// To avoid unexpected operations, plugins do not read the binary content body
	if IsBinaryRequestBody() {
		ctx.needRequestBody = false
	}
	if ctx.plugin.vm.onHttpRequestHeaders == nil {
		return types.ActionContinue
	}
	return ctx.plugin.vm.onHttpRequestHeaders(ctx, *config)
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpRequestBody(bodySize int, endOfStream bool) types.Action {
	defer recoverFunc()
	if ctx.config == nil {
		return types.ActionContinue
	}
	if !ctx.needRequestBody {
		return types.ActionContinue
	}
	if ctx.plugin.vm.onHttpStreamingRequestBody != nil && ctx.streamingRequestBody {
		chunk, _ := proxywasm.GetHttpRequestBody(0, bodySize)
		modifiedChunk := ctx.plugin.vm.onHttpStreamingRequestBody(ctx, *ctx.config, chunk, endOfStream)
		err := proxywasm.ReplaceHttpRequestBody(modifiedChunk)
		if err != nil {
			ctx.plugin.vm.log.Warnf("replace request body chunk failed: %v", err)
			return types.ActionContinue
		}
		return types.ActionContinue
	}
	if ctx.plugin.vm.onHttpRequestBody != nil {
		ctx.requestBodySize += bodySize
		if !endOfStream {
			return types.ActionPause
		}
		body, err := proxywasm.GetHttpRequestBody(0, ctx.requestBodySize)
		if err != nil {
			ctx.plugin.vm.log.Warnf("get request body failed: %v", err)
			return types.ActionContinue
		}
		return ctx.plugin.vm.onHttpRequestBody(ctx, *ctx.config, body)
	}
	return types.ActionContinue
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpResponseHeaders(numHeaders int, endOfStream bool) types.Action {
	defer recoverFunc()
	if ctx.config == nil {
		return types.ActionContinue
	}
	// To avoid unexpected operations, plugins do not read the binary content body
	if IsBinaryResponseBody() {
		ctx.needResponseBody = false
	}
	if ctx.responseCallback != nil {
		if endOfStream {
			statusCode := 500
			status, _ := proxywasm.GetHttpResponseHeader(":status")
			headers, _ := proxywasm.GetHttpResponseHeaders()
			if status != "" {
				statusCode, _ = strconv.Atoi(status)
			}
			ctx.responseCallback(true, statusCode, headers, nil)
			return types.HeaderStopAllIterationAndWatermark
		}
		ctx.needResponseBody = true
		return types.HeaderStopIteration
	}
	if ctx.plugin.vm.onHttpResponseHeaders == nil {
		return types.ActionContinue
	}
	return ctx.plugin.vm.onHttpResponseHeaders(ctx, *ctx.config)
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpResponseBody(bodySize int, endOfStream bool) types.Action {
	defer recoverFunc()
	if ctx.config == nil {
		return types.ActionContinue
	}
	if !ctx.needResponseBody {
		return types.ActionContinue
	}
	if ctx.responseCallback != nil {
		if !endOfStream {
			return types.ActionPause
		}
		body, err := proxywasm.GetHttpResponseBody(0, bodySize)
		if err != nil {
			ctx.plugin.vm.log.Warnf("get response body failed: %v", err)
			return types.ActionContinue
		}
		statusCode := 500
		status, _ := proxywasm.GetHttpResponseHeader(":status")
		proxywasm.RemoveHttpResponseHeader("content-length")
		headers, _ := proxywasm.GetHttpResponseHeaders()
		if status != "" {
			statusCode, _ = strconv.Atoi(status)
		}
		ctx.responseCallback(false, statusCode, headers, body)
		return types.ActionContinue
	}
	if ctx.plugin.vm.onHttpStreamingResponseBody != nil && ctx.streamingResponseBody {
		chunk, _ := proxywasm.GetHttpResponseBody(0, bodySize)
		modifiedChunk := ctx.plugin.vm.onHttpStreamingResponseBody(ctx, *ctx.config, chunk, endOfStream)
		err := proxywasm.ReplaceHttpResponseBody(modifiedChunk)
		if err != nil {
			ctx.plugin.vm.log.Warnf("replace response body chunk failed: %v", err)
			return types.ActionContinue
		}
		return types.ActionContinue
	}
	if ctx.plugin.vm.onHttpResponseBody != nil {
		if !endOfStream {
			return types.ActionPause
		}
		body, err := proxywasm.GetHttpResponseBody(0, bodySize)
		if err != nil {
			ctx.plugin.vm.log.Warnf("get response body failed: %v", err)
			return types.ActionContinue
		}
		return ctx.plugin.vm.onHttpResponseBody(ctx, *ctx.config, body)
	}
	return types.ActionContinue
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpStreamDone() {
	defer recoverFunc()
	if ctx.config == nil {
		return
	}
	if ctx.plugin.vm.onHttpStreamDone == nil {
		return
	}
	ctx.plugin.vm.onHttpStreamDone(ctx, *ctx.config)
}

// This RouteCall must only be invoked during the request body phase, and it requires that stopIteration has been returned during the request header phase.
func (ctx *CommonHttpCtx[PluginConfig]) RouteCall(method, rawURL string, headers [][2]string, body []byte, callback RouteResponseCallback) error {
	proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	proxywasm.RemoveHttpRequestHeader("Content-Length")
	requestID := uuid.New().String()
	ctx.responseCallback = func(sendDirectly bool, statusCode int, responseHeaders [][2]string, responseBody []byte) {
		callback(sendDirectly, statusCode, responseHeaders, responseBody)
		log.Infof("route call end, id:%s, code:%d, headers:%#v, body:%s", requestID, statusCode, responseHeaders, strings.ReplaceAll(string(responseBody), "\n", `\n`))
	}
	orignalMethod, _ := proxywasm.GetHttpRequestHeader(":method")
	orignalPath, _ := proxywasm.GetHttpRequestHeader(":path")
	orignalHost, _ := proxywasm.GetHttpRequestHeader(":authority")
	proxywasm.ReplaceHttpRequestHeader("x-envoy-original-method", orignalMethod)
	proxywasm.ReplaceHttpRequestHeader("x-envoy-original-path", orignalPath)
	proxywasm.ReplaceHttpRequestHeader("x-envoy-original-host", orignalHost)

	proxywasm.ReplaceHttpRequestHeader(":method", method)
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url:%s, err:%w", rawURL, err)
	}
	var authority string
	if parsedURL.Host != "" {
		authority = parsedURL.Host
	}
	path := "/" + strings.TrimPrefix(parsedURL.Path, "/")
	if parsedURL.RawQuery != "" {
		path = fmt.Sprintf("%s?%s", path, parsedURL.RawQuery)
	}
	proxywasm.ReplaceHttpRequestHeader(":path", path)
	if authority != "" {
		proxywasm.ReplaceHttpRequestHeader(":authority", authority)
	}
	for _, kv := range headers {
		proxywasm.ReplaceHttpRequestHeader(kv[0], kv[1])
	}
	proxywasm.ReplaceHttpRequestBody(body)
	reqHeaders, _ := proxywasm.GetHttpRequestHeaders()
	clusterName, _ := proxywasm.GetProperty([]string{"cluster_name"})
	log.Infof("route call start, id:%s, method:%s, url:%s, cluster:%s, headers:%#v, body:%s", requestID, method, rawURL, clusterName, reqHeaders, strings.ReplaceAll(string(body), "\n", `\n`))
	return nil
}

func recoverFunc() {
	if r := recover(); r != nil {
		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		log.Errorf("recovered from panic %v, stack: %s", r, buf)
	}
}
