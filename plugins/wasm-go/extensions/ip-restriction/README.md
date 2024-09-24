---
title: IP 访问限制
keywords: [higress, ip restriction]
description: IP 访问限制插件配置参考
---

## 功能说明

`ip-restriction `插件可以通过将 IP 地址列入白名单或黑名单来限制对服务或路由的访问.支持对单个 IP 地址、多个 IP 地址和类似
10.10.10.0/24 的 CIDR范围的限制.

## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`210`


## 配置说明

| 配置项            | 类型     | 必填 | 默认值                         | 说明                                       |
|----------------|--------|----|-----------------------------|------------------------------------------|
| ip_source_type | string | 否  | origin-source               | 可选值：1. 对端socket ip：`origin-source`; 2. 通过header获取：`header` |
| ip_header_name | string | 否  | x-forwarded-for             | 当`ip_source_type`为`header`时，指定自定义IP来源头                                 |
| allow          | array  | 否  | []                          | 白名单列表                                    |
| deny           | array  | 否  | []                          | 黑名单列表                                    |
| status         | int    | 否  | 403                         | 拒绝访问时的 HTTP 状态码                          |
| message        | string | 否  | Your IP address is blocked. | 拒绝访问时的返回信息                               |


```yaml
ip_source_type: origin-source
allow:
  - 10.0.0.1
  - 192.168.0.0/16
```

```yaml
ip_source_type: header
ip_header_name: x-real-iP
deny:
  - 10.0.0.1
  - 192.169.0.0/16   
```
