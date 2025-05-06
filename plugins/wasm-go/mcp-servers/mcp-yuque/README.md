# Yuque MCP Server

Implementation of the MCP server based on the Yuque Open Service API, enabling the editing, updating, and publishing of Yuque knowledge bases and documents through the MCP protocol.


## Features

Currently supports the following operations:
- **Knowledge Base Management**: Create, search, update, delete knowledge bases, etc.
- **Document Management**: Create, update, view history details, search documents, etc.

For enterprise team users:
- **Member Management**: Manage knowledge base members and permissions.
- **Data Aggregation**: Statistics on knowledge bases, documents, members, etc.


## Usage Guide

### Get AccessToken

Refer to the [Yuque Developer Documentation](https://www.yuque.com/yuque/developer/api) for personal user authentication or enterprise team identity authentication.
   
### Generate SSE URL

On the MCP Server interface, log in and enter the AccessToken to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "yuque": {
      "url": "https://mcp.higress.ai/mcp-yuque/{generate_key}",
    }
}
```
