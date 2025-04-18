# Firecrawl MCP Server

一个集成了[Firecrawl](https://github.com/mendableai/firecrawl)的模型上下文协议（MCP）服务器实现，提供网页抓取功能。

## 功能

- 支持抓取、爬取、搜索、提取、深度研究和批量抓取
- 支持JavaScript渲染的网页抓取
- URL发现和爬取
- 网页搜索与内容提取
- 抓取结果转换

## 使用教程

### 获取 API-KEY
1. 注册Firecrawl 账号 [访问官网](https://www.firecrawl.dev/app)
2. 通过开发者控制台生成 API Key [前往控制台](https://www.firecrawl.dev/app/api-keys)

### 生成 SSE URL

在 MCP Server 界面，登录后输入 API-KEY，生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

```json
"mcpServers": {
    "firecrawl": {
      "url": "http://mcp.higress.ai/mcp-firecrawl/{generate_key}",
    }
}
```

