# Weather MCP Server

A weather query MCP service based on OpenWeather API, retrieving weather information for specified cities.

Source code: [https://github.com/MrCare/mcp_tool](https://github.com/MrCare/mcp_tool)

## Usage Guide
   
### Generate SSE URL

On the MCP Server interface, log in to generate the URL.

### Configure MCP Client

On the user's MCP Client interface, add the generated SSE URL to the MCP Server list.

```json
"mcpServers": {
    "weather": {
      "url": "https://mcp.higress.ai/mcp-weather/{generate_key}",
    }
}
```
