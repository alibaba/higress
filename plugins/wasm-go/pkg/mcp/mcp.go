package mcp

import (
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/filter"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/server"
)

type MCPServer server.MCPServer

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
