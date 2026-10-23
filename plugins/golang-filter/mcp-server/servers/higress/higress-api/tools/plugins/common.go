package plugins

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterCommonPluginTools registers all common plugin management tools
func RegisterCommonPluginTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// List plugin instances for a specific scope
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("list-plugin-instances", "List all plugin instances for a specific scope (e.g., a route, domain, or service)", getListPluginInstancesSchema()),
		handleListPluginInstances(client),
	)

	// Get plugin configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get-plugin", "Get configuration for a specific plugin", getPluginConfigSchema()),
		handleGetPluginConfig(client),
	)

	// Delete plugin configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("delete-plugin", "Delete configuration for a specific plugin", getPluginConfigSchema()),
		handleDeletePluginConfig(client),
	)
}

func handleListPluginInstances(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments

		// Parse required parameters
		scope, ok := arguments["scope"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'scope' argument")
		}

		if !IsValidScope(scope) {
			return nil, fmt.Errorf("invalid scope '%s', must be one of: %v", scope, ValidScopes)
		}

		// Parse resource_name (required for non-global scopes)
		var resourceName string
		if scope != ScopeGlobal {
			resourceName, ok = arguments["resource_name"].(string)
			if !ok || resourceName == "" {
				return nil, fmt.Errorf("'resource_name' is required for scope '%s'", scope)
			}
		}

		// Build API path and make request
		// The API endpoint for listing all plugin instances at a specific scope
		var path string
		switch scope {
		case ScopeGlobal:
			path = "/v1/global/plugin-instances"
		case ScopeDomain:
			path = fmt.Sprintf("/v1/domains/%s/plugin-instances", resourceName)
		case ScopeService:
			path = fmt.Sprintf("/v1/services/%s/plugin-instances", resourceName)
		case ScopeRoute:
			path = fmt.Sprintf("/v1/routes/%s/plugin-instances", resourceName)
		default:
			path = "/v1/global/plugin-instances"
		}

		respBody, err := client.Get(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to list plugin instances at scope '%s': %w", scope, err)
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

func handleGetPluginConfig(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments

		// Parse required parameters
		pluginName, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		scope, ok := arguments["scope"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'scope' argument")
		}

		if !IsValidScope(scope) {
			return nil, fmt.Errorf("invalid scope '%s', must be one of: %v", scope, ValidScopes)
		}

		// Parse resource_name (required for non-global scopes)
		var resourceName string
		if scope != ScopeGlobal {
			resourceName, ok = arguments["resource_name"].(string)
			if !ok || resourceName == "" {
				return nil, fmt.Errorf("'resource_name' is required for scope '%s'", scope)
			}
		}

		// Build API path and make request
		path := BuildPluginPath(pluginName, scope, resourceName)
		respBody, err := client.Get(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to get plugin config for '%s' at scope '%s': %w", pluginName, scope, err)
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

func handleDeletePluginConfig(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments

		// Parse required parameters
		pluginName, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		scope, ok := arguments["scope"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'scope' argument")
		}

		if !IsValidScope(scope) {
			return nil, fmt.Errorf("invalid scope '%s', must be one of: %v", scope, ValidScopes)
		}

		// Parse resource_name (required for non-global scopes)
		var resourceName string
		if scope != ScopeGlobal {
			resourceName, ok = arguments["resource_name"].(string)
			if !ok || resourceName == "" {
				return nil, fmt.Errorf("'resource_name' is required for scope '%s'", scope)
			}
		}

		// Build API path and make request
		path := BuildPluginPath(pluginName, scope, resourceName)
		respBody, err := client.Delete(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to delete plugin config for '%s' at scope '%s': %w", pluginName, scope, err)
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

func getListPluginInstancesSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"scope": {
				"type": "string",
				"enum": ["GLOBAL", "DOMAIN", "SERVICE", "ROUTE"],
				"description": "The scope at which to list plugin instances"
			},
			"resource_name": {
				"type": "string",
				"description": "The name of the resource (required for DOMAIN, SERVICE, ROUTE scopes). For example, the route name, domain name, or service name"
			}
		},
		"required": ["scope"],
		"additionalProperties": false
	}`)
}

func getPluginConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the plugin"
			},
			"scope": {
				"type": "string",
				"enum": ["GLOBAL", "DOMAIN", "SERVICE", "ROUTE"],
				"description": "The scope at which the plugin is applied"
			},
			"resource_name": {
				"type": "string",
				"description": "The name of the resource (required for DOMAIN, SERVICE, ROUTE scopes)"
			}
		},
		"required": ["name", "scope"],
		"additionalProperties": false
	}`)
}
