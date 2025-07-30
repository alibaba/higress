package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// McpServer represents an MCP server configuration
type McpServer struct {
	ID                 string               `json:"id,omitempty"`
	Name               string               `json:"name"`
	Description        string               `json:"description,omitempty"`
	Domains            []string             `json:"domains,omitempty"`
	Services           []McpUpstreamService `json:"services,omitempty"`
	Type               string               `json:"type"`
	ConsumerAuthInfo   *ConsumerAuthInfo    `json:"consumerAuthInfo,omitempty"`
	RawConfigurations  string               `json:"rawConfigurations,omitempty"`
	DSN                string               `json:"dsn,omitempty"`
	DBType             string               `json:"dbType,omitempty"`
	UpstreamPathPrefix string               `json:"upstreamPathPrefix,omitempty"`
}

// McpUpstreamService represents a service in MCP server
type McpUpstreamService struct {
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Version string `json:"version,omitempty"`
	Weight  int    `json:"weight"`
}

// ConsumerAuthInfo represents consumer authentication information
type ConsumerAuthInfo struct {
	Type             string   `json:"type,omitempty"`
	Enable           bool     `json:"enable,omitempty"`
	AllowedConsumers []string `json:"allowedConsumers,omitempty"`
}

// McpServerConsumers represents MCP server consumers configuration
type McpServerConsumers struct {
	McpServerName string   `json:"mcpServerName"`
	Consumers     []string `json:"consumers"`
}

// McpServerConsumerDetail represents detailed consumer information
type McpServerConsumerDetail struct {
	McpServerName string `json:"mcpServerName"`
	ConsumerName  string `json:"consumerName"`
	Type          string `json:"type,omitempty"`
}

// SwaggerContent represents swagger content for conversion
type SwaggerContent struct {
	Content string `json:"content"`
}

// McpServerResponse represents the API response for MCP server operations
type McpServerResponse = higress.APIResponse[McpServer]

// McpServerConsumerDetailResponse represents the API response for MCP server consumer operations
type McpServerConsumerDetailResponse = higress.APIResponse[[]McpServerConsumerDetail]

// RegisterMcpServerTools registers all MCP server management tools
func RegisterMcpServerTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// List MCP servers
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("list-mcp-servers", "List all MCP servers", listMcpServersSchema()),
		handleListMcpServers(client),
	)

	// Get specific MCP server
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get-mcp-server", "Get detailed information about a specific MCP server", getMcpServerSchema()),
		handleGetMcpServer(client),
	)

	// Add or update MCP server
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("add-or-update-mcp-server", "Add or update an MCP server instance", getAddOrUpdateMcpServerSchema()),
		handleAddOrUpdateMcpServer(client),
	)

	// Delete MCP server
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("delete-mcp-server", "Delete an MCP server", getMcpServerSchema()),
		handleDeleteMcpServer(client),
	)

	// List MCP server consumers
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("list-mcp-server-consumers", "List MCP server allowed consumers", listMcpServerConsumersSchema()),
		handleListMcpServerConsumers(client),
	)

	// Add MCP server consumers
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("add-mcp-server-consumers", "Add MCP server allowed consumers", getMcpServerConsumersSchema()),
		handleAddMcpServerConsumers(client),
	)

	// Delete MCP server consumers
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("delete-mcp-server-consumers", "Delete MCP server allowed consumers", getMcpServerConsumersSchema()),
		handleDeleteMcpServerConsumers(client),
	)

	// Convert Swagger to MCP config
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("swagger-to-mcp-config", "Convert Swagger content to MCP configuration", getSwaggerToMcpConfigSchema()),
		handleSwaggerToMcpConfig(client),
	)
}

