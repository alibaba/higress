package tools

import (
	"context"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// RegisterIstiodTools registers all Istiod debug tools
func RegisterIstiodTools(mcpServer *common.MCPServer, client OpsClient) {
	// Sync status tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-syncz",
			"Get synchronization status information between Istiod and Envoy proxies",
			CreateSimpleSchema(),
		),
		handleIstiodSyncz(client),
	)

	// Endpoints debug tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-endpointz",
			"Get all service endpoint information discovered by Istiod",
			CreateSimpleSchema(),
		),
		handleIstiodEndpointz(client),
	)

	// Config status tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-configz",
			"Get Istiod configuration status and error information",
			CreateSimpleSchema(),
		),
		handleIstiodConfigz(client),
	)

	// Clusters debug tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-clusters",
			"Get all cluster information discovered by Istiod",
			CreateSimpleSchema(),
		),
		handleIstiodClusters(client),
	)

	// Version info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-version",
			"Get Istiod version information",
			CreateSimpleSchema(),
		),
		handleIstiodVersion(client),
	)

	// Registry info tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema(
			"get-istiod-registryz",
			"Get Istiod service registry information",
			CreateSimpleSchema(),
		),
		handleIstiodRegistryz(client),
	)
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
