// Common types for nginx migration MCP server - Standalone Mode
package standalone

import (
	"nginx-migration-mcp/internal/rag"
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

// MCPServer implements the tools.MCPServer interface
// Method implementations are in server.go
