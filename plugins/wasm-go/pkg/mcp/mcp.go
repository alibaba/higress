package mcp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/filter"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/server"
)

var _ server.Server = &MCPServer{}

type MCPServer struct {
	tools  map[string]server.Tool
	config []byte
}

func (s MCPServer) Clone() server.Server {
	return &MCPServer{tools: s.tools}
}

func (s *MCPServer) AddMCPTool(name string, tool server.Tool) server.Server {
	if s.tools == nil {
		s.tools = make(map[string]server.Tool)
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

func (s *MCPServer) GetMCPTools() map[string]server.Tool {
	return s.tools
}

func (s *MCPServer) SetConfig(config []byte) {
	s.config = config

}

// mcp server function
var (
	LoadMCPServer = server.Load

	InitMCPServer = server.Initialize

	AddMCPServer = server.AddMCPServer
)

// mcp filter function
var (
	LoadMCPFilter = filter.Load

	InitMCPFIlter = filter.Initialize

	SetConfigParser = filter.SetConfigParser

	FilterName = filter.FilterName

	SetRequestFilter = filter.SetRequestFilter

	SetResponseFilter = filter.SetResponseFilter

	OnJsonRpcError = filter.OnJsonRpcError
)
