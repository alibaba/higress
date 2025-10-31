# Higress Ops MCP Server

Higress Ops MCP Server 提供了 MCP 工具来调试和监控 Istio 和 Envoy 组件，帮助运维人员进行故障诊断和性能分析。

## 功能特性

### Istiod 调试接口

#### 配置相关
- `get-istiod-configz`: 获取 Istiod 的配置状态和错误信息

#### 服务发现相关
- `get-istiod-endpointz`: 获取 Istiod 发现的所有服务端点信息
- `get-istiod-clusters`: 获取 Istiod 发现的所有集群信息
- `get-istiod-registryz`: 获取 Istiod 的服务注册表信息

#### 状态监控相关
- `get-istiod-syncz`: 获取 Istiod 与 Envoy 代理的同步状态信息
- `get-istiod-metrics`: 获取 Istiod 的 Prometheus 指标数据

#### 系统信息相关
- `get-istiod-version`: 获取 Istiod 的版本信息
- `get-istiod-debug-vars`: 获取 Istiod 的调试变量信息

### Envoy 调试接口

#### 配置相关
- `get-envoy-config-dump`: 获取 Envoy 的完整配置快照，支持资源过滤和敏感信息掩码
- `get-envoy-listeners`: 获取 Envoy 的所有监听器信息
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
| `namespace` | string | 可选 | Kubernetes 命名空间，默认为 `higress-system` |
| `description` | string | 可选 | 服务器描述信息，默认为 "Higress Ops MCP Server, which provides debug interfaces for Istio and Envoy components." |

## 配置示例

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: higress-config
  namespace: higress-system
  resourceVersion: '107160'
data:
  higress: |
    mcpServer:
      sse_path_suffix: /sse  # SSE 连接的路径后缀
      enable: true           # 启用 MCP Server
      redis:
        address: redis-stack-server.higress-system.svc.cluster.local:6379  # Redis服务地址
        username: ""         # Redis用户名（可选）
        password: ""         # Redis密码（可选）
        db: 0                # Redis数据库（可选）
      match_list:            # MCP Server 会话保持路由规则
        - match_rule_domain: "*"
          match_rule_path: /higress-ops
          match_rule_type: "prefix"
      servers:
        - name: higress-ops-mcp-server
          path: /higress-ops
          type: higress-ops
          config:
            istiodURL: http://higress-controller.higress-system.svc.cluster.local:15014   # istiod url
            envoyAdminURL: http://127.0.0.1:15000 # envoy url 填127.0.0.1就行，和 gateway 于同一容器
            namespace: higress-system
            description: "Higress Ops MCP Server for Istio and Envoy debugging"
  mesh: |-
    accessLogEncoding: TEXT
    accessLogFile: /dev/stdout
    accessLogFormat: '{"ai_log":"%FILTER_STATE(wasm.ai_log:PLAIN)%","authority":"%REQ(X-ENVOY-ORIGINAL-HOST?:AUTHORITY)%","bytes_received":"%BYTES_RECEIVED%","bytes_sent":"%BYTES_SENT%","downstream_local_address":"%DOWNSTREAM_LOCAL_ADDRESS%","downstream_remote_address":"%DOWNSTREAM_REMOTE_ADDRESS%","duration":"%DURATION%","istio_policy_status":"%DYNAMIC_METADATA(istio.mixer:status)%","method":"%REQ(:METHOD)%","path":"%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%","protocol":"%PROTOCOL%","request_id":"%REQ(X-REQUEST-ID)%","requested_server_name":"%REQUESTED_SERVER_NAME%","response_code":"%RESPONSE_CODE%","response_flags":"%RESPONSE_FLAGS%","route_name":"%ROUTE_NAME%","start_time":"%START_TIME%","trace_id":"%REQ(X-B3-TRACEID)%","upstream_cluster":"%UPSTREAM_CLUSTER%","upstream_host":"%UPSTREAM_HOST%","upstream_local_address":"%UPSTREAM_LOCAL_ADDRESS%","upstream_service_time":"%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%","upstream_transport_failure_reason":"%UPSTREAM_TRANSPORT_FAILURE_REASON%","user_agent":"%REQ(USER-AGENT)%","x_forwarded_for":"%REQ(X-FORWARDED-FOR)%","response_code_details":"%RESPONSE_CODE_DETAILS%"}'
    configSources:
    - address: xds://127.0.0.1:15051
    - address: k8s://
    defaultConfig:
      discoveryAddress: higress-controller.higress-system.svc:15012
      proxyStatsMatcher:
        inclusionRegexps:
        - .*
      tracing: {}
    dnsRefreshRate: 200s
    enableAutoMtls: false
    enablePrometheusMerge: true
    ingressControllerMode: "OFF"
    mseIngressGlobalConfig:
      enableH3: false
      enableProxyProtocol: false
    protocolDetectionTimeout: 100ms
    rootNamespace: higress-system
    trustDomain: cluster.local
  meshNetworks: 'networks: {}'

