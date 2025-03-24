package gorm

import (
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	"github.com/mark3labs/mcp-go/mcp"
)

func init() {
	internal.GlobalRegistry.RegisterServer("database", &DBConfig{})
}

type DBConfig struct {
	name   string
	dbType string
	dsn    string
}

func (c *DBConfig) ParseConfig(config map[string]any) error {
	name, ok := config["name"].(string)
	if !ok {
		return errors.New("missing servername")
	}
	c.name = name

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
	return nil
}

func (c *DBConfig) NewServer() (*internal.MCPServer, error) {
	mcpServer := internal.NewMCPServer(
		c.name,
		"1.0.0",
	)

	dbClient, err := NewDBClient(c.dsn, c.dbType)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DBClient: %w", err)
	}

	// Add query tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("query", "Run a read-only SQL query in clickhouse database with repository git data", GetQueryToolSchema()),
		HandleQueryTool(dbClient),
	)

	return mcpServer, nil
}
