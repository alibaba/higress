package tools

import (
	"context"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterIstiodTools registers all Istiod debug tools
func RegisterIstiodTools(mcpServer *common.MCPServer, client OpsClient) {
	// Config dump tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-config-dump",
			"获取 Istiod 的完整配置快照，包括所有 xDS 配置",
			CreateSimpleSchema(),
		),
		handleIstiodConfigDump(client),
	)

	// Metrics tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-metrics",
			"获取 Istiod 的 Prometheus 指标数据",
			CreateSimpleSchema(),
		),
		handleIstiodMetrics(client),
	)

	// Sync status tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-syncz",
			"获取 Istiod 与 Envoy 代理的同步状态信息",
			CreateSimpleSchema(),
		),
		handleIstiodSyncz(client),
	)

	// Endpoints debug tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-endpointz",
			"获取 Istiod 发现的所有服务端点信息",
			CreateSimpleSchema(),
		),
		handleIstiodEndpointz(client),
	)

	// Config status tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-configz",
			"获取 Istiod 的配置状态和错误信息",
			CreateSimpleSchema(),
		),
		handleIstiodConfigz(client),
	)

	// Clusters debug tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-clusters",
			"获取 Istiod 发现的所有集群信息",
			CreateSimpleSchema(),
		),
		handleIstiodClusters(client),
	)

	// Version info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-version",
			"获取 Istiod 的版本信息",
			CreateSimpleSchema(),
		),
		handleIstiodVersion(client),
	)

	// Registry info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-registryz",
			"获取 Istiod 的服务注册表信息",
			CreateSimpleSchema(),
		),
		handleIstiodRegistryz(client),
	)

	// Proxy status tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-proxy-status",
			"获取所有连接到 Istiod 的代理状态",
			CreateParameterSchema(
				map[string]interface{}{
					"proxy": map[string]interface{}{
						"type":        "string",
						"description": "特定代理的名称（可选）",
					},
				},
				[]string{},
			),
		),
		handleIstiodProxyStatus(client),
	)

	// Debug vars tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-debug-vars",
			"获取 Istiod 的调试变量信息",
			CreateSimpleSchema(),
		),
		handleIstiodDebugVars(client),
	)
}

func handleIstiodConfigDump(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetIstiodDebug("/debug/config_dump")
		if err != nil {
			return CreateErrorResult("failed to get Istiod config dump: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleIstiodMetrics(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetIstiodDebug("/stats/prometheus")
		if err != nil {
			return CreateErrorResult("failed to get Istiod metrics: " + err.Error())
		}
		return CreateToolResult(data, "text")
	}
}

func handleIstiodSyncz(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetIstiodDebug("/debug/syncz")
		if err != nil {
			return CreateErrorResult("failed to get Istiod sync status: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleIstiodEndpointz(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetIstiodDebug("/debug/endpointz")
		if err != nil {
			return CreateErrorResult("failed to get Istiod endpoints: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleIstiodConfigz(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetIstiodDebug("/debug/configz")
		if err != nil {
			return CreateErrorResult("failed to get Istiod config status: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleIstiodClusters(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetIstiodDebug("/debug/clusterz")
		if err != nil {
			return CreateErrorResult("failed to get Istiod clusters: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleIstiodVersion(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetIstiodDebug("/version")
		if err != nil {
			return CreateErrorResult("failed to get Istiod version: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleIstiodRegistryz(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetIstiodDebug("/debug/registryz")
		if err != nil {
			return CreateErrorResult("failed to get Istiod registry: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleIstiodProxyStatus(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		proxy := GetStringParam(arguments, "proxy", "")

		path := "/debug/proxy_status"
		params := make(map[string]string)

		if proxy != "" {
			params["proxy"] = proxy
		}

		var data []byte
		var err error

		if len(params) > 0 {
			data, err = client.GetIstiodDebugWithParams(path, params)
		} else {
			data, err = client.GetIstiodDebug(path)
		}

		if err != nil {
			return CreateErrorResult("failed to get Istiod proxy status: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}

func handleIstiodDebugVars(client OpsClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := client.GetIstiodDebug("/debug/vars")
		if err != nil {
			return CreateErrorResult("failed to get Istiod debug vars: " + err.Error())
		}
		return CreateToolResult(data, "json")
	}
}
