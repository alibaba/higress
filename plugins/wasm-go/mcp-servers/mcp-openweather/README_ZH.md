# Weather MCP Server

基于 OpenWeather API 的天气查询 MCP 服务，获取指定城市的天气信息。

源码地址：[https://github.com/MrCare/mcp_tool](https://github.com/MrCare/mcp_tool)

## 使用教程
   
### 生成 SSE URL

在 MCP Server 界面，登录后生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

```json
"mcpServers": {
    "weather": {
      "url": "https://mcp.higress.ai/mcp-weather/{generate_key}",
    }
}
```

