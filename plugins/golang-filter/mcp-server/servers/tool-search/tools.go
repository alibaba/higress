package tool_search

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleToolSearch handles the x_higress_tool_search tool
func HandleToolSearch(searchService *SearchService) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		api.LogInfo("HandleToolSearch called")

		arguments := request.Params.Arguments
		api.LogDebugf("Request arguments: %+v", arguments)

		// Get query parameter
		query, ok := arguments["query"].(string)
		if !ok {
			api.LogErrorf("Invalid query argument type: %T", arguments["query"])
			return nil, fmt.Errorf("invalid query argument")
		}

		// Validate query
		if query == "" {
			api.LogError("Empty query provided")
			return nil, fmt.Errorf("query cannot be empty")
		}

		// Get topK parameter (optional, default to 10)
		topK := 10
		if topKVal, ok := arguments["topK"]; ok {
			switch v := topKVal.(type) {
			case float64:
				topK = int(v)
			case int:
				topK = v
			case int64:
				topK = int(v)
			default:
				api.LogWarnf("Invalid topK argument type: %T, using default: %d", topKVal, topK)
			}

			// Validate topK range
			if topK <= 0 || topK > 100 {
				api.LogWarnf("Invalid topK value: %d, using default: 10", topK)
				topK = 10
			}
		}

		api.LogInfof("Parsed parameters - query: '%s', topK: %d", query, topK)

		// Add timeout to context
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Perform search
		result, err := searchService.SearchTools(ctx, query, topK)
		if err != nil {
			api.LogErrorf("Search failed: %v", err)
			return nil, fmt.Errorf("failed to search tools: %w", err)
		}

		api.LogInfof("Search completed successfully, found %d tools", len(result.Tools))

		// Build response
		response := map[string]interface{}{
			"tools": result.Tools,
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			api.LogErrorf("Failed to marshal response: %v", err)
			return nil, fmt.Errorf("failed to marshal search results: %w", err)
		}

		api.LogDebugf("Response marshaled successfully, JSON length: %d", len(jsonData))
		api.LogDebugf("Returning MCP CallToolResult")

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

// GetToolSearchSchema returns the schema for the tool search tool
func GetToolSearchSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Query statement for semantic similarity comparison with tool descriptions"
			},
			"topK": {
				"type": "integer",
				"description": "Specify how many tools need to be selected, default is to select the top 10 tools.",
				"minimum": 1,
				"maximum": 100
			}
		},
		"required": ["query"]
	}`)
}
