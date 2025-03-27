package registry

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

type McpApplicationDescription struct {
	Protocol         string              `json:"protocol"`
	ToolsDescription []*ToolDescription  `json:"tools"`
	ToolsMeta        map[string]ToolMeta `json:"toolsMeta"`
}

type ToolMeta struct {
	InvokeContext     map[string]string           `json:"invokeContext"`
	ParametersMapping map[string]ParameterMapInfo `json:"parametersMapping"`
	CredentialRef     *string                     `json:"credentialRef"`
}

type ParameterMapInfo struct {
	ParamName   string `json:"name"`
	BackendName string `json:"backendName"`
	ParamType   string `json:"type"`
	Position    string `json:"position"`
}

type ToolDescription struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type ToolChangeEventListener interface {
	OnToolChanged(McpServerRegistry)
}

type McpServerRegistry interface {
	ListToolsDesciption() []*ToolDescription
	GetToolRpcContext(toolname string) (*RpcContext, bool)
	RegisterToolChangeEventListener(listener ToolChangeEventListener)
}

type RpcContext struct {
	Instances  *[]Instance
	ToolMeta   ToolMeta
	Protocol   string
	Credential *CredentialInfo
}

type CredentialInfo struct {
	CredentialType string         `json:"type"`
	Credentials    map[string]any `json:"credentialsMap"`
}

type Instance struct {
	Host string
	Port uint64
	Meta map[string]string
}

type RemoteCallHandle interface {
	HandleToolCall(ctx *RpcContext, parameters map[string]any) (*mcp.CallToolResult, error)
}
