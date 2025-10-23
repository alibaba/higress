//go:build higress_integration
// +build higress_integration

package nginx_migration

import (
	"errors"

	"nginx-migration-mcp/integration/mcptools"
	"nginx-migration-mcp/internal/rag"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

const Version = "1.0.0"

func init() {
	common.GlobalRegistry.RegisterServer("nginx-migration", &NginxMigrationConfig{})
}

// NginxMigrationConfig holds configuration for the Nginx Migration MCP Server
type NginxMigrationConfig struct {
	gatewayName      string
	gatewayNamespace string
	defaultNamespace string
	defaultHostname  string
	description      string
	ragConfigPath    string // RAG 配置文件路径
}

// ParseConfig parses the configuration map for the Nginx Migration server
func (c *NginxMigrationConfig) ParseConfig(config map[string]interface{}) error {
	// Optional configurations with defaults
	if gatewayName, ok := config["gatewayName"].(string); ok {
		c.gatewayName = gatewayName
	} else {
		c.gatewayName = "higress-gateway"
	}

	if gatewayNamespace, ok := config["gatewayNamespace"].(string); ok {
		c.gatewayNamespace = gatewayNamespace
	} else {
		c.gatewayNamespace = "higress-system"
	}

	if defaultNamespace, ok := config["defaultNamespace"].(string); ok {
		c.defaultNamespace = defaultNamespace
	} else {
		c.defaultNamespace = "default"
	}

	if defaultHostname, ok := config["defaultHostname"].(string); ok {
		c.defaultHostname = defaultHostname
	} else {
		c.defaultHostname = "example.com"
	}

	if desc, ok := config["description"].(string); ok {
		c.description = desc
	} else {
		c.description = "Nginx Migration MCP Server - Convert Nginx configs and Lua plugins to Higress"
	}

	// RAG 配置路径（可选）
	if ragPath, ok := config["ragConfigPath"].(string); ok {
		c.ragConfigPath = ragPath
	} else {
		c.ragConfigPath = "config/rag.json" // 默认路径
	}

	api.LogDebugf("NginxMigrationConfig ParseConfig: gatewayName=%s, gatewayNamespace=%s, defaultNamespace=%s, ragConfig=%s",
		c.gatewayName, c.gatewayNamespace, c.defaultNamespace, c.ragConfigPath)

	return nil
}

// NewServer creates a new MCP server instance for Nginx Migration
func (c *NginxMigrationConfig) NewServer(serverName string) (*common.MCPServer, error) {
	if serverName == "" {
		return nil, errors.New("server name cannot be empty")
	}

	mcpServer := common.NewMCPServer(
		serverName,
		Version,
		common.WithInstructions("Nginx Migration MCP Server: Analyze and convert Nginx configurations and Lua plugins to Higress"),
	)

	// Create migration context with configuration
	migrationCtx := &mcptools.MigrationContext{
		GatewayName:      c.gatewayName,
		GatewayNamespace: c.gatewayNamespace,
		DefaultNamespace: c.defaultNamespace,
		DefaultHostname:  c.defaultHostname,
	}

	// 初始化 RAG Manager（如果配置了）
	if c.ragConfigPath != "" {
		api.LogInfof("Loading RAG config from: %s", c.ragConfigPath)
		ragConfig, err := rag.LoadRAGConfig(c.ragConfigPath)
		if err != nil {
			api.LogWarnf("Failed to load RAG config: %v, RAG will be disabled", err)
			// 不返回错误，继续使用无 RAG 的模式
			ragConfig = &rag.RAGConfig{Enabled: false}
		}

		// 创建 RAG Manager
		migrationCtx.RAGManager = rag.NewRAGManager(ragConfig)

		if migrationCtx.RAGManager.IsEnabled() {
			api.LogInfof("✅ RAG enabled for Nginx Migration MCP Server")
		} else {
			api.LogInfof("📖 RAG disabled, using rule-based approach")
		}
	}

	// Register all migration tools
	mcptools.RegisterNginxConfigTools(mcpServer, migrationCtx)
	mcptools.RegisterLuaPluginTools(mcpServer, migrationCtx)
	mcptools.RegisterToolChainTools(mcpServer, migrationCtx)

	api.LogInfof("Nginx Migration MCP Server initialized: %s (tools registered)", serverName)

	return mcpServer, nil
}
