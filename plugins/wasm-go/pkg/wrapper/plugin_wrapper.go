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
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/iface"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/matcher"
)

type Log log.Log
type PluginContext iface.PluginContext
type HttpContext iface.HttpContext

const (
	CustomLogKey         = "custom_log"
	AILogKey             = "ai_log"
	TraceSpanTagPrefix   = "trace_span_tag."
	PluginIDKey          = "_plugin_id_"
	VMLeaseKeyPrefix     = "higress_wasm_vm_lease"
	ConfigStoreLeaderKey = "config_store"
)

type oldParseConfigFunc[PluginConfig any] func(json gjson.Result, config *PluginConfig, log log.Log) error
type oldParseRuleConfigFunc[PluginConfig any] func(json gjson.Result, global PluginConfig, config *PluginConfig, log log.Log) error
type oldOnHttpHeadersFunc[PluginConfig any] func(context HttpContext, config PluginConfig, log log.Log) types.Action
type oldOnHttpBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, body []byte, log log.Log) types.Action
type oldOnHttpStreamingBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, chunk []byte, isLastChunk bool, log log.Log) []byte
type oldOnHttpStreamDoneFunc[PluginConfig any] func(context HttpContext, config PluginConfig, log log.Log)

type ParseConfigWithContextFunc[PluginConfig any] func(context PluginContext, json gjson.Result, config *PluginConfig) error
type ParseConfigFunc[PluginConfig any] func(json gjson.Result, config *PluginConfig) error
type ParseRawConfigFunc[PluginConfig any] func(configBytes []byte, config *PluginConfig) error
type ParseRawConfigWithContextFunc[PluginConfig any] func(context PluginContext, configBytes []byte, config *PluginConfig) error
type ParseRuleConfigFunc[PluginConfig any] func(json gjson.Result, global PluginConfig, config *PluginConfig) error
type ParseRawRuleConfigFunc[PluginConfig any] func(configBytes []byte, global PluginConfig, config *PluginConfig) error
type ParseRawRuleConfigWithContextFunc[PluginConfig any] func(context PluginContext, configBytes []byte, global PluginConfig, config *PluginConfig) error
type onHttpHeadersFunc[PluginConfig any] func(context HttpContext, config PluginConfig) types.Action
type onHttpBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, body []byte) types.Action
type onHttpStreamingBodyFunc[PluginConfig any] func(context HttpContext, config PluginConfig, chunk []byte, isLastChunk bool) []byte
type onHttpStreamDoneFunc[PluginConfig any] func(context HttpContext, config PluginConfig)

type onPluginStartOrReload func(context PluginContext) error

type CommonVmCtx[PluginConfig any] struct {
	types.DefaultVMContext
	pluginName                  string
	log                         log.Log
	hasCustomConfig             bool
	vmID                        string
	prePluginStartOrReload      onPluginStartOrReload
	parseConfig                 ParseRawConfigWithContextFunc[PluginConfig]
	parseRuleConfig             ParseRawRuleConfigWithContextFunc[PluginConfig]
	onHttpRequestHeaders        onHttpHeadersFunc[PluginConfig]
	onHttpRequestBody           onHttpBodyFunc[PluginConfig]
	onHttpStreamingRequestBody  onHttpStreamingBodyFunc[PluginConfig]
	onHttpResponseHeaders       onHttpHeadersFunc[PluginConfig]
	onHttpResponseBody          onHttpBodyFunc[PluginConfig]
	onHttpStreamingResponseBody onHttpStreamingBodyFunc[PluginConfig]
	onHttpStreamDone            onHttpStreamDoneFunc[PluginConfig]
	rebuildAfterRequests        uint64 // Number of requests after which to trigger rebuild
	requestCount                uint64 // Current request count
	rebuildMaxMem               uint64 // Maximum memory size in bytes before triggering rebuild
	maxRequestsPerIoCycle       uint64 // Maximum concurrent requests per IO cycle (0 means not set)
}

type TickFuncEntry struct {
	lastExecuted int64
	tickPeriod   int64
	tickFunc     func()
}

var globalOnTickFuncs []TickFuncEntry = []TickFuncEntry{}

