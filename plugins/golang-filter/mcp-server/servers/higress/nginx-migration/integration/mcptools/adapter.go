//go:build higress_integration
// +build higress_integration

package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// SimpleToolHandler is a simplified handler function that takes arguments and returns a string result
type SimpleToolHandler func(args map[string]interface{}) (string, error)

// AdaptSimpleHandler converts a SimpleToolHandler to an MCP ToolHandlerFunc
func AdaptSimpleHandler(handler SimpleToolHandler) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract arguments
		var args map[string]interface{}
		if request.Params.Arguments != nil {
			args = request.Params.Arguments
		} else {
			args = make(map[string]interface{})
		}

		// Call the simple handler
		result, err := handler(args)
		if err != nil {
			return nil, fmt.Errorf("tool execution failed: %w", err)
		}

		// Return MCP result
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	}
}

// RegisterSimpleTool registers a tool with a simplified handler
func RegisterSimpleTool(
	server *common.MCPServer,
	name string,
	description string,
	inputSchema map[string]interface{},
	handler SimpleToolHandler,
) {
	// Create tool with schema
	schemaBytes, _ := json.Marshal(inputSchema)

	tool := mcp.NewToolWithRawSchema(name, description, schemaBytes)
	server.AddTool(tool, AdaptSimpleHandler(handler))
}
