---
title: HTTP 日志推送
keywords: [HTTP, 日志推送]
description: HTTP 日志推送插件配置参考
---

## 功能说明

`HTTP 日志推送`插件实现了将 HTTP 请求和响应日志异步推送到指定收集器的功能。该插件可以捕获请求的详细信息，包括请求方法、路径、状态码、延迟等，并将这些信息以 JSON 格式推送到配置的收集器服务。

## 运行属性

插件执行阶段：`日志阶段`
插件执行优先级：`默认`

## 配置字段

| 名称 | 数据类型 | 填写要求 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- | ---- |
| `collector_cluster` | string | 非必填 | `outbound|80||log-collector.higress-system.svc.cluster.local` | 日志收集器的集群名称 |
| `service_name` | string | 非必填 | `higress-gateway` | 服务名称 |
| `endpoint` | string | 非必填 | `/ingest` | 日志推送的端点路径 |
| `timeout` | uint32 | 非必填 | `50` | 推送超时时间（毫秒） |
| `extract_headers` | array of string | 非必填 | `[]` | 需要提取的请求头列表 |

## 用法示例

### 基本配置

使用默认配置推送日志到默认收集器：

```yaml
collector_cluster: "outbound|80||log-collector.higress-system.svc.cluster.local"
service_name: "higress-gateway"
endpoint: "/ingest"
timeout: 50	extract_headers: ["x-user-id", "x-request-id"]
```

### 自定义配置

自定义收集器和服务名称：

```yaml
collector_cluster: "outbound|8080||custom-collector.default.svc.cluster.local"
service_name: "my-service"
endpoint: "/logs"
timeout: 100	extract_headers: ["x-user-id", "x-api-key"]
```

## 推送数据格式

插件推送的 JSON 数据格式如下：

```json
{
  "ts": 1704067200,
  "service": "higress-gateway",
  "trace_id": "abc123",
  "method": "GET",
  "path": "/api/users",
  "host": "example.com",
  "protocol": "http",
  "status": 200,
  "latency": 12.34,
  "headers": {
    "x-user-id": "123",
    "x-request-id": "abc123"
  }
}
```

## 注意事项

1. 插件使用异步方式推送日志，不会阻塞请求处理
2. 请确保配置的收集器集群存在且可访问
3. 提取的请求头如果不存在，不会包含在推送数据中
4. 超时时间建议设置为较小值，避免影响主请求流程
