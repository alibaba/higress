package gorm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleQueryTool handles SQL query execution
func HandleQueryTool(dbClient *DBClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		message, ok := arguments["sql"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid message argument")
		}

		results, err := dbClient.Query(message)
		if err != nil {
			return nil, fmt.Errorf("failed to execute SQL query: %w", err)
		}

		return buildCallToolResult(results)
	}
}

// HandleExecuteTool handles SQL INSERT, UPDATE, or DELETE execution
func HandleExecuteTool(dbClient *DBClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		message, ok := arguments["sql"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid message argument")
		}

		results, err := dbClient.Execute(message)
		if err != nil {
			return nil, fmt.Errorf("failed to execute SQL query: %w", err)
		}

		return buildCallToolResult(results)
	}
}

// HandleListTablesTool handles list all tables
func HandleListTablesTool(dbClient *DBClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		results, err := dbClient.ListTables()
		if err != nil {
			return nil, fmt.Errorf("failed to execute SQL query: %w", err)
		}

		return buildCallToolResult(results)
	}
}

// HandleDescribeTableTool handles describe table
func HandleDescribeTableTool(dbClient *DBClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		message, ok := arguments["table"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid message argument")
		}

		results, err := dbClient.DescribeTable(message)
		if err != nil {
			return nil, fmt.Errorf("failed to execute SQL query: %w", err)
		}

		return buildCallToolResult(results)
	}
}

// buildCallToolResult builds the call tool result
func buildCallToolResult(results any) (*mcp.CallToolResult, error) {
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

// GetExecuteToolSchema returns the schema for execute tool
func GetExecuteToolSchema() json.RawMessage {
	return json.RawMessage(`
	{
		"type": "object",
		"properties": {
		"sql": { 
				"type": "string",
				"description": "The sql to execute"
			}
		}
	}
	`)
}

// GetDescribeTableToolSchema returns the schema for DescribeTable tool
func GetDescribeTableToolSchema() json.RawMessage {
	return json.RawMessage(`
	{
		"type": "object",
		"properties": {
		"table": { 
				"type": "string",
				"description": "table name"
			}
		}
	}
	`)
}

// GetListTablesToolSchema returns the schema for ListTables tool
func GetListTablesToolSchema() json.RawMessage {
	return json.RawMessage(`
	{
		"type": "object",
		"properties": {
		}
	}
	`)
}
