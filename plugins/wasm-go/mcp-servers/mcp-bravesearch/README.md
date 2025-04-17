# Brave Search MCP Server

An MCP server implementation that integrates the Brave Search API, providing web and local search capabilities.

## Features

- **Web Search**: Supports general queries, news, articles, with pagination and time control
- **Local Search**: Find businesses, restaurants, and services with detailed information

Source code: [https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search](https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search)

# Usage Guide

## Get API-KEY

1. Register for a Brave Search API account [Visit official website](https://brave.com/search/api/)
2. Choose a plan (free plan includes 2000 queries per month)
3. Generate API key through developer console [Go to console](https://api.search.brave.com/app/keys)

## Generate SSE URL

On the MCP Server interface, log in and enter the API-KEY to generate the URL.

## Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "bravesearch": {
      "url": "http://mcp.higress.ai/mcp-brave-search/{generate_key}",
    }
}
```
