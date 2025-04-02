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

package utils

import (
	"fmt"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
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

// JsonRpcID represents a JSON-RPC ID which can be either a string or a number
type JsonRpcID struct {
	StringValue string
	IntValue    int64
	IsString    bool
}

// NewJsonRpcIDFromGjson creates a JsonRpcID from a gjson.Result
func NewJsonRpcIDFromGjson(result gjson.Result) JsonRpcID {
	if result.Type == gjson.String {
		return JsonRpcID{
			StringValue: result.String(),
			IsString:    true,
		}
	}
	return JsonRpcID{
		IntValue: result.Int(),
		IsString: false,
	}
}

type JsonRpcRequestHandler func(context wrapper.HttpContext, id JsonRpcID, method string, params gjson.Result) types.Action

type JsonRpcResponseHandler func(context wrapper.HttpContext, id JsonRpcID, result gjson.Result, error gjson.Result) types.Action

type JsonRpcMethodHandler func(context wrapper.HttpContext, id JsonRpcID, params gjson.Result) error

type MethodHandlers map[string]JsonRpcMethodHandler

func sendJsonRpcResponse(id JsonRpcID, extras map[string]any, debugInfo string) {
	body := []byte(`{"jsonrpc": "2.0"}`)
	if id.IsString {
		body, _ = sjson.SetBytes(body, "id", id.StringValue)
	} else {
		body, _ = sjson.SetBytes(body, "id", id.IntValue)
	}
	for key, value := range extras {
		body, _ = sjson.SetBytes(body, key, value)
	}
	proxywasm.SendHttpResponseWithDetail(200, debugInfo, [][2]string{{"Content-Type", "application/json; charset=utf-8"}}, body, -1)
}

func OnJsonRpcResponseSuccess(ctx wrapper.HttpContext, result map[string]any, debugInfo ...string) {
	var (
		id JsonRpcID
		ok bool
	)
	idRaw := ctx.GetContext(CtxJsonRpcID)
	if id, ok = idRaw.(JsonRpcID); !ok {
		proxywasm.SendHttpResponseWithDetail(500, "not_found_json_rpc_id", nil, []byte("not found json rpc id"), -1)
		return
	}
	responseDebugInfo := "json_rpc_success"
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}
	sendJsonRpcResponse(id, map[string]any{JResult: result}, responseDebugInfo)
}

func OnJsonRpcResponseError(ctx wrapper.HttpContext, err error, errorCode int, debugInfo ...string) {
	var (
		id JsonRpcID
		ok bool
	)
	idRaw := ctx.GetContext(CtxJsonRpcID)
	if id, ok = idRaw.(JsonRpcID); !ok {
		proxywasm.SendHttpResponseWithDetail(500, "not_found_json_rpc_id", nil, []byte("not found json rpc id"), -1)
		return
	}
	responseDebugInfo := fmt.Sprintf("json_rpc_error(%s)", err)
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}
	sendJsonRpcResponse(id, map[string]any{JError: map[string]any{
		JMessage: err.Error(),
		JCode:    errorCode,
	}}, responseDebugInfo)
}

func HandleJsonRpcMethod(ctx wrapper.HttpContext, body []byte, handles MethodHandlers) types.Action {
	idResult := gjson.GetBytes(body, "id")
	id := NewJsonRpcIDFromGjson(idResult)
	ctx.SetContext(CtxJsonRpcID, id)
	method := gjson.GetBytes(body, "method").String()
	params := gjson.GetBytes(body, "params")
	if handle, ok := handles[method]; ok {
		log.Debugf("json rpc call method[%s] with params[%s]", method, params.Raw)
		err := handle(ctx, id, params)
		if err != nil {
			OnJsonRpcResponseError(ctx, err, ErrInvalidRequest)
			return types.ActionContinue
		}
		// Waiting for the response
		return types.ActionPause
	}
	OnJsonRpcResponseError(ctx, fmt.Errorf("method not found:%s", method), ErrMethodNotFound)
	return types.ActionContinue
}

func HandleJsonRpcRequest(ctx wrapper.HttpContext, body []byte, handle JsonRpcRequestHandler) types.Action {
	idResult := gjson.GetBytes(body, "id")
	id := NewJsonRpcIDFromGjson(idResult)
	ctx.SetContext(CtxJsonRpcID, id)
	method := gjson.GetBytes(body, "method").String()
	params := gjson.GetBytes(body, "params")
	log.Debugf("json rpc call method[%s] with params[%s]", method, params.Raw)
	return handle(ctx, id, method, params)
}

func HandleJsonRpcResponse(ctx wrapper.HttpContext, body []byte, handle JsonRpcResponseHandler) types.Action {
	idResult := gjson.GetBytes(body, "id")
	id := NewJsonRpcIDFromGjson(idResult)
	error := gjson.GetBytes(body, "error")
	result := gjson.GetBytes(body, "result")
	log.Debugf("json rpc response error[%s] result[%s]", error.Raw, result.Raw)
	return handle(ctx, id, result, error)
}
