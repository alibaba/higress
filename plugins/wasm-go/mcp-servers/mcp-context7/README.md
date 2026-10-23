# Context7 MCP Server

An implementation of the Model Context Protocol (MCP) server that integrates [Context7](https://context7.com), providing up-to-date, version-specific documentation and code examples.

Source Code: [https://github.com/upstash/context7](https://github.com/upstash/context7)

## Features

- Get up-to-date, version-specific documentation
- Extract real, working code examples from source
- Provide concise, relevant information without filler
- Free for personal use
- Integration with your MCP server and tools

## Usage Guide

### Generate SSE URL

On the MCP Server interface, log in and enter the API-KEY to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "context7": {
      "url": "https://mcp.higress.ai/mcp-context7/{generate_key}",
    }
}
```

### Available Tools

#### resolve-library-id
Resolves a general package name into a Context7-compatible library ID. This is a required first step before using the get-library-docs tool.

Parameters:
- query: Library name to search for and retrieve a Context7-compatible library ID (required)

#### get-library-docs
Fetches up-to-date documentation for a library. You must call resolve-library-id first to obtain the exact Context7-compatible library ID.

Parameters:
- folders: Folders filter for organizing documentation
- libraryId: Unique identifier of the library (required)
- tokens: Maximum number of tokens to return (default: 5000)
- topic: Specific topic within the documentation
- type: Type of documentation to retrieve (currently only "txt" supported)
