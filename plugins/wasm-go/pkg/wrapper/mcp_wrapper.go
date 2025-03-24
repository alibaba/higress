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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/invopop/jsonschema"
	"github.com/tidwall/gjson"
)

type MCPTool[PluginConfig any] interface {
	Create(params []byte) MCPTool[PluginConfig]
	Call(context HttpContext, config PluginConfig) error
	Description() string
	InputSchema() map[string]any
}

type MCPTools[PluginConfig any] map[string]MCPTool[PluginConfig]

type addMCPToolOption[PluginConfig any] struct {
	name string
	tool MCPTool[PluginConfig]
}

func AddMCPTool[PluginConfig any](name string, tool MCPTool[PluginConfig]) CtxOption[PluginConfig] {
	return &addMCPToolOption[PluginConfig]{
		name: name,
		tool: tool,
	}
}

func (o *addMCPToolOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	ctx.isJsonRpcSever = true
	ctx.handleJsonRpcMethod = true
	if _, exist := ctx.mcpTools[o.name]; exist {
		panic(fmt.Sprintf("Conflict! There is a tool with the same name:%s",
			o.name))
	}
	ctx.mcpTools[o.name] = o.tool
}

func (ctx *CommonHttpCtx[PluginConfig]) OnMCPResponseSuccess(result map[string]any) {
	ctx.OnJsonRpcResponseSuccess(result)
	// TODO: support pub to redis when use POST + SSE
}

func (ctx *CommonHttpCtx[PluginConfig]) OnMCPResponseError(err error, code ...int) {
	ctx.OnJsonRpcResponseError(err, code...)
	// TODO: support pub to redis when use POST + SSE
}

func (ctx *CommonHttpCtx[PluginConfig]) OnMCPToolCallSuccess(content []map[string]any) {
	ctx.OnMCPResponseSuccess(map[string]any{
		"content": content,
		"isError": false,
	})
}

func (ctx *CommonHttpCtx[PluginConfig]) OnMCPToolCallError(err error) {
	ctx.OnMCPResponseSuccess(map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": err.Error(),
			},
		},
		"isError": true,
	})
}

func (ctx *CommonHttpCtx[PluginConfig]) SendMCPToolTextResult(result string) {
	ctx.OnMCPToolCallSuccess([]map[string]any{
		{
			"type": "text",
			"text": result,
		},
	})
}

func (ctx *CommonHttpCtx[PluginConfig]) registerMCPTools(mcpTools MCPTools[PluginConfig]) {
	if !ctx.plugin.vm.isJsonRpcSever {
		return
	}
	if !ctx.plugin.vm.handleJsonRpcMethod {
		return
	}
	ctx.plugin.vm.jsonRpcMethodHandlers["ping"] = func(context HttpContext, config PluginConfig, id int64, params gjson.Result) error {
		ctx.OnMCPResponseSuccess(map[string]any{})
		return nil
	}
	ctx.plugin.vm.jsonRpcMethodHandlers["initialize"] = func(context HttpContext, config PluginConfig, id int64, params gjson.Result) error {
		version := params.Get("protocolVersion").String()
		if version == "" {
			ctx.OnMCPResponseError(errors.New("Unsupported protocol version"), ErrInvalidParams)
		}
		ctx.OnMCPResponseSuccess(map[string]any{
			"protocolVersion": version,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    context.GetPluginName(),
				"version": "1.0.0",
			},
		})
		return nil
	}
	ctx.plugin.vm.jsonRpcMethodHandlers["tools/list"] = func(context HttpContext, config PluginConfig, id int64, params gjson.Result) error {
		var tools []map[string]any
		for name, tool := range mcpTools {
			tools = append(tools, map[string]any{
				"name":        name,
				"description": tool.Description(),
				"inputSchema": tool.InputSchema(),
			})
		}
		ctx.OnMCPResponseSuccess(map[string]any{
			"tools":      tools,
			"nextCursor": "",
		})
		return nil
	}
	ctx.plugin.vm.jsonRpcMethodHandlers["tools/call"] = func(context HttpContext, config PluginConfig, id int64, params gjson.Result) error {
		name := params.Get("name").String()
		args := params.Get("arguments")
		if tool, ok := mcpTools[name]; ok {
			log.Debugf("mcp call tool[%s] with arguments[%s]", name, args.Raw)
			toolInstance := tool.Create([]byte(args.Raw))
			err := toolInstance.Call(context, config)
			// TODO: validate the json schema through github.com/kaptinlin/jsonschema
			if err != nil {
				ctx.OnMCPToolCallError(err)
				return nil
			}
			return nil
		}
		ctx.OnMCPResponseError(errors.New("Unknown tool: invalid_tool_name"), ErrInvalidParams)
		return nil
	}
}

