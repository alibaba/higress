package plugins

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

const RequestBlockPluginName = "request-block"

// RegisterRequestBlockPluginTools registers all request block plugin management tools
func RegisterRequestBlockPluginTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// Get request block configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get_request_block_config", "Get request block plugin configuration", getRequestBlockConfigSchema()),
		handleGetRequestBlockConfig(client),
	)

	// Update request block configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("update_request_block_config", "Update request block plugin configuration (SENSITIVE OPERATION)", getUpdateRequestBlockConfigSchema()),
		handleUpdateRequestBlockConfig(client),
	)
}

// handleGetRequestBlockConfig handles the get_request_block_config tool call
func handleGetRequestBlockConfig(client *higress.HigressClient) common.ToolHandlerFunc {
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
		path := BuildPluginPath(RequestBlockPluginName, scope, resourceName)
		respBody, err := client.Get(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get request block config at scope '%s': %w", scope, err)
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

// handleUpdateRequestBlockConfig handles the update_request_block_config tool call
func handleUpdateRequestBlockConfig(client *higress.HigressClient) common.ToolHandlerFunc {
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

		enabled, ok := arguments["enabled"].(bool)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'enabled' argument")
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
		path := BuildPluginPath(RequestBlockPluginName, scope, resourceName)

		// Get current request block configuration to merge with updates
		currentBody, err := client.Get(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get current request block configuration: %w", err)
		}

		var currentConfig map[string]interface{}
		if err := json.Unmarshal(currentBody, &currentConfig); err != nil {
			return nil, fmt.Errorf("failed to parse current request block configuration: %w", err)
		}

		// Update enabled status
		currentConfig["enabled"] = enabled

		// Merge configurations
		for key, value := range configurations {
			currentConfig[key] = value
		}

		respBody, err := client.Put(path, currentConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to update request block config at scope '%s': %w", scope, err)
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

// getRequestBlockConfigSchema returns the JSON schema for get_request_block_config tool
func getRequestBlockConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
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
		"required": ["scope"],
		"additionalProperties": false
	}`)
}

// getUpdateRequestBlockConfigSchema returns the JSON schema for update_request_block_config tool
func getUpdateRequestBlockConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"scope": {
				"type": "string",
				"enum": ["global", "domain", "service", "route"],
				"description": "The scope at which the plugin is applied"
			},
			"resource_name": {
				"type": "string",
				"description": "The name of the resource (required for domain, service, route scopes)"
			},
			"enabled": {
				"type": "boolean",
				"description": "Whether the plugin is enabled"
			},
			"configurations": {
				"type": "object",
				"properties": {
					"block_bodies": {
						"type": "array",
						"items": {"type": "string"},
						"description": "List of patterns to match against request body content"
					},
					"block_headers": {
						"type": "array",
						"items": {"type": "string"},
						"description": "List of patterns to match against request headers"
					},
					"block_urls": {
						"type": "array",
						"items": {"type": "string"},
						"description": "List of patterns to match against request URLs"
					},
					"blocked_code": {
						"type": "integer",
						"minimum": 100,
						"maximum": 599,
						"description": "HTTP status code to return when a block is matched"
					},
					"case_sensitive": {
						"type": "boolean",
						"description": "Whether the block matching is case sensitive"
					}
				},
				"additionalProperties": false
			}
		},
		"required": ["scope", "enabled", "configurations"],
		"additionalProperties": false
	}`)
}
