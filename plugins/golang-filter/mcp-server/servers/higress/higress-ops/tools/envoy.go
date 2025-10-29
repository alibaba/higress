package tools

import (
	"context"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterEnvoyTools registers all Envoy admin tools
func RegisterEnvoyTools(mcpServer *common.MCPServer, client OpsClient) {
	// Config dump tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-config-dump",
			"Get complete Envoy configuration snapshot, including all listeners, clusters, routes, etc.",
			CreateParameterSchema(
				map[string]interface{}{
					"resource": map[string]interface{}{
						"type":        "string",
						"description": "Specify resource type: listeners, clusters, routes, endpoints, secrets, etc. (optional)",
					},
					"mask": map[string]interface{}{
						"type":        "string",
						"description": "Configuration mask for filtering sensitive information (optional)",
					},
				},
				[]string{},
			),
		),
		handleEnvoyConfigDump(client),
	)

	// Clusters info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-clusters",
			"Get all Envoy cluster information and health status",
			CreateParameterSchema(
				map[string]interface{}{
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Output format: json or text (default text)",
						"enum":        []string{"json", "text"},
					},
				},
				[]string{},
			),
		),
		handleEnvoyClusters(client),
	)

	// Listeners info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-listeners",
			"Get all Envoy listener information",
			CreateParameterSchema(
				map[string]interface{}{
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Output format: json or text (default text)",
						"enum":        []string{"json", "text"},
					},
				},
				[]string{},
			),
		),
		handleEnvoyListeners(client),
	)

	// Routes info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-routes",
			"Get Envoy route configuration information",
			CreateParameterSchema(
				map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Specific route table name (optional)",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Output format: json or text (default text)",
						"enum":        []string{"json", "text"},
					},
				},
				[]string{},
			),
		),
		handleEnvoyRoutes(client),
	)

	// Stats tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-stats",
			"Get Envoy statistics information",
			CreateParameterSchema(
				map[string]interface{}{
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "Statistics filter, supports regular expressions (optional)",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Output format: json, prometheus or text (default text)",
						"enum":        []string{"json", "prometheus", "text"},
					},
				},
				[]string{},
			),
		),
		handleEnvoyStats(client),
	)

	// Server info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-server-info",
			"Get Envoy server basic information",
			CreateSimpleSchema(),
		),
		handleEnvoyServerInfo(client),
	)

	// Ready check tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-ready",
			"Check if Envoy is ready",
			CreateSimpleSchema(),
		),
		handleEnvoyReady(client),
	)

	// Hot restart epoch tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-hot-restart-version",
			"Get Envoy hot restart version information",
			CreateSimpleSchema(),
		),
		handleEnvoyHotRestartVersion(client),
	)

	// Certs info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-certs",
			"Get Envoy certificate information",
			CreateSimpleSchema(),
		),
		handleEnvoyCerts(client),
	)
}

func handleEnvoyConfigDump(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments

		path := "/config_dump"
		params := make(map[string]string)

		if resource := GetStringParam(arguments, "resource", ""); resource != "" {
			params["resource"] = resource
		}
		if mask := GetStringParam(arguments, "mask", ""); mask != "" {
			params["mask"] = mask
		}

		var data []byte
		var err error

		if len(params) > 0 {
			data, err = client.GetEnvoyAdminWithParams(ctx, path, params)
		} else {
			data, err = client.GetEnvoyAdmin(ctx, path)
		}

		if err != nil {
			return CreateErrorResult("failed to get Envoy config dump: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleEnvoyClusters(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		format := GetStringParam(arguments, "format", "text")

		path := "/clusters"
		params := make(map[string]string)

		if format == "json" {
			params["format"] = "json"
		}

		var data []byte
		var err error

		if len(params) > 0 {
			data, err = client.GetEnvoyAdminWithParams(ctx, path, params)
		} else {
			data, err = client.GetEnvoyAdmin(ctx, path)
		}

		if err != nil {
			return CreateErrorResult("failed to get Envoy clusters: " + err.Error())
		}
		return CreateToolResult(data, format)
	}
}

func handleEnvoyListeners(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		format := GetStringParam(arguments, "format", "text")

		path := "/listeners"
		params := make(map[string]string)

		if format == "json" {
			params["format"] = "json"
		}

		var data []byte
		var err error

		if len(params) > 0 {
			data, err = client.GetEnvoyAdminWithParams(ctx, path, params)
		} else {
			data, err = client.GetEnvoyAdmin(ctx, path)
		}

		if err != nil {
			return CreateErrorResult("failed to get Envoy listeners: " + err.Error())
		}
		return CreateToolResult(data, format)
	}
}

func handleEnvoyRoutes(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		name := GetStringParam(arguments, "name", "")
		format := GetStringParam(arguments, "format", "text")

		var path string
		if name != "" {
			path = "/routes/" + name
		} else {
			path = "/routes"
		}

		params := make(map[string]string)
		if format == "json" {
			params["format"] = "json"
		}

		var data []byte
		var err error

		if len(params) > 0 {
			data, err = client.GetEnvoyAdminWithParams(ctx, path, params)
		} else {
			data, err = client.GetEnvoyAdmin(ctx, path)
		}

		if err != nil {
			return CreateErrorResult("failed to get Envoy routes: " + err.Error())
		}
		return CreateToolResult(data, format)
	}
}

func handleEnvoyStats(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		filter := GetStringParam(arguments, "filter", "")
		format := GetStringParam(arguments, "format", "text")

		var path string
		switch format {
		case "json":
			path = "/stats?format=json"
		case "prometheus":
			path = "/stats/prometheus"
		default:
			path = "/stats"
		}

		params := make(map[string]string)
		if filter != "" {
			params["filter"] = filter
		}

		var data []byte
		var err error

		if len(params) > 0 {
			data, err = client.GetEnvoyAdminWithParams(ctx, path, params)
		} else {
			data, err = client.GetEnvoyAdmin(ctx, path)
		}

		if err != nil {
			return CreateErrorResult("failed to get Envoy stats: " + err.Error())
		}
		return CreateToolResult(data, format)
	}
}

func handleEnvoyServerInfo(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin(ctx, "/server_info")
		if err != nil {
			return CreateErrorResult("failed to get Envoy server info: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleEnvoyReady(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin(ctx, "/ready")
		if err != nil {
			return CreateErrorResult("failed to get Envoy ready status: " + err.Error())
		}
		return CreateToolResult(data, "text")
	}
}

func handleEnvoyHotRestartVersion(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin(ctx, "/hot_restart_version")
		if err != nil {
			return CreateErrorResult("failed to get Envoy hot restart version: " + err.Error())
		}
		return CreateToolResult(data, "text")
	}
}

func handleEnvoyCerts(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin(ctx, "/certs")
		if err != nil {
			return CreateErrorResult("failed to get Envoy certs: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}
