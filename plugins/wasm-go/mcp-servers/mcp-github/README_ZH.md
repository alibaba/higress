# GitHub MCP Server

GitHub API 的 MCP 服务器实现，支持文件操作、仓库管理、搜索等功能。

源码地址：[https://github.com/modelcontextprotocol/servers/tree/main/src/github](https://github.com/modelcontextprotocol/servers/tree/main/src/github)

## 功能

- **自动分支创建**: 在创建/更新文件或推送更改时，如果分支不存在会自动创建
- **全面的错误处理**: 提供常见问题的清晰错误信息
- **Git 历史保留**: 操作会保留完整的 Git 历史记录，不会强制推送
- **批量操作**: 支持单文件和批量文件操作
- **高级搜索**: 支持代码、issues/PRs 和用户的搜索

## 使用教程

### 获取 AccessToken
[创建 GitHub 个人访问令牌](https://docs.github.com/zh/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens):
   1. 访问 [个人访问令牌](https://github.com/settings/tokens)（在 GitHub 设置 > 开发者设置中）
   2. 选择该令牌可以访问的仓库（公开、所有或选择）
   3. 创建具有 `repo` 权限的令牌（"对私有仓库的完全控制"）
      - 或者，如果仅使用公开仓库，选择仅 `public_repo` 权限
   4. 复制生成的令牌
   
### 生成 SSE URL

在 MCP Server 界面，登录后输入 AccessToken，生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

```json
"mcpServers": {
    "github": {
      "url": "http://mcp.higress.ai/mcp-github/{generate_key}",
    }
}
```

