# WolframAlpha MCP Server

一个集成了[WolframAlpha](https://www.wolframalpha.com/)的模型上下文协议（MCP）服务器实现，提供自然语言计算和知识查询功能。

## 功能

- 支持自然语言查询，涵盖数学、物理、化学、地理、历史、艺术、天文等领域
- 执行数学计算、日期和单位转换、公式求解等
- 支持图像结果展示
- 自动将复杂查询转换为简化关键词查询
- 支持多语言查询（自动翻译为英文处理，返回原语言结果）

## 使用教程

### 获取 AppID
1. 注册 WolframAlpha 开发者账号 [Create a Wolfram ID](https://account.wolfram.com/login/create)
2. 生成LLM-API 的 App ID [Get An App ID](https://developer.wolframalpha.com/access)

### 生成 SSE URL

在 MCP Server 界面，登录后输入 AppID，生成URL。

### 配置 MCP Client

在用户的 MCP Client 界面，将生成的 SSE URL添加到 MCP Server列表中。

```json
"mcpServers": {
    "wolframalpha": {
      "url": "http://mcp.higress.ai/mcp-wolframalpha/{generate_key}",
    }
}
