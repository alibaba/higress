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

package filter

import (
	"github.com/tidwall/gjson"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/utils"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	defaultMaxBodyBytes uint32 = 100 * 1024 * 1024
)

type RequestFilterF func(context wrapper.HttpContext, config any, toolName string, toolArgs gjson.Result) types.Action

type ResponseFilterF func(context wrapper.HttpContext, config any, isError bool, content gjson.Result) types.Action

type OnJsonRpcErrorF func(context wrapper.HttpContext, config any, code int64, message string) types.Action

type Context struct {
	filterName        string
	requestFilter     RequestFilterF
	responseFilter    ResponseFilterF
	onJsonRpcError    OnJsonRpcErrorF
	parseFilterConfig ParseFilterConfigF
}

type CtxOption interface {
	Apply(*Context)
}

var globalContext Context

type ParseFilterConfigF func(configBytes []byte, filterConfig any) error

type setConfigParserOption struct {
	f ParseFilterConfigF
}

func SetConfigParser(f ParseFilterConfigF) CtxOption {
	return &setConfigParserOption{f}
}

func (o *setConfigParserOption) Apply(ctx *Context) {
	ctx.parseFilterConfig = o.f
}

type filterNameOption struct {
	name string
}

func FilterName(name string) CtxOption {
	return &filterNameOption{name}
}

func (o *filterNameOption) Apply(ctx *Context) {
	ctx.filterName = o.name
}

type setRequestFilterOption struct {
	f RequestFilterF
}

func SetRequestFilter(f RequestFilterF) CtxOption {
	return &setRequestFilterOption{f}
}

func (o *setRequestFilterOption) Apply(ctx *Context) {
	ctx.requestFilter = o.f
}

type setResponseFilterOption struct {
	f ResponseFilterF
}

func SetResponseFilter(f ResponseFilterF) CtxOption {
	return &setResponseFilterOption{f}
}

func (o *setResponseFilterOption) Apply(ctx *Context) {
	ctx.responseFilter = o.f
}

type onJsonRpcErrorOption struct {
	f OnJsonRpcErrorF
}

func OnJsonRpcError(f OnJsonRpcErrorF) CtxOption {
	return &onJsonRpcErrorOption{f}
}

func (o *onJsonRpcErrorOption) Apply(ctx *Context) {
	ctx.onJsonRpcError = o.f
}

func Load(options ...CtxOption) {
	for _, opt := range options {
		opt.Apply(&globalContext)
	}
}

func Initialize() {
	if globalContext.filterName == "" {
		panic("FilterName not set")
	}
	if globalContext.parseFilterConfig == nil {
		panic("SetConfigParser not set")
	}
	if globalContext.requestFilter == nil && globalContext.responseFilter == nil {
		panic("At least one of SetRequestFilter or SetResponseFilter needs to be set.")
	}
	wrapper.SetCtx(
		globalContext.filterName,
		wrapper.ParseRawConfig(parseRawConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseBody(onHttpResponseBody),
	)
}

type mcpFilterConfig struct {
	config          any
	requestHandler  utils.JsonRpcRequestHandler
	responseHandler utils.JsonRpcResponseHandler
}

func parseRawConfig(configBytes []byte, config *mcpFilterConfig) error {
	err := globalContext.parseFilterConfig(configBytes, config.config)
	if err != nil {
		return err
	}
	config.requestHandler = func(context wrapper.HttpContext, id utils.JsonRpcID, method string, params gjson.Result) types.Action {
		if globalContext.requestFilter == nil {
			return types.ActionContinue
		}
		toolName := params.Get("name").String()
		toolArgs := params.Get("arguments")
		return globalContext.requestFilter(context, config.config, toolName, toolArgs)
	}
	config.responseHandler = func(context wrapper.HttpContext, id utils.JsonRpcID, result, error gjson.Result) types.Action {
		if result.Exists() && globalContext.responseFilter != nil {
			isError := result.Get("isError").Bool()
			content := result.Get("content")
			return globalContext.responseFilter(context, config.config, isError, content)
		}
		if error.Exists() && globalContext.onJsonRpcError != nil {
			return globalContext.onJsonRpcError(context, config.config, error.Get("code").Int(), error.Get("message").String())
		}
		return types.ActionContinue
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config mcpFilterConfig) types.Action {
	if !wrapper.HasRequestBody() || globalContext.requestFilter == nil {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	ctx.SetRequestBodyBufferLimit(defaultMaxBodyBytes)
	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, config mcpFilterConfig, body []byte) types.Action {
	return utils.HandleJsonRpcRequest(ctx, body, config.requestHandler)
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config mcpFilterConfig) types.Action {
	if !wrapper.HasResponseBody() || globalContext.responseFilter == nil {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	ctx.SetResponseBodyBufferLimit(defaultMaxBodyBytes)
	return types.HeaderStopIteration
}

func onHttpResponseBody(ctx wrapper.HttpContext, config mcpFilterConfig, body []byte) types.Action {
	return utils.HandleJsonRpcResponse(ctx, body, config.responseHandler)
}
