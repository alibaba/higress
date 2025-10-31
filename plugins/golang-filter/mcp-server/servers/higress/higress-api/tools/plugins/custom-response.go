package plugins

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

const CustomResponsePluginName = "custom-response"

// CustomResponseConfig represents the configuration for custom-response plugin
type CustomResponseConfig struct {
	Body           string   `json:"body,omitempty"`
	Headers        []string `json:"headers,omitempty"`
	StatusCode     int      `json:"status_code,omitempty"`
	EnableOnStatus []int    `json:"enable_on_status,omitempty"`
}

// CustomResponseInstance represents a custom-response plugin instance
type CustomResponseInstance = PluginInstance[CustomResponseConfig]

// CustomResponseResponse represents the API response for custom-response plugin
type CustomResponseResponse = higress.APIResponse[CustomResponseInstance]

// RegisterCustomResponsePluginTools registers all custom response plugin management tools
func RegisterCustomResponsePluginTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// Update custom response configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(fmt.Sprintf("update-%s-plugin", CustomResponsePluginName), "Update custom response plugin configuration", getAddOrUpdateCustomResponseConfigSchema()),
		handleAddOrUpdateCustomResponseConfig(client),
	)
}

func handleAddOrUpdateCustomResponseConfig(client *higress.HigressClient) common.ToolHandlerFunc {
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
		path := BuildPluginPath(CustomResponsePluginName, scope, resourceName)

		// Get current custom response configuration to merge with updates
		currentBody, err := client.Get(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to get current custom response configuration: %w", err)
		}

		var response CustomResponseResponse
		if err := json.Unmarshal(currentBody, &response); err != nil {
			return nil, fmt.Errorf("failed to parse current custom response response: %w", err)
		}

		currentConfig := response.Data
		currentConfig.Enabled = enabled
		currentConfig.Scope = scope

		// Convert the input configurations to CustomResponseConfig and merge
		configBytes, err := json.Marshal(configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal configurations: %w", err)
		}

		var newConfig CustomResponseConfig
		if err := json.Unmarshal(configBytes, &newConfig); err != nil {
			return nil, fmt.Errorf("failed to parse custom response configurations: %w", err)
		}

		// Update configurations (overwrite with new values where provided)
		if newConfig.Body != "" {
			currentConfig.Configurations.Body = newConfig.Body
		}
		if newConfig.Headers != nil {
			currentConfig.Configurations.Headers = newConfig.Headers
		}
		if newConfig.StatusCode != 0 {
			currentConfig.Configurations.StatusCode = newConfig.StatusCode
		}
		if newConfig.EnableOnStatus != nil {
			currentConfig.Configurations.EnableOnStatus = newConfig.EnableOnStatus
		}

		respBody, err := client.Put(ctx, path, currentConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to update custom response config at scope '%s': %w", scope, err)
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

func getAddOrUpdateCustomResponseConfigSchema() json.RawMessage {
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
					"body": {
						"type": "string",
						"description": "Custom response body content"
					},
					"headers": {
						"type": "array",
						"items": {"type": "string"},
						"description": "List of custom response headers in the format 'Header-Name=value'"
					},
					"status_code": {
						"type": "integer",
						"minimum": 100,
						"maximum": 599,
						"description": "HTTP status code to return in the custom response"
					},
					"enable_on_status": {
						"type": "array",
						"items": {"type": "integer"},
						"description": "List of upstream status codes that trigger this custom response"
					}
				},
				"additionalProperties": false
			}
		},
		"required": ["scope", "enabled", "configurations"],
		"additionalProperties": false
	}`)
}
