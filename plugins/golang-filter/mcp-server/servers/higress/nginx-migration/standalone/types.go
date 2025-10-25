// Common types for nginx migration MCP server - Standalone Mode
package standalone

import (
	"nginx-migration-mcp/internal/rag"
	"nginx-migration-mcp/tools"
)

// MCPMessage represents a Model Context Protocol message structure
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type MCPServer struct {
	config     *ServerConfig
	ragManager *rag.RAGManager
}

// Implement the tools.MCPServer interface
func (s *MCPServer) ParseNginxConfig(args map[string]interface{}) tools.ToolResult {
	return s.parseNginxConfig(args)
}

func (s *MCPServer) ConvertToHigress(args map[string]interface{}) tools.ToolResult {
	return s.convertToHigress(args)
}

func (s *MCPServer) AnalyzeLuaPlugin(args map[string]interface{}) tools.ToolResult {
	return s.analyzeLuaPlugin(args)
}

func (s *MCPServer) ConvertLuaToWasm(args map[string]interface{}) tools.ToolResult {
	return s.convertLuaToWasm(args)
}