func handleListMcpServers(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments

		// Build query parameters
		queryParams := ""
		if mcpServerName, ok := arguments["mcpServerName"].(string); ok && mcpServerName != "" {
			queryParams += "?mcpServerName=" + mcpServerName
		}
		if mcpType, ok := arguments["type"].(string); ok && mcpType != "" {
			if queryParams == "" {
				queryParams += "?type=" + mcpType
			} else {
				queryParams += "&type=" + mcpType
			}
		}
		if pageNum, ok := arguments["pageNum"].(string); ok && pageNum != "" {
			if queryParams == "" {
				queryParams += "?pageNum=" + pageNum
			} else {
				queryParams += "&pageNum=" + pageNum
			}
		}
		if pageSize, ok := arguments["pageSize"].(string); ok && pageSize != "" {
			if queryParams == "" {
				queryParams += "?pageSize=" + pageSize
			} else {
				queryParams += "&pageSize=" + pageSize
			}
		}

		respBody, err := client.Get("/v1/mcpServer" + queryParams)
		if err != nil {
			return nil, fmt.Errorf("failed to list MCP servers: %w", err)
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

func handleGetMcpServer(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Get(fmt.Sprintf("/v1/mcpServer/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to get MCP server '%s': %w", name, err)
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

func handleAddOrUpdateMcpServer(client *higress.HigressClient) common.ToolHandlerFunc {
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

		respBody, err := client.Put("/v1/mcpServer", configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to add or update MCP server: %w", err)
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

func handleDeleteMcpServer(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name, ok := arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' argument")
		}

		respBody, err := client.Delete(fmt.Sprintf("/v1/mcpServer/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to delete MCP server '%s': %w", name, err)
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

func handleListMcpServerConsumers(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments

		// Build query parameters
		queryParams := ""
		if mcpServerName, ok := arguments["mcpServerName"].(string); ok && mcpServerName != "" {
			queryParams += "?mcpServerName=" + mcpServerName
		}
		if consumerName, ok := arguments["consumerName"].(string); ok && consumerName != "" {
			if queryParams == "" {
				queryParams += "?consumerName=" + consumerName
			} else {
				queryParams += "&consumerName=" + consumerName
			}
		}
		if pageNum, ok := arguments["pageNum"].(string); ok && pageNum != "" {
			if queryParams == "" {
				queryParams += "?pageNum=" + pageNum
			} else {
				queryParams += "&pageNum=" + pageNum
			}
		}
		if pageSize, ok := arguments["pageSize"].(string); ok && pageSize != "" {
			if queryParams == "" {
				queryParams += "?pageSize=" + pageSize
			} else {
				queryParams += "&pageSize=" + pageSize
			}
		}

		respBody, err := client.Get("/v1/mcpServer/consumers" + queryParams)
		if err != nil {
			return nil, fmt.Errorf("failed to list MCP server consumers: %w", err)
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

func handleAddMcpServerConsumers(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		configurations, ok := arguments["configurations"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'configurations' argument")
		}

		// Validate required fields
		if _, ok := configurations["mcpServerName"]; !ok {
			return nil, fmt.Errorf("missing required field 'mcpServerName' in configurations")
		}
		if _, ok := configurations["consumers"]; !ok {
			return nil, fmt.Errorf("missing required field 'consumers' in configurations")
		}

		respBody, err := client.Put("/v1/mcpServer/consumers", configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to add MCP server consumers: %w", err)
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

func handleDeleteMcpServerConsumers(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		configurations, ok := arguments["configurations"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'configurations' argument")
		}

		// Validate required fields
		if _, ok := configurations["mcpServerName"]; !ok {
			return nil, fmt.Errorf("missing required field 'mcpServerName' in configurations")
		}
		if _, ok := configurations["consumers"]; !ok {
			return nil, fmt.Errorf("missing required field 'consumers' in configurations")
		}

		respBody, err := client.DeleteWithBody("/v1/mcpServer/consumers", configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to delete MCP server consumers: %w", err)
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

func handleSwaggerToMcpConfig(client *higress.HigressClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		configurations, ok := arguments["configurations"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'configurations' argument")
		}

		// Validate required fields
		if _, ok := configurations["content"]; !ok {
			return nil, fmt.Errorf("missing required field 'content' in configurations")
		}

		respBody, err := client.Post("/v1/mcpServer/swaggerToMcpConfig", configurations)
		if err != nil {
			return nil, fmt.Errorf("failed to convert swagger to MCP config: %w", err)
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

// Schema definitions

func listMcpServersSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"mcpServerName": {
				"type": "string",
				"description": "McpServer name associated with route"
			},
			"type": {
				"type": "string",
				"description": "Mcp server type"
			},
			"pageNum": {
				"type": "string",
				"description": "Page number, starting from 1. If omitted, all items will be returned"
			},
			"pageSize": {
				"type": "string",
				"description": "Number of items per page. If omitted, all items will be returned"
			}
		},
		"required": [],
		"additionalProperties": false
	}`)
}

func getMcpServerSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "The name of the MCP server"
			}
		},
		"required": ["name"],
		"additionalProperties": false
	}`)
}

func getAddOrUpdateMcpServerSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"configurations": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "Mcp server name"
					},
					"description": {
						"type": "string",
						"description": "Mcp server description"
					},
					"domains": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Domains that the mcp server applies to"
					},
					"services": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string", "description": "Service name"},
								"port": {"type": "integer", "description": "Service port"},
								"version": {"type": "string", "description": "Service version"},
								"weight": {"type": "integer", "description": "Service weight"}
							},
							"required": ["name", "port", "weight"]
						},
						"description": "Mcp server upstream services"
					},
					"type": {
						"type": "string",
						"enum": ["OPEN_API", "DATABASE", "DIRECT_ROUTE"],
						"description": "Mcp Server Type"
					},
					"consumerAuthInfo": {
						"type": "object",
						"properties": {
							"type": {"type": "string", "description": "Consumer auth type"},
							"enable": {"type": "boolean", "description": "Whether consumer auth is enabled"},
							"allowedConsumers": {
								"type": "array",
								"items": {"type": "string"},
								"description": "Allowed consumer names"
							}
						},
						"description": "Mcp server consumer auth info"
					},
					"rawConfigurations": {
						"type": "string",
						"description": "Raw configurations in YAML format"
					},
					"dsn": {
						"type": "string",
						"description": "Data Source Name. For DB type server, it is required"
					},
					"dbType": {
						"type": "string",
						"enum": ["MYSQL", "POSTGRESQL", "SQLITE", "CLICKHOUSE"],
						"description": "Mcp Server DB Type"
					},
					"upstreamPathPrefix": {
						"type": "string",
						"description": "The upstream MCP server will redirect requests based on the path prefix"
					}
				},
				"required": ["name", "type"],
				"additionalProperties": false
			}
		},
		"required": ["configurations"],
		"additionalProperties": false
	}`)
}

func listMcpServerConsumersSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"mcpServerName": {
				"type": "string",
				"description": "McpServer name associated with route"
			},
			"consumerName": {
				"type": "string",
				"description": "Consumer name for search"
			},
			"pageNum": {
				"type": "string",
				"description": "Page number, starting from 1. If omitted, all items will be returned"
			},
			"pageSize": {
				"type": "string",
				"description": "Number of items per page. If omitted, all items will be returned"
			}
		},
		"required": [],
		"additionalProperties": false
	}`)
}

func getMcpServerConsumersSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"configurations": {
				"type": "object",
				"properties": {
					"mcpServerName": {
						"type": "string",
						"description": "Mcp server route name"
					},
					"consumers": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Consumer names"
					}
				},
				"required": ["mcpServerName", "consumers"],
				"additionalProperties": false
			}
		},
		"required": ["configurations"],
		"additionalProperties": false
	}`)
}

func getSwaggerToMcpConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"configurations": {
				"type": "object",
				"properties": {
					"content": {
						"type": "string",
						"description": "Swagger content"
					}
				},
				"required": ["content"],
				"additionalProperties": false
			}
		},
		"required": ["configurations"],
		"additionalProperties": false
	}`)
}
