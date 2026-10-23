# Context7 MCP Server

一个集成了[Context7](https://context7.com)的模型上下文协议（MCP）服务器实现，提供最新、版本特定的文档和代码示例。

源码地址：[https://github.com/upstash/context7](https://github.com/upstash/context7)

## 功能

- 获取最新、版本特定的文档
- 从源码中提取真实可用的代码示例
- 提供简洁、相关的信息，无冗余内容
- 支持个人免费使用
- 与MCP服务器和工具集成

## 使用教程

### 生成 SSE URL

在 MCP Server 界面，登录后输入 API-KEY，生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

```json
"mcpServers": {
    "context7": {
      "url": "https://mcp.higress.ai/mcp-context7/{generate_key}",
    }
}
```

### 可用工具

#### resolve-library-id
用于将通用包名解析为Context7兼容的库ID，是使用get-library-docs工具获取文档的必要前置步骤。

参数说明：
- query: 要搜索的库名称，用于获取Context7兼容的库ID (必填)

#### get-library-docs
获取库的最新文档。使用前必须先调用resolve-library-id工具获取Context7兼容的库ID。

参数说明：
- folders: 用于组织文档的文件夹过滤器
- libraryId: 库的唯一标识符 (必填)
- tokens: 返回的最大token数，默认5000
- topic: 文档中的特定主题
- type: 要检索的文档类型，目前仅支持"txt"
