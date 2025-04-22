# LibreChat MCP Server

An implementation of the Librechat Code Interpreter MCP server that follows the OpenAPI specification, providing code execution and file management capabilities.

## Features

- Supports code execution in multiple programming languages
- Supports file upload, download and deletion
- Provides detailed API response information

## Usage Guide

### Get API-KEY
1. Register for a LibreChat account [Visit official website](https://code.librechat.ai)
2. Manage your plan and then generate API Key through developer console.

### Generate SSE URL

On the MCP Server interface, log in and enter the API-KEY to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "librechat": {
      "url": "http://mcp.higress.ai/mcp-librechat/{generate_key}",
    }
}
```

### Available Tools

#### delete_file
Delete specified file

Parameters:
- fileId: File ID (required)
- session_id: Session ID (required)

#### executeCode
Execute code in specified programming language

Parameters:
- code: Source code to execute (required)
- lang: Programming language (required, options: c, cpp, d, f90, go, java, js, php, py, rs, ts, r)
- args: Command line arguments (optional)
- entity_id: Assistant/agent identifier (optional)
- files: Array of file references (optional)
- user_id: User identifier (optional)

#### get_file
Get file information

Parameters:
- session_id: Session ID (required)
- detail: Detail information (optional)
