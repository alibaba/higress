# MCP Server
English | [简体中文](./README.md)

## Overview

MCP Server is a Golang Filter plugin based on Envoy, designed to implement Server-Sent Events (SSE) and message communication functionality. This plugin supports various database types and uses Redis as a message queue to enable load-balanced requests to be sent through corresponding SSE connections.

> **Note**: MCP Server requires Higress 2.1.0 or higher version.

## Project Structure
```
mcp-server/
├── config.go                # Configuration parsing code
├── filter.go                # Request processing code
├── internal/                # Internal implementation logic
├── servers/                 # MCP server implementation
├── go.mod                   # Go module dependency definition
└── go.sum                   # Go module dependency checksum
```

## MCP Server Development Guide

```go
// Register your server in the init function
// Param 1: Server name
// Param 2: Config struct instance
func init() {
	internal.GlobalRegistry.RegisterServer("demo", &DemoConfig{})
}

// Server configuration struct
type DemoConfig struct {
	helloworld string
}

// Configuration parsing method
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
func (c *DBConfig) NewServer(serverName string) (*internal.MCPServer, error) {
	mcpServer := internal.NewMCPServer(serverName, Version)
    
	// Add tool methods to server
	// mcpServer.AddTool()	
	
	// Add resources to server
	// mcpServer.AddResource()
	
	return mcpServer, nil
}
```

**Note**: 
Need to use underscore import in config.go to execute the package's init function
```go
import (
	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/gorm"
)
```