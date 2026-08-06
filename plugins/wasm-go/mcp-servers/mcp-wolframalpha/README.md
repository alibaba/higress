# WolframAlpha MCP Server

An implementation of the Model Context Protocol (MCP) server that integrates [WolframAlpha](https://www.wolframalpha.com/), providing natural language computation and knowledge query capabilities.

## Features

- Supports natural language queries in mathematics, physics, chemistry, geography, history, art, astronomy, and more
- Performs mathematical calculations, date and unit conversions, formula solving, etc.
- Supports image result display
- Automatically converts complex queries into simplified keyword queries
- Supports multilingual queries (automatically translates to English for processing, returns results in original language)

## Usage Guide

### Get AppID
1. Register for a WolframAlpha developer account [Create a Wolfram ID](https://account.wolfram.com/login/create)
2. Generate LLM-API AppID [Get An App ID](https://developer.wolframalpha.com/access)

### Generate SSE URL

On the MCP Server interface, log in and enter the AppID to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "wolframalpha": {
      "url": "https://mcp.higress.ai/mcp-wolframalpha/{generate_key}",
    }
}
