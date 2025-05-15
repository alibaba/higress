// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/invopop/jsonschema"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/utils"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	defaultMaxBodyBytes uint32 = 100 * 1024 * 1024
)

type HttpContext wrapper.HttpContext

type Context struct {
	servers map[string]Server
}

type CtxOption interface {
	Apply(*Context)
}

var globalContext Context

type Server interface {
	AddMCPTool(name string, tool Tool) Server
	GetMCPTools() map[string]Tool
	SetConfig(config []byte)
	GetConfig(v any)
	Clone() Server
}

type Tool interface {
	Create(params []byte) Tool
	Call(httpCtx HttpContext, server Server) error
	Description() string
	InputSchema() map[string]any
}

type mcpServerConfig struct {
	server         Server
	methodHandlers utils.MethodHandlers
}

func parseConfig(configJson gjson.Result, config *mcpServerConfig) error {
	serverJson := configJson.Get("server")
	if !serverJson.Exists() {
		return errors.New("server field is missing")
	}
	serverName := serverJson.Get("name").String()
	if serverName == "" {
		return errors.New("server.name field is missing")
	}
	serverConfigJson := serverJson.Get("config").Raw

	// Parse allowTools
	allowToolsArray := configJson.Get("allowTools").Array()
	allowTools := make(map[string]struct{})
	for _, toolJson := range allowToolsArray {
		allowTools[toolJson.String()] = struct{}{}
	}

	// Check if we have REST tools defined
	toolsJson := configJson.Get("tools")
	if toolsJson.Exists() && len(toolsJson.Array()) > 0 {
		// Create REST-to-MCP server
		restServer := NewRestMCPServer(serverName)
		restServer.SetConfig([]byte(serverConfigJson))

		// Parse security schemes
		securitySchemesJson := serverJson.Get("securitySchemes")
		if securitySchemesJson.Exists() {
			for _, schemeJson := range securitySchemesJson.Array() {
				var scheme SecurityScheme
				if err := json.Unmarshal([]byte(schemeJson.Raw), &scheme); err != nil {
					return fmt.Errorf("failed to parse security scheme config: %v", err)
				}
				restServer.AddSecurityScheme(scheme)
			}
		}

		// Parse and add tools
		for _, toolJson := range toolsJson.Array() {
			var restTool RestTool
			if err := json.Unmarshal([]byte(toolJson.Raw), &restTool); err != nil {
				return fmt.Errorf("failed to parse tool config: %v", err)
			}

			if err := restServer.AddRestTool(restTool); err != nil {
				return fmt.Errorf("failed to add tool %s: %v", restTool.Name, err)
			}
		}
		config.server = restServer
	} else {
		// Original logic for registered servers
		if server, exist := globalContext.servers[serverName]; exist {
			config.server = server.Clone()
			config.server.SetConfig([]byte(serverConfigJson))
		} else {
			return fmt.Errorf("mcp server not found:%s", serverName)
		}
	}
	config.methodHandlers = make(utils.MethodHandlers)
	config.methodHandlers["ping"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		utils.OnMCPResponseSuccess(true, ctx, map[string]any{}, "mcp:ping")
		return nil
	}
	config.methodHandlers["notifications/initialized"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		proxywasm.SendHttpResponseWithDetail(202, "mcp:notifications/initialized", nil, nil, -1)
		return nil
	}
	config.methodHandlers["notifications/cancelled"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		proxywasm.SendHttpResponseWithDetail(202, "mcp:notifications/cancelled", nil, nil, -1)
		return nil
	}
	config.methodHandlers["initialize"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		version := params.Get("protocolVersion").String()
		if version == "" {
			utils.OnMCPResponseError(true, ctx, errors.New("Unsupported protocol version"), utils.ErrInvalidParams, "mcp:initialize:error")
		}
		utils.OnMCPResponseSuccess(true, ctx, map[string]any{
			"protocolVersion": version,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    serverName,
				"version": "1.0.0",
			},
		}, "mcp:initialize")
		return nil
	}
	var tools []map[string]any
	for name, tool := range config.server.GetMCPTools() {
		if len(allowTools) != 0 {
			if _, allow := allowTools[name]; !allow {
				continue
			}
		}
		tools = append(tools, map[string]any{
			"name":        name,
			"description": tool.Description(),
			"inputSchema": tool.InputSchema(),
		})
	}
	config.methodHandlers["tools/list"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		utils.OnMCPResponseSuccess(true, ctx, map[string]any{
			"tools":      tools,
			"nextCursor": "",
		}, "mcp:tools/list")
		return nil
	}
	config.methodHandlers["tools/call"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		name := params.Get("name").String()
		args := params.Get("arguments")
		if len(allowTools) != 0 {
			if _, allow := allowTools[name]; !allow {
				utils.OnMCPResponseError(true, ctx, errors.New("Unknown tool: invalid_tool_name"), utils.ErrInvalidParams, "mcp:tools/call:invalid_tool_name")
				return nil
			}
		}
		proxywasm.SetProperty([]string{"mcp_server_name"}, []byte(serverName))
		proxywasm.SetProperty([]string{"mcp_tool_name"}, []byte(name))
		if tool, ok := config.server.GetMCPTools()[name]; ok {
			log.Debugf("tool call [%s] with arguments[%s]", name, args.Raw)
			toolInstance := tool.Create([]byte(args.Raw))
			err := toolInstance.Call(ctx, config.server)
			// TODO: validate the json schema through github.com/kaptinlin/jsonschema
			if err != nil {
				utils.OnMCPToolCallError(true, ctx, err)
				return nil
			}
			return nil
		}
		utils.OnMCPResponseError(true, ctx, errors.New("Unknown tool: invalid_tool_name"), utils.ErrInvalidParams, "mcp:tools/call:invalid_tool_name")
		return nil
	}

	return nil
}

