---
title: OPA
keywords: [higress,opa]
description: OPA 策略控制插件配置参考
---

## 功能说明

该插件实现了 `OPA` 策略控制

## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`225`

## 配置字段

| 字段            | 数据类型   | 填写要求 | 默认值 | 描述                                   |
|---------------|--------|------|-----|--------------------------------------|
| policy        | string | 必填   | -   | opa 策略                               |
| timeout       | string | 必填   | -   | 访问超时时间设置                             |
| serviceSource | string | 必填   | -   | k8s,nacos,ip,route                   |
| host          | string | 非必填  | -   | 服务主机（serviceSource为`ip`必填）           |
| serviceName   | string | 非必填  | -   | 服务名称（serviceSource为`k8s,nacos,ip`必填） |
| servicePort   | string | 非必填  | -   | 服务端口（serviceSource为`k8s,nacos,ip`必填） |
| namespace     | string | 非必填  | -   | 服务端口（serviceSource为`k8s,nacos`必填）    |

## 配置示例

```yaml
serviceSource: k8s
serviceName: opa
servicePort: 8181
namespace: higress-backend
policy: example1
timeout: 5s
```

## OPA 服务安装参考

### 启动 OPA 服务

```shell
docker run -d --name opa -p 8181:8181 openpolicyagent/opa:0.35.0 run -s
```

### 创建 OPA 策略

```shell
curl -X PUT '127.0.0.1:8181/v1/policies/example1' \
  -H 'Content-Type: text/plain' \
  -d 'package example1

import input.request

default allow = false

allow {
    # HTTP method must GET
    request.method == "GET"
}'
```

### 查询策略

```shell
curl -X POST '127.0.0.1:8181/v1/data/example1/allow' \
  -H 'Content-Type: application/json' \
  -d '{"input":{"request":{"method":"GET"}}}'
```
