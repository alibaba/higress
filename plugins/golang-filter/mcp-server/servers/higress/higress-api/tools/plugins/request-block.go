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
	// Update request block configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("update"+RequestBlockPluginName, "Update request block plugin configuration", getAddOrUpdateRequestBlockConfigSchema()),
		handleAddOrUpdateRequestBlockConfig(client),
	)
}

func handleAddOrUpdateRequestBlockConfig(client *higress.HigressClient) common.ToolHandlerFunc {
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

		// Parse resource_name for non-global scopes
		var resourceName string
		if scope != ScopeGlobal {
			// Validate and get resource_name
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

		currentConfig["enabled"] = enabled
		currentConfig["scope"] = scope

		// Handle non-global scopes: validate and set target and targets
		if scope != ScopeGlobal {
			// Validate target field
			if target, ok := arguments["target"].(string); !ok || target == "" {
				return nil, fmt.Errorf("'target' is required for scope '%s'", scope)
			} else {
				currentConfig["target"] = target
			}

			// Validate targets field
			if targets, ok := arguments["targets"].(map[string]interface{}); !ok {
				return nil, fmt.Errorf("'targets' is required for scope '%s'", scope)
			} else {
				currentConfig["targets"] = targets
			}
		}

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

func getAddOrUpdateRequestBlockConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"scope": {
				"type": "string",
				"enum": ["GLOBAL", "DOMAIN", "SERVICE", "ROUTE"],
				"description": "The scope at which the plugin is applied"
			},
			"target": {
				"type": "string",
				"description": "The name of the target (required for DOMAIN, SERVICE, ROUTE scopes), it should be same as the resource_name"
			},
			"targets": {
				"type": "object",
				"oneOf": [
					{
						"properties": {
							"DOMAIN": {
								"type": "string",
								"description": "The name of the domain"
							}
						},
						"additionalProperties": false
					},
					{
						"properties": {
							"SERVICE": {
								"type": "string",
								"description": "The name of the service"
							}
						},
						"additionalProperties": false
					},
					{
						"properties": {
							"ROUTE": {
								"type": "string",
								"description": "The name of the route"
							}
						},
						"additionalProperties": false
					}
				],
				"description": "The target resource name (required for DOMAIN, SERVICE, ROUTE scopes)"
			},
			"resource_name": {
				"type": "string",
				"description": "The name of the resource (required for DOMAIN, SERVICE, ROUTE scopes)"
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
