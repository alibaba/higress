# Higress API MCP Server

Higress API MCP Server 提供了 MCP 工具来管理 Higress 路由、服务来源和插件等资源。

## 功能特性

### 路由管理
- `list-routes`: 列出路由
- `get-route`: 获取路由
- `add-route`: 添加路由
- `update-route`: 更新路由

### 服务来源管理
- `list-service-sources`: 列出服务来源
- `get-service-source`: 获取服务来源
- `add-service-source`: 添加服务来源
- `update-service-source`: 更新服务来源

### 插件管理
- `list-plugins`: 列出插件
- `get-plugin-config`: 获取插件配置
- `update-request-block-config`: 更新 request-block 插件配置

## 配置参数

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `higressURL` | string | 必填 | Higress Console 的 URL 地址 |
| `username` | string | 必填 | Higress Console 登录用户名 |
| `password` | string | 必填 | Higress Console 登录密码 |
| `description` | string | 可选 | 服务器描述信息 |

配置示例：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    meta.helm.sh/release-name: higress
    meta.helm.sh/release-namespace: higress-system
  labels:
    app: higress-gateway
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: higress-gateway
    app.kubernetes.io/version: 2.1.4
    helm.sh/chart: higress-core-2.1.4
    higress: higress-system-higress-gateway
  name: higress-config
  namespace: higress-system
data:
  higress: |-
    mcpServer:
      sse_path_suffix: /sse # SSE 连接的路径后缀
      enable: true # 启用 MCP Server
      redis:
        address: redis-stack-server.higress-system.svc.cluster.local:6379 # Redis服务地址
        username: "" # Redis用户名（可选）
        password: "" # Redis密码（可选）
        db: 0 # Redis数据库（可选）
      match_list: # MCP Server 会话保持路由规则（当匹配下面路径时，将被识别为一个 MCP 会话，通过 SSE 等机制进行会话保持）
        - match_rule_domain: "*"
          match_rule_path: /higress-api
          match_rule_type: "prefix"
      servers:
        - name: higress-api-mcp-server # MCP Server 名称
          path: /higress-api # 访问路径，需要与 match_list 中的配置匹配
          type: higress-api # 类型和 RegisterServer 一致
          config:
            higressURL: http://higress-console.higress-system.svc.cluster.local:8080
            username: admin
            password: admin
```
