## 简介
---
title: 通用响应缓存
keywords: [higress,response cache]
description: 通用响应缓存插件配置参考
---

## 功能说明

通用响应缓存插件，支持从请求头/请求体中提取key，从响应体中提取value并缓存起来；下次请求时，如果请求头/请求体中携带了相同的key，则直接返回缓存中的value，而不会请求后端服务。

**提示**

携带请求头`x-higress-skip-response-cache: on`时，当前请求将不会使用缓存中的内容，而是直接转发给后端服务，同时也不会缓存该请求返回响应的内容


## 运行属性

插件执行阶段：`认证阶段`
插件执行优先级：`10`

## 配置说明
配置包括 缓存数据库（cache）配置部分，以及配置缓存内容部分

## 配置说明

## 缓存服务（cache）
| cache.type | string | required | "" | 缓存服务类型，例如 redis |
| --- | --- | --- | --- | --- |
| cache.serviceName | string | required | "" | 缓存服务名称 |
| cache.serviceHost | string | required | "" | 缓存服务域名 |
| cache.servicePort | int64 | optional | 6379 | 缓存服务端口 |
| cache.username | string | optional | ""  | 缓存服务用户名 |
| cache.password | string | optional | "" | 缓存服务密码 |
| cache.timeout | uint32 | optional | 10000 | 缓存服务的超时时间，单位为毫秒。默认值是10000，即10秒 |
| cache.cacheTTL | int | optional | 0 | 缓存过期时间，单位为秒。默认值是 0，即 永不过期|
| cacheKeyPrefix | string | optional | "higress-response-cache:" | 缓存 Key 的前缀，默认值为 "higress-response-cache:" |


## 其他配置
| Name | Type | Requirement | Default | Description |
| --- | --- | --- | --- | --- |
| cacheResponseCode | array of number | optional | 200 | 表示支持缓存的响应状态码列表；默认为200|
| cacheKeyFromHeader | string | optional | "" | 表示提取header中的固定字段的值作为缓存key；配置此项时会从请求头提取key，不会读取请求body；cacheKeyFromHeader和cacheKeyFromBody**非空情况下只支持配置一项**|
| cacheKeyFromBody | string | optional | "" | 配置为空时，表示提取所有body作为缓存key；否则按JSON响应格式，从请求 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串；仅在cacheKeyFromHeader为空或未配置时生效 |
| cacheValueFromBodyType | string | optional | "application/json" | 表示缓存body的类型，命中cache时content-type会返回该值；默认为"application/json" |
| cacheValueFromBody | string | optional | "" | 配置为空时，表示缓存所有body；当cacheValueFromBodyType为"application/json"时，支持从响应 Body 中基于 [GJSON PATH](https://github.com/tidwall/gjson/blob/master/SYNTAX.md) 语法提取字符串 |

其中，缓存key的拼接逻辑为以下中一个： 
1. `cacheKeyPrefix` + 从请求头中`cacheKeyFromHeader`对应字段提取的内容
2. `cacheKeyPrefix` + 从请求体中`cacheKeyFromBody`对应字段提取的内容

**注意**：`cacheKeyFromHeader` 和 `cacheKeyFromBody` 不能同时配置（非空情况下只支持配置一项）。如果同时配置，插件在配置解析阶段会报错。


命中缓存插件的情况下，返回的响应头中有三种状态：
- `x-cache-status: hit` ，表示命中缓存，直接返回缓存内容
- `x-cache-status: miss` ，表示未命中缓存，返回后端响应结果
- `x-cache-status: skip` ，表示跳过缓存检查


## 配置示例
### 基础配置
```yaml
cache:
  type: redis
  serviceName: my-redis.dns
  servicePort: 6379
  timeout: 2000

cacheKeyFromHeader: "x-http-cache-key"

cacheValueFromBodyType: "application/json"
cacheValueFromBody: "messages.@reverse.0.content"
```

假设请求为

```bash
# Request
curl -H "x-http-cache-key: abcd" <url>

# Response
{"messages":[{"content":"1"}, {"content":"2"}, {"content":"3"}]}
```

则缓存的key为`higress-response-cache:abcd`，缓存的value为`3`。

后续请求命中缓存时，响应Content-type返回为 `application/json`。


### 响应body作为value

如果缓存所有响应body，则可以配置为

```yaml
cacheValueFromBodyType: "text/html"
cacheValueFromBody: ""

```

后续请求命中缓存时，响应Content-type返回为 `text/html`。

### 请求body作为key

使用请求body作为key，则可以配置为

```yaml
cacheKeyFromBody: ""
```

配置支持GJSON PATH语法。

## 进阶用法
Body为`application/json`时，支持基于 GJSON PATH 语法：

比如表达式：`messages.@reverse.0.content` ，含义是把 messages 数组反转后取第一项的 content；

GJSON PATH 也支持条件判断语法，例如希望取最后一个 role 为 user 的 content 作为 key，可以写成： `messages.@reverse.#(role=="user").content`；

如果希望将所有 role 为 user 的 content 拼成一个数组作为 key，可以写成：`messages.@reverse.#(role=="user")#.content`；

还可以支持管道语法，例如希望取到数第二个 role 为 user 的 content 作为 key，可以写成：`messages.@reverse.#(role=="user")#.content|1`。

更多用法可以参考[官方文档](https://github.com/tidwall/gjson/blob/master/SYNTAX.md)，可以使用 [GJSON Playground](https://gjson.dev/) 进行语法测试。

## 常见问题

1. 如果返回的错误为 `error status returned by host: bad argument`，请检查：
   - `serviceName`是否正确包含了服务的类型后缀(.dns等)
   - `servicePort`配置是否正确，尤其是 `static` 类型的服务端口现在固定为 80
