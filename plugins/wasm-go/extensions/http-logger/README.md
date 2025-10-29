---
title: HTTP 请求日志
keywords: [higress, logging, debug]
description: HTTP 请求日志插件配置参考
---

## 功能说明

`http-logger` 插件用于在 Higress 日志中记录 HTTP 请求和响应的详细信息，包括请求头、请求体、响应头、响应体，便于 API 调试和监控。

## 运行属性

插件执行阶段：`默认阶段`
插件执行优先级：`10`

## 配置字段

| 名称               | 数据类型 | 填写要求 | 默认值 | 描述                                                                                           |
| ------------------ | -------- | -------- | ------ | ---------------------------------------------------------------------------------------------- |
| log_request_headers  | bool   | 选填     | true   | 是否记录请求头，默认开启                                                                            |
| log_request_body     | bool   | 选填     | true   | 是否记录请求体，默认开启。仅记录 POST、PUT、PATCH 请求且 Content-Type 为支持的格式                        |
| log_response_headers | bool   | 选填     | true   | 是否记录响应头，默认开启                                                                            |
| log_response_body    | bool   | 选填     | true   | 是否记录响应体，默认开启。仅记录 Content-Type 为支持格式且没有 content-encoding 压缩的情况               |

## 支持的内容类型

插件仅记录以下 Content-Type 的请求/响应体：

- `application/x-www-form-urlencoded`
- `application/json`
- `text/plain`

## 注意事项

1. 请求/响应体大小超过 1KB 的部分会被截断并标记为 `<truncated>`
2. 带有 `content-encoding` 的响应体不会被记录（避免记录压缩内容）
3. 换行符会被转义为 `\n` 以确保日志可读性
4. 日志输出会包含完整的请求和响应信息，请注意敏感信息的保护
5. 该插件主要用于开发调试和 API 监控场景

## 配置示例

### 记录所有请求和响应信息

```yaml
log_request_headers: true
log_request_body: true
log_response_headers: true
log_response_body: true
```

### 仅记录请求和响应头

```yaml
log_request_headers: true
log_request_body: false
log_response_headers: true
log_response_body: false
```

### 仅记录响应信息

```yaml
log_request_headers: false
log_request_body: false
log_response_headers: true
log_response_body: true
```

## 日志输出示例

```
[info] request Headers: [:method=POST, :path=/api/test, content-type=application/json, user-agent=curl/7.68.0]
[info] request Body: [{"name": "test", "message": "hello\nworld"}]
[info] response Headers: [:status=200, content-type=application/json]
[info] response Body: [{"result": "success"}]
```
