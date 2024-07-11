---
title: 基于 Key 集群限流
keywords: [higress, rate-limit]
description: 基于 Key 集群限流插件配置参考
---

## 功能说明

`cluster-key-rate-limit` 插件基于 Redis 实现集群限流，适用于需要跨多个 Higress Gateway 实例实现全局一致速率限制的场景。
限流所使用的 Key 可以来源于 URL 参数、HTTP 请求头、客户端 IP 地址、消费者名称或 Cookie 中的 Key。

## 配置说明

| 配置项                  | 类型   | 必填 | 默认值 | 说明                                                                          |
| ----------------------- | ------ | ---- | ------ |-----------------------------------------------------------------------------|
| rule_name               | string | 是 | - | 限流规则名称，根据限流规则名称 + 限流类型 + 限流 key 名称 + 限流 key 对应的实际值来拼装 redis key             |
| rule_items | array of object | 是   | -                 | 限流规则项，按照 rule_items 下的排列顺序，匹配第一个 rule_item 后命中限流规则，后续规则将被忽略                 |
| show_limit_quota_header | bool | 否 | false | 响应头中是否显示 `X-RateLimit-Limit`（限制的总请求数）和 `X-RateLimit-Remaining`（剩余还可以发送的请求数） |
| rejected_code           | int | 否 | 429 | 请求被限流时，返回的 HTTP 状态码                                                         |
| rejected_msg            | string | 否 | Too many requests | 请求被限流时，返回的响应体                                                               |
| redis                   | object          | 是                                                           | -                 | redis 相关配置                                                                  |

`rule_items` 中每一项的配置字段说明。

| 配置项                | 类型            | 必填                   | 默认值 | 说明                                                                                                                                                       |
| --------------------- | --------------- |----------------------| ------ |----------------------------------------------------------------------------------------------------------------------------------------------------------|
| limit_by_header       | string          | 否，`limit_by_*` 中选填一项 | -      | 配置获取限流键值的来源 HTTP 请求头名称                                                                                                                                   |
| limit_by_param        | string          | 否，`limit_by_*` 中选填一项 | -      | 配置获取限流键值的来源 URL 参数名称                                                                                                                                     |
| limit_by_consumer     | string          | 否，`limit_by_*` 中选填一项 | -      | 根据 consumer 名称进行限流，无需添加实际值                                                                                                                               |
| limit_by_cookie       | string          | 否，`limit_by_*` 中选填一项 | -      | 配置获取限流键值的来源 Cookie中 key 名称                                                                                                                               |
| limit_by_per_header   | string          | 否，`limit_by_*` 中选填一项 | -      | 按规则匹配特定 HTTP 请求头，并对每个请求头分别计算限流，配置获取限流键值的来源 HTTP 请求头名称，配置 `limit_keys` 时支持正则表达式或 `*`                                                                      |
| limit_by_per_param    | string          | 否，`limit_by_*` 中选填一项 | -      | 按规则匹配特定 URL 参数，并对每个参数分别计算限流，配置获取限流键值的来源 URL 参数名称，配置 `limit_keys` 时支持正则表达式或 `*`                                                                           |
| limit_by_per_consumer | string          | 否，`limit_by_*` 中选填一项 | -      | 按规则匹配特定 consumer，并对每个 consumer 分别计算限流，根据 consumer 名称进行限流，无需添加实际值，配置 `limit_keys` 时支持正则表达式或 `*`                                                           |
| limit_by_per_cookie   | string          | 否，`limit_by_*` 中选填一项 | -      | 按规则匹配特定 Cookie，并对每个 Cookie 分别计算限流，配置获取限流键值的来源 Cookie中 key 名称，配置 `limit_keys` 时支持正则表达式或 `*`                                                               |
| limit_by_per_ip       | string          | 否，`limit_by_*` 中选填一项 | -      | 按规则匹配特定 IP，并对每个 IP 分别计算限流，配置获取限流键值的来源 IP 参数名称，从请求头获取，以 `from-header-对应的header名`，示例：`from-header-x-forwarded-for`，直接获取对端 socket ip，配置为 `from-remote-addr` |
| limit_keys            | array of object | 是                    | -      | 配置匹配键值后的限流次数                                                                                                                                             |

`limit_keys` 中每一项的配置字段说明。

| 配置项           | 类型   | 必填                                                         | 默认值 | 说明                                                         |
| ---------------- | ------ | ------------------------------------------------------------ | ------ | ------------------------------------------------------------ |
| key              | string | 是                                                           | -      | 匹配的键值，`limit_by_per_header`,`limit_by_per_param`,`limit_by_per_consumer`,`limit_by_per_cookie` 类型支持配置正则表达式（以regexp:开头后面跟正则表达式）或者*（代表所有），正则表达式示例：`regexp:^d.*`（以d开头的所有字符串）；`limit_by_per_ip`支持配置 IP 地址或 IP 段 |
| query_per_second | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每秒请求次数                                             |
| query_per_minute | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每分钟请求次数                                           |
| query_per_hour   | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每小时请求次数                                           |
| query_per_day    | int    | 否，`query_per_second`,`query_per_minute`,`query_per_hour`,`query_per_day` 中选填一项 | -      | 允许每天请求次数                                             |

