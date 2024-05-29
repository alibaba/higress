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
	"time"
	"unsafe"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/matcher"
	_ "github.com/higress-group/nottinygc"
)

//export sched_yield
func sched_yield() int32 {
	return 0
}

type HttpContext interface {
	Scheme() string
	Host() string
	Path() string
	Method() string
	SetContext(key string, value interface{})
	GetContext(key string) interface{}
	// If the onHttpRequestBody handle is not set, the request body will not be read by default
	DontReadRequestBody()
	// If the onHttpResponseBody handle is not set, the request body will not be read by default
	DontReadResponseBody()
	// If the onHttpStreamingRequestBody handle is not set, and the onHttpRequestBody handle is set, the request body will be buffered by default
	BufferRequestBody()
	// If the onHttpStreamingResponseBody handle is not set, and the onHttpResponseBody handle is set, the response body will be buffered by default
	BufferResponseBody()
	// If any request header is changed in onHttpRequestHeaders, envoy will re-calculate the route. Call this function to disable the re-routing.
	DisableReroute()
}

type ParseConfigFunc[PluginConfig any] func(json gjson.Result, config *PluginConfig, log Log) error
type ParseRuleConfigFunc[PluginConfig any] func(json gjson.Result, global PluginConfig, config *PluginConfig, log Log) error
type onHttpHeadersFunc[PluginConfig any] func(context HttpContext, config PluginConfig, log Log) types.Action
type onHttpBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, body []byte, log Log) types.Action
type onHttpStreamingBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log Log) []byte
type onHttpStreamDoneFunc[PluginConfig any] func(context HttpContext, config PluginConfig, log Log)

