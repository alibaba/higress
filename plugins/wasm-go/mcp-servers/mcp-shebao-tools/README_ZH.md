# Shebao Tools MCP Server

一个集成了社保、公积金、残保金、个税、工伤赔付和工亡赔付计算功能的模型上下文协议（MCP）服务器实现。

## 功能

- 根据城市信息计算社保、公积金费用。输入城市名称和薪资信息，返回详细计算结果。
- 根据企业规模计算残保金。输入企业员工数量和平均薪资，返回计算结果。
- 根据个人薪资计算个税缴纳费用。输入个人薪资，返回缴纳费用。
- 根据工伤情况计算赔付费用。输入工伤等级和薪资信息，返回赔付费用。
- 根据工亡情况计算赔付费用。输入相关信息，返回赔付费用。

## 使用教程

### 获取 apikey
1. 注册账号 [Create a  ID](https://check.junrunrenli.com/#/index?src=higress)
2. 发送邮件to: yuanpeng@junrunrenli.com   标题：MCP  内容：申请MCP社保计算工具服务，并提供你的账号。

### 生成 SSE URL

在 MCP Server 界面，登录后输入 apikey，生成URL。

### 配置 API Key

在 `mcp-server.yaml` 文件中，将 `apikey` 字段设置为有效的 API 密钥。

### 集成到 MCP Client

在用户的 MCP Client 界面，将相关配置添加到 MCP Server 列表中。

```json
"mcpServers": {
    "wolframalpha": {
      "url": "https://open-api.junrunrenli.com/agent/tools?jr-api-key={apikey}",
    }
}