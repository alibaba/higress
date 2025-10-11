package higress_ops

import (
	"errors"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress/higress-ops/tools"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

const Version = "1.0.0"

func init() {
	common.GlobalRegistry.RegisterServer("higress-ops", &HigressOpsConfig{})
}

type HigressOpsConfig struct {
	istiodURL     string
	envoyAdminURL string
	namespace     string
	description   string
}

func (c *HigressOpsConfig) ParseConfig(config map[string]interface{}) error {
	istiodURL, ok := config["istiodURL"].(string)
	if !ok {
		return errors.New("missing istiodURL")
	}
	c.istiodURL = istiodURL

	envoyAdminURL, ok := config["envoyAdminURL"].(string)
	if !ok {
		return errors.New("missing envoyAdminURL")
	}
	c.envoyAdminURL = envoyAdminURL

	if namespace, ok := config["namespace"].(string); ok {
		c.namespace = namespace
	} else {
		c.namespace = "istio-system"
	}

	if desc, ok := config["description"].(string); ok {
		c.description = desc
	} else {
		c.description = "Higress Ops MCP Server, which provides debug interfaces for Istio and Envoy components."
	}

	api.LogInfof("Higress Ops MCP Server configuration parsed successfully. IstiodURL: %s, EnvoyAdminURL: %s, Namespace: %s, Description: %s",
		c.istiodURL, c.envoyAdminURL, c.namespace, c.description)

	return nil
}

func (c *HigressOpsConfig) NewServer(serverName string) (*common.MCPServer, error) {
	mcpServer := common.NewMCPServer(
		serverName,
		Version,
		common.WithInstructions("This is a Higress Ops MCP Server that provides debug interfaces for Istio and Envoy components"),
	)

	// Initialize Ops client
	client := NewOpsClient(c.istiodURL, c.envoyAdminURL, c.namespace)

	// Register all tools with the client as an interface
	tools.RegisterIstiodTools(mcpServer, tools.OpsClient(client))
	tools.RegisterEnvoyTools(mcpServer, tools.OpsClient(client))

	api.LogInfof("Higress Ops MCP Server initialized: %s", serverName)

	return mcpServer, nil
}
