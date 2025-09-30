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
			"获取 Envoy 的完整配置快照，包括所有监听器、集群、路由等配置",
			CreateParameterSchema(
				map[string]interface{}{
					"resource": map[string]interface{}{
						"type":        "string",
						"description": "指定资源类型: listeners, clusters, routes, endpoints, secrets 等（可选）",
					},
					"mask": map[string]interface{}{
						"type":        "string",
						"description": "配置掩码，用于过滤敏感信息（可选）",
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
			"获取 Envoy 的所有集群信息和健康状态",
			CreateParameterSchema(
				map[string]interface{}{
					"format": map[string]interface{}{
						"type":        "string",
						"description": "输出格式: json 或 text（默认text）",
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
			"获取 Envoy 的所有监听器信息",
			CreateParameterSchema(
				map[string]interface{}{
					"format": map[string]interface{}{
						"type":        "string",
						"description": "输出格式: json 或 text（默认text）",
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
			"获取 Envoy 的路由配置信息",
			CreateParameterSchema(
				map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "特定路由表名称（可选）",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "输出格式: json 或 text（默认text）",
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
			"获取 Envoy 的统计信息",
			CreateParameterSchema(
				map[string]interface{}{
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "统计项过滤器，支持正则表达式（可选）",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "输出格式: json, prometheus 或 text（默认text）",
						"enum":        []string{"json", "prometheus", "text"},
					},
				},
				[]string{},
			),
		),
		handleEnvoyStats(client),
	)

	// Runtime info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-runtime",
			"获取 Envoy 的运行时配置信息",
			CreateSimpleSchema(),
		),
		handleEnvoyRuntime(client),
	)

	// Memory usage tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-memory",
			"获取 Envoy 的内存使用情况",
			CreateSimpleSchema(),
		),
		handleEnvoyMemory(client),
	)

	// Server info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-server-info",
			"获取 Envoy 服务器的基本信息",
			CreateSimpleSchema(),
		),
		handleEnvoyServerInfo(client),
	)

	// Ready check tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-ready",
			"检查 Envoy 是否准备就绪",
			CreateSimpleSchema(),
		),
		handleEnvoyReady(client),
	)

	// Hot restart epoch tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-hot-restart-version",
			"获取 Envoy 热重启版本信息",
			CreateSimpleSchema(),
		),
		handleEnvoyHotRestartVersion(client),
	)

	// Certs info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-envoy-certs",
			"获取 Envoy 的证书信息",
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
			data, err = client.GetEnvoyAdminWithParams(path, params)
		} else {
			data, err = client.GetEnvoyAdmin(path)
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
			data, err = client.GetEnvoyAdminWithParams(path, params)
		} else {
			data, err = client.GetEnvoyAdmin(path)
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
			data, err = client.GetEnvoyAdminWithParams(path, params)
		} else {
			data, err = client.GetEnvoyAdmin(path)
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
			data, err = client.GetEnvoyAdminWithParams(path, params)
		} else {
			data, err = client.GetEnvoyAdmin(path)
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
			data, err = client.GetEnvoyAdminWithParams(path, params)
		} else {
			data, err = client.GetEnvoyAdmin(path)
		}

		if err != nil {
			return CreateErrorResult("failed to get Envoy stats: " + err.Error())
		}
		return CreateToolResult(data, format)
	}
}

func handleEnvoyRuntime(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin("/runtime")
		if err != nil {
			return CreateErrorResult("failed to get Envoy runtime: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleEnvoyMemory(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin("/memory")
		if err != nil {
			return CreateErrorResult("failed to get Envoy memory: " + err.Error())
		}
		return CreateToolResult(data, "text")
	}
}

func handleEnvoyServerInfo(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin("/server_info")
		if err != nil {
			return CreateErrorResult("failed to get Envoy server info: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleEnvoyReady(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin("/ready")
		if err != nil {
			return CreateErrorResult("failed to get Envoy ready status: " + err.Error())
		}
		return CreateToolResult(data, "text")
	}
}

func handleEnvoyHotRestartVersion(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin("/hot_restart_version")
		if err != nil {
			return CreateErrorResult("failed to get Envoy hot restart version: " + err.Error())
		}
		return CreateToolResult(data, "text")
	}
}

func handleEnvoyCerts(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetEnvoyAdmin("/certs")
		if err != nil {
			return CreateErrorResult("failed to get Envoy certs: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}
