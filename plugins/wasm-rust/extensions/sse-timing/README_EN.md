---
title: sse-timing
keywords: [sse, performance, timing]
description: 
---

## Description

An alternative implementation of the server-timing protocol (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Server-Timing),
It's useful to profile AI backend latency which uses event-stream MIME type.

## Runtime

phase：`UNSPECIFIED_PHASE`
priority：`10`

## Config

| Name   | Type   | Requirement | Default | Description                                     |
|--------|--------|-------------|---------|-------------------------------------------------|
| vendor | string | false       | higress | the proxy vendor that sse response pass through |

## 配置示例

```yaml
vendor: higress
```
