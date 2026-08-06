package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// AiRoute represents an AI route configuration
type AiRoute struct {
	Name               string                  `json:"name"`
	Version            string                  `json:"version,omitempty"`
	Domains            []string                `json:"domains,omitempty"`
	PathPredicate      *AiRoutePredicate       `json:"pathPredicate,omitempty"`
	HeaderPredicates   []AiKeyedRoutePredicate `json:"headerPredicates,omitempty"`
	URLParamPredicates []AiKeyedRoutePredicate `json:"urlParamPredicates,omitempty"`
	Upstreams          []AiUpstream            `json:"upstreams,omitempty"`
	ModelPredicates    []AiModelPredicate      `json:"modelPredicates,omitempty"`
	AuthConfig         *RouteAuthConfig        `json:"authConfig,omitempty"`
	FallbackConfig     *AiRouteFallbackConfig  `json:"fallbackConfig,omitempty"`
}

// AiRoutePredicate represents an AI route predicate
type AiRoutePredicate struct {
	MatchType     string `json:"matchType"`
	MatchValue    string `json:"matchValue"`
	CaseSensitive bool   `json:"caseSensitive,omitempty"`
}

// AiKeyedRoutePredicate represents an AI route predicate with a key
type AiKeyedRoutePredicate struct {
	Key           string `json:"key"`
	MatchType     string `json:"matchType"`
	MatchValue    string `json:"matchValue"`
	CaseSensitive bool   `json:"caseSensitive,omitempty"`
}

// AiUpstream represents an AI upstream configuration
type AiUpstream struct {
	Provider     string            `json:"provider"`
	Weight       int               `json:"weight"`
	ModelMapping map[string]string `json:"modelMapping,omitempty"`
}

// AiModelPredicate represents an AI model predicate
type AiModelPredicate struct {
	MatchType     string `json:"matchType"`
	MatchValue    string `json:"matchValue"`
	CaseSensitive bool   `json:"caseSensitive,omitempty"`
}

// AiRouteFallbackConfig represents AI route fallback configuration
type AiRouteFallbackConfig struct {
	Enabled          bool         `json:"enabled"`
	Upstreams        []AiUpstream `json:"upstreams,omitempty"`
	FallbackStrategy string       `json:"fallbackStrategy,omitempty"`
	ResponseCodes    []string     `json:"responseCodes,omitempty"`
}

// AiRouteResponse represents the API response for AI route operations
type AiRouteResponse = higress.APIResponse[AiRoute]

// RegisterAiRouteTools registers all AI route management tools
func RegisterAiRouteTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// List all AI routes
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("list-ai-routes", "List all available AI routes", listAiRoutesSchema()),
		handleListAiRoutes(client),
	)

	// Get specific AI route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get-ai-route", "Get detailed information about a specific AI route", getAiRouteSchema()),
		handleGetAiRoute(client),
	)

	// Add new AI route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("add-ai-route", "Add a new AI route", getAddAiRouteSchema()),
		handleAddAiRoute(client),
	)

	// Update existing AI route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("update-ai-route", "Update an existing AI route", getUpdateAiRouteSchema()),
		handleUpdateAiRoute(client),
	)

	// Delete existing AI route
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("delete-ai-route", "Delete an existing AI route", getAiRouteSchema()),
		handleDeleteAiRoute(client),
	)
}

