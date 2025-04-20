# Brave Search MCP Server

一个集成Brave搜索API的MCP服务器实现，提供网页和本地搜索功能。

## 功能

- **网页搜索**：支持通用查询、新闻、文章，具备分页和时效性控制
- **本地搜索**：查找带有详细信息的企业、餐厅和服务

源码地址：[https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search](https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search)

# 使用教程

## 获取 API-KEY

1. 注册Brave搜索API账号 [访问官网](https://brave.com/search/api/)
2. 选择套餐（免费套餐每月包含2000次查询）
3. 通过开发者控制台生成 API 密钥 [前往控制台](https://api.search.brave.com/app/keys)

## 生成 SSE URL

在 MCP Server 界面，登录后输入 API-KEY，生成URL。



## 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到MCP Server列表中。

```json
"mcpServers": {
    "bravesearch": {
      "url": "http://mcp.higress.ai/mcp-brave-search/{generate_key}",
    }
}
```


