# ChatPPT MCP Server

Biyou Technology's MCP Server currently covers 18 intelligent document processing interfaces, including but not limited to PPT creation, PPT beautification, PPT generation, resume creation, resume analysis, and person-job matching. Users can build their own document creation tools through the server, enabling more possibilities for intelligent document creation.

Source code: [https://github.com/YOOTeam/chatppt-mcp](https://github.com/YOOTeam/chatppt-mcp)

## Usage Guide

### Get API-KEY

Refer to the official documentation to get API-KEY [Create Application and Get Token](https://wiki.yoo-ai.com/mcp/McpServe/serve1.3.html)

### Generate SSE URL

On the MCP Server interface, log in and enter the API-KEY to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "chatppt": {
      "url": "http://mcp.higress.ai/mcp-chatppt/{generate_key}",
    }
}
```
