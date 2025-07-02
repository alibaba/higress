package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterServiceTools registers all service source management tools
func RegisterServiceTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// List all service sources
	mcpServer.AddTool(
		mcp.NewTool("list-service-sources", mcp.WithDescription("List all available service sources")),
		handleListServiceSources(client),
	)

	// Get specific service source
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get-service-source", "Get detailed information about a specific service source", getServiceSourceSchema()),
		handleGetServiceSource(client),
	)

	// Add new service source
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("add-service-source", "Add a new service source", getAddServiceSourceSchema()),
		handleAddServiceSource(client),
	)

	// Update existing service source
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("update-service-source", "Update an existing service source", getUpdateServiceSourceSchema()),
		handleUpdateServiceSource(client),
	)

	// Delete existing service source
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("delete-service-source", "Delete an existing service source", getServiceSourceSchema()),
		handleDeleteServiceSource(client),
	)
}

func handleListServiceSources(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		respBody, err := client.Get("/v1/service-sources")
		if err != nil {
			return nil, fmt.Errorf("failed to list service sources: %w", err)
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

func handleGetServiceSource(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Get(fmt.Sprintf("/v1/service-sources/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get service source '%s': %w", name, err)
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

func handleAddServiceSource(client *higress.HigressClient) common.ToolHandlerFunc {
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
		if _, ok := configurations["type"]; !ok {
			return nil, fmt.Errorf("missing required field 'type' in configurations")
		}
		if _, ok := configurations["domain"]; !ok {
			return nil, fmt.Errorf("missing required field 'domain' in configurations")
		}
		if _, ok := configurations["port"]; !ok {
			return nil, fmt.Errorf("missing required field 'port' in configurations")
		}

		respBody, err := client.Post("/v1/service-sources", configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to add service source: %w", err)
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

func handleUpdateServiceSource(client *higress.HigressClient) common.ToolHandlerFunc {
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

		// Get current service source configuration to merge with updates
		currentBody, err := client.Get(fmt.Sprintf("/v1/service-sources/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get current service source configuration: %w", err)
		}

		var currentSource map[string]interface{}
		if err := json.Unmarshal(currentBody, &currentSource); err != nil {
			return nil, fmt.Errorf("failed to parse current service source configuration: %w", err)
		}

		// Merge configurations
		for key, value := range configurations {
			currentSource[key] = value
		}

		respBody, err := client.Put(fmt.Sprintf("/v1/service-sources/%s", name), currentSource)
		if err != nil {
			return nil, fmt.Errorf("failed to update service source '%s': %w", name, err)
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

func handleDeleteServiceSource(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Delete(fmt.Sprintf("/v1/service-sources/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to delete service source '%s': %w", name, err)
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

func getServiceSourceSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the service source to retrieve"
			}
		},
		"required": ["name"],
		"additionalProperties": false
	}`)
}

// TODO: extend other types of service sources, e.g., nacos, zookeeper, euraka.
func getAddServiceSourceSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"configurations": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "The name of the service source"
					},
					"type": {
						"type": "string",
						"enum": ["static", "dns"],
						"description": "The type of service source: 'static' for static IPs, 'dns' for DNS resolution"
					},
					"domain": {
						"type": "string",
						"description": "The domain name or IP address (required)"
					},
					"port": {
						"type": "integer",
						"minimum": 1,
						"maximum": 65535,
						"description": "The port number (required)"
					},
					"protocol": {
						"type": "string",
						"enum": ["http", "https"],
						"description": "The protocol to use (optional, defaults to http)"
					},
					"sni": {
						"type": "string",
						"description": "Server Name Indication for HTTPS connections (optional)"
					}
				},
				"required": ["name", "type", "domain", "port"],
				"additionalProperties": false
			}
		},
		"required": ["configurations"],
		"additionalProperties": false
	}`)
}

// TODO: extend other types of service sources, e.g., nacos, zookeeper, euraka.
func getUpdateServiceSourceSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the service source to update"
			},
			"configurations": {
				"type": "object",
				"properties": {
					"type": {
						"type": "string",
						"enum": ["static", "dns"],
						"description": "The type of service source: 'static' for static IPs, 'dns' for DNS resolution"
					},
					"domain": {
						"type": "string",
						"description": "The domain name or IP address"
					},
					"port": {
						"type": "integer",
						"minimum": 1,
						"maximum": 65535,
						"description": "The port number"
					},
					"protocol": {
						"type": "string",
						"enum": ["http", "https"],
						"description": "The protocol to use (optional, defaults to http)"
					},
					"sni": {
						"type": "string",
						"description": "Server Name Indication for HTTPS connections"
					}
				},
				"additionalProperties": false
			}
		},
		"required": ["name", "configurations"],
		"additionalProperties": false
	}`)
}
