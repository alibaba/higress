package gorm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/envoyproxy/envoy/examples/golang-http/simple/internal"
	"github.com/mark3labs/mcp-go/mcp"
)

const favoriteFilesTemplate = `
WITH current_files AS (
    SELECT path
    FROM (
        SELECT
            old_path AS path,
            max(time) AS last_time,
            2 AS change_type
        FROM git.file_changes
        GROUP BY old_path
        UNION ALL
        SELECT
            path,
            max(time) AS last_time,
            argMax(change_type, time) AS change_type
        FROM git.file_changes
        GROUP BY path
    )
    GROUP BY path
    HAVING (argMax(change_type, last_time) != 2) AND (NOT match(path, '(^dbms/)|(^libs/)|(^tests/testflows/)|(^programs/server/store/)'))
    ORDER BY path ASC
)
SELECT
    path,
    count() AS c
FROM git.file_changes
WHERE (author = '%s') AND (path IN (current_files))
GROUP BY path
ORDER BY c DESC
LIMIT 10`

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

// HandleFavoriteTool handles author's favorite files query
func HandleFavoriteTool(dbClient *DBClient) internal.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		author, ok := arguments["author"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid author argument")
		}
		query := fmt.Sprintf(favoriteFilesTemplate, author)

		results, err := dbClient.ExecuteSQL(query)
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

// GetFavoriteToolSchema returns the schema for favorite files tool
func GetFavoriteToolSchema() json.RawMessage {
	return json.RawMessage(`
	{
		"type": "object",
		"properties": {
		"author": { 
				"type": "string",
				"description": "the author name"
			}
		}
	}
	`)
}
