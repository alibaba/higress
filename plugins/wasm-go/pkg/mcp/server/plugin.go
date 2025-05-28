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
var globalToolRegistry GlobalToolRegistry

// ToolInfo stores information about a tool for the global registry.
type ToolInfo struct {
	Name        string
	Description string
	InputSchema map[string]any
	ServerName  string // Original server name
	Tool        Tool   // The actual tool instance for cloning
}

// GlobalToolRegistry holds all tools from all servers.
type GlobalToolRegistry struct {
	// serverName -> toolName -> toolInfo
	serverTools map[string]map[string]ToolInfo
}

func init() {
	globalToolRegistry = GlobalToolRegistry{
		serverTools: make(map[string]map[string]ToolInfo),
	}
}

// RegisterTool registers a tool into the global registry.
func (r *GlobalToolRegistry) RegisterTool(serverName string, toolName string, tool Tool) {
	if _, ok := r.serverTools[serverName]; !ok {
		r.serverTools[serverName] = make(map[string]ToolInfo)
	}
	r.serverTools[serverName][toolName] = ToolInfo{
		Name:        toolName,
		Description: tool.Description(),
		InputSchema: tool.InputSchema(),
		ServerName:  serverName,
		Tool:        tool,
	}
	log.Debugf("Registered tool %s/%s", serverName, toolName)
}

// GetToolInfo retrieves tool information from the global registry.
func (r *GlobalToolRegistry) GetToolInfo(serverName string, toolName string) (ToolInfo, bool) {
	if serverTools, ok := r.serverTools[serverName]; ok {
		toolInfo, found := serverTools[toolName]
		return toolInfo, found
	}
	return ToolInfo{}, false
}

// GetServer retrieves a server instance from the global context.
// This is needed by ComposedMCPServer to get original server instances.
func GetServerFromGlobalContext(serverName string) (Server, bool) {
	server, exist := globalContext.servers[serverName]
	return server, exist
}

type Server interface {
	AddMCPTool(name string, tool Tool) Server
	GetMCPTools() map[string]Tool // For single server, returns its tools. For composed, returns composed tools.
	SetConfig(config []byte)
	GetConfig(v any)
	Clone() Server
	// GetName() string // Returns the server name - REMOVED
}

type Tool interface {
	Create(params []byte) Tool
	Call(httpCtx HttpContext, server Server) error
	Description() string
	InputSchema() map[string]any
}

// ToolSetConfig defines the configuration for a toolset.
type ToolSetConfig struct {
	Name        string             `json:"name"`
	ServerTools []ServerToolConfig `json:"serverTools"`
}

// ServerToolConfig specifies which tools from a server to include in a toolset.
type ServerToolConfig struct {
	ServerName string   `json:"serverName"`
	Tools      []string `json:"tools"`
}

type mcpServerConfig struct {
	serverName     string // Store the server name directly
	server         Server // Can be a single server or a composed server
	methodHandlers utils.MethodHandlers
	toolSet        *ToolSetConfig // Parsed toolset configuration
	isComposed     bool
}