type mcpToolRequestFunc[PluginConfig any] func(context HttpContext, config PluginConfig, toolName string, toolArgs gjson.Result) types.Action
type mcpToolResponseFunc[PluginConfig any] func(context HttpContext, config PluginConfig, isError bool, content gjson.Result) types.Action
type jsonRpcErrorFunc[PluginConfig any] func(context HttpContext, config PluginConfig, errorCode int64, errorMessage string) types.Action

type mcpToolRequestOption[PluginConfig any] struct {
	f mcpToolRequestFunc[PluginConfig]
}

func OnMCPToolRequest[PluginConfig any](f mcpToolRequestFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &mcpToolRequestOption[PluginConfig]{f}
}

func (o *mcpToolRequestOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	ctx.isJsonRpcSever = true
	ctx.onMcpToolRequest = o.f
}

type mcpToolResponseOption[PluginConfig any] struct {
	f mcpToolResponseFunc[PluginConfig]
}

func OnMCPToolResponse[PluginConfig any](f mcpToolResponseFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &mcpToolResponseOption[PluginConfig]{f}
}

func (o *mcpToolResponseOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	ctx.isJsonRpcSever = true
	ctx.onMcpToolResponse = o.f
}

type jsonRpcErrorOption[PluginConfig any] struct {
	f jsonRpcErrorFunc[PluginConfig]
}

func OnJsonRpcError[PluginConfig any](f jsonRpcErrorFunc[PluginConfig]) CtxOption[PluginConfig] {
	return &jsonRpcErrorOption[PluginConfig]{f}
}

func (o *jsonRpcErrorOption[PluginConfig]) Apply(ctx *CommonVmCtx[PluginConfig]) {
	ctx.isJsonRpcSever = true
	ctx.onJsonRpcError = o.f
}

func (ctx *CommonHttpCtx[PluginConfig]) registerMCPToolProcessor() {
	if !ctx.plugin.vm.isJsonRpcSever {
		return
	}
	if ctx.plugin.vm.handleJsonRpcMethod {
		return
	}
	if ctx.plugin.vm.onMcpToolRequest != nil {
		ctx.plugin.vm.jsonRpcRequestHandler = func(context HttpContext, config PluginConfig, id int64, method string, params gjson.Result) types.Action {
			toolName := params.Get("name").String()
			toolArgs := params.Get("arguments")
			return ctx.plugin.vm.onMcpToolRequest(context, config, toolName, toolArgs)
		}
	}
	if ctx.plugin.vm.onMcpToolResponse != nil {
		ctx.plugin.vm.jsonRpcResponseHandler = func(context HttpContext, config PluginConfig, id int64, result, error gjson.Result) types.Action {
			if result.Exists() {
				isError := result.Get("isError").Bool()
				content := result.Get("content")
				return ctx.plugin.vm.onMcpToolResponse(context, config, isError, content)
			}
			if error.Exists() && ctx.plugin.vm.onJsonRpcError != nil {
				return ctx.plugin.vm.onJsonRpcError(context, config, error.Get("code").Int(), error.Get("message").String())
			}
			return types.ActionContinue
		}
	}
}

func ToInputSchema(v any) map[string]any {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	inputSchema := jsonschema.Reflect(v).Definitions[t.Name()]
	inputSchemaBytes, _ := json.Marshal(inputSchema)
	var result map[string]any
	json.Unmarshal(inputSchemaBytes, &result)
	return result
}

func (ctx *CommonHttpCtx[PluginConfig]) ParseMCPServerConfig(config any) error {
	serverConfigBase64, _ := proxywasm.GetHttpRequestHeader("x-higress-mcpserver-config")
	if serverConfigBase64 == "" {
		log.Info("mcp server config from request is empty")
		return nil
	}
	serverConfig, err := base64.StdEncoding.DecodeString(serverConfigBase64)
	if err != nil {
		return fmt.Errorf("base64 decode mcp server config failed:%s, bytes:%s", err, serverConfigBase64)
	}
	return json.Unmarshal(serverConfig, config)
}
