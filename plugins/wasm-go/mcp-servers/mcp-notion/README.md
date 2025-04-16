# Notion MCP Server

A Notion workspace is a collaborative environment where teams can organize work, manage projects, and store information in a highly customizable way. Notion's REST API facilitates direct interactions with workspace elements through programming. 

Source code: [https://github.com/makenotion/notion-mcp-server/tree/main](https://github.com/makenotion/notion-mcp-server/tree/main)

## Feature

Notion MCP Server provides the following features:

- **Pages**: Create, update, and retrieve page content.
- **Databases**: Manage database, properties, entries, and schemas.
- **Users**: Access user profiles and permissions.
- **Comments**: Handle page and inline comments.
- **Content Queries**: Search through workspace content.

## Usage Guide

### Get Notion Integration token

Go to [https://www.notion.so/profile/integrations](https://www.notion.so/profile/integrations) and create a new internal integration or select an existing one.


### Generate SSE URL

On the MCP Server interface, log in and enter the token to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "notion": {
      "url": "http://mcp.higress.ai/mcp-notion/{generate_key}",
    }
}
```