```

## 鉴权配置

Higress Ops MCP Server 使用自定义 HTTP Header 进行鉴权。客户端需要在请求头中携带 Istiod 认证 Token。

### Token 生成方式

使用以下命令生成长期有效的 Istiod 认证 Token：

```bash
kubectl create token higress-gateway -n higress-system --audience istio-ca --duration 87600h
```

**参数说明：**
- `higress-gateway`: ServiceAccount 名称（与 Higress Gateway Pod 使用的 ServiceAccount 一致）
- `-n higress-system`: 命名空间（需要与配置参数 `namespace` 一致）
- `--audience istio-ca`: Token 的受众，必须为 `istio-ca`
- `--duration 87600h`: Token 有效期（87600小时 ≈ 10年）

### 配置示例

```json
{
  "mcpServers": {
    "higress_ops_mcp": {
      "url": "http://127.0.0.1:80/higress-ops/sse",
      "headers": {
        "X-Istiod-Token": "eyJhbGciOiJSUzI1NiIsImtpZCI6Im1IUlI0Z01ISUNBNVlZbDBHcVVBMjFhMklwQ3hFaHIxSlVlamtzTFRLOTQifQ..."
      }
    }
  }
}
```

**说明：**
- `X-Istiod-Token` 头用于携带 Istiod 认证 Token
- Token 值由上述 `kubectl create token` 命令生成
- 如果未配置 Token，跨 Pod 访问 Istiod 接口时会遇到 401 认证错误

## 演示

1. get envoy route information

https://private-user-images.githubusercontent.com/153273766/507769115-d8e20b70-db1a-4a82-b89a-9eefeb3c8982.mov?jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3NjE4Nzg4NjAsIm5iZiI6MTc2MTg3ODU2MCwicGF0aCI6Ii8xNTMyNzM3NjYvNTA3NzY5MTE1LWQ4ZTIwYjcwLWRiMWEtNGE4Mi1iODlhLTllZWZlYjNjODk4Mi5tb3Y_WC1BbXotQWxnb3JpdGhtPUFXUzQtSE1BQy1TSEEyNTYmWC1BbXotQ3JlZGVudGlhbD1BS0lBVkNPRFlMU0E1M1BRSzRaQSUyRjIwMjUxMDMxJTJGdXMtZWFzdC0xJTJGczMlMkZhd3M0X3JlcXVlc3QmWC1BbXotRGF0ZT0yMDI1MTAzMVQwMjQyNDBaJlgtQW16LUV4cGlyZXM9MzAwJlgtQW16LVNpZ25hdHVyZT1kYzg1Y2FiOTdiN2FiOTNkMmQ0OTc1NzEyZGMyMTlkNDQ4YjQ0NGYyOGUwNTlhYzYyYzA1ODJhOWM0M2Y3ZTQyJlgtQW16LVNpZ25lZEhlYWRlcnM9aG9zdCJ9.Uz-HfM9tOzl7zrhGsPP1suunGg_K9ZbUN1BzAU5Oquo

2. get istiod cluster information

https://private-user-images.githubusercontent.com/153273766/507769013-9f598593-1251-4304-8e41-8bf4d1588897.mov?jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3NjE4Nzg4NjAsIm5iZiI6MTc2MTg3ODU2MCwicGF0aCI6Ii8xNTMyNzM3NjYvNTA3NzY5MDEzLTlmNTk4NTkzLTEyNTEtNDMwNC04ZTQxLThiZjRkMTU4ODg5Ny5tb3Y_WC1BbXotQWxnb3JpdGhtPUFXUzQtSE1BQy1TSEEyNTYmWC1BbXotQ3JlZGVudGlhbD1BS0lBVkNPRFlMU0E1M1BRSzRaQSUyRjIwMjUxMDMxJTJGdXMtZWFzdC0xJTJGczMlMkZhd3M0X3JlcXVlc3QmWC1BbXotRGF0ZT0yMDI1MTAzMVQwMjQyNDBaJlgtQW16LUV4cGlyZXM9MzAwJlgtQW16LVNpZ25hdHVyZT1hZDQwYWE3MjM5OTU1NGNkMDcwNTgzNDMzZGI4NDRkYzdiNWRlNGJhODMwNjFlYjZiZjUzNzM3YWFhYzIyMjBjJlgtQW16LVNpZ25lZEhlYWRlcnM9aG9zdCJ9.g19-rxOHSLIIszdGYAI7CmRzLTlrbA1fJ0hB6duuDBI



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
- 使用 `get-istiod-configz` 验证 Istiod 配置状态
- 使用 `get-envoy-config-dump` 验证 Envoy 配置

### 4. 安全审计
- 使用 `get-envoy-certs` 检查证书状态
- 使用 `get-istiod-debug-vars` 查看调试变量

## 工具参数示例

### Istiod 工具示例

```bash
# 获取配置状态
get-istiod-configz

# 获取同步状态
get-istiod-syncz

# 获取端点信息
get-istiod-endpointz
```

### Envoy 工具示例

```bash
# 获取配置快照，过滤监听器配置
get-envoy-config-dump --resource="listeners"

# 获取集群信息，JSON 格式输出
get-envoy-clusters --format="json"

# 获取统计信息，只显示包含 "cluster" 的统计项
get-envoy-stats --filter="cluster.*" --format="json"
```

## 常见问题

### Q: 如何获取特定集群的详细信息？
A: 使用 `get-envoy-clusters` 工具，然后使用 `get-envoy-config-dump --resource="clusters"` 获取详细配置。

### Q: 如何监控配置同步状态？
A: 使用 `get-istiod-syncz` 查看整体同步状态，使用 `get-istiod-configz` 查看配置状态和错误信息。

### Q: 如何排查路由问题？
A: 使用 `get-envoy-config-dump` 获取详细路由信息。

### Q: 支持哪些输出格式？
A: 大部分工具支持 text 和 json 格式，统计信息还支持 prometheus 格式。
