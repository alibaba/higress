# MCP Server
English | [简体中文](./README.md)

## Overview

MCP Server is a Golang Filter plugin based on Envoy that provides a unified MCP (Model Context Protocol) service interface. It supports integration with various backend services, including:

- Database Services: Supports multiple database access and management through GORM
- Configuration Service: Supports integration with Nacos configuration service
- Extensibility: Supports custom server implementations for easy integration with other services

> **Note**: MCP Server requires Higress version 2.1.0 or higher to be used.

## MCP Server Development Guide

```go
// Register your server in the init function
// Parameter 1: Server name
// Parameter 2: Configuration struct instance
func init() {
	common.GlobalRegistry.RegisterServer("demo", &DemoConfig{})
}

// Server configuration struct
type DemoConfig struct {
	helloworld string
}

// Parse configuration method
// Parse and validate configuration items from the config map
func (c *DBConfig) ParseConfig(config map[string]any) error {
	helloworld, ok := config["helloworld"].(string)
	if !ok { return errors.New("missing helloworld")}
	c.helloworld = helloworld
	return nil
}

// Create a new MCP server instance
// serverName: Server name
// Returns: MCP server instance and possible error
func (c *DBConfig) NewServer(serverName string) (*common.MCPServer, error) {
	mcpServer := common.NewMCPServer(serverName, Version)
    
	// Add tool methods to the server
	// mcpServer.AddTool()	
	
	// Add resources to the server
	// mcpServer.AddResource()
	
	return mcpServer, nil
}
```

**Note**: 
You need to use underscore imports in config.go to execute the package's init function
```go
import (
	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/gorm"
)
```