func handleListAiRoutes(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		respBody, err := client.Get(ctx, "/v1/ai/routes")
		if err != nil {
			return nil, fmt.Errorf("failed to list AI routes: %w", err)
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

func handleGetAiRoute(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Get(ctx, fmt.Sprintf("/v1/ai/routes/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get AI route '%s': %w", name, err)
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

func handleAddAiRoute(client *higress.HigressClient) common.ToolHandlerFunc {
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
		if _, ok := configurations["upstreams"]; !ok {
			return nil, fmt.Errorf("missing required field 'upstreams' in configurations")
		}

		// Validate AI providers exist in upstreams
		if upstreams, ok := configurations["upstreams"].([]interface{}); ok && len(upstreams) > 0 {
			for _, upstream := range upstreams {
				if upstreamMap, ok := upstream.(map[string]interface{}); ok {
					if providerName, ok := upstreamMap["provider"].(string); ok {
						// Check if AI provider exists
						_, err := client.Get(ctx, fmt.Sprintf("/v1/ai/providers/%s", providerName))
						if err != nil {
							return nil, fmt.Errorf("Please create the AI provider '%s' first and then create the AI route", providerName)
						}
					}
				}
			}
		}

		// Validate AI providers exist in fallback upstreams
		if fallbackConfig, ok := configurations["fallbackConfig"].(map[string]interface{}); ok {
			if fallbackUpstreams, ok := fallbackConfig["upstreams"].([]interface{}); ok && len(fallbackUpstreams) > 0 {
				for _, upstream := range fallbackUpstreams {
					if upstreamMap, ok := upstream.(map[string]interface{}); ok {
						if providerName, ok := upstreamMap["provider"].(string); ok {
							// Check if AI provider exists
							_, err := client.Get(ctx, fmt.Sprintf("/v1/ai/providers/%s", providerName))
							if err != nil {
								return nil, fmt.Errorf("Please create the AI provider '%s' first and then create the AI route", providerName)
							}
						}
					}
				}
			}
		}

		respBody, err := client.Post(ctx, "/v1/ai/routes", configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to add AI route: %w", err)
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

func handleUpdateAiRoute(client *higress.HigressClient) common.ToolHandlerFunc {
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

		// Get current AI route configuration to merge with updates
		currentBody, err := client.Get(ctx, fmt.Sprintf("/v1/ai/routes/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get current AI route configuration: %w", err)
		}

		var response AiRouteResponse
		if err := json.Unmarshal(currentBody, &response); err != nil {
			return nil, fmt.Errorf("failed to parse current AI route response: %w", err)
		}

		currentConfig := response.Data

		// Update configurations using JSON marshal/unmarshal for type conversion
		configBytes, err := json.Marshal(configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal configurations: %w", err)
		}

		var newConfig AiRoute
		if err := json.Unmarshal(configBytes, &newConfig); err != nil {
			return nil, fmt.Errorf("failed to parse AI route configurations: %w", err)
		}

		// Merge configurations (overwrite with new values where provided)
		if newConfig.Domains != nil {
			currentConfig.Domains = newConfig.Domains
		}
		if newConfig.PathPredicate != nil {
			currentConfig.PathPredicate = newConfig.PathPredicate
		}
		if newConfig.HeaderPredicates != nil {
			currentConfig.HeaderPredicates = newConfig.HeaderPredicates
		}
		if newConfig.URLParamPredicates != nil {
			currentConfig.URLParamPredicates = newConfig.URLParamPredicates
		}
		if newConfig.Upstreams != nil {
			currentConfig.Upstreams = newConfig.Upstreams
		}
		if newConfig.ModelPredicates != nil {
			currentConfig.ModelPredicates = newConfig.ModelPredicates
		}
		if newConfig.AuthConfig != nil {
			currentConfig.AuthConfig = newConfig.AuthConfig
		}
		if newConfig.FallbackConfig != nil {
			currentConfig.FallbackConfig = newConfig.FallbackConfig
		}

		respBody, err := client.Put(ctx, fmt.Sprintf("/v1/ai/routes/%s", name), currentConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to update AI route '%s': %w", name, err)
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

func handleDeleteAiRoute(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Delete(ctx, fmt.Sprintf("/v1/ai/routes/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to delete AI route '%s': %w", name, err)
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

func listAiRoutesSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {},
		"required": [],
		"additionalProperties": false
	}`)
}

func getAiRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the AI route"
			}
		},
		"required": ["name"],
		"additionalProperties": false
	}`)
}

func getAddAiRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"configurations": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "AI route name"
					},
					"domains": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Domains that the route applies to. If empty, the route applies to all domains."
					},
					"pathPredicate": {
						"type": "object",
						"properties": {
							"matchType": {"type": "string", "enum": ["PRE"], "description": "Match type"},
							"matchValue": {"type": "string", "description": "The value to match against"},
							"caseSensitive": {"type": "boolean", "description": "Whether to match the value case-sensitively"}
						},
						"required": ["matchType", "matchValue"],
						"description": "Path predicate"
					},
					"headerPredicates": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"key": {"type": "string", "description": "Header key"},
								"matchType": {"type": "string", "enum": ["EQUAL", "PRE", "REGULAR"], "description": "Match type"},
								"matchValue": {"type": "string", "description": "The value to match against"},
								"caseSensitive": {"type": "boolean", "description": "Whether to match the value case-sensitively"}
							},
							"required": ["key", "matchType", "matchValue"]
						},
						"description": "Header predicates"
					},
					"urlParamPredicates": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"key": {"type": "string", "description": "URL parameter key"},
								"matchType": {"type": "string", "enum": ["EQUAL", "PRE", "REGULAR"], "description": "Match type"},
								"matchValue": {"type": "string", "description": "The value to match against"},
								"caseSensitive": {"type": "boolean", "description": "Whether to match the value case-sensitively"}
							},
							"required": ["key", "matchType", "matchValue"]
						},
						"description": "URL parameter predicates"
					},
					"upstreams": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"provider": {"type": "string", "description": "LLM provider name"},
								"weight": {"type": "integer", "description": "Weight of the upstream,The sum of upstream weights must be 100"},
								"modelMapping": {
									"type": "object",
									"additionalProperties": {"type": "string"},
									"description": "Model mapping"
								}
							},
							"required": ["provider", "weight"]
						},
						"description": "Route upstreams"
					},
					"modelPredicates": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"matchType": {"type": "string", "enum": ["EQUAL", "PRE"], "description": "Match type"},
								"matchValue": {"type": "string", "description": "The value to match against"},
								"caseSensitive": {"type": "boolean", "description": "Whether to match the value case-sensitively"}
							},
							"required": ["matchType", "matchValue"]
						},
						"description": "Model predicates"
					},
					"authConfig": {
						"type": "object",
						"properties": {
							"enabled": {"type": "boolean", "description": "Whether auth is enabled"},
							"allowedConsumers": {
								"type": "array",
								"items": {"type": "string"},
								"description": "Allowed consumer names"
							}
						},
						"description": "Route auth configuration"
					},
					"fallbackConfig": {
						"type": "object",
						"properties": {
							"enabled": {"type": "boolean", "description": "Whether fallback is enabled"},
							"upstreams": {
								"type": "array",
								"items": {
									"type": "object",
									"properties": {
										"provider": {"type": "string", "description": "LLM provider name"},
										"weight": {"type": "integer", "description": "Weight of the upstream"},
										"modelMapping": {
											"type": "object",
											"additionalProperties": {"type": "string"},
											"description": "Model mapping"
										}
									},
									"required": ["provider", "weight"]
								},
								"description": "Fallback upstreams. Only one upstream is allowed when fallbackStrategy is SEQ."
							},
							"fallbackStrategy": {"type": "string", "enum": ["RAND", "SEQ"], "description": "Fallback strategy"},
							"responseCodes": {
								"type": "array",
								"items": {"type": "string"},
								"description": "Response codes that need fallback"
							}
						},
						"description": "AI Route fallback configuration"
					}
				},
				"required": ["name", "upstreams"],
				"additionalProperties": false
			}
		},
		"required": ["configurations"],
		"additionalProperties": false
	}`)
}

func getUpdateAiRouteSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the AI route"
			},
			"configurations": {
				"type": "object",
				"properties": {
					"domains": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Domains that the route applies to. If empty, the route applies to all domains."
					},
					"pathPredicate": {
						"type": "object",
						"properties": {
							"matchType": {"type": "string", "enum": ["EQUAL", "PRE", "REGULAR"], "description": "Match type"},
							"matchValue": {"type": "string", "description": "The value to match against"},
							"caseSensitive": {"type": "boolean", "description": "Whether to match the value case-sensitively"}
						},
						"required": ["matchType", "matchValue"],
						"description": "Path predicate"
					},
					"headerPredicates": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"key": {"type": "string", "description": "Header key"},
								"matchType": {"type": "string", "enum": ["EQUAL", "PRE", "REGULAR"], "description": "Match type"},
								"matchValue": {"type": "string", "description": "The value to match against"},
								"caseSensitive": {"type": "boolean", "description": "Whether to match the value case-sensitively"}
							},
							"required": ["key", "matchType", "matchValue"]
						},
						"description": "Header predicates"
					},
					"urlParamPredicates": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"key": {"type": "string", "description": "URL parameter key"},
								"matchType": {"type": "string", "enum": ["EQUAL", "PRE", "REGULAR"], "description": "Match type"},
								"matchValue": {"type": "string", "description": "The value to match against"},
								"caseSensitive": {"type": "boolean", "description": "Whether to match the value case-sensitively"}
							},
							"required": ["key", "matchType", "matchValue"]
						},
						"description": "URL parameter predicates"
					},
					"upstreams": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"provider": {"type": "string", "description": "LLM provider name"},
								"weight": {"type": "integer", "description": "Weight of the upstream"},
								"modelMapping": {
									"type": "object",
									"additionalProperties": {"type": "string"},
									"description": "Model mapping"
								}
							},
							"required": ["provider", "weight"]
						},
						"description": "Route upstreams"
					},
					"modelPredicates": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"matchType": {"type": "string", "enum": ["EQUAL", "PRE", "REGULAR"], "description": "Match type"},
								"matchValue": {"type": "string", "description": "The value to match against"},
								"caseSensitive": {"type": "boolean", "description": "Whether to match the value case-sensitively"}
							},
							"required": ["matchType", "matchValue"]
						},
						"description": "Model predicates"
					},
					"authConfig": {
						"type": "object",
						"properties": {
							"enabled": {"type": "boolean", "description": "Whether auth is enabled"},
							"allowedConsumers": {
								"type": "array",
								"items": {"type": "string"},
								"description": "Allowed consumer names"
							}
						},
						"description": "Route auth configuration"
					},
					"fallbackConfig": {
						"type": "object",
						"properties": {
							"enabled": {"type": "boolean", "description": "Whether fallback is enabled"},
							"upstreams": {
								"type": "array",
								"items": {
									"type": "object",
									"properties": {
										"provider": {"type": "string", "description": "LLM provider name"},
										"weight": {"type": "integer", "description": "Weight of the upstream"},
										"modelMapping": {
											"type": "object",
											"additionalProperties": {"type": "string"},
											"description": "Model mapping"
										}
									},
									"required": ["provider", "weight"]
								},
								"description": "Fallback upstreams. Only one upstream is allowed when fallbackStrategy is SEQ."
							},
							"fallbackStrategy": {"type": "string", "enum": ["RAND", "SEQ"], "description": "Fallback strategy"},
							"responseCodes": {
								"type": "array",
								"items": {"type": "string"},
								"description": "Response codes that need fallback"
							}
						},
						"description": "AI Route fallback configuration"
					}
				},
				"additionalProperties": false
			}
		},
		"required": ["name", "configurations"],
		"additionalProperties": false
	}`)
}
