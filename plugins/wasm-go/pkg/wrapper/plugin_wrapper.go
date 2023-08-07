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
	"unsafe"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/matcher"
	_ "github.com/wasilibs/nottinygc"
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
}

type ParseConfigFunc[PluginConfig any] func(json gjson.Result, config *PluginConfig, log Log) error
type onHttpHeadersFunc[PluginConfig any] func(context HttpContext, config PluginConfig, log Log) types.Action
type onHttpBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, body []byte, log Log) types.Action
type onHttpStreamDoneFunc[PluginConfig any] func(context HttpContext, config PluginConfig, log Log)

type CommonVmCtx[PluginConfig any] struct {
	types.DefaultVMContext
	pluginName            string
	log                   Log
	hasCustomConfig       bool
	parseConfig           ParseConfigFunc[PluginConfig]
	onHttpRequestHeaders  onHttpHeadersFunc[PluginConfig]
	onHttpRequestBody     onHttpBodyFunc[PluginConfig]
	onHttpResponseHeaders onHttpHeadersFunc[PluginConfig]
	onHttpResponseBody    onHttpBodyFunc[PluginConfig]
	onHttpStreamDone      onHttpStreamDoneFunc[PluginConfig]
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
	vm *CommonVmCtx[PluginConfig]
}

func (ctx *CommonPluginCtx[PluginConfig]) OnPluginStart(int) types.OnPluginStartStatus {
	data, err := proxywasm.GetPluginConfiguration()
	if err != nil && err != types.ErrorStatusNotFound {
		ctx.vm.log.Criticalf("error reading plugin configuration: %v", err)
		return types.OnPluginStartStatusFailed
	}
	var jsonData gjson.Result
	if len(data) == 0 {
		if ctx.vm.hasCustomConfig {
			ctx.vm.log.Warn("need config")
			return types.OnPluginStartStatusFailed
		}
	} else {
		if !gjson.ValidBytes(data) {
			ctx.vm.log.Warnf("the plugin configuration is not a valid json: %s", string(data))
			return types.OnPluginStartStatusFailed

		}
		jsonData = gjson.ParseBytes(data)
	}
	err = ctx.ParseRuleConfig(jsonData, func(js gjson.Result, cfg *PluginConfig) error {
		return ctx.vm.parseConfig(js, cfg, ctx.vm.log)
	})
	if err != nil {
		ctx.vm.log.Warnf("parse rule config failed: %v", err)
		return types.OnPluginStartStatusFailed
	}
	return types.OnPluginStartStatusOK
}

func (ctx *CommonPluginCtx[PluginConfig]) NewHttpContext(contextID uint32) types.HttpContext {
	httpCtx := &CommonHttpCtx[PluginConfig]{
		plugin:      ctx,
		contextID:   contextID,
		userContext: map[string]interface{}{},
	}
	if ctx.vm.onHttpRequestBody != nil {
		httpCtx.needRequestBody = true
	}
	if ctx.vm.onHttpResponseBody != nil {
		httpCtx.needResponseBody = true
	}
	return httpCtx
}

type CommonHttpCtx[PluginConfig any] struct {
	types.DefaultHttpContext
	plugin           *CommonPluginCtx[PluginConfig]
	config           *PluginConfig
	needRequestBody  bool
	needResponseBody bool
	requestBodySize  int
	responseBodySize int
	contextID        uint32
	userContext      map[string]interface{}
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
	if ctx.plugin.vm.onHttpRequestHeaders == nil {
		return types.ActionContinue
	}
	return ctx.plugin.vm.onHttpRequestHeaders(ctx, *config, ctx.plugin.vm.log)
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpRequestBody(bodySize int, endOfStream bool) types.Action {
	if ctx.config == nil {
		return types.ActionContinue
	}
	if ctx.plugin.vm.onHttpRequestBody == nil {
		return types.ActionContinue
	}
	if !ctx.needRequestBody {
		return types.ActionContinue
	}
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

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpResponseHeaders(numHeaders int, endOfStream bool) types.Action {
	if ctx.config == nil {
		return types.ActionContinue
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
	if ctx.plugin.vm.onHttpResponseBody == nil {
		return types.ActionContinue
	}
	if !ctx.needResponseBody {
		return types.ActionContinue
	}
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

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpStreamDone() {
	if ctx.config == nil {
		return
	}
	if ctx.plugin.vm.onHttpStreamDone == nil {
		return
	}
	ctx.plugin.vm.onHttpStreamDone(ctx, *ctx.config, ctx.plugin.vm.log)
}
