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
		mcp.NewToolWithRawSchema("list-routes", "List all available routes", json.RawMessage(`{}`)),
		handleListRoutes(client),
	)

	// Get specific route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get-route", "Get detailed information about a specific route", getRouteSchema()),
		handleGetRoute(client),
	)

	// Add new route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("add-route", "Add a new route", getAddRouteSchema()),
		handleAddRoute(client),
	)

	// Update existing route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("update-route", "Update an existing route", getUpdateRouteSchema()),
		handleUpdateRoute(client),
	)

	// Delete existing route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("delete-route", "Delete an existing route", getRouteSchema()),
		handleDeleteRoute(client),
	)
}

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

func handleDeleteRoute(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Delete(fmt.Sprintf("/v1/routes/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to delete route '%s': %w", name, err)
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

func getRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the route"
			}
		},
		"required": ["name"],
		"additionalProperties": false
	}`)
}

func getAddRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"configurations": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "The name of the route"
					},
					"domains": {
						"type": "array",
						"items": {"type": "string"},
						"description": "List of domain names, but only one domain is allowed"
					},
					"path": {
						"type": "object",
						"properties": {
							"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"], "description": "Match type of path"},
							"matchValue": {"type": "string", "description": "Value to match"},
							"caseSensitive": {"type": "boolean", "description": "Whether matching is case sensitive"}
						},
						"required": ["matchType", "matchValue"],
						"description": "List of path match conditions"
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
								"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"], "description": "Match type of header"},
								"matchValue": {"type": "string", "description": "Value to match"},
								"caseSensitive": {"type": "boolean", "description": "Whether matching is case sensitive"},
								"key": {"type": "string", "description": "Header key name"}
							},
							"required": ["matchType", "matchValue", "key"]
						},
						"description": "List of header match conditions"
					},
					"urlParams": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"], "description": "Match type of URL parameter"},
								"matchValue": {"type": "string", "description": "Value to match"},
								"caseSensitive": {"type": "boolean", "description": "Whether matching is case sensitive"},
								"key": {"type": "string", "description": "Parameter key name"}
							},
							"required": ["matchType", "matchValue", "key"]
						},
						"description": "List of URL parameter match conditions"
					},
					"services": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string", "description": "Service name"},
								"port": {"type": "integer", "description": "Service port"},
								"weight": {"type": "integer", "description": "Service weight"}
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
				"required": ["name", "path", "services"],
				"additionalProperties": false
			}
		},
		"required": ["configurations"],
		"additionalProperties": false
	}`)
}

func getUpdateRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the route"
			},
			"configurations": {
				"type": "object",
				"properties": {
					"domains": {
						"type": "array",
						"items": {"type": "string"},
						"description": "List of domain names, but only one domain is allowed"
					},
					"path": {
						"type": "object",
						"properties": {
							"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"], "description": "Match type of path"},
							"matchValue": {"type": "string", "description": "Value to match"},
							"caseSensitive": {"type": "boolean", "description": "Whether matching is case sensitive"}
						},
						"required": ["matchType", "matchValue"],
						"description": "The path configuration"
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
								"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"], "description": "Match type of header"},
								"matchValue": {"type": "string", "description": "Value to match"},
								"caseSensitive": {"type": "boolean", "description": "Whether matching is case sensitive"},
								"key": {"type": "string", "description": "Header key name"}
							},
							"required": ["matchType", "matchValue", "key"]
						},
						"description": "List of header match conditions"
					},
					"urlParams": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"matchType": {"type": "string", "enum": ["PRE", "EQUAL", "REGULAR"], "description": "Match type of URL parameter"},
								"matchValue": {"type": "string", "description": "Value to match"},
								"caseSensitive": {"type": "boolean", "description": "Whether matching is case sensitive"},
								"key": {"type": "string", "description": "Parameter key name"}
							},
							"required": ["matchType", "matchValue", "key"]
						},
						"description": "List of URL parameter match conditions"
					},
					"services": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string", "description": "Service name"},
								"port": {"type": "integer", "description": "Service port"},
								"weight": {"type": "integer", "description": "Service weight"}
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
