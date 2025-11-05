# ChatPPT MCP Server

必优科技 MCP Server 目前已经覆盖了 18 个智能文档的接口能力，包括但不限于 PPT 创作，PPT 美化，PPT 生成，简历创作，简历分析，人岗匹配等场景下的文档处理能力，用户可通过 server 搭建自己的文档创作工具，让智能文档创作有更多可能。

源码地址： [https://github.com/YOOTeam/chatppt-mcp](https://github.com/YOOTeam/chatppt-mcp)

## 使用教程

### 获取 API-KEY

参考官方文档获取 API-KEY [创建应用获取 Token](https://wiki.yoo-ai.com/mcp/McpServe/serve1.3.html)

### 生成 SSE URL

在 MCP Server 界面，登录后输入 API-KEY，生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

```json
"mcpServers": {
    "chatppt": {
      "url": "https://mcp.higress.ai/mcp-chatppt/{generate_key}",
    }
}
```

