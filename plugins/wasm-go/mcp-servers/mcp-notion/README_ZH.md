# Notion MCP Server

Notion 工作区是一个协作环境，团队可以在其中以高度可定制的方式组织工作、管理项目和存储信息。Notion 的 REST API 方便通过编程与工作区元素直接交互。

源码地址：[https://github.com/makenotion/notion-mcp-server/tree/main](https://github.com/makenotion/notion-mcp-server/tree/main)

## 功能

Notion MCP Server 提供了以下功能：

- **页面**：创建、更新和检索页面内容。
- **数据库**：管理数据库、属性、条目和模式。
- **用户**：访问用户配置文件和权限。
- **评论**：处理页面和内联评论。
- **内容查询**：搜索工作区内容。

## 使用教程

### 获取 Notion 集成 Key

在Notion中设置集成，转到 [https://www.notion.so/profile/integrations](https://www.notion.so/profile/integrations) 并创建一个新的内部集成或选择一个现有的集成。

   
### 生成 SSE URL

在 MCP Server 界面，登录后输入 AccessToken，生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

```json
"mcpServers": {
    "notion": {
      "url": "http://mcp.higress.ai/mcp-notion/{generate_key}",
    }
}
```
