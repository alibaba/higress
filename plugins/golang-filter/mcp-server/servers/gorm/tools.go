package gorm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleQueryTool handles SQL query execution
func HandleQueryTool(dbClient *DBClient) internal.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		message, ok := arguments["sql"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid message argument")
		}

		results, err := dbClient.ExecuteSQL(message)
		if err != nil {
			return nil, fmt.Errorf("failed to execute SQL query: %w", err)
		}

		jsonData, err := json.Marshal(results)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal SQL results: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(jsonData),
				},
			},
		}, nil
	}
}

// GetQueryToolSchema returns the schema for query tool
func GetQueryToolSchema() json.RawMessage {
	return json.RawMessage(`
	{
		"type": "object",
		"properties": {
		"sql": { 
				"type": "string",
				"description": "The sql query to execute"
			}
		}
	}
	`)
}