func parseConfig(configJson gjson.Result, config *mcpServerConfig) error {
	toolSetJson := configJson.Get("toolSet")
	serverJson := configJson.Get("server")                        // This is for single server or REST server definition
	pluginServerConfigJson := configJson.Get("server.config").Raw // Config for the plugin instance itself, if any.

	// serverConfigJsonForInstance is the config passed to the specific server instance (single or REST)
	// It's distinct from pluginServerConfigJson which might be for the mcp-server plugin itself.
	var serverConfigJsonForInstance string

	if toolSetJson.Exists() {
		config.isComposed = true
		var tsConfig ToolSetConfig
		if err := json.Unmarshal([]byte(toolSetJson.Raw), &tsConfig); err != nil {
			return fmt.Errorf("failed to parse toolSet config: %v", err)
		}
		config.toolSet = &tsConfig
		config.serverName = tsConfig.Name // Use toolSet name as the server name for composed server
		log.Infof("Parsing toolSet configuration: %s", config.serverName)

		composedServer := NewComposedMCPServer(config.serverName, tsConfig.ServerTools, &globalToolRegistry)
		// A composed server itself might have a config block, e.g. for shared settings, though not typical.
		composedServer.SetConfig([]byte(pluginServerConfigJson))
		config.server = composedServer
	} else if serverJson.Exists() {
		config.isComposed = false
		config.serverName = serverJson.Get("name").String()
		if config.serverName == "" {
			return errors.New("server.name field is missing for single server config")
		}
		// This is the config for the specific server being defined (e.g. REST server's own config)
		serverConfigJsonForInstance = serverJson.Get("config").Raw
		log.Infof("Parsing single server configuration: %s", config.serverName)

		// Original logic for single server
		toolsJson := configJson.Get("tools") // These are REST tools for this server instance
		if toolsJson.Exists() && len(toolsJson.Array()) > 0 {
			// Create REST-to-MCP server
			restServer := NewRestMCPServer(config.serverName)         // Pass the server name
			restServer.SetConfig([]byte(serverConfigJsonForInstance)) // Pass the server's specific config

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

			for _, toolJson := range toolsJson.Array() {
				var restTool RestTool
				if err := json.Unmarshal([]byte(toolJson.Raw), &restTool); err != nil {
					return fmt.Errorf("failed to parse tool config: %v", err)
				}

				if err := restServer.AddRestTool(restTool); err != nil {
					return fmt.Errorf("failed to add tool %s: %v", restTool.Name, err)
				}
				// Register tool to global registry
				globalToolRegistry.RegisterTool(config.serverName, restTool.Name, restServer.GetMCPTools()[restTool.Name])
			}
			config.server = restServer
		} else {
			// Logic for pre-registered Go-based servers (non-REST)
			if serverInstance, exist := globalContext.servers[config.serverName]; exist {
				clonedServer := serverInstance.Clone()
				clonedServer.SetConfig([]byte(serverConfigJsonForInstance)) // Pass the server's specific config
				config.server = clonedServer
				// Register tools from this server to global registry
				for toolName, toolInstance := range clonedServer.GetMCPTools() {
					globalToolRegistry.RegisterTool(config.serverName, toolName, toolInstance)
				}
			} else {
				return fmt.Errorf("mcp server type '%s' not registered in globalContext.servers", config.serverName)
			}
		}
	} else {
		return errors.New("either 'server' or 'toolSet' field must be present in the configuration")
	}

	// Parse allowTools - this might need adjustment for composed servers
	allowToolsArray := configJson.Get("allowTools").Array()
	allowTools := make(map[string]struct{}) // For single server, tool name. For composed, serverName/toolName.
	for _, toolJson := range allowToolsArray {
		allowTools[toolJson.String()] = struct{}{}
	}

	config.methodHandlers = make(utils.MethodHandlers)
	// Use config.serverName which is now reliably set
	currentServerNameForHandlers := config.serverName

	config.methodHandlers["ping"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		utils.OnMCPResponseSuccess(true, ctx, map[string]any{}, fmt.Sprintf("mcp:%s:ping", currentServerNameForHandlers))
		return nil
	}
	config.methodHandlers["notifications/initialized"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		proxywasm.SendHttpResponseWithDetail(202, fmt.Sprintf("mcp:%s:notifications/initialized", currentServerNameForHandlers), nil, nil, -1)
		return nil
	}
	config.methodHandlers["notifications/cancelled"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		proxywasm.SendHttpResponseWithDetail(202, fmt.Sprintf("mcp:%s:notifications/cancelled", currentServerNameForHandlers), nil, nil, -1)
		return nil
	}
	config.methodHandlers["initialize"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		version := params.Get("protocolVersion").String()
		if version == "" {
			utils.OnMCPResponseError(true, ctx, errors.New("Unsupported protocol version"), utils.ErrInvalidParams, fmt.Sprintf("mcp:%s:initialize:error", currentServerNameForHandlers))
		}
		utils.OnMCPResponseSuccess(true, ctx, map[string]any{
			"protocolVersion": version,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    currentServerNameForHandlers, // Use the actual server name (single or composed)
				"version": "1.0.0",
			},
		}, fmt.Sprintf("mcp:%s:initialize", currentServerNameForHandlers))
		return nil
	}

	config.methodHandlers["tools/list"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		var listedTools []map[string]any
		// GetMCPTools() will return appropriately formatted tools for both single and composed servers
		allTools := config.server.GetMCPTools() // For composed, keys are "serverName/toolName"

		for toolFullName, tool := range allTools {
			// For composed server, toolFullName is "originalServerName/originalToolName"
			// For single server, toolFullName is "originalToolName"
			// The allowTools map should use the same format as toolFullName
			if len(allowTools) != 0 {
				if _, allow := allowTools[toolFullName]; !allow {
					continue
				}
			}
			listedTools = append(listedTools, map[string]any{
				"name":        toolFullName,
				"description": tool.Description(),
				"inputSchema": tool.InputSchema(),
			})
		}
		utils.OnMCPResponseSuccess(true, ctx, map[string]any{
			"tools":      listedTools,
			"nextCursor": "",
		}, fmt.Sprintf("mcp:%s:tools/list", currentServerNameForHandlers))
		return nil
	}

	config.methodHandlers["tools/call"] = func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
		if config.isComposed {
			// This endpoint is for a composed server (toolSet).
			// Actual tool calls should be routed by mcp-router to individual servers.
			// If a tools/call request reaches here, it's a misconfiguration or unexpected.
			errMsg := fmt.Sprintf("tools/call is not supported on a composed toolSet endpoint ('%s'). It should be routed by mcp-router to the target server.", currentServerNameForHandlers)
			log.Errorf(errMsg)
			utils.OnMCPResponseError(true, ctx, errors.New(errMsg), utils.ErrMethodNotFound, fmt.Sprintf("mcp:%s:tools/call:not_supported_on_toolset", currentServerNameForHandlers))
			return nil
		}

		// Logic for single (non-composed) server
		toolName := params.Get("name").String() // For single server, this is the direct tool name
		args := params.Get("arguments")

		if len(allowTools) != 0 {
			if _, allow := allowTools[toolName]; !allow {
				utils.OnMCPResponseError(true, ctx, fmt.Errorf("Tool not allowed: %s", toolName), utils.ErrInvalidParams, fmt.Sprintf("mcp:%s:tools/call:tool_not_allowed", currentServerNameForHandlers))
				return nil
			}
		}

		proxywasm.SetProperty([]string{"mcp_server_name"}, []byte(currentServerNameForHandlers))
		proxywasm.SetProperty([]string{"mcp_tool_name"}, []byte(toolName))

		toolToCall, ok := config.server.GetMCPTools()[toolName]
		if !ok {
			utils.OnMCPResponseError(true, ctx, fmt.Errorf("Unknown tool: %s", toolName), utils.ErrInvalidParams, fmt.Sprintf("mcp:%s:tools/call:invalid_tool_name", currentServerNameForHandlers))
			return nil
		}

		log.Debugf("Tool call [%s] on server [%s] with arguments[%s]", toolName, currentServerNameForHandlers, args.Raw)
		toolInstance := toolToCall.Create([]byte(args.Raw))
		err := toolInstance.Call(ctx, config.server) // Pass the single server instance
		if err != nil {
			utils.OnMCPToolCallError(true, ctx, err)
			return nil
		}
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
