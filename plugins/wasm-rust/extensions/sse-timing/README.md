---
title: sse耗时追踪
keywords: [sse, 性能, 耗时]
description: 
---

## 功能说明

An alternative implementation of the server-timing protocol (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Server-Timing),
It's useful to profile AI backend latency which uses event-stream MIME type.

## 运行属性

stage：`默认阶段`
level：`10`

### 配置说明

| Name   | Type   | Requirement | Default | Description |
|--------|--------|-------------|---------|-------------|
| vendor | string | false       | higress | sse响应经过的代理  |

#### 配置示例

```yaml
vendor: higress
```
