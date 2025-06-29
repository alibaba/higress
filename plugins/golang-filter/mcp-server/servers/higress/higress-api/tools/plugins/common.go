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
		mcp.NewToolWithRawSchema("get-plugin-config", "Get configuration for a specific plugin", getPluginConfigSchema()),
		handleGetPluginConfig(client),
	)
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
				"enum": ["global", "domain", "service", "route"],
				"description": "The scope at which the plugin is applied"
			},
			"resource_name": {
				"type": "string",
				"description": "The name of the resource (required for domain, service, route scopes)"
			}
		},
		"required": ["name", "scope"],
		"additionalProperties": false
	}`)
}
