package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// LlmProvider represents an LLM provider configuration
type LlmProvider struct {
	Name                string                 `json:"name"`
	Type                string                 `json:"type"`
	Protocol            string                 `json:"protocol"`
	Tokens              []string               `json:"tokens,omitempty"`
	TokenFailoverConfig *TokenFailoverConfig   `json:"tokenFailoverConfig,omitempty"`
	RawConfigs          map[string]interface{} `json:"rawConfigs,omitempty"`
}

// TokenFailoverConfig represents token failover configuration
type TokenFailoverConfig struct {
	Enabled             bool   `json:"enabled,omitempty"`
	FailureThreshold    int    `json:"failureThreshold,omitempty"`
	SuccessThreshold    int    `json:"successThreshold,omitempty"`
	HealthCheckInterval int    `json:"healthCheckInterval,omitempty"`
	HealthCheckTimeout  int    `json:"healthCheckTimeout,omitempty"`
	HealthCheckModel    string `json:"healthCheckModel,omitempty"`
}

// LlmProviderResponse represents the API response for LLM provider operations
type LlmProviderResponse = higress.APIResponse[LlmProvider]

// RegisterAiProviderTools registers all AI provider management tools
func RegisterAiProviderTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// List all LLM providers
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("list-ai-providers", "List all available LLM providers", listAiProvidersSchema()),
		handleListAiProviders(client),
	)

	// Get specific LLM provider
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get-ai-provider", "Get detailed information about a specific LLM provider", getAiProviderSchema()),
		handleGetAiProvider(client),
	)

	// Add new LLM provider
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("add-ai-provider", "Add a new LLM provider", getAddAiProviderSchema()),
		handleAddAiProvider(client),
	)

	// Update existing LLM provider
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("update-ai-provider", "Update an existing LLM provider", getUpdateAiProviderSchema()),
		handleUpdateAiProvider(client),
	)

	// Delete existing LLM provider
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("delete-ai-provider", "Delete an existing LLM provider", getAiProviderSchema()),
		handleDeleteAiProvider(client),
	)
}

