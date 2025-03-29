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
	"encoding/base64"
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
	getMCPTools() map[string]Tool
	setConfig(config []byte)
	clone() Server

	Valid() error
	AddMCPTool(name string, tool Tool) Server
	GetConfig(v any)
}

type MCPServer struct {
	tools  map[string]Tool
	config []byte
}

func (s MCPServer) clone() Server {
	return &MCPServer{tools: s.tools}
}

func (s MCPServer) Valid() error {
	return nil
}

func (s *MCPServer) AddMCPTool(name string, tool Tool) Server {
	if s.tools == nil {
		s.tools = make(map[string]Tool)
	}
	if _, exist := s.tools[name]; exist {
		panic(fmt.Sprintf("Conflict! There is a tool with the same name:%s",
			name))
	}
	s.tools[name] = tool
	return s
}

// Can only be called during a tool call
func (s *MCPServer) GetConfig(v any) {
	var config []byte
	serverConfigBase64, _ := proxywasm.GetHttpRequestHeader("x-higress-mcpserver-config")
	proxywasm.RemoveHttpRequestHeader("x-higress-mcpserver-config")
	if serverConfigBase64 != "" {
		log.Info("parse server config from request")
		serverConfig, err := base64.StdEncoding.DecodeString(serverConfigBase64)
		if err != nil {
			log.Errorf("base64 decode mcp server config failed:%s, bytes:%s", err, serverConfigBase64)
		} else {
			config = serverConfig
		}
	} else {
		config = s.config
	}
	err := json.Unmarshal(config, v)
	if err != nil {
		log.Errorf("json unmarshal server config failed:%v, config:%s", err, config)
	}
}

func (s *MCPServer) getMCPTools() map[string]Tool {
	return s.tools
}

func (s *MCPServer) setConfig(config []byte) {
	s.config = config

}

type Tool interface {
	Create(params []byte) Tool
	Call(httpCtx HttpContext, server Server) error
	Description() string
	InputSchema() map[string]any
}

type mcpServerConfig struct {
	name           string
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
	if server, exist := globalContext.servers[serverName]; exist {
		config.server = server.clone()
		config.server.setConfig([]byte(serverJson.Get("config").Raw))
	} else {
		return fmt.Errorf("mcp server not found:%s", serverName)
	}
	config.methodHandlers = make(utils.MethodHandlers)
	config.methodHandlers["ping"] = func(ctx wrapper.HttpContext, id int64, params gjson.Result) error {
		proxywasm.SetProperty([]string{"mcp_server_name"}, []byte(serverName))
		utils.OnMCPResponseSuccess(ctx, map[string]any{})
		return nil
	}
	config.methodHandlers["initialize"] = func(ctx wrapper.HttpContext, id int64, params gjson.Result) error {
		proxywasm.SetProperty([]string{"mcp_server_name"}, []byte(serverName))
		version := params.Get("protocolVersion").String()
		if version == "" {
			utils.OnMCPResponseError(ctx, errors.New("Unsupported protocol version"), utils.ErrInvalidParams)
		}
		utils.OnMCPResponseSuccess(ctx, map[string]any{
			"protocolVersion": version,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    serverName,
				"version": "1.0.0",
			},
		})
		return nil
	}
	config.methodHandlers["tools/list"] = func(ctx wrapper.HttpContext, id int64, params gjson.Result) error {
		proxywasm.SetProperty([]string{"mcp_server_name"}, []byte(serverName))
		var tools []map[string]any
		for name, tool := range config.server.getMCPTools() {
			tools = append(tools, map[string]any{
				"name":        name,
				"description": tool.Description(),
				"inputSchema": tool.InputSchema(),
			})
		}
		utils.OnMCPResponseSuccess(ctx, map[string]any{
			"tools":      tools,
			"nextCursor": "",
		})
		return nil
	}
	config.methodHandlers["tools/call"] = func(ctx wrapper.HttpContext, id int64, params gjson.Result) error {
		name := params.Get("name").String()
		args := params.Get("arguments")
		proxywasm.SetProperty([]string{"mcp_server_name"}, []byte(serverName))
		proxywasm.SetProperty([]string{"mcp_tool_name"}, []byte(name))
		if tool, ok := config.server.getMCPTools()[name]; ok {
			log.Debugf("tool call with arguments[%s]", name, args.Raw)
			toolInstance := tool.Create([]byte(args.Raw))
			err := toolInstance.Call(ctx, config.server)
			// TODO: validate the json schema through github.com/kaptinlin/jsonschema
			if err != nil {
				utils.OnMCPToolCallError(ctx, err)
				return nil
			}
			return nil
		}
		utils.OnMCPResponseError(ctx, errors.New("Unknown tool: invalid_tool_name"), utils.ErrInvalidParams)
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
	ctx.SetRequestBodyBufferLimit(defaultMaxBodyBytes)
	ctx.SetResponseBodyBufferLimit(defaultMaxBodyBytes)
	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, config mcpServerConfig, body []byte) types.Action {
	return utils.HandleJsonRpcMethod(ctx, body, config.methodHandlers)
}
