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

// RequestBlockConfig represents the configuration for request-block plugin
type RequestBlockConfig struct {
	BlockBodies   []string `json:"block_bodies,omitempty"`
	BlockHeaders  []string `json:"block_headers,omitempty"`
	BlockUrls     []string `json:"block_urls,omitempty"`
	BlockedCode   int      `json:"blocked_code,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
}

// RequestBlockInstance represents a request-block plugin instance
type RequestBlockInstance = PluginInstance[RequestBlockConfig]

// RequestBlockResponse represents the API response for request-block plugin
type RequestBlockResponse = higress.APIResponse[RequestBlockInstance]

// RegisterRequestBlockPluginTools registers all request block plugin management tools
func RegisterRequestBlockPluginTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// Update request block configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(fmt.Sprintf("update-%s-plugin", RequestBlockPluginName), "Update request block plugin configuration", getAddOrUpdateRequestBlockConfigSchema()),
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

		configurations, ok := arguments["configurations"]
		if !ok {
			return nil, fmt.Errorf("missing 'configurations' argument")
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

		var response RequestBlockResponse
		if err := json.Unmarshal(currentBody, &response); err != nil {
			return nil, fmt.Errorf("failed to parse current request block response: %w", err)
		}

		currentConfig := response.Data
		currentConfig.Enabled = enabled
		currentConfig.Scope = scope

		// Convert the input configurations to RequestBlockConfig and merge
		configBytes, err := json.Marshal(configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal configurations: %w", err)
		}

		var newConfig RequestBlockConfig
		if err := json.Unmarshal(configBytes, &newConfig); err != nil {
			return nil, fmt.Errorf("failed to parse request block configurations: %w", err)
		}

		// Update configurations (overwrite with new values where provided)
		if newConfig.BlockBodies != nil {
			currentConfig.Configurations.BlockBodies = newConfig.BlockBodies
		}
		if newConfig.BlockHeaders != nil {
			currentConfig.Configurations.BlockHeaders = newConfig.BlockHeaders
		}
		if newConfig.BlockUrls != nil {
			currentConfig.Configurations.BlockUrls = newConfig.BlockUrls
		}
		if newConfig.BlockedCode != 0 {
			currentConfig.Configurations.BlockedCode = newConfig.BlockedCode
		}
		currentConfig.Configurations.CaseSensitive = newConfig.CaseSensitive

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
