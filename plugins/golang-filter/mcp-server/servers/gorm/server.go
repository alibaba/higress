package gorm

import (
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/mark3labs/mcp-go/mcp"
)

const Version = "1.0.0"

func init() {
	internal.GlobalRegistry.RegisterServer("database", &DBConfig{})
}

type DBConfig struct {
	dbType string
	dsn    string
}

func (c *DBConfig) ParseConfig(config map[string]any) error {
	dsn, ok := config["dsn"].(string)
	if !ok {
		return errors.New("missing dsn")
	}
	c.dsn = dsn

	dbType, ok := config["dbType"].(string)
	if !ok {
		return errors.New("missing database type")
	}
	c.dbType = dbType
	api.LogDebugf("DBConfig ParseConfig: %+v", config)
	return nil
}

func (c *DBConfig) NewServer(serverName string) (*internal.MCPServer, error) {
	mcpServer := internal.NewMCPServer(
		serverName,
		Version,
		internal.WithInstructions(fmt.Sprintf("This is a %s database server", c.dbType)),
	)

	dbClient, err := NewDBClient(c.dsn, c.dbType)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DBClient: %w", err)
	}

	// Add query tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("query", fmt.Sprintf("Run a read-only SQL query in database %s", c.dbType), GetQueryToolSchema()),
		HandleQueryTool(dbClient),
	)

	return mcpServer, nil
}
