# Higress API MCP Server

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