// Register multiple onTick functions. Parameters include:
// 1) tickPeriod: the execution period of tickFunc, this value should be a multiple of 100
// 2) tickFunc: the function to be executed
//
// You should call this function in parseConfig phase, for example:
//
//	func parseConfig(json gjson.Result, config *HelloWorldConfig, log log.Log) error {
//	  wrapper.RegisterTickFunc(1000, func() { proxywasm.LogInfo("onTick 1s") })
//		 wrapper.RegisterTickFunc(3000, func() { proxywasm.LogInfo("onTick 3s") })
//		 return nil
//	}
func RegisterTickFunc(tickPeriod int64, tickFunc func()) {
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
	ctxF ParseConfigWithContextFunc[PluginConfig]
}

func (o parseConfigOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	if o.rawF != nil {
		ctx.parseConfig = func(context PluginContext, configBytes []byte, config *PluginConfig) error {
			return o.rawF(configBytes, config)
		}
	} else if o.f != nil {
		ctx.parseConfig = func(context PluginContext, configBytes []byte, config *PluginConfig) error {
			return o.f(gjson.ParseBytes(configBytes), config)
		}
	} else if o.oldF != nil {
		ctx.parseConfig = func(context PluginContext, configBytes []byte, config *PluginConfig) error {
			return o.oldF(gjson.ParseBytes(configBytes), config, ctx.log)
		}
	} else if o.ctxF != nil {
		ctx.parseConfig = func(context PluginContext, configBytes []byte, config *PluginConfig) error {
			return o.ctxF(context, gjson.ParseBytes(configBytes), config)
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

func ParseConfigWithContext[PluginConfig any](f ParseConfigWithContextFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &parseConfigOption[PluginConfig]{ctxF: f}
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
		ctx.parseConfig = func(context PluginContext, configBytes []byte, config *PluginConfig) error {
			return o.parseRawConfigF(configBytes, config)
		}
		ctx.parseRuleConfig = func(context PluginContext, configBytes []byte, global PluginConfig, config *PluginConfig) error {
			return o.parseRawRuleConfigF(configBytes, global, config)
		}
	} else if o.parseConfigF != nil && o.parseRuleConfigF != nil {
		ctx.parseConfig = func(context PluginContext, configBytes []byte, config *PluginConfig) error {
			return o.parseConfigF(gjson.ParseBytes(configBytes), config)
		}
		ctx.parseRuleConfig = func(context PluginContext, configBytes []byte, global PluginConfig, config *PluginConfig) error {
			return o.parseRuleConfigF(gjson.ParseBytes(configBytes), global, config)
		}
	} else {
		ctx.parseConfig = func(context PluginContext, configBytes []byte, config *PluginConfig) error {
			return o.oldParseConfigF(gjson.ParseBytes(configBytes), config, ctx.log)
		}
		ctx.parseRuleConfig = func(context PluginContext, configBytes []byte, global PluginConfig, config *PluginConfig) error {
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

type safeLogOption[PluginConfig any] struct{}

func (o *safeLogOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	log.SetSafeLogEnabled(true)
}

// EnableSafeLog enables safe log mode to prevent logging sensitive information.
// When enabled, sensitive logs such as HTTP request/response headers and bodies
// from external service calls will be suppressed.
//
// Usage in plugin init:
//
//	func main() {
//	    wrapper.SetCtx(
//	        "my-plugin",
//	        wrapper.ParseConfig(parseConfig),
//	        wrapper.EnableSafeLog[PluginConfig](),
//	    )
//	}
func EnableSafeLog[PluginConfig any]() CtxOption[PluginConfig] {
	return &safeLogOption[PluginConfig]{}
}

type rebuildOption[PluginConfig any] struct {
	rebuildAfterRequests uint64
}

func (o *rebuildOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	ctx.rebuildAfterRequests = o.rebuildAfterRequests
	ctx.requestCount = 0
}

func WithRebuildAfterRequests[PluginConfig any](requestCount uint64) CtxOption[PluginConfig] {
	return &rebuildOption[PluginConfig]{rebuildAfterRequests: requestCount}
}

type rebuildMaxMemOption[PluginConfig any] struct {
	rebuildMaxMem uint64
}

func (o *rebuildMaxMemOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	ctx.rebuildMaxMem = o.rebuildMaxMem
}

// WithRebuildMaxMemBytes sets the maximum memory size in bytes before triggering a plugin rebuild.
// When the VM memory reaches this threshold, the rebuild flag will be set.
// memSizeBytes: The maximum memory size in bytes (e.g., 100*1024*1024 for 100MB)
func WithRebuildMaxMemBytes[PluginConfig any](memSizeBytes uint64) CtxOption[PluginConfig] {
	return &rebuildMaxMemOption[PluginConfig]{rebuildMaxMem: memSizeBytes}
}

type maxRequestsPerIoCycleOption[PluginConfig any] struct {
	maxRequestsPerIoCycle uint64
}

func (o *maxRequestsPerIoCycleOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	ctx.maxRequestsPerIoCycle = o.maxRequestsPerIoCycle
}

// WithMaxRequestsPerIoCycle sets the global max requests per IO cycle.
// This controls how many concurrent requests can be processed in a single IO cycle.
// The setting is applied during plugin start via the "set_global_max_requests_per_io_cycle" foreign function.
//
// Background:
// When plugin logic is complex, external call callbacks (HTTP/Redis) may timeout within the current
// IO cycle even if the backend has already returned a response quickly. This is because the plugin
// execution blocks the IO cycle from processing the callback in time.
//
// Recommendation:
// For plugins with complex logic that need to make external calls (HTTP/Redis), it is recommended
// to use this setting to limit concurrent requests per IO cycle.
//
// Important notes:
//   - This setting takes effect GLOBALLY, not just for the current plugin.
//   - When multiple plugins set this value, the SMALLEST limit will take effect.
//   - Setting this value too small may cause additional CPU overhead.
//   - Recommended value = external call timeout / total plugin logic execution time across all plugins
//
// maxRequests: The maximum number of requests per IO cycle (e.g., 20)
func WithMaxRequestsPerIoCycle[PluginConfig any](maxRequests uint64) CtxOption[PluginConfig] {
	return &maxRequestsPerIoCycleOption[PluginConfig]{maxRequestsPerIoCycle: maxRequests}
}

// setGlobalMaxRequestsPerIoCycle sets the global max requests per IO cycle via foreign function call.
func setGlobalMaxRequestsPerIoCycle(maxRequests uint64) error {
	param := make([]byte, 8)
	binary.LittleEndian.PutUint64(param, maxRequests)
	_, err := proxywasm.CallForeignFunction("set_global_max_requests_per_io_cycle", param)
	if err != nil {
		return fmt.Errorf("set global max requests failed: %w", err)
	}
	return nil
}

type prePluginOption[PluginConfig any] struct {
	f onPluginStartOrReload
}

func (o *prePluginOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	ctx.prePluginStartOrReload = o.f
}

func PrePluginStartOrReload[PluginConfig any](f onPluginStartOrReload) CtxOption[PluginConfig] {
	return &prePluginOption[PluginConfig]{f}
}

func parseEmptyPluginConfig[PluginConfig any](PluginContext, []byte, *PluginConfig) error {
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
		vmID:            uuid.New().String(),
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
		vm:          ctx,
		userContext: map[string]interface{}{},
	}
}

type CommonPluginCtx[PluginConfig any] struct {
	types.DefaultPluginContext
	matcher.RuleMatcher[PluginConfig]
	vm                 *CommonVmCtx[PluginConfig]
	onTickFuncs        []TickFuncEntry
	userContext        map[string]interface{}
	fingerPrint        string
	ruleLevelIsolation bool
	isLeader           bool
}

type Lease struct {
	VMID      string `json:"vmID"`
	Timestamp int64  `json:"timestamp"`
}

func (ctx *CommonPluginCtx[PluginConfig]) DoLeaderElection() {
	ctx.isLeader = ctx.tryAcquireOrRenewLease()
}

func (ctx *CommonPluginCtx[PluginConfig]) leaderLeaseKey() string {
	return fmt.Sprintf("%s:%s", VMLeaseKeyPrefix, ctx.fingerPrint)
}

func (ctx *CommonPluginCtx[PluginConfig]) tryAcquireOrRenewLease() bool {
	now := time.Now().Unix()

	data, cas, err := proxywasm.GetSharedData(ctx.leaderLeaseKey())
	if err != nil {
		if errors.Is(err, types.ErrorStatusNotFound) {
			return ctx.setLease(now, cas)
		} else {
			log.Errorf("Failed to get lease: %v", err)
			return false
		}
	}
	if data == nil {
		return ctx.setLease(now, cas)
	}

	var lease Lease
	err = json.Unmarshal(data, &lease)
	if err != nil {
		log.Errorf("Failed to unmarshal lease data: %v", err)
		return false
	}
	// If vmID is itself, try to renew the lease directly
	// If the lease is expired (60s), try to acquire the lease
	if lease.VMID == ctx.vm.vmID || now-lease.Timestamp > 60 {
		lease.VMID = ctx.vm.vmID
		lease.Timestamp = now
		return ctx.setLease(now, cas)
	}

	return false
}

func (ctx *CommonPluginCtx[PluginConfig]) setLease(timestamp int64, cas uint32) bool {
	lease := Lease{
		VMID:      ctx.vm.vmID,
		Timestamp: timestamp,
	}
	leaseByte, err := json.Marshal(lease)
	if err != nil {
		log.Errorf("Failed to marshal lease data: %v", err)
		return false
	}

	if err := proxywasm.SetSharedData(ctx.leaderLeaseKey(), leaseByte, cas); err != nil {
		log.Errorf("Failed to set or renew lease: %v", err)
		return false
	}
	return true
}

func (ctx *CommonPluginCtx[PluginConfig]) IsLeader() bool {
	return ctx.isLeader
}

func (ctx *CommonPluginCtx[PluginConfig]) IsRuleLevelConfigIsolation() bool {
	return ctx.ruleLevelIsolation
}

func (ctx *CommonPluginCtx[PluginConfig]) GetFingerPrint() string {
	return ctx.fingerPrint
}

func (ctx *CommonPluginCtx[PluginConfig]) EnableRuleLevelConfigIsolation() {
	ctx.ruleLevelIsolation = true
}

func (ctx *CommonPluginCtx[PluginConfig]) GetContext(key string) interface{} {
	return ctx.userContext[key]
}

func (ctx *CommonPluginCtx[PluginConfig]) SetContext(key string, value interface{}) {
	ctx.userContext[key] = value
}

func (ctx *CommonPluginCtx[PluginConfig]) OnPluginStart(int) types.OnPluginStartStatus {
	if ctx.vm.prePluginStartOrReload != nil {
		err := ctx.vm.prePluginStartOrReload(ctx)
		if err != nil {
			log.Errorf("prePluginStartOrReload hook failed: %v", err)
			return types.OnPluginStartStatusFailed
		}
	}
	// Set max requests per IO cycle if configured
	if ctx.vm.maxRequestsPerIoCycle > 0 {
		if err := setGlobalMaxRequestsPerIoCycle(ctx.vm.maxRequestsPerIoCycle); err != nil {
			log.Warnf("set global max requests per IO cycle failed: %v", err)
		} else {
			log.Infof("set global max requests per IO cycle to %d", ctx.vm.maxRequestsPerIoCycle)
		}
	}
	data, err := proxywasm.GetPluginConfiguration()
	globalOnTickFuncs = nil
	if err != nil && err != types.ErrorStatusNotFound {
		log.Criticalf("error reading plugin configuration: %v", err)
		return types.OnPluginStartStatusFailed
	}
	var jsonData gjson.Result
	if len(data) == 0 {
		if ctx.vm.hasCustomConfig {
			log.Warn("config is empty, but has ParseConfigFunc")
		}
	} else {
		if !gjson.ValidBytes(data) {
			ctx.vm.log.Warnf("the plugin configuration is not a valid json: %s", string(data))
			return types.OnPluginStartStatusFailed
		}
		pluginID := gjson.GetBytes(data, PluginIDKey).String()
		if pluginID != "" {
			ctx.vm.log.ResetID(pluginID)
			data, _ = sjson.DeleteBytes([]byte(data), PluginIDKey)
		}
		jsonData = gjson.ParseBytes(data)
	}
	var parseOverrideConfig func(gjson.Result, PluginConfig, *PluginConfig) error
	if ctx.vm.parseRuleConfig != nil {
		parseOverrideConfig = func(js gjson.Result, global PluginConfig, cfg *PluginConfig) error {
			return ctx.vm.parseRuleConfig(ctx, []byte(js.Raw), global, cfg)
		}
	}
	err = ctx.ParseRuleConfig(ctx, jsonData,
		func(js gjson.Result, cfg *PluginConfig) error {
			return ctx.vm.parseConfig(ctx, []byte(js.Raw), cfg)
		},
		parseOverrideConfig,
	)
	if err != nil {
		log.Warnf("parse rule config failed: %v", err)
		log.Error("plugin start failed")
		return types.OnPluginStartStatusFailed
	}
	if globalOnTickFuncs != nil {
		ctx.onTickFuncs = globalOnTickFuncs
		if err := proxywasm.SetTickPeriodMilliSeconds(100); err != nil {
			log.Error("SetTickPeriodMilliSeconds failed, onTick functions will not take effect.")
			log.Error("plugin start failed")
			return types.OnPluginStartStatusFailed
		}
	}
	log.Info("plugin start successfully")
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

type CommonHttpCtx[PluginConfig any] struct {
	types.DefaultHttpContext
	plugin                    *CommonPluginCtx[PluginConfig]
	config                    *PluginConfig
	needRequestBody           bool
	needResponseBody          bool
	streamingRequestBody      bool
	streamingResponseBody     bool
	pauseStreamingResponse    bool
	requestBodySize           int
	responseBodySize          int
	contextID                 uint32
	userContext               map[string]interface{}
	userAttribute             map[string]interface{}
	bufferQueue               [][]byte
	responseCallback          iface.RouteResponseCallback
	executionPhase            iface.HTTPExecutionPhase
	requestHeaderEndOfStream  bool
	responseHeaderEndOfStream bool
	// Cached request pseudo-headers from the header phase
	scheme string
	host   string
	path   string
	method string
	// Cached request headers from the header phase
	requestConnection      string
	requestUpgrade         string
	requestContentType     string
	requestContentEncoding string
	// Cached response headers from the header phase
	responseContentType     string
	responseContentEncoding string
}

func (ctx *CommonHttpCtx[PluginConfig]) GetExecutionPhase() iface.HTTPExecutionPhase {
	return ctx.executionPhase
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
		preJsonLogStr := UnmarshalStr(fmt.Sprintf(`"%s"`, string(preMarshalledJsonLogStr)))
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
	marshalledJsonStr := MarshalStr(string(jsonStr))
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

func (ctx *CommonHttpCtx[PluginConfig]) GetMatchConfig() (*PluginConfig, error) {
	config, err := ctx.plugin.GetMatchConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (ctx *CommonHttpCtx[PluginConfig]) Scheme() string {
	return ctx.scheme
}

func (ctx *CommonHttpCtx[PluginConfig]) Host() string {
	return ctx.host
}

func (ctx *CommonHttpCtx[PluginConfig]) Path() string {
	return ctx.path
}

func (ctx *CommonHttpCtx[PluginConfig]) Method() string {
	return ctx.method
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

func (ctx *CommonHttpCtx[PluginConfig]) NeedPauseStreamingResponse() {
	ctx.pauseStreamingResponse = true
}

func (ctx *CommonHttpCtx[PluginConfig]) PushBuffer(buffer []byte) {
	ctx.bufferQueue = append(ctx.bufferQueue, buffer)
}

func (ctx *CommonHttpCtx[PluginConfig]) PopBuffer() []byte {
	var buffer []byte
	if len(ctx.bufferQueue) > 0 {
		buffer = ctx.bufferQueue[0]
		ctx.bufferQueue = ctx.bufferQueue[1:]
	}
	return buffer
}

func (ctx *CommonHttpCtx[PluginConfig]) BufferQueueSize() int {
	return len(ctx.bufferQueue)
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

// HasRequestBody checks if the request has a body.
// It directly checks whether endOfStream was received during OnHttpRequestHeaders.
// If endOfStream was true in the header phase, there's no body; otherwise there is a body.
func (ctx *CommonHttpCtx[PluginConfig]) HasRequestBody() bool {
	return !ctx.requestHeaderEndOfStream
}

// HasResponseBody checks if the response has a body.
// It directly checks whether endOfStream was received during OnHttpResponseHeaders.
// If endOfStream was true in the header phase, there's no body; otherwise there is a body.
func (ctx *CommonHttpCtx[PluginConfig]) HasResponseBody() bool {
	return !ctx.responseHeaderEndOfStream
}

// IsWebsocket checks if the request is a WebSocket upgrade request.
// It uses cached header values from the header phase and can be called at any time.
func (ctx *CommonHttpCtx[PluginConfig]) IsWebsocket() bool {
	return strings.EqualFold(ctx.requestConnection, "upgrade") && strings.EqualFold(ctx.requestUpgrade, "websocket")
}

// IsBinaryRequestBody checks if the request body is binary content.
// It uses cached header values from the header phase and can be called at any time.
func (ctx *CommonHttpCtx[PluginConfig]) IsBinaryRequestBody() bool {
	if strings.Contains(ctx.requestContentType, "octet-stream") ||
		strings.Contains(ctx.requestContentType, "grpc") {
		return true
	}
	return ctx.requestContentEncoding != ""
}

// IsBinaryResponseBody checks if the response body is binary content.
// It uses cached header values from the header phase and can be called at any time.
func (ctx *CommonHttpCtx[PluginConfig]) IsBinaryResponseBody() bool {
	if strings.Contains(ctx.responseContentType, "octet-stream") ||
		strings.Contains(ctx.responseContentType, "grpc") {
		return true
	}
	return ctx.responseContentEncoding != ""
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	defer recoverFunc()
	ctx.executionPhase = iface.DecodeHeader
	// Track if endOfStream was received in the header phase
	ctx.requestHeaderEndOfStream = endOfStream

	// Cache request pseudo-headers for later access outside of header phase
	ctx.scheme, _ = proxywasm.GetHttpRequestHeader(":scheme")
	ctx.host, _ = proxywasm.GetHttpRequestHeader(":authority")
	ctx.path, _ = proxywasm.GetHttpRequestHeader(":path")
	ctx.method, _ = proxywasm.GetHttpRequestHeader(":method")

	// Cache request headers for later access outside of header phase
	ctx.requestConnection, _ = proxywasm.GetHttpRequestHeader("connection")
	ctx.requestUpgrade, _ = proxywasm.GetHttpRequestHeader("upgrade")
	ctx.requestContentType, _ = proxywasm.GetHttpRequestHeader("content-type")
	ctx.requestContentEncoding, _ = proxywasm.GetHttpRequestHeader("content-encoding")

	requestID, _ := proxywasm.GetHttpRequestHeader("x-request-id")
	_ = proxywasm.SetProperty([]string{"x_request_id"}, []byte(requestID))

	// Increment request count and check rebuild condition
	if ctx.plugin.vm.rebuildAfterRequests > 0 {
		ctx.plugin.vm.requestCount++
		if ctx.plugin.vm.requestCount >= ctx.plugin.vm.rebuildAfterRequests {
			proxywasm.SetProperty([]string{"wasm_need_rebuild"}, []byte("true"))
			ctx.plugin.vm.log.Debugf("Plugin reached rebuild threshold after %d requests, rebuild flag set", ctx.plugin.vm.requestCount)
			ctx.plugin.vm.requestCount = 0
		}
	}

	// Check memory usage and rebuild condition
	if ctx.plugin.vm.rebuildMaxMem > 0 {
		data, err := proxywasm.GetProperty([]string{"plugin_vm_memory"})
		if err != nil {
			ctx.plugin.vm.log.Debugf("Failed to get VM memory: %v", err)
		} else if len(data) == 8 {
			memorySize := binary.LittleEndian.Uint64([]byte(data))
			ctx.plugin.vm.log.Debugf("Current VM memory usage: %d bytes (%.2f MB)",
				memorySize,
				float64(memorySize)/(1024*1024))

			if memorySize >= ctx.plugin.vm.rebuildMaxMem {
				proxywasm.SetProperty([]string{"wasm_need_rebuild"}, []byte("true"))
				ctx.plugin.vm.log.Debugf("Plugin reached rebuild memory threshold: %d bytes (%.2f MB), rebuild flag set",
					memorySize,
					float64(memorySize)/(1024*1024))
			}
		}
	}

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
	if ctx.IsBinaryRequestBody() {
		ctx.needRequestBody = false
	}
	if ctx.IsWebsocket() {
		ctx.needRequestBody = false
		ctx.needResponseBody = false
	}
	if ctx.plugin.vm.onHttpRequestHeaders == nil {
		return types.ActionContinue
	}
	return ctx.plugin.vm.onHttpRequestHeaders(ctx, *config)
}

func (ctx *CommonHttpCtx[PluginConfig]) OnHttpRequestBody(bodySize int, endOfStream bool) types.Action {
	defer recoverFunc()
	ctx.executionPhase = iface.DecodeData
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
	ctx.executionPhase = iface.EncodeHeader
	// Track if endOfStream was received in the header phase
	ctx.responseHeaderEndOfStream = endOfStream

	// Cache response headers for later access outside of header phase
	ctx.responseContentType, _ = proxywasm.GetHttpResponseHeader("content-type")
	ctx.responseContentEncoding, _ = proxywasm.GetHttpResponseHeader("content-encoding")

	if ctx.config == nil {
		return types.ActionContinue
	}
	// To avoid unexpected operations, plugins do not read the binary content body
	if ctx.IsBinaryResponseBody() {
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
			ctx.responseCallback(statusCode, headers, nil)
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
	ctx.executionPhase = iface.EncodeData
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
		ctx.responseCallback(statusCode, headers, body)
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
		if ctx.pauseStreamingResponse {
			return types.DataStopIterationNoBuffer
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
	ctx.executionPhase = iface.Done
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
func (ctx *CommonHttpCtx[PluginConfig]) RouteCall(method, rawURL string, headers [][2]string, body []byte, callback iface.RouteResponseCallback) error {
	proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	proxywasm.RemoveHttpRequestHeader("Content-Length")
	requestID := uuid.New().String()
	ctx.responseCallback = func(statusCode int, responseHeaders [][2]string, responseBody []byte) {
		callback(statusCode, responseHeaders, responseBody)
		log.UnsafeInfof("route call end, id:%s, code:%d, headers:%#v, body:%s", requestID, statusCode, responseHeaders, strings.ReplaceAll(string(responseBody), "\n", `\n`))
	}
	originalMethod, _ := proxywasm.GetHttpRequestHeader(":method")
	originalPath, _ := proxywasm.GetHttpRequestHeader(":path")
	originalHost, _ := proxywasm.GetHttpRequestHeader(":authority")
	proxywasm.ReplaceHttpRequestHeader("x-envoy-original-method", originalMethod)
	proxywasm.ReplaceHttpRequestHeader("x-envoy-original-path", originalPath)
	proxywasm.ReplaceHttpRequestHeader("x-envoy-original-host", originalHost)

	proxywasm.ReplaceHttpRequestHeader(":method", method)
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url:%s, err:%w", rawURL, err)
	}
	var authority string
	if parsedURL.Host != "" {
		authority = parsedURL.Host
	}
	path := "/" + strings.TrimPrefix(parsedURL.EscapedPath(), "/")
	if parsedURL.RawQuery != "" {
		path = fmt.Sprintf("%s?%s", path, parsedURL.RawQuery)
	}
	if parsedURL.Fragment != "" {
		path = fmt.Sprintf("%s#%s", path, parsedURL.Fragment)
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
	log.UnsafeInfof("route call start, id:%s, method:%s, url:%s, cluster:%s, headers:%#v, body:%s", requestID, method, rawURL, clusterName, reqHeaders, strings.ReplaceAll(string(body), "\n", `\n`))
	return nil
}

func recoverFunc() {
	if r := recover(); r != nil {
		// Check if panic recovery is disabled via environment variable
		if os.Getenv("WASM_DISABLE_PANIC_RECOVERY") == "true" {
			// Re-panic to preserve the original panic for debugging
			panic(r)
		}

		// Default behavior: recover and log the panic
		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		// Escape newlines to ensure the entire stack trace is printed on a single line,
		// which prevents log collection systems from splitting the stack trace into multiple entries
		escapedStack := strings.ReplaceAll(string(buf), "\n", "\\n")
		log.Errorf("recovered from panic %v, stack: %s", r, escapedStack)
	}
}
