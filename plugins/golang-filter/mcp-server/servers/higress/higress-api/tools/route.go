package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterRouteTools registers all route management tools
func RegisterRouteTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// List all routes
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("list_routes", "List all available routes", getListRoutesSchema()),
		handleListRoutes(client),
	)

	// Get specific route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get_route", "Get detailed information about a specific route", getRouteSchema()),
		handleGetRoute(client),
	)

	// Add new route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("add_route", "Add a new route (SENSITIVE OPERATION)", getAddRouteSchema()),
		handleAddRoute(client),
	)

	// Update existing route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("update_route", "Update an existing route (SENSITIVE OPERATION)", getUpdateRouteSchema()),
		handleUpdateRoute(client),
	)
}

// handleListRoutes handles the list_routes tool call
func handleListRoutes(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		respBody, err := client.Get("/v1/routes")
		if err != nil {
			return nil, fmt.Errorf("failed to list routes: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(respBody),
				},
			},
		}, nil
	}
}

// handleGetRoute handles the get_route tool call
func handleGetRoute(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Get(fmt.Sprintf("/v1/routes/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get route '%s': %w", name, err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(respBody),
				},
			},
		}, nil
	}
}

// handleAddRoute handles the add_route tool call
func handleAddRoute(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		configurations, ok := arguments["configurations"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'configurations' argument")
		}

		// Validate required fields
		if _, ok := configurations["name"]; !ok {
			return nil, fmt.Errorf("missing required field 'name' in configurations")
		}
		if _, ok := configurations["path"]; !ok {
			return nil, fmt.Errorf("missing required field 'path' in configurations")
		}
		if _, ok := configurations["services"]; !ok {
			return nil, fmt.Errorf("missing required field 'services' in configurations")
		}

		respBody, err := client.Post("/v1/routes", configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to add route: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(respBody),
				},
			},
		}, nil
	}
}

// handleUpdateRoute handles the update_route tool call
func handleUpdateRoute(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		configurations, ok := arguments["configurations"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'configurations' argument")
		}

		// Get current route configuration to merge with updates
		currentBody, err := client.Get(fmt.Sprintf("/v1/routes/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get current route configuration: %w", err)
		}

		var currentRoute map[string]interface{}
		if err := json.Unmarshal(currentBody, &currentRoute); err != nil {
			return nil, fmt.Errorf("failed to parse current route configuration: %w", err)
		}

		// Merge configurations
		for key, value := range configurations {
			currentRoute[key] = value
		}

		respBody, err := client.Put(fmt.Sprintf("/v1/routes/%s", name), currentRoute)
		if err != nil {
			return nil, fmt.Errorf("failed to update route '%s': %w", name, err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(respBody),
				},
			},
		}, nil
	}
}

// getListRoutesSchema returns the JSON schema for list_routes tool
func getListRoutesSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`)
}

// getRouteSchema returns the JSON schema for get_route tool
func getRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the route to retrieve"
			}
		},
		"required": ["name"],
		"additionalProperties": false
	}`)
}

// getAddRouteSchema returns the JSON schema for add_route tool
func getAddRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"configurations": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "The name of the route (required)"
					},
					"domains": {
						"type": "array",
						"items": {"type": "string"},
						"description": "List of domain names (only one domain is allowed)"
					},
					"path": {
						"type": "object",
						"properties": {
							"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"]},
							"matchValue": {"type": "string"},
							"caseSensitive": {"type": "boolean"}
						},
						"required": ["matchType", "matchValue"],
						"description": "Path matching configuration (required)"
					},
					"methods": {
						"type": "array",
						"items": {"type": "string", "enum": ["GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH", "TRACE", "CONNECT"]},
						"description": "List of HTTP methods"
					},
					"headers": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"key": {"type": "string"},
								"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"]},
								"matchValue": {"type": "string"},
								"caseSensitive": {"type": "boolean"}
							},
							"required": ["key", "matchType", "matchValue"]
						},
						"description": "List of header match conditions"
					},
					"urlParams": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"key": {"type": "string"},
								"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"]},
								"matchValue": {"type": "string"},
								"caseSensitive": {"type": "boolean"}
							},
							"required": ["key", "matchType", "matchValue"]
						},
						"description": "List of URL parameter match conditions"
					},
					"services": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"port": {"type": "integer"},
								"weight": {"type": "integer"}
							},
							"required": ["name", "port", "weight"]
						},
						"description": "List of services for this route (required)"
					},
					"customConfigs": {
						"type": "object",
						"additionalProperties": {"type": "string"},
						"description": "Dictionary of custom configurations"
					}
				},
				"required": ["name", "path", "services"],
				"additionalProperties": false
			}
		},
		"required": ["configurations"],
		"additionalProperties": false
	}`)
}

// getUpdateRouteSchema returns the JSON schema for update_route tool
func getUpdateRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the route to update (required)"
			},
			"configurations": {
				"type": "object",
				"properties": {
					"domains": {
						"type": "array",
						"items": {"type": "string"},
						"description": "List of domain names (only one domain is allowed)"
					},
					"path": {
						"type": "object",
						"properties": {
							"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"]},
							"matchValue": {"type": "string"},
							"caseSensitive": {"type": "boolean"}
						},
						"required": ["matchType", "matchValue"],
						"description": "Path matching configuration"
					},
					"methods": {
						"type": "array",
						"items": {"type": "string", "enum": ["GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH", "TRACE", "CONNECT"]},
						"description": "List of HTTP methods"
					},
					"headers": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"key": {"type": "string"},
								"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"]},
								"matchValue": {"type": "string"},
								"caseSensitive": {"type": "boolean"}
							},
							"required": ["key", "matchType", "matchValue"]
						},
						"description": "List of header match conditions"
					},
					"urlParams": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"key": {"type": "string"},
								"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"]},
								"matchValue": {"type": "string"},
								"caseSensitive": {"type": "boolean"}
							},
							"required": ["key", "matchType", "matchValue"]
						},
						"description": "List of URL parameter match conditions"
					},
					"services": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"port": {"type": "integer"},
								"weight": {"type": "integer"}
							},
							"required": ["name", "port", "weight"]
						},
						"description": "List of services for this route"
					},
					"customConfigs": {
						"type": "object",
						"additionalProperties": {"type": "string"},
						"description": "Dictionary of custom configurations"
					}
				},
				"additionalProperties": false
			}
		},
		"required": ["name", "configurations"],
		"additionalProperties": false
	}`)
}
