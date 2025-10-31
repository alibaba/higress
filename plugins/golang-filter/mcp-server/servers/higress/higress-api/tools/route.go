package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// Route represents a route configuration
type Route struct {
	Name          string                 `json:"name"`
	Version       string                 `json:"version,omitempty"`
	Domains       []string               `json:"domains,omitempty"`
	Path          *RoutePath             `json:"path,omitempty"`
	Methods       []string               `json:"methods,omitempty"`
	Headers       []RouteMatch           `json:"headers,omitempty"`
	URLParams     []RouteMatch           `json:"urlParams,omitempty"`
	Services      []RouteService         `json:"services,omitempty"`
	AuthConfig    *RouteAuthConfig       `json:"authConfig,omitempty"`
	CustomConfigs map[string]interface{} `json:"customConfigs,omitempty"`
}

// RoutePath represents path matching configuration
type RoutePath struct {
	MatchType     string `json:"matchType"`
	MatchValue    string `json:"matchValue"`
	CaseSensitive bool   `json:"caseSensitive,omitempty"`
}

// RouteMatch represents header or URL parameter matching configuration
type RouteMatch struct {
	Key        string `json:"key"`
	MatchType  string `json:"matchType"`
	MatchValue string `json:"matchValue"`
}

// RouteService represents a service in the route
type RouteService struct {
	Name   string `json:"name"`
	Port   int    `json:"port"`
	Weight int    `json:"weight"`
}

// RouteAuthConfig represents authentication configuration for a route
type RouteAuthConfig struct {
	Enabled          bool     `json:"enabled"`
	AllowedConsumers []string `json:"allowedConsumers,omitempty"`
}

// RouteResponse represents the API response for route operations
type RouteResponse = higress.APIResponse[Route]

// RegisterRouteTools registers all route management tools
func RegisterRouteTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// List all routes
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("list-routes", "List all available routes", listRouteSchema()),
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
		respBody, err := client.Get(ctx, "/v1/routes")
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

		respBody, err := client.Get(ctx, fmt.Sprintf("/v1/routes/%s", name))
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

		// Validate service sources exist
		if services, ok := configurations["services"].([]interface{}); ok && len(services) > 0 {
			for _, svc := range services {
				if serviceMap, ok := svc.(map[string]interface{}); ok {
					if serviceName, ok := serviceMap["name"].(string); ok {
						// Extract service source name from "serviceName.serviceType" format
						var serviceSourceName string
						for i := len(serviceName) - 1; i >= 0; i-- {
							if serviceName[i] == '.' {
								serviceSourceName = serviceName[:i]
								break
							}
						}

						if serviceSourceName == "" {
							return nil, fmt.Errorf("invalid service name format '%s', expected 'serviceName.serviceType'", serviceName)
						}

						// Check if service source exists
						_, err := client.Get(ctx, fmt.Sprintf("/v1/service-sources/%s", serviceSourceName))
						if err != nil {
							return nil, fmt.Errorf("Please create the service source '%s' first and then create the route", serviceSourceName)
						}
					}
				}
			}
		}

		respBody, err := client.Post(ctx, "/v1/routes", configurations)
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
		currentBody, err := client.Get(ctx, fmt.Sprintf("/v1/routes/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get current route configuration: %w", err)
		}

		var response RouteResponse
		if err := json.Unmarshal(currentBody, &response); err != nil {
			return nil, fmt.Errorf("failed to parse current route response: %w", err)
		}

		currentConfig := response.Data

		// Update configurations using JSON marshal/unmarshal for type conversion
		configBytes, err := json.Marshal(configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal configurations: %w", err)
		}

		var newConfig Route
		if err := json.Unmarshal(configBytes, &newConfig); err != nil {
			return nil, fmt.Errorf("failed to parse route configurations: %w", err)
		}

		// Merge configurations (overwrite with new values where provided)
		if newConfig.Domains != nil {
			currentConfig.Domains = newConfig.Domains
		}
		if newConfig.Path != nil {
			currentConfig.Path = newConfig.Path
		}
		if newConfig.Methods != nil {
			currentConfig.Methods = newConfig.Methods
		}
		if newConfig.Headers != nil {
			currentConfig.Headers = newConfig.Headers
		}
		if newConfig.URLParams != nil {
			currentConfig.URLParams = newConfig.URLParams
		}
		if newConfig.Services != nil {
			currentConfig.Services = newConfig.Services
		}
		if newConfig.AuthConfig != nil {
			currentConfig.AuthConfig = newConfig.AuthConfig
		}
		if newConfig.CustomConfigs != nil {
			currentConfig.CustomConfigs = newConfig.CustomConfigs
		}

		respBody, err := client.Put(ctx, fmt.Sprintf("/v1/routes/%s", name), currentConfig)
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

		respBody, err := client.Delete(ctx, fmt.Sprintf("/v1/routes/%s", name))
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

func listRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": [],
		"additionalProperties": false
	}`)
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
						"description": "List of domain names, but only one domain is allowed,Do not fill in the code to match all"
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
						"description": "List of domain names, but only one domain is allowed",
						"maxItems": 1
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
