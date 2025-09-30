// Package nginxmigration provides nginx configuration migration to Higress functionality
package nginxmigration

import (
	"errors"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress/nginx-migration/tools"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

const Version = "1.0.0"

func init() {
	common.GlobalRegistry.RegisterServer("nginx-migration", &NginxMigrationConfig{})
}

type NginxMigrationConfig struct {
	higressURL  string
	username    string
	password    string
	description string
	// LLM integration options
	enableLLM  bool
	llmAPIKey  string
	llmBaseURL string
}

func (c *NginxMigrationConfig) ParseConfig(config map[string]interface{}) error {
	higressURL, ok := config["higressURL"].(string)
	if !ok {
		return errors.New("missing higressURL")
	}
	c.higressURL = higressURL

	username, ok := config["username"].(string)
	if !ok {
		return errors.New("missing username")
	}
	c.username = username

	password, ok := config["password"].(string)
	if !ok {
		return errors.New("missing password")
	}
	c.password = password

	if desc, ok := config["description"].(string); ok {
		c.description = desc
	} else {
		c.description = "Nginx Configuration Migration MCP Server, which helps migrate Nginx configurations to Higress format."
	}

	// Parse LLM configuration
	if enableLLM, ok := config["enableLLM"].(bool); ok {
		c.enableLLM = enableLLM
	}

	if llmAPIKey, ok := config["llmAPIKey"].(string); ok {
		c.llmAPIKey = llmAPIKey
	}

	if llmBaseURL, ok := config["llmBaseURL"].(string); ok {
		c.llmBaseURL = llmBaseURL
	}

	api.LogDebugf("NginxMigrationConfig ParseConfig: higressURL=%s, username=%s, enableLLM=%t, description=%s",
		c.higressURL, c.username, c.enableLLM, c.description)

	return nil
}

func (c *NginxMigrationConfig) NewServer(serverName string) (*common.MCPServer, error) {
	mcpServer := common.NewMCPServer(
		serverName,
		Version,
		common.WithInstructions("This is a Nginx Configuration Migration MCP Server that helps convert Nginx configurations to Higress format"),
	)

	// Initialize Higress API client
	client := higress.NewHigressClient(c.higressURL, c.username, c.password)

	// Register basic migration tools
	tools.RegisterMigrationTools(mcpServer, client)

	// Register plugin migration tools
	tools.RegisterPluginMigrationTools(mcpServer, client)

	// Register LLM-enhanced tools if enabled
	if c.enableLLM && c.llmAPIKey != "" {
		llmClient := tools.NewOpenAIClient(c.llmAPIKey, c.llmBaseURL)
		tools.RegisterLLMEnhancedMigrationTools(mcpServer, client, llmClient)
		api.LogInfof("LLM-enhanced migration tools registered")
	}

	api.LogInfof("Nginx Migration MCP Server initialized: %s (LLM: %t)", serverName, c.enableLLM)

	return mcpServer, nil
}
