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
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	CtxJsonRpcID = "jsonRpcID"
	JError       = "error"
	JCode        = "code"
	JMessage     = "message"
	JResult      = "result"

	ErrParseError     = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternalError  = -32603
)

type JsonRpcRequestHandler[PluginConfig any] func(context HttpContext, config PluginConfig, id int64, method string, params gjson.Result) types.Action

type JsonRpcResponseHandler[PluginConfig any] func(context HttpContext, config PluginConfig, id int64, result gjson.Result, error gjson.Result) types.Action

type JsonRpcMethodHandler[PluginConfig any] func(context HttpContext, config PluginConfig, id int64, params gjson.Result) error

type MethodHandlers[PluginConfig any] map[string]JsonRpcMethodHandler[PluginConfig]

func sendJsonRpcResponse(id int64, extras map[string]any, debugInfo string) {
	body := []byte(`{"jsonrpc": "2.0"}`)
	body, _ = sjson.SetBytes(body, "id", id)
	for key, value := range extras {
		body, _ = sjson.SetBytes(body, key, value)
	}
	proxywasm.SendHttpResponseWithDetail(200, debugInfo, [][2]string{{"Content-Type", "application/json; charset=utf-8"}}, body, -1)
}

func (ctx *CommonHttpCtx[PluginConfig]) OnJsonRpcResponseSuccess(result map[string]any) {
	var (
		id int64
		ok bool
	)
	if id, ok = ctx.userContext[CtxJsonRpcID].(int64); !ok {
		proxywasm.SendHttpResponseWithDetail(500, "not_found_json_rpc_id", nil, []byte("not found json rpc id"), -1)
		return
	}
	sendJsonRpcResponse(id, map[string]any{JResult: result}, "json_rpc_success")
}

func (ctx *CommonHttpCtx[PluginConfig]) OnJsonRpcResponseError(err error, code ...int) {
	var (
		id int64
		ok bool
	)
	if id, ok = ctx.userContext[CtxJsonRpcID].(int64); !ok {
		proxywasm.SendHttpResponseWithDetail(500, "not_found_json_rpc_id", nil, []byte("not found json rpc id"), -1)
		return
	}
	errorCode := ErrInternalError
	if len(code) > 0 {
		errorCode = code[0]
	}
	sendJsonRpcResponse(id, map[string]any{JError: map[string]any{
		JMessage: err.Error(),
		JCode:    errorCode,
	}}, "json_rpc_error")
}

func (ctx *CommonHttpCtx[PluginConfig]) HandleJsonRpcMethod(context HttpContext, config PluginConfig, body []byte, handles MethodHandlers[PluginConfig]) types.Action {
	id := gjson.GetBytes(body, "id").Int()
	ctx.userContext[CtxJsonRpcID] = id
	method := gjson.GetBytes(body, "method").String()
	params := gjson.GetBytes(body, "params")
	if handle, ok := handles[method]; ok {
		log.Debugf("json rpc call id[%d] method[%s] with params[%s]", id, method, params.Raw)
		err := handle(context, config, id, params)
		if err != nil {
			ctx.OnJsonRpcResponseError(err)
			return types.ActionContinue
		}
		// Waiting for the response
		return types.ActionPause
	}
	ctx.OnJsonRpcResponseError(fmt.Errorf("method not found:%s", method), ErrMethodNotFound)
	return types.ActionContinue
}

func (ctx *CommonHttpCtx[PluginConfig]) HandleJsonRpcRequest(context HttpContext, config PluginConfig, body []byte, handle JsonRpcRequestHandler[PluginConfig]) types.Action {
	id := gjson.GetBytes(body, "id").Int()
	ctx.userContext[CtxJsonRpcID] = id
	method := gjson.GetBytes(body, "method").String()
	params := gjson.GetBytes(body, "params")
	log.Debugf("json rpc call id[%d] method[%s] with params[%s]", id, method, params.Raw)
	return handle(context, config, id, method, params)
}

func (ctx *CommonHttpCtx[PluginConfig]) HandleJsonRpcResponse(context HttpContext, config PluginConfig, body []byte, handle JsonRpcResponseHandler[PluginConfig]) types.Action {
	id := gjson.GetBytes(body, "id").Int()
	error := gjson.GetBytes(body, "error")
	result := gjson.GetBytes(body, "result")
	log.Debugf("json rpc response id[%d] error[%s] result[%s]", id, error.Raw, result.Raw)
	return handle(context, config, id, result, error)
}