type CommonVmCtx[PluginConfig any] struct {
	types.DefaultVMContext
	pluginName                  string
	log                         Log
	hasCustomConfig             bool
	parseConfig                 ParseConfigFunc[PluginConfig]
	parseRuleConfig             ParseRuleConfigFunc[PluginConfig]
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
//	func parseConfig(json gjson.Result, config *HelloWorldConfig, log wrapper.Log) error {
//	  wrapper.RegisteTickFunc(1000, func() { proxywasm.LogInfo("onTick 1s") })
//		 wrapper.RegisteTickFunc(3000, func() { proxywasm.LogInfo("onTick 3s") })
//		 return nil
//	}
func RegisteTickFunc(tickPeriod int64, tickFunc func()) {
	globalOnTickFuncs = append(globalOnTickFuncs, TickFuncEntry{0, tickPeriod, tickFunc})
}

func SetCtx[PluginConfig any](pluginName string, setFuncs ...SetPluginFunc[PluginConfig]) {
	proxywasm.SetVMContext(NewCommonVmCtx(pluginName, setFuncs...))
}

type SetPluginFunc[PluginConfig any] func(*CommonVmCtx[PluginConfig])

func ParseConfigBy[PluginConfig any](f ParseConfigFunc[PluginConfig]) SetPluginFunc[PluginConfig] {
	return func(ctx *CommonVmCtx[PluginConfig]) {
		ctx.parseConfig = f
	}
}

func ParseOverrideConfigBy[PluginConfig any](f ParseConfigFunc[PluginConfig], g ParseRuleConfigFunc[PluginConfig]) SetPluginFunc[PluginConfig] {
	return func(ctx *CommonVmCtx[PluginConfig]) {
		ctx.parseConfig = f
		ctx.parseRuleConfig = g
	}
}

func ProcessRequestHeadersBy[PluginConfig any](f onHttpHeadersFunc[PluginConfig]) SetPluginFunc[PluginConfig] {
	return func(ctx *CommonVmCtx[PluginConfig]) {
		ctx.onHttpRequestHeaders = f
	}
}

func ProcessRequestBodyBy[PluginConfig any](f onHttpBodyFunc[PluginConfig]) SetPluginFunc[PluginConfig] {
	return func(ctx *CommonVmCtx[PluginConfig]) {
		ctx.onHttpRequestBody = f
	}
}

func ProcessStreamingRequestBodyBy[PluginConfig any](f onHttpStreamingBodyFunc[PluginConfig]) SetPluginFunc[PluginConfig] {
	return func(ctx *CommonVmCtx[PluginConfig]) {
		ctx.onHttpStreamingRequestBody = f
	}
}

func ProcessResponseHeadersBy[PluginConfig any](f onHttpHeadersFunc[PluginConfig]) SetPluginFunc[PluginConfig] {
	return func(ctx *CommonVmCtx[PluginConfig]) {
		ctx.onHttpResponseHeaders = f
	}
}

func ProcessResponseBodyBy[PluginConfig any](f onHttpBodyFunc[PluginConfig]) SetPluginFunc[PluginConfig] {
	return func(ctx *CommonVmCtx[PluginConfig]) {
		ctx.onHttpResponseBody = f
	}
}

func ProcessStreamingResponseBodyBy[PluginConfig any](f onHttpStreamingBodyFunc[PluginConfig]) SetPluginFunc[PluginConfig] {
	return func(ctx *CommonVmCtx[PluginConfig]) {
		ctx.onHttpStreamingResponseBody = f
	}
}

func ProcessStreamDoneBy[PluginConfig any](f onHttpStreamDoneFunc[PluginConfig]) SetPluginFunc[PluginConfig] {
	return func(ctx *CommonVmCtx[PluginConfig]) {
		ctx.onHttpStreamDone = f
	}
}

func parseEmptyPluginConfig[PluginConfig any](gjson.Result, *PluginConfig, Log) error {
	return nil
}

func NewCommonVmCtx[PluginConfig any](pluginName string, setFuncs ...SetPluginFunc[PluginConfig]) *CommonVmCtx[PluginConfig] {
	ctx := &CommonVmCtx[PluginConfig]{
		pluginName:      pluginName,
		log:             Log{pluginName},
		hasCustomConfig: true,
	}
	for _, set := range setFuncs {
		set(ctx)
	}
	if ctx.parseConfig == nil {
		var config PluginConfig
		if unsafe.Sizeof(config) != 0 {
			msg := "the `parseConfig` is missing in NewCommonVmCtx's arguments"
			ctx.log.Critical(msg)
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

	var parseOverrideConfig func(gjson.Result, PluginConfig, *PluginConfig) error
	if ctx.vm.parseRuleConfig != nil {
		parseOverrideConfig = func(js gjson.Result, global PluginConfig, cfg *PluginConfig) error {
			return ctx.vm.parseRuleConfig(js, global, cfg, ctx.vm.log)
		}
	}
	err = ctx.ParseRuleConfig(jsonData,
		func(js gjson.Result, cfg *PluginConfig) error {
			return ctx.vm.parseConfig(js, cfg, ctx.vm.log)
		},
		parseOverrideConfig,
	)
	if err != nil {
		ctx.vm.log.Warnf("parse rule config failed: %v", err)
		return types.OnPluginStartStatusFailed
	}
	if globalOnTickFuncs != nil {
		ctx.onTickFuncs = globalOnTickFuncs
		if err := proxywasm.SetTickPeriodMilliSeconds(100); err != nil {
			ctx.vm.log.Error("SetTickPeriodMilliSeconds failed, onTick functions will not take effect.")
			return types.OnPluginStartStatusFailed
		}
	}
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
		plugin:      ctx,
		contextID:   contextID,
		userContext: map[string]interface{}{},
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
}

func (ctx *CommonHttpCtx[PluginConfig]) SetContext(key string, value interface{}) {
	ctx.userContext[key] = value
}

func (ctx *CommonHttpCtx[PluginConfig]) GetContext(key string) interface{} {
	return ctx.userContext[key]
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

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
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
	return ctx.plugin.vm.onHttpRequestHeaders(ctx, *config, ctx.plugin.vm.log)
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpRequestBody(bodySize int, endOfStream bool) types.Action {
	if ctx.config == nil {
		return types.ActionContinue
	}
	if !ctx.needRequestBody {
		return types.ActionContinue
	}
	if ctx.plugin.vm.onHttpStreamingRequestBody != nil && ctx.streamingRequestBody {
		chunk, _ := proxywasm.GetHttpRequestBody(0, bodySize)
		modifiedChunk := ctx.plugin.vm.onHttpStreamingRequestBody(ctx, *ctx.config, chunk, endOfStream, ctx.plugin.vm.log)
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
		return ctx.plugin.vm.onHttpRequestBody(ctx, *ctx.config, body, ctx.plugin.vm.log)
	}
	return types.ActionContinue
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpResponseHeaders(numHeaders int, endOfStream bool) types.Action {
	if ctx.config == nil {
		return types.ActionContinue
	}
	// To avoid unexpected operations, plugins do not read the binary content body
	if IsBinaryResponseBody() {
		ctx.needResponseBody = false
	}
	if ctx.plugin.vm.onHttpResponseHeaders == nil {
		return types.ActionContinue
	}
	return ctx.plugin.vm.onHttpResponseHeaders(ctx, *ctx.config, ctx.plugin.vm.log)
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpResponseBody(bodySize int, endOfStream bool) types.Action {
	if ctx.config == nil {
		return types.ActionContinue
	}
	if !ctx.needResponseBody {
		return types.ActionContinue
	}
	if ctx.plugin.vm.onHttpStreamingResponseBody != nil && ctx.streamingResponseBody {
		chunk, _ := proxywasm.GetHttpResponseBody(0, bodySize)
		modifiedChunk := ctx.plugin.vm.onHttpStreamingResponseBody(ctx, *ctx.config, chunk, endOfStream, ctx.plugin.vm.log)
		err := proxywasm.ReplaceHttpResponseBody(modifiedChunk)
		if err != nil {
			ctx.plugin.vm.log.Warnf("replace response body chunk failed: %v", err)
			return types.ActionContinue
		}
		return types.ActionContinue
	}
	if ctx.plugin.vm.onHttpResponseBody != nil {
		ctx.responseBodySize += bodySize
		if !endOfStream {
			return types.ActionPause
		}
		body, err := proxywasm.GetHttpResponseBody(0, ctx.responseBodySize)
		if err != nil {
			ctx.plugin.vm.log.Warnf("get response body failed: %v", err)
			return types.ActionContinue
		}
		return ctx.plugin.vm.onHttpResponseBody(ctx, *ctx.config, body, ctx.plugin.vm.log)
	}
	return types.ActionContinue
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpStreamDone() {
	if ctx.config == nil {
		return
	}
	if ctx.plugin.vm.onHttpStreamDone == nil {
		return
	}
	ctx.plugin.vm.onHttpStreamDone(ctx, *ctx.config, ctx.plugin.vm.log)
}
