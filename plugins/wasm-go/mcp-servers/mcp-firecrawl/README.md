# Firecrawl MCP Server

An implementation of the Model Context Protocol (MCP) server that integrates [Firecrawl](https://github.com/mendableai/firecrawl), providing web scraping capabilities.

## Features

- Supports scraping, crawling, searching, extracting, deep research, and batch scraping
- Supports JavaScript-rendered web page scraping
- URL discovery and crawling
- Web search and content extraction
- Scraping result transformation

## Usage Guide

### Get API-KEY
1. Register for a Firecrawl account [Visit official website](https://www.firecrawl.dev/app)
2. Generate API Key through developer console [Go to console](https://www.firecrawl.dev/app/api-keys)

### Generate SSE URL

On the MCP Server interface, log in and enter the API-KEY to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "firecrawl": {
      "url": "https://mcp.higress.ai/mcp-firecrawl/{generate_key}",
    }
}
```
