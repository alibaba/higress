# Higress OPS MCP Server

Higress OPS MCP Server 是一个基于 Model Context Protocol (MCP) 的 Higress Console 管理服务器，提供了通过 MCP 协议管理 Higress 路由、服务源和插件的能力。

## 功能特性

### 🚦 路由管理
- **列出路由** (`list_routes`) - 获取所有可用路由列表
- **获取路由** (`get_route`) - 获取指定路由的详细配置
- **添加路由** (`add_route`) - 创建新的路由配置 ⚠️ *敏感操作*
- **更新路由** (`update_route`) - 修改现有路由配置 ⚠️ *敏感操作*

### 🏢 服务源管理
- **列出服务源** (`list_service_sources`) - 获取所有服务源列表
- **获取服务源** (`get_service_source`) - 获取指定服务源的详细信息
- **添加服务源** (`add_service_source`) - 创建新的服务源 ⚠️ *敏感操作*
- **更新服务源** (`update_service_source`) - 修改现有服务源 ⚠️ *敏感操作*

### 🔌 插件管理
- **获取插件配置** (`get_plugin_config`) - 获取路由的插件配置
- **更新插件配置** (`update_plugin_config`) - 修改插件配置 ⚠️ *敏感操作*
- **获取请求阻断配置** (`get_request_block_config`) - 获取 request-block 插件配置
- **更新请求阻断配置** (`update_request_block_config`) - 修改 request-block 插件配置 ⚠️ *敏感操作*

### 🔧 通用工具
- **健康检查** (`health_check`) - 检查 Higress Console 连接状态
- **系统信息** (`get_system_info`) - 获取 Higress Console 系统信息
- **列出插件** (`list_plugins`) - 获取所有可用插件列表

## 配置说明

### 基本配置

```yaml
servers:
  - type: higress                    # 服务器类型
    name: higress-console           # 服务器实例名称
    path: /higress                  # MCP 服务路径
    domain_list:                    # 允许访问的域名
      - "console.example.com"
    config:
      higressURL: "https://console.example.com"  # Higress Console URL (必需)
      username: "admin"                          # 用户名 (必需)
      password: "your-password"                  # 密码 (必需)
      description: "Higress Console Management"  # 描述 (可选)
```

### 配置参数详解

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `higressURL` | string | ✅ | Higress Console 的 URL 地址 |
| `username` | string | ✅ | Higress Console 登录用户名 |
| `password` | string | ✅ | Higress Console 登录密码 |
| `description` | string | ❌ | 服务器描述信息，默认为 "Higress Console Management Server" |

## 使用示例

### 1. 建立 SSE 连接

```bash
curl -X GET "https://your-gateway.com/higress/sse"
```

返回：
```json
{
  "endpoint": "https://your-gateway.com/higress?sessionId=abc123",
  "sessionId": "abc123"
}
```

### 2. 列出所有路由

```bash
curl -X POST "https://your-gateway.com/higress?sessionId=abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "list_routes",
      "arguments": {}
    }
  }'
```

### 3. 获取特定路由信息

```bash
curl -X POST "https://your-gateway.com/higress?sessionId=abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "get_route",
      "arguments": {
        "name": "my-api-route"
      }
    }
  }'
```

### 4. 添加新路由

```bash
curl -X POST "https://your-gateway.com/higress?sessionId=abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "add_route",
      "arguments": {
        "configurations": {
          "name": "new-api-route",
          "domains": ["api.example.com"],
          "path": {
            "matchType": "PRE",
            "matchValue": "/api/v1/"
          },
          "methods": ["GET", "POST"],
          "services": [
            {
              "name": "backend-service",
              "port": 8080,
              "weight": 100
            }
          ]
        }
      }
    }
  }'
```

### 5. 健康检查

```bash
curl -X POST "https://your-gateway.com/higress?sessionId=abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "health_check",
      "arguments": {}
    }
  }'
```

## 安全注意事项

⚠️ **敏感操作警告**

以下操作被标记为敏感操作，执行时需要特别注意：

- `add_route` - 添加新路由
- `update_route` - 更新路由配置
- `add_service_source` - 添加新服务源
- `update_service_source` - 更新服务源配置
- `update_plugin_config` - 更新插件配置
- `update_request_block_config` - 更新请求阻断配置

建议在生产环境中：
1. 启用 mcp-session 的认证机制
2. 配置适当的速率限制
3. 限制访问域名列表
4. 定期轮换 Higress Console 密码
5. 监控敏感操作的执行日志

## 错误处理

服务器提供详细的错误信息，包括：

- **配置错误**: 缺少必需参数或参数格式错误
- **连接错误**: 无法连接到 Higress Console
- **认证错误**: 用户名或密码错误
- **API 错误**: Higress Console API 返回错误
- **网络错误**: 网络连接问题

所有错误都会包含详细的错误描述，便于问题诊断。

## 架构说明

Higress OPS MCP Server 基于现有的 MCP 框架构建：

```
mcp-session (会话管理)
    ↓
mcp-server (服务器管理)
    ↓
higress-api (Higress 专用服务器)
    ↓
Higress Console API
```

- **mcp-session**: 提供会话管理、SSE 连接、认证和限流
- **mcp-server**: 提供 MCP 协议实现和服务器注册机制
- **higress-api**: 实现 Higress Console 的具体业务逻辑
- **Higress Console API**: 实际的 Higress 管理接口

## 开发说明

### 文件结构

```
mcp-server/servers/higress/higress-api/
├── server.go           # 服务器配置和注册
├── client.go          # Higress Console API 客户端
├── types.go           # API 类型定义
├── tools_route.go     # 路由管理工具
├── tools_service.go   # 服务源管理工具
├── tools_plugin.go    # 插件管理工具
├── tools_common.go    # 通用工具
├── example-config.yaml # 配置示例
└── README.md          # 本文档
```

### 扩展新功能

1. 在相应的 `tools_*.go` 文件中添加新的工具函数
2. 在 `types.go` 中定义相关的数据结构
3. 在 `server.go` 的 `NewServer` 方法中注册新工具
4. 更新 JSON Schema 定义

## 版本信息

- **版本**: 1.0.0
- **MCP 协议版本**: 兼容
- **Go 版本要求**: 1.19+
- **依赖**: 基于现有的 mcp-session/common 框架

## 许可证

本项目遵循与 Higress 主项目相同的许可证。 