func handleListAiProviders(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		respBody, err := client.Get("/v1/ai/providers")
		if err != nil {
			return nil, fmt.Errorf("failed to list LLM providers: %w", err)
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

func handleGetAiProvider(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Get(fmt.Sprintf("/v1/ai/providers/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get LLM provider '%s': %w", name, err)
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

func handleAddAiProvider(client *higress.HigressClient) common.ToolHandlerFunc {
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
		if _, ok := configurations["protocol"]; !ok {
			return nil, fmt.Errorf("missing required field 'protocol' in configurations")
		}

		respBody, err := client.Post("/v1/ai/providers", configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to add LLM provider: %w", err)
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

func handleUpdateAiProvider(client *higress.HigressClient) common.ToolHandlerFunc {
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

		// Get current LLM provider configuration to merge with updates
		currentBody, err := client.Get(fmt.Sprintf("/v1/ai/providers/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get current LLM provider configuration: %w", err)
		}

		var response LlmProviderResponse
		if err := json.Unmarshal(currentBody, &response); err != nil {
			return nil, fmt.Errorf("failed to parse current LLM provider response: %w", err)
		}

		currentConfig := response.Data

		// Update configurations using JSON marshal/unmarshal for type conversion
		configBytes, err := json.Marshal(configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal configurations: %w", err)
		}

		var newConfig LlmProvider
		if err := json.Unmarshal(configBytes, &newConfig); err != nil {
			return nil, fmt.Errorf("failed to parse LLM provider configurations: %w", err)
		}

		// Merge configurations (overwrite with new values where provided)
		if newConfig.Type != "" {
			currentConfig.Type = newConfig.Type
		}
		if newConfig.Protocol != "" {
			currentConfig.Protocol = newConfig.Protocol
		}
		if newConfig.Tokens != nil {
			currentConfig.Tokens = newConfig.Tokens
		}
		if newConfig.TokenFailoverConfig != nil {
			currentConfig.TokenFailoverConfig = newConfig.TokenFailoverConfig
		}
		if newConfig.RawConfigs != nil {
			currentConfig.RawConfigs = newConfig.RawConfigs
		}

		respBody, err := client.Put(fmt.Sprintf("/v1/ai/providers/%s", name), currentConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to update LLM provider '%s': %w", name, err)
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

func handleDeleteAiProvider(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Delete(fmt.Sprintf("/v1/ai/providers/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to delete LLM provider '%s': %w", name, err)
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

func listAiProvidersSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": [],
		"additionalProperties": false
	}`)
}

func getAiProviderSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the LLM provider"
			}
		},
		"required": ["name"],
		"additionalProperties": false
	}`)
}

func getAddAiProviderSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"configurations": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "Provider name"
					},
					"type": {
						"type": "string",
						"enum": ["qwen", "openai", "moonshot", "azure", "ai360", "github", "groq", "baichuan", "yi", "deepseek", "zhipuai", "ollama", "claude", "baidu", "hunyuan", "stepfun", "minimax", "cloudflare", "spark", "gemini", "deepl", "mistral", "cohere", "doubao", "coze", "together-ai"],
						"description": "LLM Service Provider Type"
					},
					"protocol": {
						"type": "string",
						"enum": ["openai/v1", "original"],
						"description": "LLM Service Provider Protocol"
					},
					"tokens": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Tokens used to request the provider"
					},
					"tokenFailoverConfig": {
						"type": "object",
						"properties": {
							"enabled": {"type": "boolean", "description": "Whether token failover is enabled"},
							"failureThreshold": {"type": "integer", "description": "Failure threshold"},
							"successThreshold": {"type": "integer", "description": "Success threshold"},
							"healthCheckInterval": {"type": "integer", "description": "Health check interval"},
							"healthCheckTimeout": {"type": "integer", "description": "Health check timeout"},
							"healthCheckModel": {"type": "string", "description": "Health check model"}
						},
						"description": "Token Failover Config"
					},
					"rawConfigs": {
						"type": "object",
						"additionalProperties": true,
						"description": "Raw configuration key-value pairs used by ai-proxy plugin"
					}
				},
				"required": ["name", "type", "protocol"],
				"additionalProperties": false
			}
		},
		"required": ["configurations"],
		"additionalProperties": false
	}`)
}

func getUpdateAiProviderSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the LLM provider"
			},
			"configurations": {
				"type": "object",
				"properties": {
					"type": {
						"type": "string",
						"enum": ["qwen", "openai", "moonshot", "azure", "ai360", "github", "groq", "baichuan", "yi", "deepseek", "zhipuai", "ollama", "claude", "baidu", "hunyuan", "stepfun", "minimax", "cloudflare", "spark", "gemini", "deepl", "mistral", "cohere", "doubao", "coze", "together-ai"],
						"description": "LLM Service Provider Type"
					},
					"protocol": {
						"type": "string",
						"enum": ["openai/v1", "original"],
						"description": "LLM Service Provider Protocol"
					},
					"tokens": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Tokens used to request the provider"
					},
					"tokenFailoverConfig": {
						"type": "object",
						"properties": {
							"enabled": {"type": "boolean", "description": "Whether token failover is enabled"},
							"failureThreshold": {"type": "integer", "description": "Failure threshold"},
							"successThreshold": {"type": "integer", "description": "Success threshold"},
							"healthCheckInterval": {"type": "integer", "description": "Health check interval"},
							"healthCheckTimeout": {"type": "integer", "description": "Health check timeout"},
							"healthCheckModel": {"type": "string", "description": "Health check model"}
						},
						"description": "Token Failover Config"
					},
					"rawConfigs": {
						"type": "object",
						"additionalProperties": true,
						"description": "Raw configuration key-value pairs used by ai-proxy plugin"
					}
				},
				"additionalProperties": false
			}
		},
		"required": ["name", "configurations"],
		"additionalProperties": false
	}`)
}
