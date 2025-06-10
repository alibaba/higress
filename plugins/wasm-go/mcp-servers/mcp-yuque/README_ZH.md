# 语雀 MCP Server

基于语雀开放服务 API 的 MCP 服务器实现，通过 MCP 协议，实现语雀知识库、文档的编辑、更新发布。

## 功能

当前支持以下操作：
- **知识库管理**：新建、搜索、更新、删除知识库等。
- **文档管理**：创建、更新、历史详情、搜索文档等。


对于企业团队用户：
- **成员管理**：管理知识库成员、权限。
- **数据汇总**：知识库、文档、成员等数据统计。


## 使用教程

### 获取 AccessToken

参考[语雀开发者文档](https://www.yuque.com/yuque/developer/api)，进行个人用户认证或企业团队身份认证。
   
### 生成 SSE URL

在 MCP Server 界面，登录后输入 AccessToken，生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

```json
"mcpServers": {
    "yuque": {
      "url": "https://mcp.higress.ai/mcp-yuque/{generate_key}",
    }
}
```

