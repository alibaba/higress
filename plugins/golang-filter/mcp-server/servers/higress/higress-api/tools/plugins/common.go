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
	// Get plugin configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get_plugin_config", "Get configuration for a specific plugin", getPluginConfigSchema()),
		handleGetPluginConfig(client),
	)

	// Update plugin configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("update_plugin_config", "Update configuration for a specific plugin (SENSITIVE OPERATION)", getUpdatePluginConfigSchema()),
		handleUpdatePluginConfig(client),
	)
}

// handleGetPluginConfig handles the get_plugin_config tool call
func handleGetPluginConfig(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments

		// Parse required parameters
		pluginName, ok := arguments["plugin_name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'plugin_name' argument")
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
		respBody, err := client.Get(path)
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

// handleUpdatePluginConfig handles the update_plugin_config tool call
func handleUpdatePluginConfig(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments

		// Parse required parameters
		pluginName, ok := arguments["plugin_name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'plugin_name' argument")
		}

		scope, ok := arguments["scope"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'scope' argument")
		}

		if !IsValidScope(scope) {
			return nil, fmt.Errorf("invalid scope '%s', must be one of: %v", scope, ValidScopes)
		}

		configurations, ok := arguments["configurations"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'configurations' argument")
		}

		// Parse resource_name (required for non-global scopes)
		var resourceName string
		if scope != ScopeGlobal {
			resourceName, ok = arguments["resource_name"].(string)
			if !ok || resourceName == "" {
				return nil, fmt.Errorf("'resource_name' is required for scope '%s'", scope)
			}
		}

		// Build API path
		path := BuildPluginPath(pluginName, scope, resourceName)

		// Get current plugin configuration to merge with updates
		currentBody, err := client.Get(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get current plugin configuration: %w", err)
		}

		var currentConfig map[string]interface{}
		if err := json.Unmarshal(currentBody, &currentConfig); err != nil {
			return nil, fmt.Errorf("failed to parse current plugin configuration: %w", err)
		}

		// Merge configurations
		for key, value := range configurations {
			currentConfig[key] = value
		}

		respBody, err := client.Put(path, currentConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to update plugin config for '%s' at scope '%s': %w", pluginName, scope, err)
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

// getPluginConfigSchema returns the JSON schema for get_plugin_config tool
func getPluginConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"plugin_name": {
				"type": "string",
				"description": "The name of the plugin to retrieve configuration for"
			},
			"scope": {
				"type": "string",
				"enum": ["global", "domain", "service", "route"],
				"description": "The scope at which the plugin is applied"
			},
			"resource_name": {
				"type": "string",
				"description": "The name of the resource (required for domain, service, route scopes)"
			}
		},
		"required": ["plugin_name", "scope"],
		"additionalProperties": false
	}`)
}

// getUpdatePluginConfigSchema returns the JSON schema for update_plugin_config tool
func getUpdatePluginConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"plugin_name": {
				"type": "string",
				"description": "The name of the plugin to update configuration for"
			},
			"scope": {
				"type": "string",
				"enum": ["global", "domain", "service", "route"],
				"description": "The scope at which the plugin is applied"
			},
			"resource_name": {
				"type": "string",
				"description": "The name of the resource (required for domain, service, route scopes)"
			},
			"configurations": {
				"type": "object",
				"description": "The plugin configuration object to update",
				"additionalProperties": true
			}
		},
		"required": ["plugin_name", "scope", "configurations"],
		"additionalProperties": false
	}`)
}