func Load(options ...CtxOption) {
	for _, opt := range options {
		opt.Apply(&globalContext)
	}
}

func Initialize() {
	if globalContext.servers == nil {
		panic("At least one mcpserver needs to be added.")
	}
	wrapper.SetCtx(
		"mcp-server",
		wrapper.ParseConfig(parseConfig),
		wrapper.WithLogger[mcpServerConfig](&utils.MCPServerLog{}),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
	)
}

type addMCPServerOption struct {
	name   string
	server Server
}

func AddMCPServer(name string, server Server) CtxOption {
	return &addMCPServerOption{
		name:   name,
		server: server,
	}
}

func (o *addMCPServerOption) Apply(ctx *Context) {
	if ctx.servers == nil {
		ctx.servers = make(map[string]Server)
	}
	if _, exist := ctx.servers[o.name]; exist {
		panic(fmt.Sprintf("Conflict! There is a mcp server with the same name:%s",
			o.name))
	}
	ctx.servers[o.name] = o.server
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

func StoreServerState(ctx wrapper.HttpContext, config any) {
	if utils.IsStatefulSession(ctx) {
		log.Warnf("There is no session ID, unable to store state.")
		return
	}
	configBytes, err := json.Marshal(config)
	if err != nil {
		log.Errorf("Server config marshal failed:%v, config:%s", err, configBytes)
		return
	}
	proxywasm.SetProperty([]string{"mcp_server_config"}, configBytes)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config mcpServerConfig) types.Action {
	ctx.DisableReroute()
	ctx.SetRequestBodyBufferLimit(defaultMaxBodyBytes)
	ctx.SetResponseBodyBufferLimit(defaultMaxBodyBytes)

	if ctx.Method() == "GET" {
		proxywasm.SendHttpResponseWithDetail(405, "not_support_sse_on_this_endpoint", nil, nil, -1)
		return types.HeaderStopAllIterationAndWatermark
	}
	if !wrapper.HasRequestBody() {
		proxywasm.SendHttpResponseWithDetail(400, "missing_body_in_mcp_request", nil, nil, -1)
		return types.HeaderStopAllIterationAndWatermark
	}
	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, config mcpServerConfig, body []byte) types.Action {
	return utils.HandleJsonRpcMethod(ctx, body, config.methodHandlers)
}
