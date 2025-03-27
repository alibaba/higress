# MCP Server Implementation Guide

## Background

  Higress, as an Envoy-based API gateway, supports hosting MCP Servers through its plugin mechanism. MCP (Model Context Protocol) is essentially an AI-friendly API that enables AI Agents to more easily call various tools and services. Higress provides unified capabilities for authentication, authorization, rate limiting, and observability for tool calls, simplifying the development and deployment of AI applications.

  ![](https://img.alicdn.com/imgextra/i1/O1CN01wv8H4g1mS4MUzC1QC_!!6000000004952-2-tps-1764-597.png)

  By hosting MCP Servers with Higress, you can achieve:
  - Unified authentication and authorization mechanisms, ensuring the security of AI tool calls
  - Fine-grained rate limiting to prevent abuse and resource exhaustion
  - Comprehensive audit logs recording all tool call behaviors
  - Rich observability for monitoring the performance and health of tool calls
  - Simplified deployment and management through Higress's plugin mechanism for quickly adding new MCP Servers

This guide explains how to implement a Model Context Protocol (MCP) server using the Higress WASM Go SDK. MCP servers provide tools and resources that extend the capabilities of AI assistants.

## Overview

An MCP server is a standalone application that communicates with AI assistants through the Model Context Protocol. It can provide:

- **Tools**: Functions that can be called by the AI to perform specific tasks
- **Resources**: Data that can be accessed by the AI

> **Note**: MCP server plugins require Higress version 2.1.0 or higher to be used.

## Project Structure

A typical MCP server project has the following structure:

```
my-mcp-server/
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
├── main.go                # Entry point that registers tools and resources
├── server/
│   └── server.go          # Server configuration and parsing
└── tools/
    └── my_tool.go         # Tool implementation
```

## Server Configuration

The server configuration defines the parameters needed for the server to function. For example:

```go
// server/server.go
package server

import (
    "encoding/json"
    "errors"

    "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

// Define your server configuration structure
type MyMCPServer struct {
    ApiKey string `json:"apiKey"`
    // Add other configuration fields as needed
}

// Validate the configuration
func (s MyMCPServer) ConfigHasError() error {
    if s.ApiKey == "" {
        return errors.New("missing api key")
    }
    return nil
}

// Parse configuration from JSON
func ParseFromConfig(configBytes []byte, server *MyMCPServer) error {
    return json.Unmarshal(configBytes, server)
}

// Parse configuration from HTTP request
func ParseFromRequest(ctx wrapper.HttpContext, server *MyMCPServer) error {
    return ctx.ParseMCPServerConfig(server)
}
```

## Tool Implementation

Each tool should be implemented as a struct with the following methods:

1. `Description()`: Returns a description of the tool
2. `InputSchema()`: Returns the JSON schema for the tool's input parameters
3. `Create()`: Creates a new instance of the tool with the provided parameters
4. `Call()`: Executes the tool's functionality

Example:

```go
// tools/my_tool.go
package tools

import (
    "encoding/json"
    "fmt"
    "net/http"
    
    "my-mcp-server/server"
    
    "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

// Define your tool structure with input parameters
type MyTool struct {
    Param1 string `json:"param1" jsonschema_description:"Description of param1" jsonschema:"example=example value"`
    Param2 int    `json:"param2,omitempty" jsonschema_description:"Description of param2" jsonschema:"default=5"`
}

// Description returns the description field for the MCP tool definition.
// This corresponds to the "description" field in the MCP tool JSON response,
// which provides a human-readable explanation of the tool's purpose and usage.
func (t MyTool) Description() string {
    return `Detailed description of what this tool does and when to use it.`
}

// InputSchema returns the inputSchema field for the MCP tool definition.
// This corresponds to the "inputSchema" field in the MCP tool JSON response,
// which defines the JSON Schema for the tool's input parameters, including
// property types, descriptions, and required fields.
func (t MyTool) InputSchema() map[string]any {
    return wrapper.ToInputSchema(&MyTool{})
}

// Create instantiates a new tool instance based on the input parameters
// from an MCP tool call. It deserializes the JSON parameters into a struct,
// applying default values for optional fields, and returns the configured tool instance.
func (t MyTool) Create(params []byte) wrapper.MCPTool[server.MyMCPServer] {
    myTool := &MyTool{
        Param2: 5, // Default value
    }
    json.Unmarshal(params, &myTool)
    return myTool
}

// Call implements the core logic for handling an MCP tool call. This method is executed
// when the tool is invoked through the MCP framework. It processes the configured parameters,
// makes any necessary API requests, and formats the results to be returned to the caller.
func (t MyTool) Call(ctx wrapper.HttpContext, config server.MyMCPServer) error {
    // Validate configuration
    err := server.ParseFromRequest(ctx, &config)
    if err != nil {
        return err
    }
    err = config.ConfigHasError()
    if err != nil {
        return err
    }
    
    // Implement your tool's logic here
    // ...
    
    // Return results
    ctx.SendMCPToolTextResult(fmt.Sprintf("Result: %s, %d", t.Param1, t.Param2))
    return nil
}
```

## Main Entry Point

The main.go file is the entry point for your MCP server. It registers your tools and resources:

```go
// main.go
package main

import (
    "my-mcp-server/server"
    "my-mcp-server/tools"
    
    "github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

func main() {}

func init() {
    wrapper.SetCtx(
        "my-mcp-server", // Server name
        wrapper.ParseRawConfig(server.ParseFromConfig),
        wrapper.AddMCPTool("my_tool", tools.MyTool{}), // Register tools
        // Add more tools as needed
    )
}
```

## Dependencies

Your MCP server must use a specific version of the wasm-go SDK that supports Go 1.24's WebAssembly compilation features:

```bash
# Add the required dependency with the specific version tag
go get github.com/alibaba/higress/plugins/wasm-go@wasm-go-1.24
```

## Building the WASM Binary

To compile your Go code into a WebAssembly (WASM) file, use the following command:

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm main.go
```

This command sets the target operating system to `wasip1` (WebAssembly System Interface) and architecture to `wasm` (WebAssembly), then builds your code as a C-shared library and outputs it as `main.wasm`.

## Using the Makefile

A Makefile is provided to simplify the build process. It includes the following targets:

- `make build`: Builds the WASM binary for your MCP server
- `make build-image`: Builds a Docker image containing your MCP server
- `make build-push`: Builds and pushes the Docker image to a registry
- `make clean`: Removes build artifacts
- `make help`: Shows available targets and variables

You can customize the build by setting the following variables:

```bash
# Build with a custom server name
make SERVER_NAME=my-mcp-server build

# Build with a custom registry
make REGISTRY=my-registry.example.com/ build-image

# Build with a specific version tag
make SERVER_VERSION=1.0.0 build-image
```

## Testing

You can create unit tests for your tools to verify their functionality:

```go
// tools/my_tool_test.go
package tools

import (
    "encoding/json"
    "fmt"
    "testing"
)

func TestMyToolInputSchema(t *testing.T) {
    myTool := MyTool{}
    schema := myTool.InputSchema()
    
    schemaJSON, err := json.MarshalIndent(schema, "", "  ")
    if err != nil {
        t.Fatalf("Failed to marshal schema to JSON: %v", err)
    }
    
    fmt.Printf("MyTool InputSchema:\n%s\n", string(schemaJSON))
    
    if len(schema) == 0 {
        t.Error("InputSchema returned an empty schema")
    }
}
```