`redis` 中每一项的配置字段说明。

| 配置项       | 类型   | 必填 | 默认值                                                     | 说明                                                                        |
| ------------ | ------ | ---- | ---------------------------------------------------------- |---------------------------------------------------------------------------|
| service_name | string | 必填 | -                                                          | redis 服务名称，带服务类型的完整 FQDN 名称，例如 my-redis.dns、redis.my-ns.svc.cluster.local |
| service_port | int    | 否   | 服务类型为固定地址（static service）默认值为80，其他为6379 | 输入redis服务的服务端口                                                            |
| username     | string | 否   | -                                                          | redis 用户名                                                                 |
| password     | string | 否   | -                                                          | redis 密码                                                                  |
| timeout      | int    | 否   | 1000                                                       | redis 连接超时时间，单位毫秒                                                         |

## 配置示例

### 识别请求参数 apikey，进行区别限流

```yaml
rule_name: default_rule
rule_items:
- limit_by_param: apikey
  limit_keys:
  - key: 9a342114-ba8a-11ec-b1bf-00163e1250b5
    query_per_minute: 10
  - key: a6a6d7f2-ba8a-11ec-bec2-00163e1250b5
    query_per_hour: 100
- limit_by_per_param: apikey
  limit_keys:
  # 正则表达式，匹配以 a 开头的所有字符串，每个 apikey 对应的请求 10qds
  - key: "regexp:^a.*"
    query_per_second: 10
  # 正则表达式，匹配以 b 开头的所有字符串，每个 apikey 对应的请求 100qd
  - key: "regexp:^b.*"
    query_per_minute: 100
  # 兜底用，匹配所有请求，每个 apikey 对应的请求 1000qdh
  - key: "*"
    query_per_hour: 1000
redis:
  service_name: redis.static
show_limit_quota_header: true
```

### 识别请求头 x-ca-key，进行区别限流

```yaml
rule_name: default_rule
rule_items:
- limit_by_header: x-ca-key
  limit_keys:
  - key: 102234
    query_per_minute: 10
  - key: 308239
    query_per_hour: 10
- limit_by_per_header: x-ca-key
  limit_keys:
  # 正则表达式，匹配以 a 开头的所有字符串，每个 apikey 对应的请求 10qds
  - key: "regexp:^a.*"
    query_per_second: 10
  # 正则表达式，匹配以b开头的所有字符串，每个 apikey 对应的请求 100qd
  - key: "regexp:^b.*"
    query_per_minute: 100
  # 兜底用，匹配所有请求，每个 apikey 对应的请求 1000qdh
  - key: "*"
    query_per_hour: 1000            
redis:
  service_name: redis.static
show_limit_quota_header: true
```

### 根据请求头 x-forwarded-for 获取对端 IP，进行区别限流

```yaml
rule_name: default_rule
rule_items:
- limit_by_per_ip: from-header-x-forwarded-for
  limit_keys:
  # 精确 IP
  - key: 1.1.1.1
    query_per_day: 10
  # IP 段，符合这个 IP 段的 IP，每个 IP 100qpd
  - key: 1.1.1.0/24
    query_per_day: 100
  # 兜底用，即默认每个 IP 1000 qpd
  - key: 0.0.0.0/0
    query_per_day: 1000
redis:
  service_name: redis.static
show_limit_quota_header: true
```

### 识别 consumer，进行区别限流

```yaml
rule_name: default_rule
rule_items:
- limit_by_consumer: ''
  limit_keys:
  - key: consumer1
    query_per_second: 10
  - key: consumer2
    query_per_hour: 100
- limit_by_per_consumer: ''
  limit_keys:
  # 正则表达式，匹配以 a 开头的所有字符串，每个 consumer 对应的请求 10qds
  - key: "regexp:^a.*"
    query_per_second: 10
  # 正则表达式，匹配以 b 开头的所有字符串，每个 consumer 对应的请求 100qd
  - key: "regexp:^b.*"
    query_per_minute: 100
  # 兜底用，匹配所有请求，每个 consumer 对应的请求 1000qdh
  - key: "*"
    query_per_hour: 1000     
redis:
  service_name: redis.static
show_limit_quota_header: true 
```

### 识别 Cookie 中的键值对，进行区别限流

```yaml
rule_name: default_rule
rule_items:
  - limit_by_cookie: key1
    limit_keys:
      - key: value1
        query_per_minute: 10
      - key: value2
        query_per_hour: 100
  - limit_by_per_cookie: key1
    limit_keys:
      # 正则表达式，匹配以 a 开头的所有字符串，每个 cookie 中的 value 对应的请求 10qds
      - key: "regexp:^a.*"
        query_per_second: 10
      # 正则表达式，匹配以 b 开头的所有字符串，每个 cookie 中的 value 对应的请求 100qd
      - key: "regexp:^b.*"
        query_per_minute: 100
      # 兜底用，匹配所有请求，每个 cookie 中的 value 对应的请求 1000qdh
      - key: "*"
        query_per_hour: 1000
rejected_code: 200
rejected_msg: '{"code":-1,"msg":"Too many requests"}'
redis:
  service_name: redis.static
show_limit_quota_header: true
```
