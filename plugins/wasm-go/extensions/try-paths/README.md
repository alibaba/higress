---
title: 请求重试
keywords: [higress,try-paths]
description: 请求重试
---

# 功能说明
`try-paths`插件支持请求基于不同的路径进行重试，直到请求到正确返回的请求，功能类似nginx的try files指令。

# 配置字段

| 名称 | 数据类型 | 填写要求 |  默认值 | 描述 |
| -------- | -------- | -------- | -------- | -------- |
| serviceSource  | string           | 必填    | -          | 支持k8s,nacos,ip,dns                                 |
| domain         | string           | 非必填  | -          | 服务主机（serviceSource为`dns`必填）                 |
| host         | string           | 非必填  | -            | 访问的域名地址(serviceSource为`k8s,nacos,ip`填写有效) |
| serviceName    | string           | 非必填  | -          | 服务名称（serviceSource为`k8s,nacos,ip,dns`必填）    |
| servicePort    | string           | 非必填  | -          | 服务端口（serviceSource为`k8s,nacos,ip,dns`必填）    |
| namespace      | string           | 非必填  | -          | 服务命名空间（serviceSource为`k8s,nacos`必填）        |
| tryPaths       | array of string  | 必填    | -          | 重试路径，比如`index.html`，`$uri`, `index.html`     |
| tryCodes       | array of int     | 非必填  | [403, 404] | 重试状态码，可自定义                                  |
| timeout        | int              | 非必填  | 1000       | 重试请求的超时时间，单位ms                             |


# 配置示例

## 配置了try-paths插件的场景

```yaml
namespace: "default"
serviceName: "oss"
servicePort: 80
serviceSource: "k8s"
host: "<bucket name>.oss-cn-hangzhou.aliyuncs.com"
tryPaths:
- "$uri/"
- "$uri.html"
- "/index.html"

```

基于该配置开启插件，触发插件的请求curl "http://a.com/a", 会依次请求
- http://<bucket name>.oss-cn-hangzhou.aliyuncs.com/a/
- http://<bucket name>.oss-cn-hangzhou.aliyuncs.com/a.html
- http://<bucket name>.oss-cn-hangzhou.aliyuncs.com/index.html
如果请求返回码不是重试状态码，会直接返回该请求体，否则继续重试下一个请求，所有请求都不是重试状态码，会继续请求默认后端服务。
