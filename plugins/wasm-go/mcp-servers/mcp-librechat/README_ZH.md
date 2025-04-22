# LibreChat MCP Server

一个基于OpenAPI规范的 LibreChat Code Interpreter MCP 服务器，提供代码执行和文件管理功能。

## 功能

- 支持多种编程语言的代码执行
- 支持文件上传、下载和删除
- 提供详细的API响应信息

## 使用教程

### 获取 API-KEY
1. 注册LibreChat账号 [访问官网](https://code.librechat.ai)
2. 在控制台界面选择付费计划，并创建 API Key

### 生成 SSE URL

在 MCP Server 界面，登录后输入 API-KEY，生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

```json
"mcpServers": {
    "librechat": {
      "url": "http://mcp.higress.ai/mcp-librechat/{generate_key}",
    }
}
```

### 可用工具

#### delete_file
删除指定文件

参数：
- fileId: 文件ID (必填)
- session_id: 会话ID (必填)

#### executeCode
执行指定编程语言的代码

参数：
- code: 要执行的源代码 (必填)
- lang: 编程语言 (必填，可选值：c, cpp, d, f90, go, java, js, php, py, rs, ts, r)
- args: 命令行参数 (可选)
- entity_id: 助手/代理标识符 (可选)
- files: 文件引用数组 (可选)
- user_id: 用户标识符 (可选)

#### get_file
获取文件信息

参数：
- session_id: 会话ID (必填)
- detail: 详细信息 (可选)
