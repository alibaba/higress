# Higress Ops MCP Server

Higress Ops MCP Server 提供了 MCP 工具来调试和监控 Istio 和 Envoy 组件，帮助运维人员进行故障诊断和性能分析。

## 功能特性

### Istiod 调试接口

#### 配置相关
- `get-istiod-config-dump`: 获取 Istiod 的完整配置快照，包括所有 xDS 配置
- `get-istiod-configz`: 获取 Istiod 的配置状态和错误信息

#### 服务发现相关
- `get-istiod-endpointz`: 获取 Istiod 发现的所有服务端点信息
- `get-istiod-clusters`: 获取 Istiod 发现的所有集群信息
- `get-istiod-registryz`: 获取 Istiod 的服务注册表信息

#### 状态监控相关
- `get-istiod-syncz`: 获取 Istiod 与 Envoy 代理的同步状态信息
- `get-istiod-proxy-status`: 获取所有连接到 Istiod 的代理状态
- `get-istiod-metrics`: 获取 Istiod 的 Prometheus 指标数据

#### 系统信息相关
- `get-istiod-version`: 获取 Istiod 的版本信息
- `get-istiod-debug-vars`: 获取 Istiod 的调试变量信息

### Envoy 调试接口

#### 配置相关
- `get-envoy-config-dump`: 获取 Envoy 的完整配置快照，支持资源过滤和敏感信息掩码
- `get-envoy-listeners`: 获取 Envoy 的所有监听器信息
- `get-envoy-routes`: 获取 Envoy 的路由配置信息
- `get-envoy-clusters`: 获取 Envoy 的所有集群信息和健康状态

#### 运行时相关
- `get-envoy-stats`: 获取 Envoy 的统计信息，支持过滤器和多种输出格式
- `get-envoy-runtime`: 获取 Envoy 的运行时配置信息
- `get-envoy-memory`: 获取 Envoy 的内存使用情况

#### 状态检查相关
- `get-envoy-server-info`: 获取 Envoy 服务器的基本信息
- `get-envoy-ready`: 检查 Envoy 是否准备就绪
- `get-envoy-hot-restart-version`: 获取 Envoy 热重启版本信息

#### 安全相关
- `get-envoy-certs`: 获取 Envoy 的证书信息

## 配置参数

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `istiodURL` | string | 必填 | Istiod 调试接口的 URL 地址 |
| `envoyAdminURL` | string | 必填 | Envoy Admin 接口的 URL 地址 |
| `namespace` | string | 可选 | Kubernetes 命名空间，默认为 istio-system |
| `description` | string | 可选 | 服务器描述信息 |

## 配置示例

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
      match_list: # MCP Server 会话保持路由规则
        - match_rule_domain: "*"
          match_rule_path: /higress-ops
          match_rule_type: "prefix"
      servers:
        - name: higress-ops-mcp-server # MCP Server 名称
          path: /higress-ops # 访问路径，需要与 match_list 中的配置匹配
          type: higress-ops # 类型和 RegisterServer 一致
          config:
            istiodURL: http://istiod.istio-system.svc.cluster.local:15014
            envoyAdminURL: http://higress-gateway.higress-system.svc.cluster.local:15000
            namespace: istio-system
```

## 使用场景

### 1. 故障诊断
- 使用 `get-istiod-syncz` 检查配置同步状态
- 使用 `get-envoy-clusters` 检查集群健康状态  
- 使用 `get-envoy-listeners` 检查监听器配置

### 2. 性能分析
- 使用 `get-istiod-metrics` 获取 Istiod 性能指标
- 使用 `get-envoy-stats` 获取 Envoy 统计信息
- 使用 `get-envoy-memory` 监控内存使用

### 3. 配置验证
- 使用 `get-istiod-config-dump` 验证 Istiod 配置
- 使用 `get-envoy-config-dump` 验证 Envoy 配置
- 使用 `get-envoy-routes` 检查路由配置

### 4. 安全审计
- 使用 `get-envoy-certs` 检查证书状态
- 使用 `get-istiod-debug-vars` 查看调试变量

## 工具参数示例

### Istiod 工具示例

```bash
# 获取特定代理的状态
get-istiod-proxy-status --proxy="gateway-proxy.istio-system"

# 获取配置快照
get-istiod-config-dump

# 获取同步状态
get-istiod-syncz
```

### Envoy 工具示例

```bash
# 获取配置快照，过滤监听器配置
get-envoy-config-dump --resource="listeners"

# 获取集群信息，JSON 格式输出
get-envoy-clusters --format="json"

# 获取统计信息，只显示包含 "cluster" 的统计项
get-envoy-stats --filter="cluster.*" --format="json"

# 获取特定路由表信息
get-envoy-routes --name="80" --format="json"
```

## 常见问题

### Q: 如何获取特定集群的详细信息？
A: 使用 `get-envoy-clusters` 工具，然后使用 `get-envoy-config-dump --resource="clusters"` 获取详细配置。

### Q: 如何监控配置同步状态？
A: 使用 `get-istiod-syncz` 查看整体同步状态，使用 `get-istiod-proxy-status` 查看特定代理状态。

### Q: 如何排查路由问题？
A: 使用 `get-envoy-routes` 查看路由配置，使用 `get-envoy-config-dump --resource="routes"` 获取详细路由信息。

### Q: 支持哪些输出格式？
A: 大部分工具支持 text 和 json 格式，统计信息还支持 prometheus 格式。
