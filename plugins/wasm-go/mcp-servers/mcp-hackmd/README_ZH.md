# HackMD MCP Server

基于 HackMD API 的 MCP 服务器实现，通过 MCP 协议与 HackMD 平台进行交互。HackMD 是一个实时、多平台的协作 Markdown 知识库，可以让用户在桌面、平板或手机上与他人共同编写文档。

## 功能

HackMD MCP Server 提供了以下功能：

- **用户数据**：获取用户个人信息和相关配置。
    - `get_me`：获取用户数据。

- **笔记管理**：创建、读取、更新和删除个人笔记。
    - `get_notes`：获取用户的笔记列表。
    - `post_notes`：创建新笔记。
    - `get_notes_noteId`：通过 ID 获取特定笔记。
    - `patch_notes_noteId`：更新笔记内容。
    - `delete_notes_noteId`：删除笔记。

- **团队协作**：处理团队笔记相关操作。
    - `get_teams`：获取用户参与的团队列表。
    - `get_teams_teamPath_notes`：获取团队中的笔记列表。
    - `patch_teams_teamPath_notes_noteId`：更新团队中的笔记内容。
    - `delete_teams_teamPath_notes_noteId`：从团队中删除笔记。

- **浏览历史**：查看用户的历史记录。
    - `get_history`：获取用户的浏览历史。

## 使用教程

### 获取 AccessToken

参考 [HackMD API 文档](https://hackmd.io/@hackmd-api/developer-portal/https%3A%2F%2Fhackmd.io%2F%40hackmd-api%2FrkoVeBXkq) 获取 AccessToken。

### 生成 SSE URL

在 MCP Server 界面，登录后输入 AccessToken，生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

``` json
"mcpServers": {
    "hackmd": {
      "url": "https://mcp.higress.ai/mcp-hackmd/{generate_key}",
    }
}
```
